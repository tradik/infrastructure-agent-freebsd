package cfgprotocol

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/agent"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	tmpDirPrefix = "newrelic-infra"
	paramBaseDir = "baseDir"
)

func assertMetrics(t *testing.T, expectedStr, actual string, ignoredEventAttributes []string) {
	var v []map[string]interface{}
	if err := json.Unmarshal([]byte(actual), &v); err != nil {
		t.Error(err)
		t.FailNow()
	}
	for i := range v {
		events := v[i]["Events"].([]interface{})
		for i := range events {
			event := events[i].(map[string]interface{})
			for _, attr := range ignoredEventAttributes {
				delete(event, attr)
			}
		}
	}
	var expected []map[string]interface{}
	json.Unmarshal([]byte(expectedStr), &expected)

	fail := assert.Equal(t, expected, v)
	fmt.Println(fail)
}

func Test_OneIntegrationIsSpawnedOnceByOther(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	timeout := 20 * time.Second
	expected := `
		[{
			"ExternalKeys":["shell-test:some-entity"],
			"IsAgent":false,
			"Events":[
				{
					"displayName":"shell-test:some-entity","entityKey":"shell-test:some-entity","entityName":"shell-test:some-entity",
					"eventType":"ShellTestSample","event_type":"ShellTestSample","integrationName":"nri-test","integrationVersion":"0.0.0",
					"reportingAgent":"my_display_name","some-metric":1
				}
			]
		}]
	`
	integrationsPath := filepath.Join("testdata", "scenarios", "scenario1")
	abs, _ := filepath.Abs(integrationsPath)
	emulator := agent.New(abs)
	go emulator.RunAgent()
	func(requests chan http.Request) {
		for {
			select {
			case req := <-requests:
				bodyBuffer, _ := ioutil.ReadAll(req.Body)
				assertMetrics(t, expected, string(bodyBuffer), []string{"timestamp"})
				return
			case <-time.After(timeout):
				assert.FailNow(t,"timeout exceeded")
				return
			}
		}
	}(emulator.ChannelHTTPRequests())
}
