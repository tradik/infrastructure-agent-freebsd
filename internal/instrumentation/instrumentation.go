// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package instrumentation

import (
	"net/http"
)

type MetricType int

const (
	Counter MetricType = iota
	Gauge
)

type MetricName int

const (
	DMRequestsForwarded MetricName = iota // integration payload received
	DMDatasetsReceived
	EntityRegisterEntitiesRegistered
	EntityRegisterEntitiesRegisteredWithWarning
	EntityRegisterEntitiesRegistrationFailed
	LoggedErrors
	EventQueueDepthCapacity
	EventQueueDepthSize
	EventQueueDepthUtilization
	BatchQueueDepthCapacity
	BatchQueueDepthSize
	BatchQueueDepthUtilization
)

var (
	CounterMetricsToRegister = map[MetricName]string{
		DMRequestsForwarded:                         "dm.requests_forwarded",
		DMDatasetsReceived:                          "dm.datasets_received",
		EntityRegisterEntitiesRegistered:            "entity_register.entities_registered",
		EntityRegisterEntitiesRegisteredWithWarning: "entity_register.entities_registered_with_warning",
		EntityRegisterEntitiesRegistrationFailed:    "entity_register.entities_registration_failed",
		LoggedErrors:                                "logged.errors",
	}
	GaugeMetricsToRegister = map[MetricName]string{
		EventQueueDepthCapacity:    "event_queue_depth_capacity",
		EventQueueDepthSize:        "event_queue_depth_size",
		EventQueueDepthUtilization: "event_queue_depth_utilization",
		BatchQueueDepthCapacity:    "batch_queue_depth_capacity",
		BatchQueueDepthSize:        "batch_queue_depth_size",
		BatchQueueDepthUtilization: "batch_queue_depth_utilization",
	}
)

type Measure func(metricType MetricType, name MetricName, val int64)

type Instrumenter interface {
	GetHandler() http.Handler
	Measure(metricType MetricType, name MetricName, val int64)
	GetHttpTransport(base http.RoundTripper) http.RoundTripper
}
