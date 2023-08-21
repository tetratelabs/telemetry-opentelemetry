package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tetratelabs/telemetry"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	opentelemetry "github.com/tetratelabs/telemetry-opentelemetry"
)

// RegisterPrometheusExporter sets the global metrics handler to the provided
// Prometheus registerer and gatherer.
// Returned is an HTTP handler that can be used to read metrics from.
func RegisterPrometheusExporter(
	ms telemetry.MetricSink, reg prometheus.Registerer, gat prometheus.Gatherer,
) (http.Handler, error) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	if gat == nil {
		gat = prometheus.DefaultGatherer
	}
	promOpts := []otelprom.Option{
		otelprom.WithoutScopeInfo(),
		otelprom.WithoutTargetInfo(),
		otelprom.WithoutUnits(),
		otelprom.WithRegisterer(reg),
		otelprom.WithoutCounterSuffixes(),
	}

	prom, err := otelprom.New(promOpts...)
	if err != nil {
		return nil, err
	}

	opts := []metric.Option{metric.WithReader(prom)}
	opts = append(opts, opentelemetry.Start(ms)...)

	mp := metric.NewMeterProvider(opts...)
	otel.SetMeterProvider(mp)
	handler := promhttp.HandlerFor(gat, promhttp.HandlerOpts{})
	return handler, nil
}
