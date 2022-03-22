// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package path_test

import (
	"path"
	. "path"
	"runtime"
	"testing"
)

type PathTest struct {
	path, result string
}

var cleantests = []PathTest{
	// Already clean
	{"", "."},
	{"abc", "abc"},
	{"abc/def", "abc/def"},
	{"a/b/c", "a/b/c"},
	{".", "."},
	{"..", ".."},
	{"../..", "../.."},
	{"../../abc", "../../abc"},
	{"/abc", "/abc"},
	{"/", "/"},

	// Remove trailing slash
	{"abc/", "abc"},
	{"abc/def/", "abc/def"},
	{"a/b/c/", "a/b/c"},
	{"./", "."},
	{"../", ".."},
	{"../../", "../.."},
	{"/abc/", "/abc"},
	{".//", "."},
	{"//", "/"},

	// Remove doubled slash
	{"abc//def//ghi", "abc/def/ghi"},
	{"//abc", "/abc"},
	{"///abc", "/abc"},
	{"//abc//", "/abc"},
	{"abc//", "abc"},

	// Remove . elements
	{"abc/./def", "abc/def"},
	{"/./abc/def", "/abc/def"},
	{"abc/.", "abc"},
	{"././a", "a"},
	{"a/././b", "a/b"},

	// Remove .. elements
	{"abc/def/ghi/../jkl", "abc/def/jkl"},
	{"abc/def/../ghi/../jkl", "abc/jkl"},
	{"abc/def/..", "abc"},
	{"abc/def/../..", "."},
	{"/abc/def/../..", "/"},
	{"abc/def/../../..", ".."},
	{"/abc/def/../../..", "/"},
	{"abc/def/../../../ghi/jkl/../../../mno", "../../mno"},
	{"/qux/../quux/foobar.com/baz", "/quux/foobar.com/baz"}, // TestRedirect
	{"/../", "/"},

	// Combinations
	{"abc/./../def", "def"},
	{"abc//./../def", "def"},
	{"abc/../../././../def", "../../def"},

	// Ignore "..."
	{"example.../b@upgrade", "example.../b@upgrade"},
}

func TestCleanWildcard(t *testing.T) {
	tests := []PathTest{
		{"example.../b@upgrade", "example.../b@upgrade"},
	}
	for _, test := range tests {
		if s := Clean(test.path); s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.path, s, test.result)
		}
		if s := Clean(test.result); s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.result, s, test.result)
		}
	}
}

func TestClean(t *testing.T) {
	for _, test := range cleantests {
		if s := Clean(test.path); s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.path, s, test.result)
		}
		if s := Clean(test.result); s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.result, s, test.result)
		}
	}
}

func TestCleanMallocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping malloc count in short mode")
	}
	if runtime.GOMAXPROCS(0) > 1 {
		t.Log("skipping AllocsPerRun checks; GOMAXPROCS>1")
		return
	}

	for _, test := range cleantests {
		allocs := testing.AllocsPerRun(100, func() { Clean(test.result) })
		if allocs > 0 {
			t.Errorf("Clean(%q): %v allocs, want zero", test.result, allocs)
		}
	}
}

type SplitTest struct {
	path, dir, file string
}

var splittests = []SplitTest{
	{"a/b", "a/", "b"},
	{"a/b/", "a/b/", ""},
	{"a/", "a/", ""},
	{"a", "", "a"},
	{"/", "/", ""},
}

func TestSplit(t *testing.T) {
	for _, test := range splittests {
		if d, f := Split(test.path); d != test.dir || f != test.file {
			t.Errorf("Split(%q) = %q, %q, want %q, %q", test.path, d, f, test.dir, test.file)
		}
	}
}

type JoinTest struct {
	elem []string
	path string
}

var jointests = []JoinTest{
	// zero parameters
	{[]string{}, ""},

	// one parameter
	{[]string{""}, ""},
	{[]string{"a"}, "a"},

	// two parameters
	{[]string{"a", "b"}, "a/b"},
	{[]string{"a", ""}, "a"},
	{[]string{"", "b"}, "b"},
	{[]string{"/", "a"}, "/a"},
	{[]string{"/", ""}, "/"},
	{[]string{"a/", "b"}, "a/b"},
	{[]string{"a/", ""}, "a"},
	{[]string{"", ""}, ""},

	{[]string{"dotname", "./", ".dot"}, "dotname/.dot"},
}

func TestJoin(t *testing.T) {
	for _, test := range jointests {
		if p := Join(test.elem...); p != test.path {
			t.Errorf("Join(%q) = %q, want %q", test.elem, p, test.path)
		}
	}
}

type ExtTest struct {
	path, ext string
}

var exttests = []ExtTest{
	{"path.go", ".go"},
	{"path.pb.go", ".go"},
	{"a.dir/b", ""},
	{"a.dir/b.go", ".go"},
	{"a.dir/", ""},
}

func TestExt(t *testing.T) {
	for _, test := range exttests {
		if x := Ext(test.path); x != test.ext {
			t.Errorf("Ext(%q) = %q, want %q", test.path, x, test.ext)
		}
	}
}

var basetests = []PathTest{
	// Already clean
	{"", "."},
	{".", "."},
	{"/.", "."},
	{"/", "/"},
	{"////", "/"},
	{"x/", "x"},
	{"abc", "abc"},
	{"abc/def", "def"},
	{"a/b/.x", ".x"},
	{"a/b/c.", "c."},
	{"a/b/c.x", "c.x"},
}

func TestBase(t *testing.T) {
	for _, test := range basetests {
		if s := Base(test.path); s != test.result {
			t.Errorf("Base(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

var dirtests = []PathTest{
	{"", "."},
	{".", "."},
	{"/.", "/"},
	{"/", "/"},
	{"////", "/"},
	{"/foo", "/"},
	{"x/", "x"},
	{"abc", "."},
	{"abc/def", "abc"},
	{"abc////def", "abc"},
	{"a/b/.x", "a/b"},
	{"a/b/c.", "a/b"},
	{"a/b/c.x", "a/b"},
}

func TestDir(t *testing.T) {
	for _, test := range dirtests {
		if s := Dir(test.path); s != test.result {
			t.Errorf("Dir(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

type IsAbsTest struct {
	path  string
	isAbs bool
}

var isAbsTests = []IsAbsTest{
	{"", false},
	{"/", true},
	{"/usr/bin/gcc", true},
	{"..", false},
	{"/a/../bb", true},
	{".", false},
	{"./", false},
	{"lala", false},
}

func TestIsAbs(t *testing.T) {
	for _, test := range isAbsTests {
		if r := IsAbs(test.path); r != test.isAbs {
			t.Errorf("IsAbs(%q) = %v, want %v", test.path, r, test.isAbs)
		}
	}
}

func BenchmarkClean(b *testing.B) {
	b.Run("Clean", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path.Clean("/home/user/go/src/golang.org/x/tools/gopls")
		}
	})
	b.Run("Dirty", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path.Clean("/home/user/go/src/../src/golang.org/x/tools/gopls/")
		}
	})
	b.Run("Mod", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path.Clean("golang.org/x/tools/gopls/...")
		}
	})
}
