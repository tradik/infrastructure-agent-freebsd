package cfgprotocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	test "github.com/newrelic/infrastructure-agent/test/databind"
	"github.com/newrelic/infrastructure-agent/test/proxy/minagent"
	"github.com/newrelic/infrastructure-agent/test/proxy/testsetup"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	log.Info("launch docker-compose...")
	if err := test.ComposeUp("./docker-compose.yml"); err != nil {
		log.Println("error on compose-up: ", err.Error())
		os.Exit(-1)
	}

	exitValChn := make(chan int, 1)
	func() {
		defer test.ComposeDown("./docker-compose.yml")
		exitValChn <- m.Run()
		log.Info("docker-compose is shutdown")
	}()

	exitVal := <-exitValChn
	os.Exit(exitVal)
}

func Test_Demo(t *testing.T) {
	fmt.Println("Agent is running")
	require.NoError(t, restartAgent(minagent.ConfigOptions{
		ConfigFile: "/fake-config-cfgprotocol.yml",
	}))
	time.Sleep(10*time.Minute)
}

// restartAgent restarts the testing containerized agent
func restartAgent(config minagent.ConfigOptions) error {
	body, err := json.Marshal(config)
	if err != nil {
		return err
	}
	response, err := http.Post(testsetup.AgentRestart, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected response while restarting agent: %v", response.StatusCode)
	}
	return nil
}
