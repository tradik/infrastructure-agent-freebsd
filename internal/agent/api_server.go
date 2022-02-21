package agent

import (
	goContext "context"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/newrelic/infrastructure-agent/pkg/config"
	"github.com/newrelic/infrastructure-agent/pkg/log"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strconv"
)

const apiConfigPath = "/config"
const apiVerbosePath = "/verbose"
const apiProfilerStartPath = "/profiler-start"
const apiProfilerStopPath = "/profiler-stop"
const apiHost = "localhost"
const apiPort = 8083

type ApiServer struct {
	logger           log.Entry
	cnf              *config.Config
	httpServer       *http.Server
	agent            *Agent
	profilerContext  goContext.Context
	profilerCancelFn goContext.CancelFunc
	started          bool
}

func NewApiServer(cnf *config.Config, agent *Agent) *ApiServer {
	logger := log.WithComponent("agent_api")

	profileContext, cancelFn := goContext.WithCancel(agent.Context.Ctx)

	return &ApiServer{logger: logger, cnf: cnf, agent: agent, profilerContext: profileContext, profilerCancelFn: cancelFn}
}

func (s *ApiServer) Toogle() {
	if s.started {
		s.Stop()
	} else {
		s.Start()
	}
	s.started = !s.started
}

func (s *ApiServer) Start() {
	s.httpServer = &http.Server{Addr: net.JoinHostPort(apiHost, strconv.Itoa(apiPort))}

	router := httprouter.New()
	router.POST(apiConfigPath, s.handleConfigPost)
	router.POST(apiVerbosePath, s.handleVerbosePost)
	router.POST(apiProfilerStartPath, s.handleProfilerStart)
	router.POST(apiProfilerStopPath, s.handleProfilerStop)

	s.httpServer.Handler = router

	go func() {
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.WithError(err).Error("unable to start Agent-API")
		}
	}()
}

func (s *ApiServer) Stop() {
	s.httpServer.Shutdown(goContext.Background())
}

type Config struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

func (s *ApiServer) handleConfigPost(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	var configPost Config
	err := json.NewDecoder(request.Body).Decode(&configPost)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if val, ok := configPost.Value.(float64); ok {
		s.logger.Info(fmt.Sprintf("%s : %v", configPost.Name, val))
		if configPost.Name == "Verbose" {
			log.SetLevel(logrus.TraceLevel)
		}
	}

}

func (s *ApiServer) handleVerbosePost(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {

	var v struct {
		Verbose int `json:"verbose"`
	}

	err := json.NewDecoder(request.Body).Decode(&v)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	switch v.Verbose {
	case 1:
		log.SetLevel(logrus.DebugLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}
}

func (s *ApiServer) handleProfilerStart(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {

	var m struct {
		MemProfile         string `json:"mem_profile"`
		MemProfileInterval int    `json:"mem_profile_interval"`
	}

	err := json.NewDecoder(request.Body).Decode(&m)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	s.logger.Info("PROFILER CHANGE")

	s.cnf.MemProfileInterval = m.MemProfileInterval
	s.cnf.MemProfile = m.MemProfile

	go s.agent.intervalMemoryProfile(s.profilerContext)

	writer.WriteHeader(http.StatusOK)
	//writer.Header().Set("Content-Type", "application/json")
	//resp := make(map[string]string)
	//jsonResp, err := json.Marshal(resp)
	//if err != nil {
	//	log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	//}
	//w.Write(jsonResp)
}

func (s *ApiServer) handleProfilerStop(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	s.logger.Info("PROFILER STOP")
	s.profilerCancelFn()
}
