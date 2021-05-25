// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package cfgprotocol

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/internal/testhelpers"
	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/agent"
	"github.com/shirou/gopsutil/process"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	timeout        = 15 * time.Second
	metricNRIOutV3 = `[{
			"ExternalKeys":["shell-test:some-entity"],
			"IsAgent":false,
			"Events":[
				{
					"displayName":"shell-test:some-entity","entityKey":"shell-test:some-entity","entityName":"shell-test:some-entity",
					"eventType":"ShellTestSample","event_type":"ShellTestSample","integrationName":"nri-test","integrationVersion":"0.0.0",
					"reportingAgent":"my_display_name","some-metric":1
				}
			]
		}]`
	metricNRIOutV4 = `[{
			"ExternalKeys": ["uniqueName"],
			"IsAgent": false,
			"Events": [{"attr.format": "attribute","attributes": {"format": "attribute"},"category": "notifications","entityKey": "uniqueName",
					"eventType": "InfrastructureEvent","format": "event","summary": "foo"}]
			}]`
	defNriOutExecution     = "go run testdata/go/spawner.go -path testdata/scenarios/shared/nri-out.json -singleLine"
	defNriOutLongExecution = "go run testdata/go/spawner.go -path testdata/scenarios/shared/nri-out.json -singleLine -forever"
)

var (
	cfgShortAndLongIntegrationTmpl = filepath.Join("testdata", "templates", "nri-config-two-integrations.json")
	cfgShortIntegrationTmpl        = filepath.Join("testdata", "templates", "nri-config-integration-short.json")
	cfgTemplatePath                = filepath.Join("testdata", "templates", "nri-config.json")
)

func createAgentAndStart(t *testing.T, scenario string) *agent.Emulator {
	integrationsPath := filepath.Join("testdata", "scenarios", scenario)
	a := agent.New(integrationsPath)
	require.NoError(t, a.RunAgent())
	return a
}

func Test_OneIntegrationIsExecutedV4(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	a := createAgentAndStart(t, "v4_payload")
	defer a.Terminate()

	// the agent sends samples from the integration
	select {
	case req := <-a.ChannelHTTPRequests():
		bodyBuffer, _ := ioutil.ReadAll(req.Body)
		fmt.Println(string(bodyBuffer))
		assertMetrics(t, metricNRIOutV4, string(bodyBuffer), []string{"timestamp"})
	case <-time.After(timeout):
		assert.FailNow(t, "timeout while waiting for a response")
		return
	}
}
func Test_OneIntegrationIsExecutedAndTerminated(t *testing.T) {
	a := createAgentAndStart(t, "default")
	defer a.Terminate()

	// the agent sends samples from the integration
	select {
	case req := <-a.ChannelHTTPRequests():
		bodyBuffer, _ := ioutil.ReadAll(req.Body)
		assertMetrics(t, metricNRIOutV3, string(bodyBuffer), []string{"timestamp"})
	case <-time.After(timeout):
		assert.FailNow(t, "timeout while waiting for a response")
		return
	}

	// and just one integrations process is running
	testhelpers.Eventually(t, timeout, func(rt require.TestingT) {
		p, err := findChildrenProcessByCmdName(defNriOutExecution)
		assert.NoError(rt, err)
		assert.Len(rt, p, 1)
	})

	// there are no process running
	testhelpers.Eventually(t, 10*time.Second, func(rt require.TestingT) {
		p, err := findAllProcessByCmd(defNriOutExecution)
		assert.NoError(rt, err)
		assert.Empty(rt, p)
	})
}

func Test_IntegrationIsRelaunchedIfTerminated(t *testing.T) {
	a := createAgentAndStart(t, "scenario1")
	defer a.Terminate()
	// and just one integrations process is running
	var p []*process.Process
	var err error
	testhelpers.Eventually(t, timeout, func(rt require.TestingT) {
		p, err = findChildrenProcessByCmdName(defNriOutExecution)
		assert.NoError(rt, err)
		assert.Len(rt, p, 1)
	})
	go traceRequests(a.ChannelHTTPRequests())
	// if the integration exits with error code
	oldPid := p[0].Pid
	assert.NoError(t, p[0].Kill())
	// is eventually spawned again by the runner
	testhelpers.Eventually(t, 40*time.Second, func(rt require.TestingT) {
		p, err = findAllProcessByCmd(defNriOutExecution)
		assert.NoError(rt, err)
		if !assert.Len(rt, p, 1) {
			return
		}
	})
	assert.NotEqual(t, oldPid, p[0].Pid)
}

