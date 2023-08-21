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
	"sync"

	"github.com/tetratelabs/telemetry"
	"go.opentelemetry.io/otel/attribute"
	api "go.opentelemetry.io/otel/metric"

	"github.com/tetratelabs/telemetry-opentelemetry/internal/tag"
)

type gauge struct {
	baseMetric
	g api.Float64ObservableGauge

	// attributeSets stores a map of attributes -> values, for gauges.
	attributeSetsMutex *sync.RWMutex
	attributeSets      map[attribute.Set]*gaugeValues
	currentGaugeSet    *gaugeValues
}

var _ telemetry.Metric = (*gauge)(nil)

func (m *metricSink) newGauge(name, description string, o telemetry.MetricOptions) *gauge {
	r := &gauge{
		attributeSetsMutex: &sync.RWMutex{},
		currentGaugeSet:    &gaugeValues{},
	}
	r.attributeSets = map[attribute.Set]*gaugeValues{
		attribute.NewSet(): r.currentGaugeSet,
	}
	g, err := m.meter.Float64ObservableGauge(name,
		api.WithFloat64Callback(func(ctx context.Context, observer api.Float64Observer) error {
			r.attributeSetsMutex.Lock()
			defer r.attributeSetsMutex.Unlock()
			for _, gv := range r.attributeSets {
				observer.Observe(gv.val, gv.opt...)
			}
			return nil
		}),
		api.WithDescription(description),
		api.WithUnit(string(o.Unit)))
	if err != nil {
		m.logger.Error("failed to create gauge", err)
	}
	r.g = g
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

func (f *gauge) Record(value float64) {
	f.RecordContext(context.Background(), value)
}

func (f *gauge) RecordContext(ctx context.Context, value float64) {
	// TODO: https://github.com/open-telemetry/opentelemetry-specification/issues/2318 use synchronous gauge so we don't need to deal with this
	f.attributeSetsMutex.Lock()
	f.currentGaugeSet.val = value
	f.attributeSetsMutex.Unlock()
}

func (f *gauge) With(labelValues ...telemetry.LabelValue) telemetry.Metric {
	nm := &gauge{
		g:                  f.g,
		attributeSetsMutex: f.attributeSetsMutex,
		attributeSets:      f.attributeSets,
	}
	lvs, set := f.baseMetric.withLabelValues(labelValues...)
	if _, f := nm.attributeSets[set]; !f {
		nm.attributeSets[set] = &gaugeValues{
			opt: []api.ObserveOption{api.WithAttributeSet(set)},
		}
	}
	nm.currentGaugeSet = nm.attributeSets[set]
	keysMap := make(map[tag.Key]bool)
	for k := range f.keys {
		keysMap[k] = true
	}
	nm.baseMetric = baseMetric{
		ms:   f.ms,
		name: f.name,
		keys: keysMap,
		lvs:  lvs,
		rest: nm,
		set:  set,
	}
	return nm
}

type gaugeValues struct {
	val float64
	opt []api.ObserveOption
}
