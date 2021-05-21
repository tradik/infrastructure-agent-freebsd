// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package cfgprotocol

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/internal/testhelpers"
	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/agent"
	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/testdata"
	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	timeout = 15 * time.Second
)

// TODO add tests:
// Generator spawning 1 long running and 1 short live integration:
// 	if long running fails its respawned
//
func Test_OneIntegrationIsExecutedAndTerminated(t *testing.T) {
	// logrus.SetLevel(logrus.DebugLevel)
	tv := testdata.TemplateVars{
		ConfigName: "simple-generator",
		NriGoFile:  testdata.Nri,
		// Scenario:   testdata.TenConfigs,
	}
	content, err := testdata.GenerateConfigFile(tv)
	require.NoError(t, err)
	dir, err := tempFiles(map[string]string{
		"parent-integration.yml": content,
	})
	require.NoError(t, err)
	// when the agent runs
	a := agent.New(dir)
	require.NoError(t, a.RunAgent())
	defer a.Terminate()

	// the agent sends samples from the integration
	select {
	case req := <-a.ChannelHTTPRequests():
		bodyBuffer, _ := ioutil.ReadAll(req.Body)
		assert.Contains(t, string(bodyBuffer), testdata.Nri)
	case <-time.After(timeout):
		assert.FailNow(t, "timeout while waiting for a response")
	}

	// and just one integrations process is running
	testhelpers.Eventually(t, timeout, func(rt require.TestingT) {
		p, err := findChildrenProcessByCmdName(testdata.Nri)
		assert.NoError(rt, err)
		assert.Len(rt, p, 1)
	})

	// when remove the parent config file
	assert.NoError(t, removeTempFiles(dir))
	// there are no process running
	testhelpers.Eventually(t, 10*time.Second, func(rt require.TestingT) {
		p, err := findAllProcessByCmd(testdata.Nri)
		assert.NoError(rt, err)
		assert.Empty(rt, p)
		p, err = findAllProcessByCmd(testdata.Generator)
		assert.NoError(rt, err)
		assert.Empty(rt, p)
	})
}

func Test_IntegrationIsRelauchedIfTerminated(t *testing.T) {
	// logrus.SetLevel(logrus.DebugLevel)
	t.Skip()
	gc := testdata.TemplateVars{
		ConfigName: "simple-generator",
		// Scenario:   testdata.TenConfigs,
	}
	file, err := testdata.GenerateConfigFile(gc)

	dir, err := tempFiles(map[string]string{
		"parent-integration.yml": file,
	})
	require.NoError(t, err)
	defer removeTempFiles(dir)
	// when the agent runs
	a := agent.New(dir)
	a.RunAgent()
	defer a.Terminate()

	// and just one integrations process is running
	var p []*process.Process
	testhelpers.Eventually(t, timeout, func(rt require.TestingT) {
		p, err = findChildrenProcessByCmdName(testdata.Nri)
		assert.NoError(rt, err)
		require.Len(rt, p, 1)
	})

	// if the integration exits with error code
	oldPid := p[0].Pid
	assert.NoError(t, p[0].Kill())

	// is eventually spawned again by the runner
	testhelpers.Eventually(t, timeout, func(rt require.TestingT) {
		p, err = findAllProcessByCmd(testdata.Nri)
		assert.NoError(rt, err)
		require.Len(rt, p, 1)
	})
	assert.NotEqual(t, oldPid, p[0].Pid)
}
