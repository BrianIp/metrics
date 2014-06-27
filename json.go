// Copyright (c) 2014 Square, Inc

package metrics

import (
	"fmt"
	"io"
	"encoding/json"
)

// EncodeJSON is a streaming encoder that writes all metrics passing filter 
// to writer w as JSON
func (m *MetricContext) EncodeJSON(w io.Writer) error {

	prependComma := false
	w.Write([]byte("[\n"))

	for name, c := range m.Counters {
		if ! m.OutputFilter(name, c) {
			continue
		}
		if prependComma {
			w.Write([]byte(",\n"))
		}
                w.Write([]byte(fmt.Sprintf(
                        `{"type": "counter", "name": "%s", "value": %d, "rate": %f}`,
                        name, c.Get(), c.ComputeRate())))
		prependComma = true
        }

	for name, g := range m.Gauges {
		if ! m.OutputFilter(name, g) {
			continue
		}
		if prependComma {
			w.Write([]byte(",\n"))
		}
                w.Write([]byte(fmt.Sprintf(
                        `{"type": "gauge", "name": "%s", "value": %d}`,
                        name, g.Get())))
		prependComma = true
        }


	for name, c := range m.BasicCounters {
		if ! m.OutputFilter(name, c) {
			continue
		}
		if prependComma {
			w.Write([]byte(",\n"))
		}
                w.Write([]byte(fmt.Sprintf(
                        `{"type": "basiccounter", "name": "%s", "value": %d}`,
                        name, c.Get())))
		prependComma = true
        }


	for name, s := range m.StatsTimers {
		if ! m.OutputFilter(name, s) {
			continue
		}
		if prependComma {
			w.Write([]byte(",\n"))
		}
		type percentileData struct {
			percentile string
			value      float64
		}

		var pctiles []percentileData
		for _, p := range percentiles {
			percentile, err := s.Percentile(p)
			stuff := fmt.Sprintf("%.6f", p)
			if err == nil {
				pctiles = append(pctiles, percentileData{stuff, percentile})
			}
		}
		data := struct {
			Type        string
			Name        string
			Percentiles []percentileData
		}{
			"statstimer",
			name,
			pctiles,
		}
		b, err := json.Marshal(data)
		if err != nil {
			continue
		}
		w.Write(b)
		prependComma = true
	}

	return nil
}
