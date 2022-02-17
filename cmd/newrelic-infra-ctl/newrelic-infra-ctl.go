// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
//go:generate goversioninfo

package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/newrelic/infrastructure-agent/pkg/config"
	"github.com/newrelic/infrastructure-agent/pkg/ctl/sender"
	"github.com/prometheus/common/expfmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/rivo/tview"
	//promInstrumentation "github.com/newrelic/infrastructure-agent/internal/instrumentation"
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
	startGridUI()
}

func startGridUI() {
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
		})

	statsView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	client := &http.Client{}
	go func(statsView *tview.TextView) {

		ticker := time.NewTicker(time.Second)

		for {
			select {
			case <-ticker.C:
				metrics, err := Get(client, "http://localhost:9090")
				if err != nil {
					panic(err)
				}
				statsView.Clear()

				fmt.Fprintf(statsView, "  Agent runtime telemetry\n  ---\n")
				fmt.Fprintf(statsView, "  go_memstats_alloc_bytes : %2.f MB\n", (*metrics["go_memstats_alloc_bytes"].Metric[0].Gauge.Value)/1024.0/1024.0)
				fmt.Fprintf(statsView, "  go_memstats_heap_inuse_bytes : %2.f MB\n", (*metrics["go_memstats_heap_inuse_bytes"].Metric[0].Gauge.Value)/1024.0/1024.0)
				fmt.Fprintf(statsView, "  go_goroutines : %0.f\n", *metrics["go_goroutines"].Metric[0].Gauge.Value)
				fmt.Fprintf(statsView, "  go_threads : %0.f\n", *metrics["go_threads"].Metric[0].Gauge.Value)
				fmt.Fprintf(statsView, "  Queues\n  ---\n")
				fmt.Fprintf(statsView, "  event_queue_depth %0.f/%0.f %0.f%%\n",
					*metrics["newrelic_infra_instrumentation_event_queue_depth_size"].Metric[0].Gauge.Value,
					*metrics["newrelic_infra_instrumentation_event_queue_depth_capacity"].Metric[0].Gauge.Value,
					*metrics["newrelic_infra_instrumentation_event_queue_depth_utilization"].Metric[0].Gauge.Value, //.Metric[0].Gauge.Value,
				)
				fmt.Fprintf(statsView, "  batch_queue_depth %0.f/%0.f %0.f%%\n",
					*metrics["newrelic_infra_instrumentation_batch_queue_depth_size"].Metric[0].Gauge.Value,
					*metrics["newrelic_infra_instrumentation_batch_queue_depth_capacity"].Metric[0].Gauge.Value,
					*metrics["newrelic_infra_instrumentation_batch_queue_depth_utilization"].Metric[0].Gauge.Value, //.Metric[0].Gauge.Value,
				)
			}
		}

	}(statsView)

	grid := tview.NewGrid().
		SetRows(1, 0, 1).
		SetBorders(true)

	grid.AddItem(list, 1, 0, 1, 1, 0, 0, true).
		AddItem(statsView, 1, 1, 1, 1, 0, 0, false)

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

// Copy from https://github.com/newrelic/nri-prometheus/blob/main/internal/pkg/prometheus/prometheus.go

// MetricFamiliesByName is a map of Prometheus metrics family names and their
// representation.
type MetricFamiliesByName map[string]dto.MetricFamily

// HTTPDoer executes http requests. It is implemented by *http.Client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// acceptHeader from Prometheus server https://github.com/prometheus/prometheus/blob/v2.33.1/scrape/scrape.go#L751
const acceptHeader = `application/openmetrics-text;version=0.0.1,text/plain;version=0.0.4;q=0.5,*/*;q=0.1`

func Get(client HTTPDoer, url string) (MetricFamiliesByName, error) {
	mfs := MetricFamiliesByName{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return mfs, err
	}

	req.Header.Add("Accept", acceptHeader)

	resp, err := client.Do(req)
	if err != nil {
		return mfs, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		return nil, fmt.Errorf("status code returned by the prometheus exporter indicates an error occurred: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return mfs, err
	}
	r := bytes.NewReader(body)

	d := expfmt.NewDecoder(r, expfmt.FmtText)
	for {
		var mf dto.MetricFamily
		if err := d.Decode(&mf); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		mfs[mf.GetName()] = mf
	}

	return mfs, nil
}
