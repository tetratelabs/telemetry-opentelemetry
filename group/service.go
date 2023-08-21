// Copyright (c) Tetrate, Inc 2021.
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

// Package group provides a tetratelabs/run Group compatible OpenCensus metrics
// service.
package group

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/tetratelabs/multierror"
	"github.com/tetratelabs/run"
	"go.opencensus.io/stats/view"
)

// Exported flags.
const (
	NoPrometheus        = "disable-prometheus"
	PrometheusAddress   = "prometheus-address"
	PrometheusEndpoint  = "prometheus-endpoint"
	PrometheusNamespace = "prometheus-namespace"

	OpenCensus          = "enable-opencensus"
	OpenCensusAddress   = "opencensus-address"
	OpenCensusService   = "opencensus-servicename"
	OpenCensusReport    = "opencensus-report-interval"
	OpenCensusReconnect = "opencensus-reconnect-interval"
)

// Default configuration values.
const (
	DefaultPrometheusAddress   = ":42422"
	DefaultPrometheusEndpoint  = "/metrics"
	DefaultPrometheusNamespace = ""

	DefaultOpenCensusAddress           = "otel-collector:9091"
	DefaultOpenCensusServiceName       = ""
	DefaultOpenCensusReportInterval    = 10 * time.Second
	DefaultOpenCensusReconnectInterval = 10 * time.Second
)

const (
	flagErr = "--%s error: %w"

	errInvalidPath run.Error = "invalid path"
)

type Config struct {
	// ExternalPromWebHandler will disable the ability to configure and run the
	// Prometheus exporter from this group Config handler, as it is handled
	// externally.
	ExternalPromWebHandler bool

	// NoPrometheus will disable configuring the Prometheus exporter in favor.
	NoPrometheus bool

	// PrometheusAddress sets the scraping address to listen on.
	PrometheusAddress string

	// PrometheusEndpoint sets the scraping endpoint.
	PrometheusEndpoint string

	// PrometheusNamespace sets the namespace (prefix) for Prometheus metrics.
	PrometheusNamespace string

	// OpenCensus enables metric streaming to OpenCensus / OpenTelemetry service.
	OpenCensus bool

	// OpenCensusAddress of OpenCensus / OpenTelemetry service to connect to.
	OpenCensusAddress string

	// OpenCensusServiceName to use for outgoing metrics.
	OpenCensusServiceName string

	// OpenCensusReportInterval sets the metric views default reporting interval.
	OpenCensusReportInterval time.Duration

	// OpenCensusReconnectInterval sets the reconnection interval to use when disconnected.
	OpenCensusReconnectInterval time.Duration
}

// Service implements tetratelabs/run Group interfaces.
type Service interface {
	run.Config
	run.PreRunner
	run.Service
}

type service struct {
	config     Config
	forwarder  *ocagent.Exporter
	prometheus *prometheus.Exporter
	server     *http.Server
	listen     net.Listener
	close      chan struct{}
}

// New takes a config and creates a tetratelabs/run Group compatible metrics
// service for our tetratelabs/telemetry metrics implementation.
func New(config Config) Service {
	return &service{
		config: config,
	}
}

// Name implements run.Unit.
func (s *service) Name() string {
	return "metrics-manager"
}

