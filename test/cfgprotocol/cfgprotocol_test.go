package cfgprotocol

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/testcase"
	"github.com/sirupsen/logrus"
)

func Test_Demo(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	tc,err:=testcase.New(filepath.Join("testdata", "scenario1"))
	if err!=nil{
		t.Fatal(err)
	}
	tc.Run(20*time.Second,func(requests chan http.Request){

	})


}


