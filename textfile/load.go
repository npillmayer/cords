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
func (leaf *fileLeaf) Ext() *leafExt {
	return leaf.ext.Load().(*leafExt)
}

// ext must be an atomic value to protect concurrent access during load
func (leaf *fileLeaf) setExt(ext *leafExt) {
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
	if ext.loadExt == nil {
		return ext.content
	}
	// if ext.isString() {
	// 	return ext.asString()
	// }
	m := ext.loadExt.mx
	pos := ext.loadExt.pos
	T().Debugf("content %d…%d not yet loaded, waiting", pos, pos+leaf.length)
	m.Lock() // wait until string fragment is loaded
	T().Debugf("content %d…%d should now be loaded", pos, pos+leaf.length)
	defer m.Unlock()
	ext = leaf.Ext() // should be swapped by loading goroutine by now
	if ext.loadExt != nil {
		panic("string fragment of leaf not available after waiting for load")
	}
	// if !ext.isString() {
	// 	panic("string fragment of leaf not available after waiting for load")
	// }
	// return ext.asString()
	return ext.content
}

// Weight returns the length of the text fragment (string). This call will return
// immediately, even if the text fragment is not yet loaded.
func (leaf *fileLeaf) Weight() uint64 {
	return uint64(leaf.length)
}

// Split splits a leaf at position i, resulting in 2 new leafs.
func (leaf *fileLeaf) Split(i uint64) (cords.Leaf, cords.Leaf) {
	s := leaf.String() // may block until loaded
	left := s[:i]
	right := s[i:]
	return cords.StringLeaf(left), cords.StringLeaf(right)
}

// Substring returns a string segment of the leaf's text fragment.
func (leaf *fileLeaf) Substring(i uint64, j uint64) string {
	s := leaf.String() // may block until loaded
	return s[i:j]
}

// To avoid wasting memory per leaf node (there may be lots of them) we simulate
// a conditional type (union), which will finally hold the string fragment of this
// leaf. Before and during load, however, we will hold administrative information
// necessary for the load. This `ext` information will be wrapped as an atomic.Value.
type leafExt struct {
	content string
	loadExt *loadingInfo
	// isString() bool
	// asString() string
	// duringLoad() *loadingInfo
}

// type stringLeafExt string

// func (ext stringLeafExt) isString() bool {
// 	return true
// }

// func (ext stringLeafExt) asString() string {
// 	return string(ext)
// }

// func (ext stringLeafExt) duringLoad() *loadingInfo {
// 	return nil
// }

//var _ leafExt = stringLeafExt("")

type loadingInfo struct {
	pos  int64       // initial start position of this fragment within the file
	next *fileLeaf   // textfile leafs are initially chained, later this is dropped
	tf   *textFile   // reference to the file this segment is from, later this is dropped
	mx   *sync.Mutex // mutex guards access to string content until it is loaded
}

// func (ext *loadingInfo) isString() bool {
// 	return false
// }

// func (ext *loadingInfo) asString() string {
// 	return ""
// }

// func (ext *loadingInfo) duringLoad() *loadingInfo {
// 	return ext
// }

