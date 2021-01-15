package cords

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

// Metric is a metric to calculate on a cord. Sometimes it's helpful to find
// information about a (large) text by collecting metrics from fragments and
// assembling them. Cords naturally break up texts into smaller fragments,
// letting us calculate metrics by applying them to (a subset of) fragments and
// propagate them upwards the nodes of the cord tree.
//
// An example of a (very simplistic) metric would be to count the number of
// bytes in a text. The total count is calculated by counting the bytes in every
// fragment and adding up intermediate sums while travelling upwards through the
// cord's tree.
//
// Metric is a simple interface whith a metric function that will be applied
// to portions of text. Clients will have no control over size or boundaries of
// the fragment. Applying metrics to different fragments may be done concurrently
// by the calculation driver, therefore it is illegal to hold unguarded global state
// in a metric.
//
// Combine requires the Metric to calculate a metric value “sum” from two
// metric values. This way the metric will bubble up metric values to the root
// of the cord tree and therewith result in a single overall metric value for
// a text.
//
type Metric interface {
	Apply(frag string) MetricValue
	Combine(leftSibling, rightSibling MetricValue, metric Metric) MetricValue
}

// MetricValue is a type returned by applying a metric to text fragments (see
// interface Metric). It holds information about the added length of the text
// fragments which this value has been calulated for, and slices of bytes
// at either end of the accumulated fragments which have to be reprocessed.
//
// Fragments of text are presented to the metric function as slices of bytes,
// without regard to rune-, grapheme- or line-boundaries. If we want to
// calculate information about, say, the maximum line length in a text, we'd
// have to count the graphemes of fragments. Graphemes will consist, however,
// of an unknown number of bytes and code points, which may only be identified
// by reading them at the grapheme start character. If a fragment is cut in
// the middle of a grapheme, the metric at the first bytes of a fragment cannot
// reliably calculated. Therefore metrics will be calculated on substrings of
// fragments where conditions allow a metric application, and any unprocessed
// bytes at the left and right boundary will be marked for reprocessing.
//
// When propagating metric values up the tree nodes, metric value of the left
// and right child node of a cord tree node will have to be combined.
// The combination must be able to reprocess any unprocessed bytes.
//
//       --- denotes unprocessed bytes
//       === denotes bytes already processed by the metric
//
//    (a)  |-----========--|   &   |----==============|     combine two sibling fragments
//
//    (b)  |-----========    ------     ==============|     reprocess 6 bytes in between
//
//    (c)  |-----============================|              combined intermediate fragment
//
// For an in-depth discussion please refer to Raph Levien's “Rope Science” series
// (https://xi-editor.io/docs/rope_science_01.html).
//
// Calculating metric values is a topic where implementation characteristics of cords
// get somewhat visible for clients of the API. This is unfortunate, but may be
// mitigated by helper types provided by this package. Clients usually will either create
// metrics on top of pre-defined basic metrics or they may embed the MetricValueBase
// helper type in their MetricValues.
//
type MetricValue interface {
	Len() int                      // added length of text fragments
	Unprocessed() ([]byte, []byte) // unprocessed bytes at either end
}

// MetricValueBase is a helper type for metric application and for combining
// metric values on text fragments. Clients who want to use it should embed a
// MetricValueBase into their type definition for metric types.
//
// MetricValueBase will implement `Len` and `Unprocessed` of interface
// MetricValue. To implement `Combine` clients should interact with
// MetricValueBase in a way that lets MetricValueBase handle the tricky parts
// of fragment boundary bytes.
//
// In Metric.Apply(…):
//
//     v := &myCoolMetricValue{ … }    // create a MetricValue which embeds MetricValueBase
//     v.InitFrom(frag)                // call this helper first
//     from, to := …                   // do some metric calculations and possibly have boundaries
//     v.Measured(from, to, frag)      // leave it to MetricValueBase to remember unprocessed bytes
//     return &v
//
// In Metric.Combine(…):
//
//     unproc, ok := leftSibling.ConcatUnprocessed(&rightSibling.MetricValueBase)  // step (b)
//     if ok {                                                                     //
//         // yes, we have to re-apply our metric to `unproc`                      //
//         x := metric.Apply(string(unproc)).(*delimiterMetricValue)               //
//         // do something with sub-value x                                        //
//     }
//     leftSibling.UnifyWith(&rightSibling.MetricValueBase)                        // step (c)
//
// Which spans of text fragments can be processed and how intermediate metric values
// are calculated and stored is up to the client's `Metric` and `MetricValue`.
//
type MetricValueBase struct {
	length       int
	openL, openR []byte //
}

// Len is part of interface MetricValue.
func (mvb MetricValueBase) Len() int {
	return mvb.length
}

// Unprocessed is part of interface MetricValue.
func (mvb MetricValueBase) Unprocessed() ([]byte, []byte) {
	return mvb.openL, mvb.openR
}

// InitFrom should be called from the enclosing client metric type at the
// beginning of `Combine`. It will set up information about fragment length
// and possibly other administrative information.
//
// This will usually be called from Metric.Apply(…).
//
func (mvb *MetricValueBase) InitFrom(frag string) {
	mvb.length = len(frag)
}

