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
	"testing"

	"github.com/tetratelabs/telemetry"

	opentelemetry "github.com/tetratelabs/telemetry-opentelemetry"
	"github.com/tetratelabs/telemetry-opentelemetry/internal/monitortest"
)

var (
	ms   opentelemetry.MetricAndDerivedMetricSink
	name telemetry.Label
	kind telemetry.Label

	testSum                telemetry.Metric
	goofySum               telemetry.Metric
	hookSum                telemetry.Metric
	testDistribution       telemetry.Metric
	testGauge              telemetry.Metric
	testDisabledSum        telemetry.Metric
	testConditionalSum     telemetry.Metric
	testDerivedGauge       telemetry.DerivedMetric
	testDerivedGaugeLabels telemetry.DerivedMetric
)

func init() {
	ms = opentelemetry.New("test")
	telemetry.SetGlobalMetricSink(ms)

	name = ms.NewLabel("name")
	kind = ms.NewLabel("kind")

	testSum = ms.NewSum(
		"events_total",
		"Number of events observed, by name and kind",
	)

	goofySum = testSum.With(kind.Upsert("goofy"))

	hookSum = ms.NewSum(
		"hook_total",
		"Number of hook events observed",
	)

	testDistribution = ms.NewDistribution(
		"test_buckets",
		"Testing distribution functionality",
		[]float64{0, 2.5, 7, 8, 10, 99, 154.3},
		telemetry.WithUnit(telemetry.Seconds),
	)

	testGauge = ms.NewGauge(
		"test_gauge",
		"Testing gauge functionality",
	)

	testDisabledSum = ms.NewSum(
		"events_disabled_total",
		"Number of events observed, by name and kind",
		telemetry.WithEnabled(func() bool { return false }),
	)

	testConditionalSum = ms.NewSum(
		"events_conditional_total",
		"Number of events observed, by name and kind",
		telemetry.WithEnabled(func() bool { return true }),
	)

	testDerivedGauge = ms.NewDerivedGauge(
		"test_derived_gauge",
		"Testing derived gauge functionality",
	).ValueFrom(func() float64 {
		return 17.76
	})

	testDerivedGaugeLabels = ms.NewDerivedGauge(
		"test_derived_gauge_labels",
		"Testing derived gauge functionality",
	)

	monitortest.SetSink(ms)
}

func TestMonitorTestReset(t *testing.T) {
	t.Run("initial", func(t *testing.T) {
		mt := monitortest.New(t)
		testSum.With(name.Upsert("foo"), kind.Upsert("bar")).Increment()
		mt.Assert(testSum.Name(), map[string]string{"kind": "bar"}, monitortest.Exactly(1))
	})
	t.Run("secondary", func(t *testing.T) {
		mt := monitortest.New(t)
		testSum.With(name.Upsert("foo"), kind.Upsert("bar2")).Increment()
		// Should have been reset
		mt.Assert(testSum.Name(), map[string]string{"kind": "bar"}, monitortest.Exactly(0))
		mt.Assert(testSum.Name(), map[string]string{"kind": "bar2"}, monitortest.Exactly(1))
	})
}

func TestSum(t *testing.T) {
	mt := monitortest.New(t)

	testSum.With(name.Upsert("foo"), kind.Upsert("bar")).Increment()
	goofySum.With(name.Upsert("baz")).Record(45)
	goofySum.With(name.Upsert("baz")).Decrement()

	mt.Assert(goofySum.Name(), map[string]string{"name": "baz"}, monitortest.Exactly(44))
	mt.Assert(testSum.Name(), map[string]string{"kind": "bar"}, monitortest.Exactly(1))
}

func TestRegisterIfSum(t *testing.T) {
	mt := monitortest.New(t)

	testDisabledSum.With(name.Upsert("foo"), kind.Upsert("bar")).Increment()
	mt.Assert(testDisabledSum.Name(), nil, monitortest.DoesNotExist)

	testConditionalSum.With(name.Upsert("foo"), kind.Upsert("bar")).Increment()
	mt.Assert(testConditionalSum.Name(), map[string]string{"name": "foo", "kind": "bar"}, monitortest.Exactly(1))
}

func TestGauge(t *testing.T) {
	mt := monitortest.New(t)

	testGauge.Record(42)
	testGauge.Record(77)

	mt.Assert(testGauge.Name(), nil, monitortest.Exactly(77))
}

