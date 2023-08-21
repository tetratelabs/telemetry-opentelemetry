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
)

var (
	protocol telemetry.Label
	requests telemetry.Metric
)

func Example_newSum() {
	telemetry.ToGlobalMetricSink(func(m telemetry.MetricSink) {
		protocol = m.NewLabel("protocol")
		requests = m.NewSum(
			"requests_total",
			"Number of requests handled, by protocol",
		)
	})
	// increment on every http request
	requests.With(protocol.Insert("http")).Increment()

	// count gRPC requests double
	requests.With(protocol.Insert("grpc")).Record(2)
}