func Test_IntegrationIsRelaunchedIfOneIntegrationIsModified(t *testing.T) {
	a := createAgentAndStart(t, "scenario2")
	defer a.Terminate()
	// and just one integrations process is running
	var p []*process.Process
	var err error
	testhelpers.Eventually(t, timeout, func(rt require.TestingT) {
		p, err = findChildrenProcessByCmdName(defNriOutExecution)
		assert.NoError(rt, err)
		assert.Len(rt, p, 1)
	})
	go traceRequests(a.ChannelHTTPRequests())
	// if the integration exits with error code
	oldPid := p[0].Pid
	assert.NoError(t, p[0].Kill())
	// is eventually spawned again by the runner
	testhelpers.Eventually(t, 40*time.Second, func(rt require.TestingT) {
		p, err = findChildrenProcessByCmdName(defNriOutExecution)
		assert.NoError(rt, err)
		if !assert.Len(rt, p, 1) {
			return
		}
	})
	assert.NotEqual(t, oldPid, p[0].Pid)
}

func Test_IntegrationIsRelaunchedIfIntegrationDetailsAreChanged(t *testing.T) {
	assert.Nil(t, createFile(filepath.Join("testdata", "templates", "nri-config.json"), filepath.Join("testdata", "scenarios", "scenario2", "nri-config.json"), map[string]interface{}{
		"timestamp": time.Now(),
	}))
	a := createAgentAndStart(t, "scenario2")
	defer a.Terminate()
	// and just one integrations process is running
	var p []*process.Process
	var err error
	testhelpers.Eventually(t, timeout, func(rt require.TestingT) {
		p, err = findChildrenProcessByCmdName(defNriOutExecution)
		assert.NoError(rt, err)
		assert.Len(rt, p, 1)
	})
	go traceRequests(a.ChannelHTTPRequests())
	// if the integration exits with error code
	oldPid := p[0].Pid
	assert.Nil(t, createFile(filepath.Join("testdata", "templates", "nri-config.json"), filepath.Join("testdata", "scenarios", "scenario2", "nri-config.json"), map[string]interface{}{
		"timestamp": time.Now(),
	}))

	// is eventually spawned again by the runner
	testhelpers.Eventually(t, 40*time.Second, func(rt require.TestingT) {
		p, err = findChildrenProcessByCmdName(defNriOutExecution)
		assert.NoError(rt, err)
		if len(p) > 0 {
			assert.NotEqual(rt, oldPid, p[0].Pid)
		} else {
			assert.FailNow(rt, "")
		}
	})
}

func Test_IntegrationConfigContainsTwoIntegrationsAndOneIsRemoved(t *testing.T) {
	currentTimestamp := time.Now()
	localConfigFilePath := filepath.Join("testdata", "scenarios", "scenario3", "nri-config.json")
	assert.Nil(t, createFile(cfgShortAndLongIntegrationTmpl, localConfigFilePath, map[string]interface{}{
		"timestampShort": currentTimestamp,
		"timestampLong":  currentTimestamp,
	}))
	a := createAgentAndStart(t, "scenario3")
	defer a.Terminate()
	// and just one integrations process is running
	var shortPID []*process.Process
	var longPID []*process.Process
	var err error
	testhelpers.Eventually(t, timeout, func(rt require.TestingT) {
		shortPID, err = findChildrenProcessByCmdName(defNriOutExecution)
		assert.NoError(rt, err)
		assert.Len(rt, shortPID, 1)
		longPID, err = findChildrenProcessByCmdName(defNriOutLongExecution)
		assert.NoError(rt, err)
		assert.Len(rt, longPID, 1)
	})
	assert.Nil(t, createFile(cfgShortIntegrationTmpl, localConfigFilePath, map[string]interface{}{
		"timestampShort": currentTimestamp,
	}))

	// is eventually spawned again by the runner
	testhelpers.Eventually(t, 40*time.Second, func(rt require.TestingT) {
		newShortPID, err := findChildrenProcessByCmdName(defNriOutExecution)
		assert.NoError(rt, err)
		assert.Len(rt, newShortPID, 1)
		newLongPID, err := findChildrenProcessByCmdName(defNriOutLongExecution)
		assert.NoError(rt, err)
		assert.Len(rt, newLongPID, 0)
		assert.Equal(t, newShortPID[0].Pid, shortPID[0].Pid)
	})

}
