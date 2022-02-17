// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
//go:generate goversioninfo

package main

import (
	"flag"
	"fmt"
	"github.com/newrelic/infrastructure-agent/pkg/config"
	"github.com/newrelic/infrastructure-agent/pkg/ctl/sender"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/rivo/tview"
)

var (
	agentPID    int
	containerID string
	apiVersion  string
)

func init() {
	flag.IntVar(
		&agentPID,
		"pid",
		0,
		"New Relic infrastructure agent PID",
	)

	flag.StringVar(
		&containerID,
		"cid",
		"",
		"New Relic infrastructure agent container ID (Containerised agent)",
	)

	flag.StringVar(
		&apiVersion,
		"docker-api-version",
		config.DefaultDockerApiVersion,
		"Docker API version [Optional] (Containerised agent)",
	)
}

func main() {
	flag.Parse()

	//ctx, cancel := context.WithCancel(context.Background())
	//// Enables Control+C termination
	//go func() {
	//	s := make(chan os.Signal, 1)
	//	signal.Notify(s, syscall.SIGQUIT)
	//	<-s
	//	cancel()
	//}()

	//client, err := getClient()
	//if err != nil {
	//	logrus.WithError(err).Fatal("Failed to initialize the notification client.")
	//}
	//
	//// Default message is "enable verbose logging" to maintain backwards compatibility.
	//msg := ipc.EnableAgentAPI
	//logrus.Debug("Sending message to agent: " + fmt.Sprint(msg))
	//if err := client.Notify(ctx, msg); err != nil {
	//	logrus.WithError(err).Fatal("Error occurred while notifying the NRI Agent.")
	//}

	//logrus.Infof("Notification successfully sent to the NRI Agent with ID '%s'", client.GetID())
	startUI()
}

func startUI() {
	app := tview.NewApplication()
	list := tview.NewList().
		AddItem("Start memory profiler", "Start memory profiler with interval 300s ", '0', func() {
			startMemoryProfiler(5, "/tmp/agent_mem_profile_")
		}).
		AddItem("Start memory profiler", "Start memory profiler with interval 300s ", '0', func() {
			stopMemoryProfiler()
		}).
		AddItem("Quit", "Press to exit", 'q', func() {
			app.Stop()
		})
	if err := app.SetRoot(list, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func startMemoryProfiler(interval int, filepath string) {
	_, err := http.Post(
		"http://localhost:8083/profiler-start",
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"mem_profile":"%s","mem_profile_interval":%d}`, filepath, interval)),
	)
	if err != nil {
		log.Fatalln(err)
	}
}

func stopMemoryProfiler() {
	_, err := http.Post(
		"http://localhost:8083/profiler-stop",
		"application/json",
		strings.NewReader("{}"),
	)
	if err != nil {
		log.Fatalln(err)
	}
}

// getClient returns an agent notification client.
func getClient() (sender.Client, error) {
	if runtime.GOOS == "windows" || agentPID != 0 {
		return sender.NewClient(agentPID)
	}
	if containerID != "" {
		return sender.NewContainerisedClient(apiVersion, containerID)
	}
	return sender.NewAutoDetectedClient(apiVersion)
}
