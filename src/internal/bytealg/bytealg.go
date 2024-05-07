// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytealg

import (
	"internal/cpu"
	"unsafe"
)

// Offsets into internal/cpu records for use in assembly.
const (
	offsetX86HasSSE42  = unsafe.Offsetof(cpu.X86.HasSSE42)
	offsetX86HasAVX2   = unsafe.Offsetof(cpu.X86.HasAVX2)
	offsetX86HasPOPCNT = unsafe.Offsetof(cpu.X86.HasPOPCNT)

	offsetS390xHasVX = unsafe.Offsetof(cpu.S390X.HasVX)

	offsetPPC64HasPOWER9 = unsafe.Offsetof(cpu.PPC64.IsPOWER9)
)

// MaxLen is the maximum length of the string to be searched for (argument b) in Index.
// If MaxLen is not 0, make sure MaxLen >= 4.
var MaxLen int

// PrimeRK is the prime base used in Rabin-Karp algorithm.
const PrimeRK = 16777619

// HashStr returns the hash and the appropriate multiplicative
// factor for use in Rabin-Karp algorithm.
func HashStr[T string | []byte](sep T) (uint32, uint32) {
	hash := uint32(0)
	for i := 0; i < len(sep); i++ {
		hash = hash*PrimeRK + uint32(sep[i])
	}
	var pow, sq uint32 = 1, PrimeRK
	for i := len(sep); i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}

// HashStrRev returns the hash of the reverse of sep and the
// appropriate multiplicative factor for use in Rabin-Karp algorithm.
func HashStrRev[T string | []byte](sep T) (uint32, uint32) {
	hash := uint32(0)
	for i := len(sep) - 1; i >= 0; i-- {
		hash = hash*PrimeRK + uint32(sep[i])
	}
	var pow, sq uint32 = 1, PrimeRK
	for i := len(sep); i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}

// IndexRabinKarp uses the Rabin-Karp search algorithm to return the index of the
// first occurrence of sep in s, or -1 if not present.
func IndexRabinKarp[T string | []byte](s, sep T) int {
	// Rabin-Karp search
	hashss, pow := HashStr(sep)
	n := len(sep)
	var h uint32
	for i := 0; i < n; i++ {
		h = h*PrimeRK + uint32(s[i])
	}
	if h == hashss && string(s[:n]) == string(sep) {
		return 0
	}
	for i := n; i < len(s); {
		h *= PrimeRK
		h += uint32(s[i])
		h -= pow * uint32(s[i-n])
		i++
		if h == hashss && string(s[i-n:i]) == string(sep) {
			return i - n
		}
	}
	return -1
}

// LastIndexRabinKarp uses the Rabin-Karp search algorithm to return the last index of the
// occurrence of sep in s, or -1 if not present.
func LastIndexRabinKarp[T string | []byte](s, sep T) int {
	// Rabin-Karp search from the end of the string
	hashss, pow := HashStrRev(sep)
	n := len(sep)
	last := len(s) - n
	var h uint32
	for i := len(s) - 1; i >= last; i-- {
		h = h*PrimeRK + uint32(s[i])
	}
	if h == hashss && string(s[last:]) == string(sep) {
		return last
	}
	for i := last - 1; i >= 0; i-- {
		h *= PrimeRK
		h += uint32(s[i])
		h -= pow * uint32(s[i+n])
		if h == hashss && string(s[i:i+n]) == string(sep) {
			return i
		}
	}
	return -1
}

