package cfgprotocol

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/testcase"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_Demo(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	tc,err:=testcase.New(filepath.Join("testdata", "scenario1"))
	if err!=nil{
		t.Fatal(err)
	}
	err=tc.Run(20*time.Second, func(requests chan http.Request){

		select {
			case req:=<-requests:
				fmt.Println("-----")
				fmt.Println(req.Method)
		}

	})
	assert.Nil(t, err)

}


