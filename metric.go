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

// Package opentelemetry provides a tetratelabs/telemetry compatible metrics
// implementation based on OpenTelemetry.
package opentelemetry

import (
	"context"

	"github.com/tetratelabs/telemetry"
	"github.com/tetratelabs/telemetry/scope"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"

	"github.com/tetratelabs/telemetry-opentelemetry/internal/tag"
)

var log = scope.Register("telemetry-otel", "Messages from the telemetry-OTel metric package")

type MetricAndDerivedMetricSink interface {
	telemetry.MetricSink
	telemetry.DerivedMetricSink
}

type SinkOption func(ms *metricSink)

func WithStrictDimensions() SinkOption {
	return func(ms *metricSink) {
		ms.strictDimensions = true
	}
}

func WithLogger(l telemetry.Logger) SinkOption {
	return func(ms *metricSink) {
		if l == nil {
			ms.logger = log
			return
		}
		ms.logger = l
	}
}

// New returns a new Telemetry facade compatible MetricSink.
func New(appName string, opts ...SinkOption) MetricAndDerivedMetricSink {
	ms := &metricSink{
		meter: otel.GetMeterProvider().Meter(appName),
		knownMetrics: &metrics{
			known: map[string]MetricDefinition{},
		},
	}
	for _, opt := range opts {
		opt(ms)
	}
	return ms
}

// Start should be used by telemetry Exporters so we can trigger final
// initialization of our histograms.
func Start(ms telemetry.MetricSink) []sdk.Option {
	if m, ok := ms.(*metricSink); ok {
		return m.knownMetrics.toHistogramViews()
	}

	return nil
}

type metricSink struct {
	logger           telemetry.Logger
	meter            metric.Meter
	knownMetrics     *metrics
	strictDimensions bool
}

// NewLabel creates a new Label to be used as a metrics dimension.
func (m *metricSink) NewLabel(name string) telemetry.Label {
	label, _ := tag.NewKey(name)
	return &labelImpl{
		label: label,
	}
}

// ContextWithLabels takes the existing LabelValues collection found in context
// and runs the Label operations as provided by the provided values on top of
// the collection which is then added to the returned context. The function can
// return an error in case the provided values contain invalid label names.
func (m *metricSink) ContextWithLabels(ctx context.Context, values ...telemetry.LabelValue) (context.Context, error) {
	if len(values) == 0 {
		return ctx, nil
	}
	mutators := make([]tag.Mutator, len(values))
	for idx, value := range values {
		mutators[idx] = value.(tag.Mutator)
	}
	return tag.New(ctx, mutators...)
}

type labelImpl struct {
	label tag.Key
}

// Insert will insert the provided value for the Label if not set.
func (l labelImpl) Insert(val string) telemetry.LabelValue {
	return tag.Insert(l.label, val)
}

// Update will update the Label with provided value if already set.
func (l labelImpl) Update(val string) telemetry.LabelValue {
	return tag.Update(l.label, val)
}

// Upsert will insert or replace the provided value for the Label.
func (l labelImpl) Upsert(val string) telemetry.LabelValue {
	return tag.Upsert(l.label, val)
}

// Delete will remove the Label's value.
func (l labelImpl) Delete() telemetry.LabelValue {
	return tag.Delete(l.label)
}
