//Copyright (c) 2014 Square, Inc

package check

import (
	"github.com/measure/metrics"
)

type Checker interface {
	// Runs metric check
	CheckAll() ([]CheckResult, error)

	// TODO: should these functions really be exported?
	NewScopeAndPackage() error
	InsertMetricValuesFromJSON() error
	InsertMetricValuesFromContext(m *metrics.MetricContext) error
}
