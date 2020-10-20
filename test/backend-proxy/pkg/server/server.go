package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"gopkg.in/pipe.v2"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/newrelic/infrastructure-agent/pkg/log"
)

var (
	slog = log.WithComponent("test.backend.handler")
)

// Server has a wrapper to hold the server handler
type Server interface {
	GetHandler() http.Handler
}

// GetRouter returns the stub router
func GetRouter() Server {
	r := mux.NewRouter()
	sh := &serverHandler{
		handler: r,
		client:  http.DefaultClient,
	}
	r.PathPrefix("/collector").HandlerFunc(sh.fakeCollectorHandler)
	r.PathPrefix("/metric").HandlerFunc(sh.fakeMetricHandler)
	r.PathPrefix("/identity").HandlerFunc(sh.fakeIdentityHandler)
	r.PathPrefix("/identity-rate-limit").HandlerFunc(sh.rateLimitIdentityHandler)
	r.PathPrefix("/command-channel").HandlerFunc(sh.fakeCommandChannelHandler)

	_ = os.MkdirAll(filepath.Join("tmp", "identity"), os.ModePerm)
	_ = os.MkdirAll(filepath.Join("tmp", "command"), os.ModePerm)
	_ = os.MkdirAll(filepath.Join("tmp", "collector"), os.ModePerm)
	return sh
}

type serverHandler struct {
	handler http.Handler
	client  *http.Client
}

func (s serverHandler) GetHandler() http.Handler {
	return s.handler
}

func (s serverHandler) fakeMetricHandler(w http.ResponseWriter, req *http.Request) {
	slog.WithField("Path", req.URL.Path).Info("Metric Handler")
}

func (s serverHandler) fakeCollectorHandler(w http.ResponseWriter, req *http.Request) {
	slog.WithField("Path", req.URL.Path).Info("Collector Handler")
	pathAsName := strings.ReplaceAll(req.URL.Path[10:], "/", "-")
	now := time.Now().Format("2006-01-02_15-04-05.000")
	orig := &bytes.Buffer{}
	body := &bytes.Buffer{}
	p := pipe.Line(
		pipe.Read(req.Body),
		pipe.Tee(orig),
		pipe.Write(body),
	)

	err := pipe.Run(p)

	if err != nil {
		slog.WithError(err).Error("Something went wrong in fakeCollectorHandler")
	}

	go func() {
		if err := write(req, body, "./tmp/collector/"+now+pathAsName+"-request"); err != nil {
			slog.WithError(err).Error("Something went wrong writing request to disk in fakeCollectorHandler")
		}
	}()

	endpoint := "https://staging-infra-api.newrelic.com" + req.URL.Path[10:]
	sendLog := slog.WithField("Endpoint", endpoint)
	sendLog.Info("Attempting to send data")
	httpReq, err := http.NewRequest(req.Method, endpoint, orig)
	if err != nil {
		sendLog.WithError(err).Error("Something went wrong making new request to send data to NR")
	}

	for key := range req.Header {
		if key != "Content-Length" {
			key := req.Header.Get(key)
			sendLog.
				WithField("Header Value", key).
				WithField("Header Key", key).Info("Adding header")
			httpReq.Header.Set(key, key)
		}
	}
	httpResp, err := s.client.Do(httpReq)
	if err != nil {
		sendLog.WithError(err).Error("Something went wrong sending data to NR")
	}

	defer httpResp.Body.Close()

	for key := range httpResp.Header {
		if key != "Content-Length" {
			val := httpResp.Header.Get(key)
			sendLog.
				WithField("Header Value", val).
				WithField("Header Key", key).Info("Sending header back")
			w.Header().Set(key, val)
		}
	}

	w.WriteHeader(httpResp.StatusCode)

	filePath := "./tmp/collector/" + now + pathAsName + "-response"
	pr := pipe.Line(
		pipe.Read(httpResp.Body),
		pipe.TeeWriteFile(filePath, 0600),
		pipe.Write(w),
	)
	err = pipe.Run(pr)
	if err != nil {
		slog.WithError(err).Error("Something went wrong writing response to disk in fakeCollectorHandler")
	}
}

func (s serverHandler) rateLimitIdentityHandler(w http.ResponseWriter, req *http.Request) {
	slog.WithField("Path", req.URL.Path).Info("Limit Identity Handler")
	now := time.Now().Format("2006-01-02_15-04-05.000")
	orig := &bytes.Buffer{}
	body := &bytes.Buffer{}
	p := pipe.Line(
		pipe.Read(req.Body),
		pipe.Tee(orig),
		pipe.Write(body),
	)

	err := pipe.Run(p)

	if err != nil {
		slog.WithError(err).Error("Something went wrong in rateLimitIdentityHandler")
	}

	go func() {
		if err := write(req, body, "./tmp/identity/"+now+"-request"); err != nil {
			slog.WithError(err).Error("Something went wrong writing request to disk in rateLimitIdentityHandler")
		}
	}()

	w.Header().Add("X-RateLimit-Docs", "https://docs.newrelic.com/docs/apis/rest-api-v2/requirements/api-overload-protection-preventing-429-errors")
	w.Header().Add("X-RateLimit-Reset", fmt.Sprintf("%v", time.Now().Add(time.Minute).Unix()))
	w.Header().Add("X-RateLimit-Remaining", "0")
	w.Header().Add("X-RateLimit-Limit", "1000")
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(429)
	_, _ = w.Write([]byte("{}"))
}

