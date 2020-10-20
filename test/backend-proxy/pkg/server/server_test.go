package server

import (
	"github.com/newrelic/infrastructure-agent/pkg/log"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	tlog = log.WithComponent("test.backend.handler.test")
)

func Test_fakeCollectorHandler(t *testing.T) {
	tsNR := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tlog.Info("Called")
	}))
	defer tsNR.Close()
	config := NewConfig()
	config.CollectorBaseURL = tsNR.URL
	router := GetRouter(config)

	ts := httptest.NewServer(router.GetHandler())
	defer ts.Close()

	get, err := http.DefaultClient.Get(ts.URL + "/collector/metrics")
	require.NoError(t, err)
	defer func() {
		_, _ = ioutil.ReadAll(get.Body)
		_ = get.Body.Close()
	}()
}
