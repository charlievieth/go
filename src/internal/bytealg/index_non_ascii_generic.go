// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !amd64 && !arm64

package bytealg

// IndexNonASCII returns the index of first non-ASCII byte in p,
// or -1 if p consists only of ASCII characters.
func IndexNonASCII(p []byte) int {
	// This optimization avoids the need to recompute the capacity
	// when generating code for p[8:], bringing it to parity with
	// ValidString, which was 20% faster on long ASCII strings.
	p = p[:len(p):len(p)]

	n := len(p)
	for len(p) >= 8 {
		// Combining two 32 bit loads allows the same code to be used
		// for 32 and 64 bit platforms.
		// The compiler can generate a 32bit load for first32 and second32
		// on many platforms. See test/codegen/memcombine.go.
		first32 := uint32(p[0]) | uint32(p[1])<<8 | uint32(p[2])<<16 | uint32(p[3])<<24
		second32 := uint32(p[4]) | uint32(p[5])<<8 | uint32(p[6])<<16 | uint32(p[7])<<24
		if (first32|second32)&0x80808080 != 0 {
			// Found a non ASCII byte (>= RuneSelf).
			break
		}
		p = p[8:]
	}
	for i := 0; i < len(p); i++ {
		if p[i] >= 0x80 {
			return n - len(p) + i
		}
	}
	return -1
}

// IndexNonASCIIString returns the index of first non-ASCII byte in s,
// or -1 if s consists only of ASCII characters.
func IndexNonASCIIString(s string) int {
	n := len(s)
	for len(s) >= 8 {
		// See above comment in IndexNonASCII.
		first32 := uint32(s[0]) | uint32(s[1])<<8 | uint32(s[2])<<16 | uint32(s[3])<<24
		second32 := uint32(s[4]) | uint32(s[5])<<8 | uint32(s[6])<<16 | uint32(s[7])<<24
		if (first32|second32)&0x80808080 != 0 {
			// Found a non ASCII byte (>= RuneSelf).
			break
		}
		s = s[8:]
	}
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return n - len(s) + i
		}
	}
	return -1
}
