//Copyright (c) 2014 Square, Inc

package check

import (
	"io"

	"github.com/measure/metrics"
)

type Checker interface {
	// Runs metric check
	// creates its own new scope and package and inserts
	// metric values automagically
	CheckAll(w io.Writer) error

	// TODO: should these functions really be exported?
	NewScopeAndPackage() error
	InsertMetricValuesFromJSON() error
	InsertMetricValuesFromContext(m *metrics.MetricContext) error
}
