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
)

type disabledMetric struct {
	name string
}

// Decrement implements Metric
func (dm *disabledMetric) Decrement() {}

// Increment implements Metric
func (dm *disabledMetric) Increment() {}

// Name implements Metric
func (dm *disabledMetric) Name() string {
	return dm.name
}

// Record implements telemetry.Metric
func (dm *disabledMetric) Record(float64) {}

// RecordContext implements telemetry.Metric
func (dm *disabledMetric) RecordContext(context.Context, float64) {}

// With implements telemetry.Metric
func (dm *disabledMetric) With(...telemetry.LabelValue) telemetry.Metric {
	return dm
}

var _ telemetry.Metric = (*disabledMetric)(nil)
