// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
//go:generate goversioninfo

package main

import (
	"flag"
	"fmt"
	"github.com/gdamore/tcell/v2"
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
	//startUI()
	createGrid()
}

func startUI() {
	app := tview.NewApplication()
	list := tview.NewList().
		AddItem("Start memory profiler", "Start memory profiler with interval 5s and filePath /tmp/agent_mem_profile_ ", 0, func() {
			startMemoryProfiler(5, "/tmp/agent_mem_profile_")
			return
		}).
		AddItem("Stop memory profiler", "Stop memory profiler if started ", 0, func() {
			stopMemoryProfiler()
			return
		}).
		AddItem("Quit", "Press to exit [ESC]", 'q', func() {
			app.Stop()
		}).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEscape {
				app.Stop()
				return nil
			}
			return event
		})

	if err := app.SetRoot(list, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func createGrid() {
	newPrimitive := func(text string) tview.Primitive {
		return tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetText(text)
	}

	app := tview.NewApplication()

	list := tview.NewList().
		AddItem("Start memory profiler", "Start memory profiler with interval 300s ", '0', func() {
			startMemoryProfiler(5, "/tmp/agent_mem_profile_")
		}).
		AddItem("Start memory profiler", "Start memory profiler with interval 300s ", '0', func() {
			stopMemoryProfiler()
		}).
		AddItem("PING", "Start memory profiler with interval 300s ", 0, func() {
			fmt.Println("PING")
		}).
		AddItem("Quit", "Press to exit", 'q', func() {
			app.Stop()
		})

	//menu := newPrimitive("Menu")
	main := newPrimitive("Main content")
	sideBar := newPrimitive("Side Bar")

	grid := tview.NewGrid().
		SetRows(3, 0, 3).
		//SetColumns(30, 0, 30).
		SetBorders(true).
		AddItem(newPrimitive("Header"), 0, 0, 1, 3, 0, 0, false).
		AddItem(newPrimitive("Footer"), 2, 0, 1, 3, 0, 0, false)

	// Layout for screens narrower than 100 cells (menu and side bar are hidden).
	grid.AddItem(list, 0, 0, 0, 0, 0, 0, false).
		AddItem(main, 1, 0, 1, 3, 0, 0, false).
		AddItem(sideBar, 0, 0, 0, 0, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.AddItem(list, 1, 0, 1, 1, 0, 100, false).
		AddItem(main, 1, 1, 1, 1, 0, 100, false).
		AddItem(sideBar, 1, 2, 1, 1, 0, 100, false)

	app.SetFocus(list)

	if err := app.SetRoot(grid, true).EnableMouse(true).Run(); err != nil {
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
