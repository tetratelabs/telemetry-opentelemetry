// Copyright (c) Tetrate, Inc 2023.
// Copyright Istio Authors
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
	"context"

	"github.com/tetratelabs/telemetry"
	"go.opentelemetry.io/otel/attribute"
	api "go.opentelemetry.io/otel/metric"

	"github.com/tetratelabs/telemetry-opentelemetry/internal/tag"
)

type distribution struct {
	baseMetric
	d api.Float64Histogram
}

var _ telemetry.Metric = (*distribution)(nil)

func (m *metricSink) newDistribution(name, description string, o telemetry.MetricOptions) *distribution {
	d, err := m.meter.Float64Histogram(name,
		api.WithDescription(description),
		api.WithUnit(string(o.Unit)))
	if err != nil {
		log.Error("failed to create distribution", err)
	}
	r := &distribution{d: d}
	keysMap := map[tag.Key]bool{}
	for _, k := range o.Labels {
		if l, ok := k.(*labelImpl); ok {
			keysMap[l.label] = true
		}
	}
	r.baseMetric = baseMetric{
		ms:   m,
		name: name,
		rest: r,
		keys: keysMap,
		set:  attribute.NewSet(),
	}
	return r
}

func (f *distribution) Record(value float64) {
	f.RecordContext(context.Background(), value)
}

func (f *distribution) RecordContext(ctx context.Context, value float64) {
	if set := f.baseMetric.toLabelValues(ctx); set.Len() > 0 {
		f.d.Record(ctx, value, api.WithAttributeSet(set))
	} else {
		f.d.Record(ctx, value)
	}
}

func (f *distribution) With(labelValues ...telemetry.LabelValue) telemetry.Metric {
	nm := &distribution{
		d: f.d,
	}
	keysMap := make(map[tag.Key]bool)
	for k := range f.keys {
		keysMap[k] = true
	}

	nm.baseMetric = baseMetric{
		ms:   f.ms,
		name: f.name,
		rest: nm,
		keys: keysMap,
	}
	nm.lvs, nm.set = f.baseMetric.withLabelValues(labelValues...)
	return nm
}