// Measured is a signal to an embedded MetricValueBase
// that a range of bytes has already been considered for metric calculation.
// The MetricValueBase will derive information about unprocessed boundary
// bytes from this.
//
// This will usually be called from Metric.Apply(…).
//
func (mvb *MetricValueBase) Measured(from, to int, frag string) {
	// TODO this should be updatable and incremental span should be
	// added up, i.e. boundary bytes should change with multiple calls to Measured
	if from < 0 || from > mvb.length {
		mvb.openL = []byte(frag)
		mvb.openR = nil
		return
	}
	mvb.openL = nil
	mvb.openR = nil
	if from > to {
		from, to = to, from
	}
	if from > 0 {
		mvb.openL = []byte(frag)[:from]
	}
	if to < mvb.length {
		mvb.openR = []byte(frag)[to:]
	}
}

// MeasuredNothing is a signal to an embedded MetricValueBase that no metric value
// could be calculated for a given text fragment. This will tell the
// metric calculation driver to reconsider the complete fragment when
// combining it with a sibling node.
//
// This will usually be called from Metric.Apply(…).
//
func (mvb *MetricValueBase) MeasuredNothing(frag string) {
	mvb.Measured(-1, -1, frag)
}

// HasBoundaries returns true if the metric value has unprocessed boundary bytes.
// Clients normally will not have to consult this.
func (mvb *MetricValueBase) HasBoundaries() bool {
	return len(mvb.openL)+len(mvb.openR) < mvb.length
}

// ConcatUnprocessed is a helper function to provide access to
// unprocessed bytes in between two text fragments.
// As described with MetricValues, refer to step (b) where unprocessed
// boundary bytes are subject to re-application of the metric.
//
//    (b)  |-----========    ------    ==============|     reprocess 6 bytes in between
//
// ConcatUnprocessed will return the 6 bytes in between and a boolean flag to indicate
// if the metric should reprocess the bytes. It is the responsibility of the client's
// metric to initiate the reprocessing.
//
func (mvb *MetricValueBase) ConcatUnprocessed(rightSibling *MetricValueBase) ([]byte, bool) {
	otherL := rightSibling.openL
	if len(otherL) > 0 {
		if mvb.HasBoundaries() {
			mvb.openR = append(mvb.openR, otherL...)
			return mvb.openR, rightSibling.HasBoundaries()
		}
		// else no boundaries in mvb => openR is empty, frag is in openL
		mvb.openL = append(mvb.openL, otherL...)
		return mvb.openL, false
	}
	return nil, false
}

// UnifyWith creates a combined metric value from two sibling values.
// Recalculation of unprocessed bytes must already have been done, i.e.
// ConcatUnprocessed must already have been called.
//
// Referring to the example for MetricValue, UnifyWith will help with step (c):
//
//    (c)  |-----============================|              combined intermediate fragment
//
// The “meat” of the metric has to be calculated by the client metric type. Clients must
// implement their own data structure to support metric calculation and propagation.
// MetricValueBase just shields clients from the details of fragment handling.
//
func (mvb *MetricValueBase) UnifyWith(rightSibling *MetricValueBase) {
	mvb.length += rightSibling.length
	mvb.openR = rightSibling.openR
}

// --- Apply a metric to a cord ----------------------------------------------

// ApplyMetric applies a metric calculation on a (section of a) text.
// i and j are text positions with Go slice semantics.
//
func ApplyMetric(cord Cord, i, j uint64, metric Metric) (MetricValue, error) {
	if cord.IsVoid() {
		return nil, nil
	}
	if i > cord.Len() || j > cord.Len() || j < i {
		return nil, ErrIndexOutOfBounds
	}
	return applyMetric(&cord.root.cordNode, i, j, metric), nil
}

func applyMetric(node *cordNode, i, j uint64, metric Metric) MetricValue {
	T().Debugf("called applyMetric([%d], %d, %d)", node.Weight(), i, j)
	if node.IsLeaf() {
		leaf := node.AsLeaf()
		T().Debugf("METRIC(%s|%d, %d, %d)", leaf, leaf.Len(), i, j)
		s := leaf.leaf.Substring(umax(0, i), umin(j, leaf.Len()))
		v := metric.Apply(s)
		T().Debugf("leaf metric value = %v", v)
		return v
	}
	var v, vl, vr MetricValue
	if i < node.Weight() && node.Left() != nil {
		vl = applyMetric(node.Left(), i, j, metric)
		T().Debugf("left metric value = %v", vl)
	}
	if node.Right() != nil && j > node.Weight() {
		w := node.Weight()
		vr = applyMetric(node.Right(), i-umin(w, i), j-w, metric)
		T().Debugf("right metric value = %v", vr)
	}
	if !isnull(vl) && !isnull(vr) {
		T().Debugf("COMBINE %v  +  %v", vl, vr)
		v = metric.Combine(vl, vr, metric)
	} else if !isnull(vl) {
		v = vl
	} else if !isnull(vr) {
		v = vr
	}
	T().Debugf("combined metric value = %v", v)
	T().Debugf("node=%v", node)
	T().Debugf("dropping out of applyMetric([%d], %d, %d)", node.Weight(), i, j)
	return v
}

func isnull(v MetricValue) bool {
	return v == nil || v.Len() == 0
}
