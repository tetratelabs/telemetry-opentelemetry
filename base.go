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

	"github.com/tetratelabs/telemetry-opentelemetry/internal/tag"
)

type baseMetric struct {
	ms   *metricSink
	name string
	// attrs stores all attrs for the metrics
	keys map[tag.Key]bool // only allow these dimensions for the metric
	lvs  []tag.Mutator    // used when needing to append to context
	set  attribute.Set    // precomputed if nothing is found in context
	rest telemetry.Metric
}

// Name returns the name value of a Metric.
func (f *baseMetric) Name() string {
	return f.name
}

// Increment records a value of 1 for the current Metric.
// For Sums, this is equivalent to adding 1 to the current value.
// For Gauges, this is equivalent to setting the value to 1.
// For Distributions, this is equivalent to making an observation of value 1.
func (f *baseMetric) Increment() {
	f.rest.Record(1)
}

// Decrement records a value of -1 for the current Metric.
// For Sums, this is equivalent to subtracting -1 to the current value.
// For Gauges, this is equivalent to setting the value to -1.
// For Distributions, this is equivalent to making an observation of value -1.
func (f *baseMetric) Decrement() {
	f.rest.Record(-1)
}

func (f *baseMetric) toLabelValues(ctx context.Context) attribute.Set {
	if f.ms.strictDimensions && len(f.keys) == 0 {
		// we have no registered dimensions
		return attribute.NewSet()
	}
	if ctx == context.Background() {
		// nothing to be found in context, current attribute.Set is up-to-date.
		return f.set
	}
	// calculate LabelValues based on context values and available tag.Mutators.
	ctx, err := tag.New(ctx, f.lvs...)
	if err != nil {
		f.ms.logger.Error("unable to parse tag.Map", err, "metric", f.name)
		return attribute.NewSet()
	}

	return f.tagsToAttributeSet(tag.FromContext(ctx))
}

func (f *baseMetric) withLabelValues(lvs ...telemetry.LabelValue) ([]tag.Mutator, attribute.Set) {
	ret := make([]tag.Mutator, len(f.lvs)+len(lvs))
	copy(ret, f.lvs)
	for i := 0; i < len(lvs); i++ {
		ret[len(f.lvs)+i] = lvs[i].(tag.Mutator)
	}
	ctx, err := tag.New(context.Background(), ret...)
	if err != nil {
		f.ms.logger.Error("unable to parse LabelValues", err, "metric", f.name)
		return ret, attribute.NewSet()
	}

	return ret, f.tagsToAttributeSet(tag.FromContext(ctx))
}

func (f *baseMetric) tagsToAttributeSet(tm *tag.Map) attribute.Set {
	var kvs []attribute.KeyValue
	if tm == nil || tm.Len() == 0 {
		return attribute.NewSet()
	}
	if f.ms.strictDimensions {
		for k := range f.keys {
			if val, ok := tm.Value(k); ok {
				kvs = append(kvs, attribute.String(k.Name(), val))
			}
		}
	} else {
		tm.Iterate(func(t tag.Tag) {
			kvs = append(kvs, attribute.String(t.Key.Name(), t.Value))
		})
	}
	return attribute.NewSet(kvs...)
}
