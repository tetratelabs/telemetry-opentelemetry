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

type derivedGauge struct {
	mu    sync.RWMutex
	attrs map[attribute.Set]func() float64

	name string
}

var _ telemetry.DerivedMetric = (*derivedGauge)(nil)

func (m *metricSink) newDerivedGauge(name, description string) telemetry.DerivedMetric {
	dm := &derivedGauge{
		name:  name,
		attrs: map[attribute.Set]func() float64{},
	}
	_, err := m.meter.Float64ObservableGauge(name,
		api.WithDescription(description),
		api.WithFloat64Callback(func(ctx context.Context, observer api.Float64Observer) error {
			dm.mu.RLock()
			defer dm.mu.RUnlock()
			for kv, compute := range dm.attrs {
				observer.Observe(compute(), api.WithAttributeSet(kv))
			}
			return nil
		}))
	if err != nil {
		log.Error("failed to create derived gauge", err)
	}
	return dm
}

func (d *derivedGauge) Name() string {
	return d.name
}

func (d *derivedGauge) ValueFrom(valueFn func() float64, labelValues ...telemetry.LabelValue) telemetry.DerivedMetric {
	d.mu.Lock()
	defer d.mu.Unlock()
	m := make([]tag.Mutator, len(labelValues))
	for i := 0; i < len(labelValues); i++ {
		m[i] = labelValues[i].(tag.Mutator)
	}
	ctx, err := tag.New(context.Background(), m...)
	if err != nil {
		log.Error("unable to parse LabelValues", err, "metric", d.name)
		as := attribute.NewSet()
		d.attrs[as] = valueFn
		return d
	}
	tm := tag.FromContext(ctx)
	lv := make([]attribute.KeyValue, tm.Len())
	tm.Iterate(func(t tag.Tag) {
		lv = append(lv, attribute.String(t.Key.Name(), t.Value))
	})
	as := attribute.NewSet(lv...)
	d.attrs[as] = valueFn
	return d
}
