package testcase

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/agent"
)

const tmpDirPrefix = "nr-infrastructure-agent"

type TestCase interface {
	Run(timeout time.Duration, evalRequestsFn func(requests chan http.Request)) error
}

type testcase struct {
	emulator *agent.Emulator
	baseDir  string
	vars     map[string]interface{}
	log      func(args ...interface{})
}

func New(log func(args ...interface{}), templatesPath string) (TestCase, error) {
	testcase := &testcase{
		log: log,
	}
	if err := testcase.setUp(); err != nil {
		return nil, err
	}
	testcase.loadFiles(templatesPath)
	return testcase, nil
}

func (t *testcase) setUp() error {
	baseDir, err := os.MkdirTemp("", tmpDirPrefix)
	if err != nil {
		return err
	}
	t.baseDir = baseDir
	t.vars = map[string]interface{}{
		"baseDir": baseDir,
	}
	t.emulator = agent.New(baseDir)
	return nil
}

func (t *testcase) teardown() {
	t.log("terminate the agent execution")
	t.emulator.Terminate()
	t.log("remove temporal directory %s", t.baseDir)
	os.Remove(t.baseDir)

}

func (t *testcase) loadFiles(templatesPath string) error {
	files, err := ioutil.ReadDir(templatesPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if err := createFile(t.baseDir, f.Name(), filepath.Join(templatesPath, f.Name()), t.vars); err != nil {
			return err
		}
	}
	return nil
}

func (t *testcase) Run(timeout time.Duration, evalRequestsFn func(requests chan http.Request)) error {
	ch := make(chan struct{}, 1)
	go t.emulator.RunAgent()
	go func() {
		evalRequestsFn(t.emulator.ChannelHTTPRequests())
		ch <- struct{}{}
	}()
	select {
	case <-ch:
		t.log("execution completed")
	case <-time.After(timeout):
		t.teardown()
		return errors.New("timeout exedeed")
	}
	return nil
}
