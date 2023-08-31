// Copyright (c) Tetrate, Inc 2023.
// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opentelemetry

import (
	"errors"
	"sync"

	"github.com/tetratelabs/telemetry"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"

	"github.com/tetratelabs/telemetry-opentelemetry/internal/maps"
	"github.com/tetratelabs/telemetry-opentelemetry/internal/slices"
)

// NewSum creates a new Metric with an aggregation type of Sum (the values will
// be cumulative). That means that data collected by the new Metric will be
// summed before export.
func (m *metricSink) NewSum(name, description string, opts ...telemetry.MetricOption) telemetry.Metric {
	m.knownMetrics.register(MetricDefinition{
		Name:        name,
		Type:        "Sum",
		Description: description,
	})
	o, dm := m.createOptions(name, opts...)
	if dm != nil {
		return dm
	}
	return m.newCounter(name, description, o)
}

// NewGauge creates a new Metric with an aggregation type of LastValue. That
// means that data collected by the new Metric will export only the last
// recorded value.
func (m *metricSink) NewGauge(name, description string, opts ...telemetry.MetricOption) telemetry.Metric {
	m.knownMetrics.register(MetricDefinition{
		Name:        name,
		Type:        "LastValue",
		Description: description,
	})
	o, dm := m.createOptions(name, opts...)
	if dm != nil {
		return dm
	}
	return m.newGauge(name, description, o)
}

// NewDerivedGauge creates a new Gauge Metric. That means that data collected by the new
// Metric will export only the last recorded value.
// Unlike NewGauge, the DerivedGauge accepts functions which are called to get the current value.
func (m *metricSink) NewDerivedGauge(name, description string) telemetry.DerivedMetric {
	m.knownMetrics.register(MetricDefinition{
		Name:        name,
		Type:        "LastValue",
		Description: description,
	})
	return m.newDerivedGauge(name, description)
}

// NewDistribution creates a new Metric with an aggregation type of Distribution.
// This means that the data collected by the Metric will be collected and
// exported as a histogram, with the specified bounds.
func (m *metricSink) NewDistribution(name, description string, bounds []float64, opts ...telemetry.MetricOption) telemetry.Metric {
	m.knownMetrics.register(MetricDefinition{
		Name:        name,
		Type:        "Distribution",
		Description: description,
		Bounds:      bounds,
	})
	o, dm := m.createOptions(name, opts...)
	if dm != nil {
		return dm
	}
	return m.newDistribution(name, description, o)
}

// MetricDefinition records a metric's metadata.
// This is used to work around two limitations of OpenTelemetry:
//   - (https://github.com/open-telemetry/opentelemetry-go/issues/4003) Histogram buckets cannot be defined per instrument.
//     instead, we record all metric definitions and add them as Views at registration time.
//   - Support pkg/collateral, which wants to query all metrics. This cannot use a simple Collect() call, as this ignores any unused metrics.
type MetricDefinition struct {
	Name        string
	Type        string
	Description string
	Bounds      []float64
}

// metrics stores known metrics
type metrics struct {
	started bool
	mu      sync.Mutex
	known   map[string]MetricDefinition
}

// ExportMetricDefinitions reports all currently registered metric definitions.
func (m *metricSink) ExportMetricDefinitions() []MetricDefinition {
	m.knownMetrics.mu.Lock()
	defer m.knownMetrics.mu.Unlock()
	return slices.SortFunc(maps.Values(m.knownMetrics.known), func(a, b MetricDefinition) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name == b.Name {
			return 0
		} else {
			return 1
		}
	})
}

// register records a newly defined metric. Only valid before an exporter is set.
func (d *metrics) register(def MetricDefinition) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started {
		log.Error("Unable to register metric", errors.New("metrics have already started"), "metric", def.Name)
	}
	d.known[def.Name] = def
}

// toHistogramViews works around https://github.com/open-telemetry/opentelemetry-go/issues/4003; in the future we can define
// this when we create the histogram.
func (d *metrics) toHistogramViews() []metric.Option {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.started = true
	opts := []metric.Option{}
	for name, def := range d.known {
		if def.Bounds == nil {
			continue
		}
		// for each histogram metric (i.e. those with bounds), set up a view explicitly defining those buckets.
		v := metric.WithView(metric.NewView(
			metric.Instrument{Name: name},
			metric.Stream{Aggregation: aggregation.ExplicitBucketHistogram{
				Boundaries: def.Bounds,
			}},
		))
		opts = append(opts, v)
	}
	return opts
}
