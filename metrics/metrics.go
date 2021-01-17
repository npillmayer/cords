package metrics

import (
	"fmt"

	"github.com/npillmayer/cords"
)

// CountingMetric is a type for metrics that count items in text. Possible
// items may be lines, words, emojis, …
type CountingMetric interface {
	cords.Metric
	Count(cords.MetricValue) int
}

// Count applies a counting metric to a text.
func Count(text cords.Cord, i, j uint64, metric CountingMetric) (int, error) {
	value, err := cords.ApplyMetric(text, i, j, metric)
	if err != nil {
		return -1, fmt.Errorf("metrics.Count could not be applied: %w", err)
	}
	return metric.Count(value), nil
}

// ---------------------------------------------------------------------------

// A ScanningMetric searches a text for item (such as lines, word, emojis, …) and
// returns their location indices.
type ScanningMetric interface {
	cords.Metric
	Locations(cords.MetricValue) [][]int
}

// Find applies a scanning metric to a text.
func Find(text cords.Cord, i, j uint64, metric ScanningMetric) ([][]int, error) {
	value, err := cords.ApplyMetric(text, i, j, metric)
	if err != nil {
		return [][]int{}, fmt.Errorf("metrics.Find could not be applied: %w", err)
	}
	return metric.Locations(value), nil
}

// ---------------------------------------------------------------------------

// Align applies a materialized metric to a text.
func Align(text cords.Cord, i, j uint64, metric cords.MaterializedMetric) (cords.Cord, error) {
	_, err := cords.ApplyMaterializedMetric(text, i, j, metric)
	if err != nil {
		return cords.Cord{}, fmt.Errorf("metrics.Find could not be applied: %w", err)
	}
	return cords.Cord{}, nil
}
