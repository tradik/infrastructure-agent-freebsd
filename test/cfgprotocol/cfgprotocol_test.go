package cfgprotocol

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
	testDataDir = "testdata"
	timeout     = 15 * time.Second
)

var integrationsDir = filepath.Join(testDataDir, "integrations")

func Test_OneIntegrationIsExecutedAndTerminated(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	generator := "generator1.go"
	integration := "integration1.go"
	dir, err := tempFiles(map[string]string{
		"parent-integration.yml": `
---
integrations:
  - name: nri-cfg-protocol
    exec:
      - go
      - run
      - ` + filepath.Join(integrationsDir, generator) + `
    interval: 15
`,
	})
	require.NoError(t, err)

	a := agent.New(dir)

	go a.RunAgent()
	defer a.Terminate()

	// integrations process are spawned
	testhelpers.Eventually(t, 10*time.Second, func(rt require.TestingT) {
		_, err := findProcessByCmd(integration)
		require.NoError(rt, err)
	})
	var body string
	select {
	case req := <-a.ChannelHTTPRequests():
		bodyBuffer, _ := ioutil.ReadAll(req.Body)
		body = string(bodyBuffer)
	case <-time.After(timeout):
		assert.FailNow(t, "timeout while waiting for a response")
	}
	assert.Contains(t, body, "ShellTestSample")
	removeTempFiles(t, dir)
	testhelpers.Eventually(t, 10*time.Second, func(rt require.TestingT) {
		p, _ := findProcessByCmd(integration)
		assert.Nil(rt, p)
		p, _ = findProcessByCmd(generator)
		assert.Nil(rt, p)
	})
}
func Test_MultipleConfigNames(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	generator := "generator2.go"
	integration := "integration1.go"
	dir, err := tempFiles(map[string]string{
		"parent-integration.yml": `
---
integrations:
  - name: nri-cfg-protocol
    exec:
      - go
      - run
      - ` + filepath.Join(integrationsDir, generator) + `
    interval: 15
`,
	})
	require.NoError(t, err)

	a := agent.New(dir)

	go a.RunAgent()
	defer a.Terminate()

	// integrations process are spawned
	testhelpers.Eventually(t, 10*time.Second, func(rt require.TestingT) {
		_, err := findProcessByCmd(integration)
		require.NoError(rt, err)
	})
	removeTempFiles(t, dir)
	time.Sleep(10 * time.Minute)
	testhelpers.Eventually(t, 10*time.Second, func(rt require.TestingT) {
		p, _ := findProcessByCmd(integration)
		require.Nil(rt, p)
		p, _ = findProcessByCmd(generator)
		require.Nil(rt, p)
	})
}

func Test_IntegrationIsRelauchedIfTerminated(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	generator := "generator1.go"
	integration := "integration1.go"
	dir, err := tempFiles(map[string]string{
		"parent-integration.yml": `
---
integrations:
  - name: nri-cfg-protocol
    exec:
      - go
      - run
      - ` + filepath.Join(integrationsDir, generator) + `
    interval: 15
`,
	})
	require.NoError(t, err)

	a := agent.New(dir)

	go a.RunAgent()
	defer a.Terminate()

	// integrations process are spawned
	p := &process.Process{}
	testhelpers.Eventually(t, 10*time.Second, func(rt require.TestingT) {
		p, err = findProcessByCmd(integration)
		require.NoError(rt, err)
	})
	pidOld := p.Pid
	err = p.Kill()
	time.Sleep(10 * time.Second)
	newP := &process.Process{}
	testhelpers.Eventually(t, 10*time.Second, func(rt require.TestingT) {
		newP, err = findProcessByCmd(integration)
		require.NoError(rt, err)
	})
	assert.NotEqual(t, newP.Pid, pidOld)
}

func findProcessByCmd(cmd string) (*process.Process, error) {
	ps, _ := process.Processes()
	for _, p := range ps {
		c, _ := p.Cmdline()
		if strings.Contains(c, cmd) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no process found")
}
func tempFiles(pathContents map[string]string) (directory string, err error) {
	dir, err := ioutil.TempDir("", "tempFiles")
	if err != nil {
		return "", err
	}
	for path, content := range pathContents {
		if err := ioutil.WriteFile(filepath.Join(dir, path), []byte(content), 0666); err != nil {
			return "", err
		}
	}
	return dir, nil
}
func removeTempFiles(t *testing.T, dir string) {
	func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Log(err)
		}
	}()
}