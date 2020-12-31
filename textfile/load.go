package textfile

import (
	"fmt"
	"io"
	"os"

	"github.com/guiguan/caster"
	"github.com/npillmayer/cords"
)

/*
BSD 3-Clause License

Copyright (c) 2020–21, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// Some constants for fragement size defaults
const (
	twoKb     = 2048
	sixKb     = 6144
	tenKb     = 10240
	hundredKb = 1024000
	oneMb     = 1048576
)

// fileLeaf is a cord leaf type to hold a fragment of a text-file's content.
// fileLeaf implements interface cords.Leaf.
type fileLeaf struct {
	content string    // content fragment carries by this leaf // TODO []byte ?
	length  int64     // length of this fragment in bytes
	pos     int64     // initial start position of this fragment within the file
	next    *fileLeaf // textfile leafs are initially chained, later this is dropped
	tf      *textFile // reference to the file this segment is from, later this is dropped
}

// textFile represents a OS file which will be loaded as a cord.
type textFile struct {
	path      string         // file name
	info      os.FileInfo    // result from Stat(path)
	file      *os.File       // file handle
	cast      *caster.Caster // broadcaster for async file loading
	lastError error          // remember last I/O error
}

// Load reads a file, which must be a text file, and loads it as a cord.
// Clients may indicate an initial cursor position and a recommended fragment length.
// Both may be 0, letting Load use sensible defaults. An initialPos of -1 means
// opening the file at the end (i.e, reading the trailing fragment first).
//
// Loading of large files may be done asynchronously, but this is transparent to
// the client. The cord can be used right away and synchronisation will happen
// correctly in the background. Opening of the file is always done synchronously.
//
func Load(name string, initialPos int64, fragSize int64) (cords.Cord, error) {
	//
	tf, err := openFile(name)
	if err != nil {
		return cords.Cord{}, err
	}
	if initialPos > tf.info.Size() || initialPos < 0 {
		initialPos = tf.info.Size()
	}
	if fragSize <= 0 || fragSize > tenKb {
		if tf.info.Size() < 64 {
			fragSize = tf.info.Size()
		} else if tf.info.Size() < 1024 {
			fragSize = 64
		} else if tf.info.Size() < tenKb {
			fragSize = 256
		} else if tf.info.Size() < hundredKb {
			fragSize = 512
		} else if tf.info.Size() < oneMb {
			fragSize = twoKb
		} else {
			fragSize = sixKb
		}
	}
	cord := startLoadingFileAsync(tf, initialPos, fragSize)
	return cord, nil
}

// openFile opens an OS file and collect some useful information on it,
// checking for error conditions.
func openFile(name string) (*textFile, error) {
	fi, err := os.Stat(name)
	if err != nil {
		return nil, err
	} else if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("file is not a regular file")
	}
	file, err := os.Open(name) // just open for read access
	if err != nil {
		return nil, err
	}
	tf := &textFile{
		path: name,
		info: fi,
		file: file,
		cast: caster.New(nil), // we will broadcast messages when fragments are loaded
	}
	return tf, nil
}

func startLoadingFileAsync(tf *textFile, initialPos int64, fragSize int64) cords.Cord {
	//
	if initialPos > tf.info.Size() || fragSize > tf.info.Size() {
		panic("inconsistent setting of text file paramenters")
	}
	var cord cords.Cord // return value
	size := tf.info.Size()
	var next *fileLeaf
	// we do not allocate all the leafs as an array, because this would prevent single
	// leafs to be garbage collected if the client deletes fragments of text
	rightmost := size / fragSize * fragSize
	if rightmost == size && size > 0 {
		rightmost -= fragSize
	}
	// create an (initially empty) fileLeaf for every to-be fragment
	var last *fileLeaf                          // last leaf shall point to first leaf
	for k := rightmost; k >= 0; k -= fragSize { // iterate backwards
		leaf := &fileLeaf{
			length: min(fragSize, size-k),
			pos:    k,
			next:   next,
			tf:     tf,
		}
		if last == nil { // we want leafs to be linked to a cycle
			last = leaf
		}
		next = leaf
		last.next = leaf // will be overwritten until leaf = first leaf
	}
	return cord
}

// --- fileLeaf methods ------------------------------------------------------

func (fl *fileLeaf) String() string {
	if fl.isLoaded() {
		return fl.content
	}
	return fl.content
}

// We flag the load status of a leaf by setting the sign of the length.
// Negative values indicated the fragment is not loaded yet.
func (fl *fileLeaf) isLoaded() bool {
	return fl.length >= 0
}

func (fl *fileLeaf) load() {
	if fl.isLoaded() {
		return
	}
	// if loading results in an error (e.g., the file has been modified or removed,
	// which should not possibly happen, but nevertheless)
	// fl.Content() should return a sequence of Xs, together with an error message.
	// if err := fl.load(); err != nil {
	// 	return "*** ERROR: File has been spuriously modified ***"
	// }
	// fl.length = abs(fl.length) // positive length signals segment has been loaded
}

// --- File loading goroutine ------------------------------------------------

func loadAllFragments(tf *textFile, startLeaf *fileLeaf) {
	if startLeaf == nil {
		panic("load-fragments may not be called for a void cord")
	}
	//
	var fragChan chan *fileLeaf
	//
	leaf := startLeaf // leaf-links have to form a cycle
	for {             // let every leaf listen to the frag channel
		// TODO hook in leaf as subscriber to fragChan publisher
		// leaf will reset it's String() and Weight() functions
		//
		// first String() and Weight() will run into another channel and wait
		// (as subscribers), until the leaf is loaded and will iself signal
		// into the other channel. Resetting the member functions must be
		// guarded by a mutex. Overall mutex in tf ? Will this lead to a
		// congestion? Do not want one mtx per leaf.
		if leaf.next == startLeaf {
			break // when iterated over cycle of leafs, stop
		}
		leaf = leaf.next // continue iteration
	}
	//
	go func(ch <-chan *fileLeaf) {
		// start a broadcaster to wait for loaded fragments and publish their ids
		defer tf.cast.Close()
		// publish loaded fragments to all waiting leafs
		for m := range ch {
			tf.cast.Pub(m)
		}
	}(fragChan)
	//
	go func(ch chan<- *fileLeaf) {
		// iterate over leafs and load the fragment of text referenced by it
		leaf := startLeaf // leaf-links have to form a cycle
		for {
			buf := make([]byte, leaf.length, leaf.length)
			cnt, err := tf.file.ReadAt(buf, leaf.pos)
			if err != nil && err != io.EOF {
				tf.lastError = fmt.Errorf("Error loading text fragment: %w", err)
				// TODO: fill with error pattern:
				// https://gist.github.com/taylorza/df2f89d5f9ab3ffd06865062a4cf015d
			} else if int64(cnt) < leaf.length {
				tf.lastError = fmt.Errorf("Not all bytes loaded for text fragment")
				// TODO: fill with error pattern
			}
			leaf.content = string(buf) // will re-allocate, unfortuntely no way around this
			ch <- leaf                 // signal that this leaf is done loading
			next := leaf.next          // put aside link to next leaf
			leaf.tf = nil              // drop admin info
			leaf.next = nil            // drop admin info
			if next == startLeaf {
				break // when iterated over cycle of leafs, stop
			}
			leaf = next // continue iteration
		}
	}(fragChan)
}

// --- Helpers ---------------------------------------------------------------

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func abs(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}