// var _ leafExt = &loadingInfo{}

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
func Load(name string, initialPos int64, fragSize int64, wg *sync.WaitGroup) (cords.Cord, *fileLeaf, error) {
	tf, err := openFile(name)
	if err != nil {
		return cords.Cord{}, nil, err
	}
	T().Infof("opened file %s", tf.info.Name())
	if initialPos > tf.info.Size() || initialPos < 0 {
		initialPos = tf.info.Size()
	}
	if fragSize <= 0 || fragSize > tenKb {
		if tf.info.Size() < 64 {
			fragSize = tf.info.Size()
		} else if tf.info.Size() < 1024 {
			fragSize = 64
		} else if tf.info.Size() < tenKb {
			fragSize = 1024 // 256 TODO TEST
		} else if tf.info.Size() < hundredKb {
			fragSize = 512
		} else if tf.info.Size() < oneMb {
			fragSize = twoKb
		} else {
			fragSize = sixKb
		}
	}
	cord, start := createLeafsAndStartLoading(tf, initialPos, fragSize, wg)
	return cord, start, nil
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

func createLeafsAndStartLoading(tf *textFile, initialPos int64, fragSize int64, wg *sync.WaitGroup) (cords.Cord, *fileLeaf) {
	if initialPos > tf.info.Size() || fragSize > tf.info.Size() {
		panic("inconsistent setting of text file paramenters")
	}
	size := tf.info.Size()
	var next *fileLeaf
	// we do not allocate all the leafs as an array, because this would prevent single
	// leafs to be garbage collected as soon as the client deletes fragments of text
	rightmost := size / fragSize * fragSize
	if rightmost == size && size > 0 {
		rightmost -= fragSize
	}
	T().Debugf("fragment.size=%d, rightmost=%d", fragSize, rightmost)
	// create an (initially empty) fileLeaf for every to-be fragment
	var leaf, last, start *fileLeaf // last leaf shall point to first leaf
	T().Debugf("creating leafs for %s (%d bytes)", tf.info.Name(), tf.info.Size())
	var setup sync.WaitGroup
	b := cords.NewBuilder()
	leafcnt := 0
	for k := rightmost; k >= 0; k -= fragSize { // iterate backwards
		leaf = &fileLeaf{
			length: min(fragSize, size-k), // negative value signals leaf not loaded yet
		}
		linfo := &loadingInfo{
			next: next,
			tf:   tf,
			pos:  k,
			mx:   &sync.Mutex{}, // must be a pointer to avoid later copying
		}
		linfo.mx.Lock() // lock leaf's String() method until loaded
		ext := &leafExt{loadExt: linfo}
		leaf.setExt(ext)
		setup.Add(1)
		//leaf.ext.Store(atomicExt)
		if wg != nil {
			wg.Add(1)
		}
		T().Debugf("created leaf for fragment %d…%d", k, k+leaf.length)
		// build up a cord structure
		b.Prepend(leaf)
		if last == nil { // we want leafs to be linked to a cycle
			last = leaf
		}
		if k <= initialPos && initialPos < k+leaf.length {
			start = leaf // start leaf contains initial position
		}
		next = leaf // predecessor leaf will set this leaf as successor (*next)
		leafcnt++
	}
	lastExt := last.Ext()
	lastExt.loadExt.next = leaf // make it a cycle of leafs
	last.setExt(lastExt)
	loadAllFragmentsAsync(tf, start, leafcnt, &setup, wg)
	T().Debugf("start.len=%d", start.length)
	return b.Cord(), start
}

// --- File loading goroutine ------------------------------------------------

type msg struct {
	leaf    *fileLeaf
	content []byte
}

func loadAllFragmentsAsync(tf *textFile, startLeaf *fileLeaf, leafcnt int, setup, wg *sync.WaitGroup) {
	if startLeaf == nil {
		panic("load-fragments may not be called for a void cord")
	}
	//
	fragChan := make(chan msg, 0) // communication channel between one loader thread and many leafs
	leaf := startLeaf             // leaf-links have to form a cycle
	// start subscriber goroutines, one per leaf
	for { // let every leaf listen to the frag channel
		leafNext := leaf.Ext().loadExt.next // remember the link to next leaf
		go func(leaf *fileLeaf, cast *caster.Caster) {
			ext := leaf.Ext()
			defer ext.loadExt.mx.Unlock()          // locking has already been done during leaf creation
			ch, ok := cast.Sub(nil, uint(leafcnt)) // subscribe to broadcast messages
			if !ok {
				T().Errorf("broadcaster already closed")
			}
			T().Debugf("leaf listener @ %d", ext.loadExt.pos)
			setup.Done()
			for loaded := range ch { // wait for signals for loaded leafs
				T().Debugf("read of %d…%d", ext.loadExt.pos, leaf.length)
				if loaded.(msg).leaf == leaf { // yes, it's our turn
					T().Debugf("received loaded-message %d…%d", ext.loadExt.pos, leaf.length)
					s := string(loaded.(msg).content) // re-alloc loaded.content as string
					ext.content = s                   // put it into leaf's ext
					ext.loadExt = nil                 // drop loading management info
					leaf.ext.Store(ext)               // swap loading-ext for string-ext
					//cast.Unsub(ch)                    // we do not want to listen any further
					if wg != nil {
						wg.Done()
					}
					break // this will unlock the leaf's mutex
				}
			} // ignore all other leaf messages
		}(leaf, tf.cast)
		//if leaf.Ext().duringLoad().next == startLeaf {
		if leafNext == startLeaf {
			break // when iterated over cycle of leafs, stop
		}
		//leaf = leaf.Ext().duringLoad().next // continue iteration
		leaf = leafNext // continue iteration
	}
	// start publisher goroutine
	go func(ch <-chan msg) {
		T().Debugf("started message publisher")
		// start a broadcaster to wait for loaded fragments and publish their ids
		defer tf.cast.Close()
		// publish loaded fragments to all waiting leafs
		//T().Debugf("channel=%v", ch)
		setup.Wait()
		for m := range ch {
			T().Debugf("publisher received message")
			tf.cast.Pub(m)
		}
		T().Debugf("stopped message publisher")
	}(fragChan)
	startFileLoader(tf, startLeaf, fragChan, wg)
}

func startFileLoader(tf *textFile, startLeaf *fileLeaf, ch chan<- msg, wg *sync.WaitGroup) {
	go func(ch chan<- msg, wg *sync.WaitGroup) {
		// iterate over leafs and load the fragment of text referenced by it
		leaf := startLeaf // leaf-links have to form a cycle
		for {
			ext := leaf.Ext().loadExt
			T().Debugf("reading file fragment %d…%d", ext.pos, leaf.length)
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
			T().Debugf("putting message onto channel: %8d='%.10s…'", ext.pos, buf)
			//T().Debugf("channel=%v", ch)
			ch <- m // signal that a leaf is done loading
			T().Debugf("message sent")
			if next == startLeaf {
				break // when iterated over cycle of leafs, stop
			}
			leaf = next // continue iteration
		}
		T().Debugf("========== LOADING OF FILE FRAGMENTS FINISHED ===============")
	}(ch, wg)
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