func (s serverHandler) fakeIdentityHandler(w http.ResponseWriter, req *http.Request) {
	slog.WithField("Path", req.URL.Path).Info("Identity Handler")
	now := time.Now().Format("2006-01-02_15-04-05.000")
	orig := &bytes.Buffer{}
	body := &bytes.Buffer{}
	p := pipe.Line(
		pipe.Read(req.Body),
		pipe.Tee(orig),
		pipe.Write(body),
	)

	err := pipe.Run(p)

	if err != nil {
		slog.WithError(err).Error("Something went wrong in fakeIdentityHandler")
	}

	go func() {
		if err := write(req, body, "./tmp/identity/"+now+"-request"); err != nil {
			slog.WithError(err).Error("Something went wrong writing request to disk in fakeIdentityHandler")
		}
	}()

	endpoint := "https://staging-identity-api.newrelic.com" + req.URL.Path
	sendLog := slog.WithField("Endpoint", endpoint)
	sendLog.Info("Attempting to send data")
	httpReq, err := http.NewRequest(req.Method, endpoint, orig)
	if err != nil {
		sendLog.WithError(err).Error("Something went wrong making new request to send data to NR")
	}

	for key := range req.Header {
		if key != "Content-Length" {
			key := req.Header.Get(key)
			sendLog.
				WithField("Header Value", key).
				WithField("Header Key", key).Info("Adding header")
			httpReq.Header.Set(key, key)
		}
	}
	httpResp, err := s.client.Do(httpReq)
	if err != nil {
		sendLog.WithError(err).Error("Something went wrong sending data to NR")
	}

	defer httpResp.Body.Close()

	for key := range httpResp.Header {
		if key != "Content-Length" {
			val := httpResp.Header.Get(key)
			sendLog.
				WithField("Header Value", val).
				WithField("Header Key", key).Info("Sending header back")
			w.Header().Set(key, val)
		}
	}

	w.WriteHeader(httpResp.StatusCode)

	pr := pipe.Line(
		pipe.Read(httpResp.Body),
		pipe.TeeWriteFile("./tmp/identity/"+now+"-response", 0600),
		pipe.Write(w),
	)
	err = pipe.Run(pr)
	if err != nil {
		slog.WithError(err).Error("Something went wrong writing response to disk in fakeIdentityHandler")
	}
}

func write(req *http.Request, buf *bytes.Buffer, filename string) (err error) {
	var body io.Reader
	if req.Header.Get("Content-Encoding") == "gzip" {
		slog.Info("Got gzip body")
		body, err = gzip.NewReader(buf)
		if err != nil {
			return err
		}
	} else {
		body = buf
	}

	p := pipe.Line(
		pipe.Read(body),
		pipe.WriteFile(filename, 0600),
	)

	return pipe.Run(p)
}

func (s serverHandler) fakeCommandChannelHandler(w http.ResponseWriter, req *http.Request) {
	slog.WithField("Path", req.URL.Path).Info("Command Channel Handler")
	now := time.Now().Format("2006-01-02_15-04-05.000")

	orig := &bytes.Buffer{}
	body := &bytes.Buffer{}
	p := pipe.Line(
		pipe.Read(req.Body),
		pipe.Tee(orig),
		pipe.Write(body),
	)

	err := pipe.Run(p)

	if err != nil {
		slog.WithError(err).Error("Something went wrong in fakeCommandChannelHandler")
	}

	go func() {
		filePath := fmt.Sprintf("./tmp/command/%s.request", now)
		if err := write(req, body, filePath); err != nil {
			slog.WithError(err).Error("Something went wrong writing request to disk in fakeCommandChannelHandler")
		}
	}()

	endpoint := "https://staging-infrastructure-command-api.newrelic.com" + req.URL.Path[16:]
	sendLog := slog.WithField("Endpoint", endpoint)
	sendLog.Info("Attempting to send data")
	httpReq, err := http.NewRequest(req.Method, endpoint, orig)
	if err != nil {
		sendLog.WithError(err).Error("Something went wrong making new request to send data to NR")
	}

	for key := range req.Header {
		if key != "Content-Length" {
			key := req.Header.Get(key)
			sendLog.
				WithField("Header Value", key).
				WithField("Header Key", key).Info("Adding header")
			httpReq.Header.Set(key, key)
		}
	}
	httpResp, err := s.client.Do(httpReq)
	if err != nil {
		sendLog.WithError(err).Error("Something went wrong sending data to NR")
	}

	defer httpResp.Body.Close()

	for key := range httpResp.Header {
		if key != "Content-Length" {
			val := httpResp.Header.Get(key)
			sendLog.
				WithField("Header Value", val).
				WithField("Header Key", key).Info("Sending header back")
			w.Header().Set(key, val)
		}
	}

	w.WriteHeader(httpResp.StatusCode)

	filePath := fmt.Sprintf("./tmp/command/%s.response", now)
	pr := pipe.Line(
		pipe.Read(httpResp.Body),
		pipe.TeeWriteFile(filePath, 0600),
		pipe.Write(w),
	)
	err = pipe.Run(pr)
	if err != nil {
		slog.WithError(err).Error("Something went wrong writing response to disk in fakeCommandChannelHandler")
	}
}
