package cfgprotocol

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/internal/testhelpers"
	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/agent"
	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/testcase"
	"github.com/shirou/gopsutil/process"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDataDir ="testdata"

func Test_OneIntegrationIsExecutedAndTerminated(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	// GIVEN a set of configuration files
	dir, err := tempFiles(map[string]string{
		"parent-integration.yml": `
---
integrations:
  - name: nri-cfg-protocol
    exec:
      - go
      - run
      - testdata/integrations/generator1.go
    interval: 16
`,
	})
	require.NoError(t, err)
	a := agent.New(dir)
	go a.RunAgent()
	testhelpers.Eventually(t, 10*time.Second, func(rt require.TestingT) {
		_, err := findProcessByCmd("generator1.go")
		require.NoError(rt, err)
	})
	removeTempFiles(t, dir)
	time.Sleep(1 * time.Second)
	p, _ := findProcessByCmd("generator1.go")
	assert.Nil(t, p)

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


func Test_Demo(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	tc, err := testcase.New(t.Log, filepath.Join(testDataDir, "scenario1"))
	if err != nil {
		t.Fatal(err)
	}
	err = tc.Run(500*time.Second, func(requests chan http.Request) {
		for {
			select {
			case req := <-requests:
				// Buffer the body
				bodyBuffer, _ := ioutil.ReadAll(req.Body)
				fmt.Println(string(bodyBuffer))
			}
		}

	})
	assert.Nil(t, err)

}
