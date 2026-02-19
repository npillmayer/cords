package cords

// Metric is a metric to calculate on a cord.
//
// Metric implementations operate on byte slices. With the btree migration, the
// driver now extracts requested ranges via Report and applies metrics on that
// contiguous segment.
type Metric interface {
	Apply(frag []byte) MetricValue
	Combine(leftSibling, rightSibling MetricValue, metric Metric) MetricValue
}

// MaterializedMetric is a metric that can produce leafs from metric values.
type MaterializedMetric interface {
	Metric
	Leafs(MetricValue, bool) []Leaf
}

// MetricValue is a type returned by applying a metric to text fragments.
type MetricValue interface {
	Len() int
	Unprocessed() ([]byte, []byte)
}

// ApplyMetric applies a metric calculation on a text slice [i,j).
func ApplyMetric(cord Cord, i, j uint64, metric Metric) (MetricValue, error) {
	if cord.IsVoid() {
		return nil, nil
	}
	if i > cord.Len() || j > cord.Len() || j < i {
		return nil, ErrIndexOutOfBounds
	}
	content, err := cord.Report(i, j-i)
	if err != nil {
		return nil, err
	}
	return metric.Apply([]byte(content)), nil
}

// ApplyMaterializedMetric applies a materialized metric to [i,j) and returns
// a metric value plus a cord managing the produced metric spans.
func ApplyMaterializedMetric(cord Cord, i, j uint64, metric MaterializedMetric) (MetricValue, Cord, error) {
	if cord.IsVoid() {
		return nil, Cord{}, nil
	}
	if i > cord.Len() || j > cord.Len() || j < i {
		return nil, Cord{}, ErrIndexOutOfBounds
	}
	content, err := cord.Report(i, j-i)
	if err != nil {
		return nil, Cord{}, err
	}
	v := metric.Apply([]byte(content))
	mid := buildFragmentCord(metric.Leafs(v, false))

	bounds := metric.Leafs(v, true)
	if len(bounds) >= 2 {
		left := buildFragmentCord(bounds[:1])
		right := buildFragmentCord(bounds[1:2])
		mid = Concat(left, mid, right)
	}
	return v, mid, nil
}

func buildFragmentCord(leafs []Leaf) Cord {
	if len(leafs) == 0 {
		return Cord{}
	}
	b := NewBuilder()
	for _, leaf := range leafs {
		if leaf == nil {
			continue
		}
		_ = b.Append(leaf)
	}
	return b.Cord()
}
