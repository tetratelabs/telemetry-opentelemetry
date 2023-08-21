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

package opentelemetry_test

import (
	"github.com/tetratelabs/telemetry"

	"github.com/tetratelabs/telemetry-opentelemetry"
)

var pushLatency telemetry.Metric

func Example_newGauge() {
	telemetry.ToGlobalMetricSink(func(m telemetry.MetricSink) {
		pushLatency = m.NewGauge(
			"push_latency_seconds",
			"Duration, measured in seconds, of the last push",
			telemetry.WithUnit(telemetry.Seconds),
		)
	})
	telemetry.SetGlobalMetricSink(opentelemetry.New("example"))

	// only the last recorded value (99.2) will be exported for this gauge
	pushLatency.Record(77.3)
	pushLatency.Record(22.8)
	pushLatency.Record(99.2)
}
