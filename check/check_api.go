//Copyright (c) 2014 Square, Inc

package check

import (
	"io"

	"github.com/measure/metrics"
)

type Checker interface {
	// Runs metric check
	CheckAll(w io.Writer) ([]string, error)

	// TODO: should these functions really be exported?
	NewScopeAndPackage() error
	InsertMetricValuesFromJSON() error
	InsertMetricValuesFromContext(m *metrics.MetricContext) error
}
