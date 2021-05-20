// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/newrelic/infrastructure-agent/cmd/newrelic-infra/initialize"
	"github.com/newrelic/infrastructure-agent/internal/feature_flags"
	"github.com/newrelic/infrastructure-agent/internal/integrations/v4/files"
	"github.com/newrelic/infrastructure-agent/internal/integrations/v4/integration"
	"github.com/newrelic/infrastructure-agent/internal/integrations/v4/v3legacy"
	"github.com/newrelic/infrastructure-agent/pkg/backend/identityapi"
	"github.com/newrelic/infrastructure-agent/pkg/config"
	"github.com/newrelic/infrastructure-agent/pkg/helpers"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/configrequest"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/legacy"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/track"
	v4 "github.com/newrelic/infrastructure-agent/pkg/integrations/v4"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/v4/dm"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/v4/emitter"
	wlog "github.com/newrelic/infrastructure-agent/pkg/log"
	metrics_sender "github.com/newrelic/infrastructure-agent/pkg/metrics/sender"
	"github.com/newrelic/infrastructure-agent/pkg/plugins"
	"github.com/newrelic/infrastructure-agent/test/infra"
	"github.com/newrelic/infrastructure-agent/test/proxy/minagent"
	"github.com/sirupsen/logrus"
)



// minimalist agent. It loads the configuration from the environment and the file passed by the -config flag.
// It just submits `FakeSample` instances to the collector.
func main() {
	malog := logrus.WithField("component", "minimal-standalone-agent")

	logrus.Info("Runing minimalistic test agent...")
	runtime.GOMAXPROCS(1)

	configFile := flag.String("config", minagent.DefaultConfig, "configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		malog.WithError(err).Fatal("can't load configuration file")
	}

	if cfg.CABundleFile == "" && cfg.CABundleDir == "" {
		cfg.CABundleDir = "/cabundle"
	}
	cfg.PayloadCompressionLevel = gzip.NoCompression


	initializeAgentAndRun(malog,cfg)

}


var aslog = wlog.WithComponent("AgentService").WithFields(logrus.Fields{
	"service": "coreint",
})

func initializeAgentAndRun(malog *logrus.Entry, cfg *config.Config) error {
	a := infra.NewAgentFromConfig(cfg)
	sender := metrics_sender.NewSender(a.Context)
	sender.RegisterSampler(&minagent.FakeSampler{})
	a.RegisterMetricsSender(sender)
	if err := a.Run(); err != nil {
		malog.WithError(err).Error("while starting agent")
	}


	pluginSourceDirs := []string{
		cfg.CustomPluginInstallationDir,
		filepath.Join(cfg.AgentDir, "custom-integrations"),
		filepath.Join(cfg.AgentDir, config.DefaultIntegrationsDir),
		filepath.Join(cfg.AgentDir, "bundled-plugins"),
		filepath.Join(cfg.AgentDir, "plugins"),
	}
	pluginSourceDirs = helpers.RemoveEmptyAndDuplicateEntries(pluginSourceDirs)

	integrationCfg := v4.NewConfig(
		cfg.Verbose,
		cfg.Features,
		cfg.PassthroughEnvironment,
		cfg.PluginInstanceDirs,
		pluginSourceDirs,
	)

	ffManager := feature_flags.NewManager(cfg.Features)


	fatal := func(err error, message string) {
		aslog.WithError(err).Error(message)
		os.Exit(1)
	}


	defer a.Terminate()

	if err := initialize.AgentService(cfg); err != nil {
		fatal(err, "Can't complete platform specific initialization.")
	}

	metricsSenderConfig := dm.NewConfig(cfg.MetricURL, cfg.License, time.Duration(cfg.DMSubmissionPeriod)*time.Second, cfg.MaxMetricBatchEntitiesCount, cfg.MaxMetricBatchEntitiesQueue)

	dmSender, err := dm.NewDMSender(metricsSenderConfig, http.DefaultTransport, a.Context.IdContext().AgentIdentity)
	if err != nil {
		return err
	}

	// queues integration run requests
	definitionQ := make(chan integration.Definition, 100)
	// queues config entries requests
	configEntryQ := make(chan configrequest.Entry, 100)
	// queues integration terminated definitions
	terminateDefinitionQ := make(chan string, 100)
	var registerClient identityapi.RegisterClient
	emitterWithRegister := dm.NewEmitter(a.GetContext(), dmSender, registerClient)
	nonRegisterEmitter := dm.NewNonRegisterEmitter(a.GetContext(), dmSender)

	dmEmitter := dm.NewEmitterWithFF(emitterWithRegister, nonRegisterEmitter, ffManager)

	// track stoppable integrations
	tracker := track.NewTracker(dmEmitter)
	il:=newInstancesLookup(integrationCfg)
	integrationEmitter := emitter.NewIntegrationEmittor(a, dmEmitter, ffManager)
	integrationManager := v4.NewManager(integrationCfg, integrationEmitter, il, definitionQ, terminateDefinitionQ, configEntryQ, tracker)


	// Start all plugins we want the agent to run.
	if err = plugins.RegisterPlugins(a, integrationEmitter); err != nil {
		aslog.WithError(err).Error("fatal error while registering plugins")
		os.Exit(1)
	}

	go integrationManager.Start(a.Context.Ctx)
	pluginRegistry := legacy.NewPluginRegistry(pluginSourceDirs, cfg.PluginInstanceDirs)
	if err := pluginRegistry.LoadPlugins(); err != nil {
		fatal(err, "Can't load plugins.")
	}
	pluginConfig, err := legacy.LoadPluginConfig(pluginRegistry, cfg.PluginConfigFiles)
	if err != nil {
		fatal(err, "Can't load plugin configuration.")
	}
	runner := legacy.NewPluginRunner(pluginRegistry, a)
	for _, pluginConf := range pluginConfig.PluginConfigs {
		if err := runner.ConfigurePlugin(pluginConf, a.Context.ActiveEntitiesChannel()); err != nil {
			fatal(err, fmt.Sprint("Can't configure plugin.", pluginConf))
		}
	}

	if err := runner.ConfigureV1Plugins(a.Context); err != nil {
		aslog.WithError(err).Debug("Can't configure integrations.")
	}

	malog.Info("New Relic infrastructure agent is running.")

	return a.Run()
}

func newInstancesLookup(cfg v4.Configuration) integration.InstancesLookup {
	const executablesSubFolder = "bin"

	var execFolders []string
	for _, df := range cfg.DefinitionFolders {
		execFolders = append(execFolders, df)
		execFolders = append(execFolders, filepath.Join(df, executablesSubFolder))
	}
	legacyDefinedCommands := v3legacy.NewDefinitionsRepo(v3legacy.LegacyConfig{
		DefinitionFolders: cfg.DefinitionFolders,
		Verbose:           cfg.Verbose,
	})
	return integration.InstancesLookup{
		Legacy: legacyDefinedCommands.NewDefinitionCommand,
		ByName: files.Executables{Folders: execFolders}.Path,
	}
}