// FlagSet implements run.Config.
func (s *service) FlagSet() *run.FlagSet {
	fs := run.NewFlagSet("Metrics options")

	if s.config.PrometheusAddress == "" {
		s.config.PrometheusAddress = DefaultPrometheusAddress
	}

	if s.config.PrometheusEndpoint == "" {
		s.config.PrometheusEndpoint = DefaultPrometheusEndpoint
	}

	if s.config.PrometheusNamespace == "" {
		s.config.PrometheusNamespace = DefaultPrometheusNamespace
	}

	if !s.config.ExternalPromWebHandler {
		fs.BoolVar(&s.config.NoPrometheus, NoPrometheus, s.config.NoPrometheus,
			"disable Prometheus scraping endpoint")
		fs.StringVar(&s.config.PrometheusAddress, PrometheusAddress, s.config.PrometheusAddress,
			"address to serve Prometheus from")
		fs.StringVar(&s.config.PrometheusEndpoint, PrometheusEndpoint, s.config.PrometheusEndpoint,
			"endpoint to serve Prometheus from")
		fs.StringVar(&s.config.PrometheusNamespace, PrometheusNamespace, s.config.PrometheusNamespace,
			"namespace (prefix) for Prometheus metrics")
	} else {
		s.config.NoPrometheus = true
	}

	if s.config.OpenCensusAddress == "" {
		s.config.OpenCensusAddress = DefaultOpenCensusAddress
	}
	if s.config.OpenCensusServiceName == "" {
		s.config.OpenCensusServiceName = DefaultOpenCensusServiceName
	}
	if s.config.OpenCensusReportInterval == 0 {
		s.config.OpenCensusReportInterval = DefaultOpenCensusReportInterval
	}
	if s.config.OpenCensusReconnectInterval == 0 {
		s.config.OpenCensusReconnectInterval = DefaultOpenCensusReconnectInterval
	}

	fs.BoolVar(&s.config.OpenCensus, OpenCensus, s.config.OpenCensus,
		"enable OpenCensus agent")
	fs.StringVar(&s.config.OpenCensusAddress, OpenCensusAddress, s.config.OpenCensusAddress,
		"address of OpenCensus / OpenTelemetry service")
	fs.StringVar(&s.config.OpenCensusServiceName, OpenCensusService, s.config.OpenCensusServiceName,
		"service name for OpenCensus metrics")
	fs.DurationVar(&s.config.OpenCensusReportInterval, OpenCensusReport, s.config.OpenCensusReportInterval,
		"OpenCensus view reporting interval")
	fs.DurationVar(&s.config.OpenCensusReconnectInterval, OpenCensusReconnect, s.config.OpenCensusReconnectInterval,
		"OpenCensus reconnection interval")

	return fs
}

// Validate implements run.Config.
func (s service) Validate() error {
	var mErr error

	if !s.config.NoPrometheus {
		if _, _, err := net.SplitHostPort(s.config.PrometheusAddress); err != nil {
			mErr = multierror.Append(mErr,
				fmt.Errorf(flagErr, PrometheusAddress, err))
		}
		if len(s.config.PrometheusEndpoint) < 1 || s.config.PrometheusEndpoint[0] != '/' {
			mErr = multierror.Append(mErr,
				fmt.Errorf(flagErr, PrometheusEndpoint, errInvalidPath))
		}
	}

	if s.config.OpenCensus {
		if _, _, err := net.SplitHostPort(s.config.OpenCensusAddress); err != nil {
			mErr = multierror.Append(mErr,
				fmt.Errorf(flagErr, OpenCensusAddress, err))
		}
	}

	return mErr
}

// PreRun implements run.PreRunner.
func (s *service) PreRun() error {
	s.close = make(chan struct{})

	if !s.config.NoPrometheus {
		var err error
		if s.prometheus, err = prometheus.NewExporter(prometheus.Options{}); err != nil {
			return fmt.Errorf("could not set up Prometheus exporter: %v", err)
		}
		view.RegisterExporter(s.prometheus)
	}

	if s.config.OpenCensus {
		var err error
		if s.forwarder, err = ocagent.NewUnstartedExporter(
			ocagent.WithAddress(s.config.OpenCensusAddress),
			ocagent.WithInsecure(),
		); err != nil {
			return fmt.Errorf("could not set up OpenCensus forwarder: %v", err)
		}
		view.RegisterExporter(s.forwarder)
		view.SetReportingPeriod(s.config.OpenCensusReportInterval)
	}
	return nil
}

// Serve implements run.Service.
func (s *service) Serve() error {
	if !s.config.NoPrometheus {
		m := http.NewServeMux()
		m.Handle(s.config.PrometheusEndpoint, s.prometheus)
		s.server = &http.Server{Handler: m}
		var err error
		if s.listen, err = net.Listen("tcp", s.config.PrometheusAddress); err != nil {
			return fmt.Errorf("unable to start prometheus service on %s%s: %w",
				s.config.PrometheusAddress, s.config.PrometheusEndpoint, err)
		}

		go func() {
			_ = s.server.Serve(s.listen)
		}()
	}
	if s.config.OpenCensus {
		err := s.forwarder.Start()
		if err != nil {
			return fmt.Errorf("could not set up OpenCensus forwarder: %v", err)
		}
	}
	<-s.close
	return nil
}

// GracefulStop implements run.Service.
func (s *service) GracefulStop() {
	if s.forwarder != nil {
		_ = s.forwarder.Stop()
	}
	if s.listen != nil {
		_ = s.listen.Close()
	}
	close(s.close)
}
