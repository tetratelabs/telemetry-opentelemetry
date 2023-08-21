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
	api "go.opentelemetry.io/otel/metric"

	"github.com/tetratelabs/telemetry-opentelemetry/internal/tag"
)

type counter struct {
	baseMetric
	c api.Float64Counter
}

var _ telemetry.Metric = (*counter)(nil)

func (m *metricSink) newCounter(name, description string, o telemetry.MetricOptions) *counter {
	c, err := m.meter.Float64Counter(name,
		api.WithDescription(description),
		api.WithUnit(string(o.Unit)))
	if err != nil {
		log.Error("failed to create counter", err)
	}
	r := &counter{c: c}
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
	}
	return r
}

func (f *counter) Record(value float64) {
	f.RecordContext(context.Background(), value)
}

func (f *counter) RecordContext(ctx context.Context, value float64) {
	if set := f.baseMetric.toLabelValues(ctx); set.Len() > 0 {
		f.c.Add(ctx, value, api.WithAttributeSet(set))
	} else {
		f.c.Add(ctx, value)
	}
}

func (f *counter) With(labelValues ...telemetry.LabelValue) telemetry.Metric {
	nm := &counter{
		c: f.c,
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
