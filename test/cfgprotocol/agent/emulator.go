// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package agent

import (
	"compress/gzip"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/newrelic/infrastructure-agent/cmd/newrelic-infra/initialize"
	"github.com/newrelic/infrastructure-agent/internal/agent"
	"github.com/newrelic/infrastructure-agent/internal/feature_flags"
	"github.com/newrelic/infrastructure-agent/internal/integrations/v4/files"
	"github.com/newrelic/infrastructure-agent/internal/integrations/v4/integration"
	"github.com/newrelic/infrastructure-agent/internal/integrations/v4/v3legacy"
	"github.com/newrelic/infrastructure-agent/pkg/backend/identityapi"
	"github.com/newrelic/infrastructure-agent/pkg/config"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/configrequest"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/track"
	v4 "github.com/newrelic/infrastructure-agent/pkg/integrations/v4"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/v4/dm"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/v4/emitter"
	"github.com/newrelic/infrastructure-agent/pkg/plugins"
	"github.com/newrelic/infrastructure-agent/test/infra"
	ihttp "github.com/newrelic/infrastructure-agent/test/infra/http"
	"github.com/sirupsen/logrus"
)

// func main() {
// 	logrus.SetLevel(logrus.DebugLevel)
// 	logrus.Info("TEXT")
// 	executor := New("/Users/icorrales/Repositories/github.com/newrelic/infrastructure-agent/test/cfgprotocol/agent/integrations.d")
// 	logrus.Debug("TEXT")
// 	go executor.RunAgent()
// 	time.Sleep(20 * time.Second)
// }

var testClient = ihttp.NewRequestRecorderClient()

const timeout = 5 * time.Second

type Emulator struct {
	recordClient   *ihttp.RequestRecorderClient
	agent          *agent.Agent
	integrationCfg v4.Configuration
}

func (ae *Emulator) Client() *ihttp.RequestRecorderClient {
	return ae.recordClient
}

func New(integrationDir string) Emulator {

	agent := infra.NewAgent(testClient.Client, func(config *config.Config) {
		config.DisplayName = "my_display_name"
		config.License = "abcdef012345"
		config.PayloadCompressionLevel = gzip.NoCompression
		config.Verbose = 1
		config.PluginDir = integrationDir
		config.LogFormat = "text"
		config.LogToStdout = true
		config.Debug = true
	})
	cfg := agent.Context.Config()

	integrationCfg := v4.NewConfig(
		cfg.Verbose,
		cfg.Features,
		cfg.PassthroughEnvironment,
		[]string{integrationDir},
		nil,
	)
	return Emulator{
		recordClient:   ihttp.NewRequestRecorderClient(),
		agent:          agent,
		integrationCfg: integrationCfg,
	}
}

func (ae *Emulator) Terminate() {
	ae.agent.Terminate()
}

// minimalist agent. It loads the configuration from the environment and the file passed by the -config flag.
// It just submits `FakeSample` instances to the collector.
func (ae *Emulator) RunAgent() error {
	malog := logrus.WithField("component", "minimal-standalone-agent")
	logrus.Info("Runing minimalistic test agent...")
	runtime.GOMAXPROCS(1)

	cfg := ae.agent.GetContext().Config()

	ffManager := feature_flags.NewManager(cfg.Features)
	fatal := func(err error, message string) {
		malog.WithError(err).Error(message)
		os.Exit(1)
	}

	if err := initialize.AgentService(cfg); err != nil {
		fatal(err, "Can't complete platform specific initialization.")
	}
	metricsSenderConfig := dm.NewConfig(cfg.MetricURL, cfg.License, time.Duration(cfg.DMSubmissionPeriod)*time.Second, cfg.MaxMetricBatchEntitiesCount, cfg.MaxMetricBatchEntitiesQueue)
	dmSender, err := dm.NewDMSender(metricsSenderConfig, http.DefaultTransport, ae.agent.Context.IdContext().AgentIdentity)
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
	emitterWithRegister := dm.NewEmitter(ae.agent.GetContext(), dmSender, registerClient)
	nonRegisterEmitter := dm.NewNonRegisterEmitter(ae.agent.GetContext(), dmSender)

	dmEmitter := dm.NewEmitterWithFF(emitterWithRegister, nonRegisterEmitter, ffManager)

	// track stoppable integrations
	tracker := track.NewTracker(dmEmitter)
	il := newInstancesLookup(ae.integrationCfg)
	integrationEmitter := emitter.NewIntegrationEmittor(ae.agent, dmEmitter, ffManager)
	integrationManager := v4.NewManager(ae.integrationCfg, integrationEmitter, il, definitionQ, terminateDefinitionQ, configEntryQ, tracker)

	// Start all plugins we want the agent to run.
	if err = plugins.RegisterPlugins(ae.agent, integrationEmitter); err != nil {
		malog.WithError(err).Error("fatal error while registering plugins")
		os.Exit(1)
	}
	go integrationManager.Start(ae.agent.Context.Ctx)

	return ae.agent.Run()
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
