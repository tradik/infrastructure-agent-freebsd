package instrumentation

import (
	"github.com/newrelic/infrastructure-agent/pkg/config"
	"strings"
)

func InitSelfInstrumentation(c *config.Config) {
	if strings.ToLower(c.SelfInstrumentation) == "apm" && c.SelfInstrumentationLicenseKey != "" {
		apmSelfInstrumentation, err := NewAgentInstrumentationApm(
			c.SelfInstrumentationLicenseKey,
			c.DisplayName,
			c.SelfInstrumentationApmEndpoint,
			c.SelfInstrumentationTelemetryEndpoint,
		)
		if err == nil {
			SelfInstrumentation = apmSelfInstrumentation
		}
	}
}
