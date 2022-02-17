// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/api/metric"
	oprometheus "go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/label"
	"net/http"
)

type instrumentation struct {
	handler  *oprometheus.Exporter
	meter    *metric.Meter
	counters map[MetricName]metric.Int64Counter
	gauges   map[MetricName]prometheus.Gauge
}

func (i instrumentation) GetHandler() http.Handler {
	return i.handler
}

func (i instrumentation) Measure(metricType MetricType, name MetricName, val int64) {
	switch metricType {
	case Gauge:
		i.gauges[name].Set(float64(val))
	case Counter:
		i.meter.RecordBatch(
			context.Background(),
			[]label.KeyValue{},
			i.counters[name].Measurement(val))
	default:
	}
}

func (i instrumentation) GetHttpTransport(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(base,
		otelhttp.WithMeterProvider(i.handler.MeterProvider()),
		otelhttp.WithMessageEvents(
			otelhttp.ReadEvents,
			otelhttp.WriteEvents))
}

// New creates a new instrumentation bundle (exporter + measure fn...).
func New() (Instrumenter, error) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())

	gauges := make(map[MetricName]prometheus.Gauge, 2)

	for metricName, metricRegistrationName := range GaugeMetricsToRegister {
		opsQueued := prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "newrelic_infra",
			Subsystem: "instrumentation",
			Name:      metricRegistrationName,
			Help:      "TODO",
		})

		registry.MustRegister(opsQueued)
		gauges[metricName] = opsQueued
	}

	prometheusExporter, err := oprometheus.InstallNewPipeline(oprometheus.Config{
		Registry: registry,
	})
	if err != nil {
		return nil, err
	}
	meter := prometheusExporter.MeterProvider().Meter("newrelic.infra")

	counters := make(map[MetricName]metric.Int64Counter, 2)

	for metricName, metricRegistrationName := range CounterMetricsToRegister {
		counters[metricName] = metric.Must(meter).NewInt64Counter("newrelic.infra/instrumentation." + metricRegistrationName)
	}

	return &instrumentation{
		handler:  prometheusExporter,
		counters: counters,
		gauges:   gauges,
		meter:    &meter,
	}, err
}
