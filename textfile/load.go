package textfile

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

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

// --- Leaf nodes getting content from a file --------------------------------

// fileLeaf is a cord leaf type to hold a fragment of a text-file's content.
// fileLeaf implements interface cords.Leaf.
type fileLeaf struct {
	ext    atomic.Value // of type leafExt; will later hold the text fragment carried by this leaf
	length int64        // length of this fragment in bytes
}

// ext must be an atomic value to protect concurrent access during load
func (leaf *fileLeaf) Ext() leafExt {
	return leaf.ext.Load().(leafExt)
}

// ext must be an atomic value to protect concurrent access during load
func (leaf *fileLeaf) setExt(ext leafExt) {
	leaf.ext.Store(ext)
}

// String returns the text fragment carried by this leaf.
// If the text fragment has not been loaded yet, the call to `String` will block
// until the fragment is available.
//
// Fragments may not be aligned with grapheme- or even rune-boundaries.
// The ``string´´ should be considered rather a read-only byte slice.
func (leaf *fileLeaf) String() string {
	ext := leaf.Ext()
	if ext.isString() {
		return ext.asString()
	}
	m := ext.duringLoad().mx
	m.Lock() // wait until string fragment is loaded
	defer m.Unlock()
	ext = leaf.Ext() // should be swapped by loading goroutine by now
	if !ext.isString() {
		panic("string fragment of leaf not available after waiting for load")
	}
	return ext.asString()
}

// Weight returns the length of the text fragment (string). This call will return
// immediately, even if the text fragment is not yet loaded.
func (leaf *fileLeaf) Weight() uint64 {
	return uint64(leaf.length)
}

// To avoid wasting memory per leaf node (there may be lots of them) we simulate
// a conditional type (union), which will finally hold the string fragment of this
// leaf. Before and during load, however, we will hold administrative information
// necessary for the load. This `ext` information will be wrapped as an atomic.Value.
type leafExt interface {
	isString() bool
	asString() string
	duringLoad() *loadingLeafExt
}

type stringLeafExt string

func (ext stringLeafExt) isString() bool {
	return true
}

func (ext stringLeafExt) asString() string {
	return string(ext)
}

func (ext stringLeafExt) duringLoad() *loadingLeafExt {
	return nil
}

var _ leafExt = stringLeafExt("")

type loadingLeafExt struct {
	pos  int64       // initial start position of this fragment within the file
	next *fileLeaf   // textfile leafs are initially chained, later this is dropped
	tf   *textFile   // reference to the file this segment is from, later this is dropped
	mx   *sync.Mutex // mutex guards access to string content until it is loaded
}

func (ext *loadingLeafExt) isString() bool {
	return false
}

func (ext *loadingLeafExt) asString() string {
	return ""
}

func (ext *loadingLeafExt) duringLoad() *loadingLeafExt {
	return ext
}

var _ leafExt = &loadingLeafExt{}

// --- Open and read a textfile ----------------------------------------------

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
	cord := startLoadingFile(tf, initialPos, fragSize)
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

func startLoadingFile(tf *textFile, initialPos int64, fragSize int64) cords.Cord {
	//
	if initialPos > tf.info.Size() || fragSize > tf.info.Size() {
		panic("inconsistent setting of text file paramenters")
	}
	var cord cords.Cord // return value
	size := tf.info.Size()
	var next *fileLeaf
	// we do not allocate all the leafs as an array, because this would prevent single
	// leafs to be garbage collected as soon as the client deletes fragments of text
	rightmost := size / fragSize * fragSize
	if rightmost == size && size > 0 {
		rightmost -= fragSize
	}
	// create an (initially empty) fileLeaf for every to-be fragment
	var leaf, last, start *fileLeaf             // last leaf shall point to first leaf
	for k := rightmost; k >= 0; k -= fragSize { // iterate backwards
		leaf = &fileLeaf{
			length: min(fragSize, size-k), // negative value signals leaf not loaded yet
		}
		atomicExt := &loadingLeafExt{
			next: next,
			tf:   tf,
			pos:  k,
			mx:   &sync.Mutex{}, // must be a pointer to avoid later copying
		}
		atomicExt.mx.Lock() // lock leaf's String() method until loaded
		leaf.ext.Store(atomicExt)
		if last == nil { // we want leafs to be linked to a cycle
			last = leaf
		}
		if k <= initialPos && initialPos < k+leaf.length {
			start = leaf // start leaf contains initial position
		}
		next = leaf // predecessor leaf will set this leaf as successor (*next)
	}
	lastExt := last.Ext()
	lastExt.duringLoad().next = leaf // make it a cycle of leafs
	last.setExt(lastExt)
	loadAllFragmentsAsync(tf, start)
	return cord
}

