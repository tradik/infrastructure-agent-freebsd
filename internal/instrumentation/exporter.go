// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/api/metric"
	oprometheus "go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/label"
)

type instrumentation struct {
	handler  *oprometheus.Exporter
	meter    *metric.Meter
	counters map[MetricName]metric.Int64Counter
	gauges   map[MetricName]metric.Int64ValueRecorder
}

func (i instrumentation) GetHandler() http.Handler {
	return i.handler
}

func (i instrumentation) Measure(metricType MetricType, name MetricName, val int64) {
	switch metricType {
	case Gauge:
		i.meter.RecordBatch(
			context.Background(),
			[]label.KeyValue{},
			i.gauges[name].Measurement(val))
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
	prometheusExporter, err := oprometheus.InstallNewPipeline(oprometheus.Config{
		Registry: registry,
	})
	if err != nil {
		return nil, err
	}
	meter := prometheusExporter.MeterProvider().Meter("newrelic.infra")

	counters := make(map[MetricName]metric.Int64Counter, 2)
	gauges := make(map[MetricName]metric.Int64ValueRecorder, 2)

	for metricName, metricRegistrationName := range MetricsToRegister {
		counters[metricName] = metric.Must(meter).NewInt64Counter("newrelic.infra/instrumentation." + metricRegistrationName)
	}

	gauges[EventQueueDepthCapacity] = metric.Must(meter).NewInt64ValueRecorder("newrelic.infra/instrumentation." + "event_queue_depth_capacity")

	return &instrumentation{
		handler:  prometheusExporter,
		counters: counters,
		gauges:   gauges,
		meter:    &meter,
	}, err
}
