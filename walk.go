// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fspath implements the Walk function from filepath, but
// operating over a http.FileSystem. The implementation is almost
// entirely taken from filepath as of go1.12
package fspath

import (
	"errors"
	"net/http"
	"os"
	filepath "path" // an http.FileSystem always uses forward slashes as the path separator
	"sort"
)

// SkipDir is used as a return value from WalkFuncs to indicate that
// the directory named in the call is to be skipped. It is not returned
// as an error by any function.
var SkipDir = errors.New("skip this directory")

// WalkFunc is the type of the function called for each file or directory
// visited by Walk. The path argument contains the argument to Walk as a
// prefix; that is, if Walk is called with "dir", which is a directory
// containing the file "a", the walk function will be called with argument
// "dir/a". The info argument is the os.FileInfo for the named path.
//
// If there was a problem walking to the file or directory named by path, the
// incoming error will describe the problem and the function can decide how
// to handle that error (and Walk will not descend into that directory). In the
// case of an error, the info argument will be nil. If an error is returned,
// processing stops. The sole exception is when the function returns the special
// value SkipDir. If the function returns SkipDir when invoked on a directory,
// Walk skips the directory's contents entirely. If the function returns SkipDir
// when invoked on a non-directory file, Walk skips the remaining files in the
// containing directory.
type WalkFunc func(path string, info os.FileInfo, err error) error

// walk recursively descends path, calling walkFn.
func walk(fs http.FileSystem, path string, info os.FileInfo, walkFn WalkFunc) error {
	if !info.IsDir() {
		return walkFn(path, info, nil)
	}
	names, err := readDirNames(fs, path)
	err1 := walkFn(path, info, err)
	// If err != nil, walk can't walk into this directory.
	// err1 != nil means walkFn want walk to skip this directory or stop walking.
	// Therefore, if one of err and err1 isn't nil, walk will return.
	if err != nil || err1 != nil {
		// The caller's behavior is controlled by the return value, which is decided
		// by walkFn. walkFn may ignore err and return nil.
		// If walkFn returns SkipDir, it will be handled by the caller.
		// So walk should return whatever walkFn returns.
		return err1
	}
	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, err := fsstat(fs, filename)
		if err != nil {
			if err := walkFn(filename, fileInfo, err); err != nil && err != SkipDir {
				return err
			}
		} else {
			err = walk(fs, filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order.
func Walk(fs http.FileSystem, root string, walkFn WalkFunc) error {
	f, err := fs.Open(root)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		err = walkFn(root, nil, err)
	} else {
		err = walk(fs, root, info, walkFn)
	}
	if err == SkipDir {
		return nil
	}
	return err
}

func fsstat(fs http.FileSystem, path string) (os.FileInfo, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	return f.Stat()
}

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(fs http.FileSystem, dirname string) ([]string, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	infos, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Name())
	}
	sort.Strings(names)
	return names, nil
}