// --- File loading goroutine ------------------------------------------------

type msg struct {
	leaf    *fileLeaf
	content []byte
}

func loadAllFragmentsAsync(tf *textFile, startLeaf *fileLeaf) {
	if startLeaf == nil {
		panic("load-fragments may not be called for a void cord")
	}
	//
	var fragChan chan msg // communication channel between one loader thread and many leafs
	leaf := startLeaf     // leaf-links have to form a cycle
	// start subscriber goroutines, one per leaf
	for { // let every leaf listen to the frag channel
		go func(leaf *fileLeaf, cast *caster.Caster) {
			ext := leaf.Ext().duringLoad()
			defer ext.mx.Unlock()     // locking has already been done during leaf creation
			ch, _ := cast.Sub(nil, 1) // subscribe to broadcast messages
			for loaded := range ch {  // wait for signals for loaded leafs
				if loaded.(msg).leaf == leaf { // yes, it's our turn
					s := string(loaded.(msg).content) // re-alloc loaded.content as string
					leaf.ext.Store(stringLeafExt(s))  // swap loading-ext for string-ext
					cast.Unsub(ch)                    // we do not want to listen any further
					break                             // this will unlock the leaf's mutex
				}
			} // ignore all other leaf messages
		}(leaf, tf.cast)
		if leaf.Ext().duringLoad().next == startLeaf {
			break // when iterated over cycle of leafs, stop
		}
		leaf = leaf.Ext().duringLoad().next // continue iteration
	}
	// start publisher goroutine
	go func(ch <-chan msg) {
		// start a broadcaster to wait for loaded fragments and publish their ids
		defer tf.cast.Close()
		// publish loaded fragments to all waiting leafs
		for m := range ch {
			tf.cast.Pub(m)
		}
	}(fragChan)
	startFileLoader(tf, startLeaf, fragChan)
}

func startFileLoader(tf *textFile, startLeaf *fileLeaf, ch chan<- msg) {
	go func(ch chan<- msg) {
		// iterate over leafs and load the fragment of text referenced by it
		leaf := startLeaf // leaf-links have to form a cycle
		for {
			ext := leaf.Ext().duringLoad()
			buf := make([]byte, leaf.length, leaf.length)
			cnt, err := tf.file.ReadAt(buf, ext.pos)
			if err != nil && err != io.EOF {
				tf.lastError = fmt.Errorf("Error loading text fragment: %w", err)
				// TODO: fill with error pattern:
				// https://gist.github.com/taylorza/df2f89d5f9ab3ffd06865062a4cf015d
			} else if int64(cnt) < leaf.length {
				tf.lastError = fmt.Errorf("Not all bytes loaded for text fragment")
				// TODO: fill with error pattern
			}
			m := msg{leaf: leaf, content: buf}
			next := ext.next // put aside link to next leaf // before channel write !
			ch <- m          // signal that this leaf is done loading
			if next == startLeaf {
				break // when iterated over cycle of leafs, stop
			}
			leaf = next // continue iteration
		}
	}(ch)
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
