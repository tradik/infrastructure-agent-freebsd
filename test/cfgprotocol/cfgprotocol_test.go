package cfgprotocol

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/testcase"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const testDataDir = "testdata"

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
