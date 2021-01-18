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
// Combine requires the Metric to calculate a metric value “sum” (monoid) from
// two metric values. This way the metric will bubble up metric values to the root
// of the cord tree and therewith result in a single overall metric value for
// a text.
// Combine must be a monoid over cords.MetricValue, with a neutral element n
// of Apply = f("") → n, i.e. the metric value of the empty string.
//
// However, for materialized metrics it is a bit
// different from plain metrics: they resemble a free monoid. This is reflected
// by the result of materialized metrics, which is a list of spans (organized through
// a cord-tree).
// As a corollary, Combine has an additional task for materialized metrics
// than it has for plain metrics. Combine has to focus on the bytes *between* the
// already recognized spans of both the left and right sibling, and be able to
// convert them to cord leafs.
//
type Metric interface {
	Apply(frag []byte) MetricValue
	Combine(leftSibling, rightSibling MetricValue, metric Metric) MetricValue
}

// MaterializedMetric is a type for metrics (please refer to interface Metric)
// that build a concrete cord tree for a text-cord.
//
// A materialized metric does metric calculations exactly as a simple metric.
// However, it additionally supports building up a tree from atomic leafs
// containing metric values.
//
// There are (at least) two ways to go about building a metric tree: one to
// preserve a homomorph of the text fragments, essentially materializing a
// catamorphism, or we can re-align the leaf boundaries with the
// continuity-boundaries of the metric. We'll go with the latter and build up
// a tree which will the be somewhat decoupled from the text cord.
//
type MaterializedMetric interface {
	Metric
	Leafs(MetricValue, bool) []Leaf
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
//    (c)  |-----============================|              combined intermediate fragment  or
//
// For an approachable discussion please refer to Raph Levien's “Rope Science” series
// (https://xi-editor.io/docs/rope_science_01.html), or—less approachable—read up
// on catamorphism.
//
// Calculating metric values is a topic where implementation characteristics of cords
// get somewhat visible for clients of the API. This is unfortunate, but may be
// mitigated by helper types provided by this package. Clients usually will either create
// metrics on top of pre-defined basic metrics or they may embed the MetricValueBase
// helper type in their MetricValues.
//
type MetricValue interface {
	Len() int                      // summed up length of text fragments
	Unprocessed() ([]byte, []byte) // unprocessed bytes at either end
}

// --- Apply a metric to a cord ----------------------------------------------

// ApplyMetric applies a metric calculation on a (section of a) text.
//
// i and j are text positions with Go slice semantics.
// If [i, j) does not specify a valid slice of the text, ErrIndexOutOfBounds will be
// returned.
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

// ApplyMaterializedMetric applies a materialized metric to a (section of a) text.
// Returns a metric value and a cord, which manages the spans of the metric
// on the text.
//
// i and j are text positions with Go slice semantics.
// If [i, j) does not specify a valid slice of the text, ErrIndexOutOfBounds will be
// returned.
//
func ApplyMaterializedMetric(cord Cord, i, j uint64, metric MaterializedMetric) (MetricValue, Cord, error) {
	if cord.IsVoid() {
		return nil, Cord{}, nil
	}
	if i > cord.Len() || j > cord.Len() || j < i {
		return nil, Cord{}, ErrIndexOutOfBounds
	}
	v, c := applyMaterializedMetric(&cord.root.cordNode, i, j, metric)
	spans := metric.Leafs(v, true)
	cl := buildFragmentCord(spans[:1])
	cr := buildFragmentCord(spans[1:])
	c = Concat(cl, c, cr)
	//sl, sr := v.Unprocessed() // there may be unprocessed bytes at either end
	// TODO apply metric and build l.leaf and r.leaf
	return v, c, nil
}

func applyMaterializedMetric(node *cordNode, i, j uint64, metric MaterializedMetric) (MetricValue, Cord) {
	T().Debugf("called applyMaterializedMetric([%d], %d, %d)", node.Weight(), i, j)
	if node.IsLeaf() {
		leaf := node.AsLeaf()
		T().Debugf("M-METRIC(%s|%d, %d, %d)", leaf, leaf.Len(), i, j)
		s := leaf.leaf.Substring(umax(0, i), umin(j, leaf.Len()))
		v := metric.Apply(s)
		c := buildFragmentCord(metric.Leafs(v, false))
		if !c.IsVoid() {
			dump(&c.root.cordNode)
		}
		return v, c
	}
	var v, vl, vr MetricValue
	var c, cl, cr Cord
	if i < node.Weight() && node.Left() != nil {
		vl, cl = applyMaterializedMetric(node.Left(), i, j, metric)
		T().Debugf("left metric value = %v", vl)
	}
	if node.Right() != nil && j > node.Weight() {
		w := node.Weight()
		vr, cr = applyMaterializedMetric(node.Right(), i-umin(w, i), j-w, metric)
		T().Debugf("right metric value = %v", vr)
	}
	if !isnull(vl) && !isnull(vr) {
		T().Debugf("COMBINE %v  +  %v", vl, vr)
		v = metric.Combine(vl, vr, metric)
		mid := buildFragmentCord(metric.Leafs(v, false))
		c = Concat(cl, mid, cr)
	} else if !isnull(vl) {
		v = vl
		c = cl
	} else if !isnull(vr) {
		v = vr
		c = cr
	}
	T().Debugf("combined metric value = %v", v)
	T().Debugf("node=%v", node)
	if !c.IsVoid() {
		dump(&c.root.cordNode)
	}
	T().Debugf("dropping out of applyMetric([%d], %d, %d)", node.Weight(), i, j)
	return v, c
}

func buildFragmentCord(leafs []Leaf) Cord {
	if len(leafs) == 0 || leafs[0] == nil {
		return Cord{}
	}
	var cord Cord
	for _, leaf := range leafs {
		lnode := makeLeafNode()
		lnode.leaf = leaf
		c := makeCord(&lnode.cordNode)
		cord = cord.concat2(c)
	}
	if !cord.IsVoid() {
		dump(&cord.root.cordNode)
	}
	return cord
}

func isnull(v MetricValue) bool {
	return v == nil || v.Len() == 0
}