func TestGaugeLabels(t *testing.T) {
	mt := monitortest.New(t)

	testGauge.With(kind.Upsert("foo")).Record(42)
	testGauge.With(kind.Upsert("bar")).Record(77)
	testGauge.With(kind.Upsert("bar")).Record(72)

	mt.Assert(testGauge.Name(), map[string]string{"kind": "foo"}, monitortest.Exactly(42))
	mt.Assert(testGauge.Name(), map[string]string{"kind": "bar"}, monitortest.Exactly(72))
}

func TestDerivedGauge(t *testing.T) {
	mt := monitortest.New(t)
	mt.Assert(testDerivedGauge.Name(), nil, monitortest.Exactly(17.76))
}

func TestDerivedGaugeWithLabels(t *testing.T) {
	t.Skipf("skip for now")
	foo := ms.NewLabel("foo")
	testDerivedGaugeLabels.ValueFrom(
		func() float64 {
			return 17.76
		},
		foo.Upsert("bar"),
	)

	testDerivedGaugeLabels.ValueFrom(
		func() float64 {
			return 18.12
		},
		foo.Upsert("baz"),
	)

	mt := monitortest.New(t)

	cases := []struct {
		wantLabel string
		wantValue float64
	}{
		{"bar", 17.76},
		{"baz", 18.12},
	}
	for _, tc := range cases {
		t.Run(tc.wantLabel, func(tt *testing.T) {
			mt.Assert(testDerivedGaugeLabels.Name(), map[string]string{"foo": tc.wantLabel}, monitortest.Exactly(tc.wantValue))
		})
	}
}

func TestDistribution(t *testing.T) {
	mt := monitortest.New(t)

	funDistribution := testDistribution.With(name.Upsert("fun"))
	funDistribution.Record(7.7773)
	testDistribution.With(name.Upsert("foo")).Record(7.4)
	testDistribution.With(name.Upsert("foo")).Record(6.8)
	testDistribution.With(name.Upsert("foo")).Record(10.2)

	mt.Assert(testDistribution.Name(), map[string]string{"name": "fun"}, monitortest.Distribution(1, 7.7773))
	mt.Assert(testDistribution.Name(), map[string]string{"name": "foo"}, monitortest.Distribution(3, 24.4))
	mt.Assert(testDistribution.Name(), map[string]string{"name": "foo"}, monitortest.Buckets(7))
}

//func TestRecordHook(t *testing.T) {
//	mt := monitortest.New(t)
//
//	// testRecordHook will record value for hookSum measure when testSum is recorded
//	rh := &testRecordHook{}
//	ms.RegisterRecordHook(testSum.Name(), rh)
//
//	testSum.With(name.Upsert("foo"), kind.Upsert("bart")).Increment()
//	testSum.With(name.Upsert("baz"), kind.Upsert("bart")).Record(45)
//
//	mt.Assert(testSum.Name(), map[string]string{"name": "foo", "kind": "bart"}, monitortest.Exactly(1))
//	mt.Assert(testSum.Name(), map[string]string{"name": "baz", "kind": "bart"}, monitortest.Exactly(45))
//	mt.Assert(hookSum.Name(), map[string]string{"name": "foo"}, monitortest.Exactly(1))
//	mt.Assert(hookSum.Name(), map[string]string{"name": "baz"}, monitortest.Exactly(45))
//}
//
//type testRecordHook struct{}
//
//func (r *testRecordHook) OnRecord(n string, tags []monitoring.LabelValue, value float64) {
//	// Check if this is `events_total` metric.
//	if n != "events_total" {
//		return
//	}
//
//	// Get name tag of recorded testSum metric, and record the corresponding hookSum metric.
//	var nv string
//	for _, tag := range tags {
//		if tag.Key() == name {
//			nv = tag.Value()
//			break
//		}
//	}
//	hookSum.With(name.Upsert(nv)).Record(value)
//}

func BenchmarkCounter(b *testing.B) {
	monitortest.New(b)
	b.Run("no labels", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			testSum.Increment()
		}
	})
	b.Run("dynamic labels", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			testSum.With(name.Upsert("test")).Increment()
		}
	})
	b.Run("static labels", func(b *testing.B) {
		testSum := testSum.With(name.Upsert("test"))
		for n := 0; n < b.N; n++ {
			testSum.Increment()
		}
	})
}

func BenchmarkGauge(b *testing.B) {
	monitortest.New(b)
	b.Run("no labels", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			testGauge.Increment()
		}
	})
	b.Run("dynamic labels", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			testGauge.With(name.Upsert("test")).Increment()
		}
	})
	b.Run("static labels", func(b *testing.B) {
		testGauge := testGauge.With(name.Upsert("test"))
		for n := 0; n < b.N; n++ {
			testGauge.Increment()
		}
	})
}
