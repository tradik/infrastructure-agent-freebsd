package cfgprotocol

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/agent"
	"github.com/sirupsen/logrus"
)

func Test_Demo(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	executor := agent.New(getIntegrationPath())
	go executor.RunAgent()
	for {
		select {
		case req := <-executor.Client().RequestCh:
			fmt.Print(req)
		}
	}

	time.Sleep(45 * time.Second)
}

func getIntegrationPath() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dir, "agent", "integrations.d")
}
