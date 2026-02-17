package metrics

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
// It is up to the client's `Metric` and `MetricValue` to decide which spans of
// text fragments can be processed and how intermediate metric values are calculated
// and stored.
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
func (mvb *MetricValueBase) InitFrom(frag []byte) {
	mvb.length = len(frag)
}

// Measured is a signal to an embedded MetricValueBase
// that a range of bytes has already been considered for metric calculation.
// The MetricValueBase will derive information about unprocessed boundary
// bytes from this.
//
// This will usually be called from Metric.Apply(…).
//
// from and to are allowed to be identical, signalling a split.
//
func (mvb *MetricValueBase) Measured(from, to int, frag []byte) {
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
func (mvb *MetricValueBase) MeasuredNothing(frag []byte) {
	mvb.Measured(-1, -1, frag)
}

// HasBoundaries returns true if the metric value has unprocessed boundary bytes.
// Clients normally will not have to consult this.
func (mvb *MetricValueBase) HasBoundaries() bool {
	return len(mvb.openL)+len(mvb.openR) < mvb.length || len(mvb.openR) > 0
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

func (mvb *MetricValueBase) Chunk() []byte {
	b := mvb.HasBoundaries()
	if b {
		return []byte("")
	}
	return mvb.openL
}

func (mvb *MetricValueBase) Suffix() []byte {
	b := mvb.HasBoundaries()
	if !b {
		return []byte("")
	}
	return mvb.openL
}

func (mvb *MetricValueBase) Prefix() []byte {
	b := mvb.HasBoundaries()
	if !b {
		return []byte("")
	}
	return mvb.openR
}
