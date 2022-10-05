// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package path implements utility routines for manipulating slash-separated
// paths.
//
// The path package should only be used for paths separated by forward
// slashes, such as the paths in URLs. This package does not deal with
// Windows paths with drive letters or backslashes; to manipulate
// operating system paths, use the path/filepath package.
package path

func cleanIndex(path string) int {
	// rooted := path[0] == '/'
	for i := 0; i < len(path)-1; i++ {
		// if path[i] == '/' && (path[i+1] == '/' || path[i+1] == '.') {
		switch {
		case path[i] == '/' && path[i+1] == '/':
			return i
		case path[i] == '.' && (path[i+1] == '.' || path[i+1] == '/'):
			if i > 0 {
				i--
			}
			for ; i > 0 && path[i] != '/'; i-- {
			}
			return i
		}
		// if path[i] == '/' && path[i+1] == '/' {
		// 	return i
		// }
		// if path[i] == '.' && (path[i+1] == '.' || path[i+1] == '/') {
		// 	// backtrack to the last '/'
		// 	// if !rooted {

		// 	// WARN WARN WARN WARN
		// 	// This is probably wrong
		// 	// WARN WARN WARN WARN

		// 	if i > 0 {
		// 		i--
		// 	}
		// 	for ; i > 0 && path[i] != '/'; i-- {
		// 	}
		// 	return i
		// }
	}
	return -1
}

// Clean returns the shortest path name equivalent to path
// by purely lexical processing. It applies the following rules
// iteratively until no further processing can be done:
//
//  1. Replace multiple slashes with a single slash.
//  2. Eliminate each . path name element (the current directory).
//  3. Eliminate each inner .. path name element (the parent directory)
//     along with the non-.. element that precedes it.
//  4. Eliminate .. elements that begin a rooted path:
//     that is, replace "/.." by "/" at the beginning of a path.
//
// The returned path ends in a slash only if it is the root "/".
//
// If the result of this process is an empty string, Clean
// returns the string ".".
//
// See also Rob Pike, “Lexical File Names in Plan 9 or
// Getting Dot-Dot Right,”
// https://9p.io/sys/doc/lexnames.html
func Clean(path string) string {
	// Remove leading "./" and any extra leading slashes (".//a" => "a").
	for len(path) >= 2 && path[:2] == "./" {
		path = path[2:]
		for path != "" && path[0] == '/' {
			path = path[1:]
		}
	}

	// WARN: can this be improved and still be fast?

	// CEV: we need to do this because cleanIndex does not detect trailing
	// slash dot ("/.") elements.

	// Remove any traling slashes.
	path = trimTrailingSlashes(path)

	// Remove any traling slashes and dot slashes ("./").
	for len(path) >= 2 && path[len(path)-2:] == "/." {
		path = path[:len(path)-2]
	}
	path = trimTrailingSlashes(path)

	if path == "" {
		return "."
	}

	// WARN WARN WARN: no longer accurate
	//
	// Invariants:
	//	reading from path; r is index of next byte to process.
	//	writing to buf; w is index of next byte to write.
	//	dotdot is index in buf where .. must stop, either because
	//		it is the leading slash or it is a leading ../../.. prefix.

	r := cleanIndex(path)
	if r == -1 {
		return path
	}
	// if r == -1 {
	// 	r = 0
	// }
	dotdot := 0
	rooted := path[0] == '/'
	if rooted {
		dotdot = 1
		if r == 0 {
			r = 1
		}
	}
	// if i != 0 {
	// 	r = i
	// }
	buf := []byte(path)
	buf = buf[:r]

	n := len(path)
	for r < n {
		switch {
		case path[r] == '/':
			// empty path element
			r++
		case path[r] == '.' && (r+1 == n || path[r+1] == '/'):
			// . element
			r++
		case path[r] == '.' && path[r+1] == '.' && (r+2 == n || path[r+2] == '/'):
			// .. element: remove to last /
			r += 2
			switch {
			case len(buf) > dotdot:
				i := len(buf) - 1
				for i > dotdot && buf[i] != '/' {
					i--
				}
				buf = buf[:i]
			case !rooted:
				// cannot backtrack, but not rooted, so append .. element.
				if len(buf) > 0 {
					buf = append(buf, '/')
				}
				buf = append(buf, '.')
				buf = append(buf, '.')
				dotdot = len(buf)
			}
		default:
			// real path element.
			// add slash if needed
			if rooted && len(buf) != 1 || !rooted && len(buf) != 0 {
				buf = append(buf, '/')
			}
			// copy element
			for ; r < n && path[r] != '/'; r++ {
				buf = append(buf, path[r])
			}
		}
	}

	if len(buf) == 0 {
		return "."
	}
	if string(buf) == path[:len(buf)] {
		return path[:len(buf)]
	}
	return string(buf)
}

// trimTrailingSlashes is strings.TrimRight(s[1:], "/") but we can't import
// strings. If s is entirely slashes, "/" is returned.
func trimTrailingSlashes(s string) string {
	i := len(s) - 1
	for i > 0 && s[i] == '/' {
		i--
	}
	return s[:i+1]
	// for i := len(s) - 1; i >= 0; i-- {
	// 	if s[i] != '/' {
	// 		return s[:i+1]
	// 	}
	// }
	// return "/"
}

// lastSlash(s) is strings.LastIndex(s, "/") but we can't import strings.
func lastSlash(s string) int {
	i := len(s) - 1
	for i >= 0 && s[i] != '/' {
		i--
	}
	return i
}

// Split splits path immediately following the final slash,
// separating it into a directory and file name component.
// If there is no slash in path, Split returns an empty dir and
// file set to path.
// The returned values have the property that path = dir+file.
func Split(path string) (dir, file string) {
	i := lastSlash(path)
	return path[:i+1], path[i+1:]
}

// Join joins any number of path elements into a single path,
// separating them with slashes. Empty elements are ignored.
// The result is Cleaned. However, if the argument list is
// empty or all its elements are empty, Join returns
// an empty string.
func Join(elem ...string) string {
	size := 0
	for _, e := range elem {
		size += len(e)
	}
	if size == 0 {
		return ""
	}
	buf := make([]byte, 0, size+len(elem)-1)
	for _, e := range elem {
		if len(buf) > 0 || e != "" {
			if len(buf) > 0 {
				buf = append(buf, '/')
			}
			buf = append(buf, e...)
		}
	}
	return Clean(string(buf))
}

// Ext returns the file name extension used by path.
// The extension is the suffix beginning at the final dot
// in the final slash-separated element of path;
// it is empty if there is no dot.
func Ext(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}

// Base returns the last element of path.
// Trailing slashes are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of slashes, Base returns "/".
func Base(path string) string {
	if path == "" {
		return "."
	}
	// Strip trailing slashes.
	for len(path) > 0 && path[len(path)-1] == '/' {
		path = path[0 : len(path)-1]
	}
	// Find the last element
	if i := lastSlash(path); i >= 0 {
		path = path[i+1:]
	}
	// If empty now, it had only slashes.
	if path == "" {
		return "/"
	}
	return path
}

// IsAbs reports whether the path is absolute.
func IsAbs(path string) bool {
	return len(path) > 0 && path[0] == '/'
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element using Split, the path is Cleaned and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of slashes followed by non-slash bytes, Dir
// returns a single slash. In any other case, the returned path does not end in a
// slash.
func Dir(path string) string {
	dir, _ := Split(path)
	return Clean(dir)
}
