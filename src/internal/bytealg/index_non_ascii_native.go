// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 || arm64

package bytealg

// IndexNonASCII returns the index of first non-ASCII byte in p,
// or -1 if p consists only of ASCII characters.
func IndexNonASCII(p []byte) int

// IndexNonASCIIString returns the index of first non-ASCII byte in s,
// or -1 if s consists only of ASCII characters.
func IndexNonASCIIString(s string) int
