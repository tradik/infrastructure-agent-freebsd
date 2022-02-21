package instrumentation

import (
	"github.com/newrelic/infrastructure-agent/pkg/config"
	"github.com/newrelic/infrastructure-agent/pkg/sysinfo/hostname"
	"strings"
)

func InitSelfInstrumentation(c *config.Config, resolver hostname.Resolver) {
	if strings.ToLower(c.SelfInstrumentation) == "apm" && c.SelfInstrumentationLicenseKey != "" {
		apmSelfInstrumentation, err := NewAgentInstrumentationApm(
			c.SelfInstrumentationLicenseKey,
			c.SelfInstrumentationApmEndpoint,
			c.SelfInstrumentationTelemetryEndpoint,
			resolver,
		)
		if err == nil {
			SelfInstrumentation = apmSelfInstrumentation
		}
	}
}