func criticalFactorization[T string | []byte](s T) (maxSuffix, period int) {
	ms := -1 // max suffix
	p := 1   // period
	// uint here is required for BCE
	for j, k := 0, 1; uint(j+k) < uint(len(s)); {
		a := s[uint(j+k)]
		b := s[ms+k]
		if a < b {
			j += k
			k = 1
			p = j - ms
		} else if a == b {
			if k != p {
				k++
			} else {
				j += p
				k = 1
			}
		} else {
			ms = j
			j++
			k = 1
			p = 1
		}
	}
	p0 := p

	msr := -1 // max suffix reverse
	p = 1
	// uint here is required for BCE
	for j, k := 0, 1; uint(j+k) < uint(len(s)); {
		a := s[uint(j+k)]
		b := s[msr+k]
		if a > b {
			j += k
			k = 1
			p = j - msr
		} else if a == b {
			if k != p {
				k++
			} else {
				j += p
				k = 1
			}
		} else {
			msr = j
			j++
			k = 1
			p = 1
		}
	}
	if msr < ms {
		return ms + 1, p0
	}
	return msr + 1, p
}

func TwoWayLongNeedle[T string | []byte](s, substr T) int {
	suffix, period := criticalFactorization(substr)

	n := len(substr)
	var shiftTable [1 << 8]int // 256
	for i := 0; i < 1<<8; i++ {
		shiftTable[i] = n
	}
	for i := 0; i < len(substr); i++ {
		// WARN: reduce conversions
		// shiftTable[substr[i]] = uint(len(substr)) - uint(i) - 1
		shiftTable[substr[i]] = len(substr) - i - 1
	}

	if string(substr[:suffix]) == string(substr[period:period+suffix]) {
		// Entire needle is periodic; a mismatch can only advance by the
		// period, so use memory to avoid rescanning known occurrences
		// of the period.

		memory := 0
		for j := 0; j <= len(s)-len(substr); {
			shift := shiftTable[s[j+len(substr)-1]]
			if 0 < shift {
				if memory != 0 && shift < period {
					shift = len(substr) - period
				}
				memory = 0
				j += shift
				continue
			}

			// Scan for matches in right half.  The last byte has
			// already been matched, by virtue of the shift table.

			// TODO: bench the difference here
			i := max(suffix, memory)
			for ; i < len(substr)-1 && substr[i] == s[i+j]; i++ {
			}
			// o := i + j // TODO: remove "o"
			// for i < len(substr)-1 && substr[i] == s[o] {
			// 	i++
			// 	o++
			// }

			if len(substr)-1 <= i {
				// Scan for matches in left half.
				i = suffix - 1
				// pneedle := needle[i:]
				// phaystack := haystack[i+j:]
				o := i + j
				// WARN: make sure this is correct
				for memory < i+1 && substr[i] == s[o] {
					i--
					o--
				}
				// if i+1 < memory+1 { // WARN: dependent on uint rollover !!!
				if i < memory {
					return j
				}
				// No match, so remember how many repetitions of period
				// on the right half were scanned.
				j += period
				memory = len(substr) - period // WARN: conversion
			} else {
				j += i - suffix + 1
				memory = 0
			}
		}
	} else {
		// The two halves of needle are distinct; no extra memory is
		// required, and any mismatch results in a maximal shift.
		period = max(suffix, len(substr)-suffix) + 1 // WARN: conversion

		for j := 0; j <= len(s)-len(substr); {
			// Check the last byte first; if it does not match, then
			// shift to the next possible match location.
			shift := shiftTable[s[j+len(substr)-1]]
			if 0 < shift {
				j += shift
				continue
			}

			// Scan for matches in right half.  The last byte has
			// already been matched, by virtue of the shift table.
			//
			// TODO: consider using a slice of s[j:]
			i := suffix
			for ; i < len(substr)-1 && substr[i] == s[i+j]; i++ {
			}
			if i >= len(substr)-1 {
				// Scan for matches in left half.
				i := suffix - 1
				for ; i >= 0 && substr[i] == s[i+j]; i-- {
				}
				if i < 0 {
					return j
				}
				j += period
			} else {
				j += i - suffix + 1
			}
		}
	}
	return -1
}

// MakeNoZero makes a slice of length n and capacity of at least n Bytes
// without zeroing the bytes (including the bytes between len and cap).
// It is the caller's responsibility to ensure uninitialized bytes
// do not leak to the end user.
func MakeNoZero(n int) []byte
