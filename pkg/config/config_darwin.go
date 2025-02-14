// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"path/filepath"

	"github.com/kelseyhightower/envconfig"
)

const (
	defaultConnectEnabled = true
)

func init() {
	defaultConfigFiles = []string{
		"newrelic-infra.yml",
		filepath.Join("/usr", "local", "etc", "newrelic-infra", "newrelic-infra.yml"),
	}
	defaultAgentDir = filepath.Join("/usr", "local", "var", "db", "newrelic-infra")

	// add PATH environment variable to all integrations
	defaultPassthroughEnvironment = []string{"PATH"}
}
func runtimeValues() (userMode, agentUser, executablePath string) {
	return ModeRoot, "", ""
}

func configOverride(cfg *Config) {
	if err := envconfig.Process(envPrefix, cfg); err != nil {
		clog.WithError(err).Error("unable to interpret environment variables")
	}
}
