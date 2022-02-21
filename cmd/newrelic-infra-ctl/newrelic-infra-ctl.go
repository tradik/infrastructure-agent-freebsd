// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
//go:generate goversioninfo

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/newrelic/infrastructure-agent/pkg/config"
	"github.com/newrelic/infrastructure-agent/pkg/ctl/sender"
	"github.com/newrelic/infrastructure-agent/pkg/ipc"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rivo/tview"
	"github.com/sirupsen/logrus"
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
	interactive := len(os.Args) == 1
	if interactive {
		interactiveMode()
		return
	}
	cliMode()
}

func interactiveMode() {
	agentPid, err := getAgentPID()
	if err != nil {
		log.Fatalln("Cannot get the infrastructure-agent PID %w", err)
	}

	ctx := context.Background()
	client, err := getClient(agentPid)
	if err != nil {
		log.Fatalln("Cannot get get client: %w", err)
	}
	toogleAgentApi(ctx, client)
	startGridUI()
}

func cliMode() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	// Enables Control+C termination
	go func() {
		s := make(chan os.Signal, 1)
		signal.Notify(s, syscall.SIGQUIT)
		<-s
		cancel()
	}()

	client, err := getClient(agentPID)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize the notification client.")
	}
	toogleAgentApi(ctx, client)
	logrus.Infof("Notification successfully sent to the NRI Agent with ID '%s'", client.GetID())

}

func toogleAgentApi(ctx context.Context, client sender.Client) {
	msg := ipc.EnableAgentAPI
	logrus.Debug("Sending message to agent: " + fmt.Sprint(msg))
	if err := client.Notify(ctx, msg); err != nil {
		logrus.WithError(err).Fatal("Error occurred while notifying the NRI Agent.")
	}
}

func getAgentPID() (int, error) {
	cmd := exec.Command("pgrep", "-n", "^newrelic-infra$")
	outputCommand, err := cmd.CombinedOutput()
	if err != nil {
		return 0, errors.New("error executing pgrep")
	}
	return strconv.Atoi(strings.TrimSpace(string(outputCommand)))
}

func startGridUI() {
	app := tview.NewApplication()

	pages := tview.NewPages()

	list := tview.NewList().
		AddItem("Start memory profiler", "Start memory profiler with interval 5s and filePath /tmp/agent_mem_profile_ ", 0, func() {
			startMemoryProfiler(5, "/tmp/agent_mem_profile_")
			return
		}).
		AddItem("Stop memory profiler", "Stop memory profiler if started ", 0, func() {
			stopMemoryProfiler()
			return
		}).
		AddItem("Start CPU profiler", "Stop memory profiler if started ", 0, func() {
			stopMemoryProfiler()
			return
		}).
		AddItem("Enable verbose logs", "", 0, func() {
			return
		}).
		AddItem("Clean cache (56MB)", "", 0, func() {
			return
		}).
		AddItem("Clean logs (3.4GB)", "", 0, func() {
			return
		}).
		AddItem("Enable Self instrumentation", "", 0, func() {
			pages.SwitchToPage("self ins example")
			return
		}).
		AddItem("Validate network connectivity", "", 0, func() {
			return
		}).
		AddItem("Quit", "Press to exit [ESC]", 'q', func() {
			client, err := getClient(agentPID)
			if err != nil {
				logrus.WithError(err).Fatal("Failed to initialize the notification client.")
			}
			toogleAgentApi(context.Background(), client)
			app.Stop()
		}).
		ShowSecondaryText(false)

	statsView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	footer := tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetText(time.Now().Format("2006-01-02 15:04:05"))

	start := time.Now().Add(-time.Hour * 25)
	client := &http.Client{}
	go func(statsView *tview.TextView, footer *tview.TextView) {

		ticker := time.NewTicker(time.Second)

		for {
			select {
			case <-ticker.C:
				metrics, err := Get(client, "http://localhost:9090")
				if err != nil {
					// TODO rethink this
					return
					//panic(err)
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

				now := time.Now()
				_, _, day, hour, min, sec := diff(start, now)
				footer.Clear()
				footer.SetText(fmt.Sprintf("%dd %dh %dm %ds | %s", day, hour, min, sec, time.Now().Format("2006-01-02 15:04:05")))
			}
		}

	}(statsView, footer)

	grid := tview.NewGrid().
		SetRows(1, 0, 1).
		SetBorders(true).
		AddItem(footer, 2, 0, 1, 2, 0, 0, false)

	grid.AddItem(list, 1, 0, 1, 1, 0, 0, true).
		AddItem(statsView, 1, 1, 1, 1, 0, 0, false)

	app.SetFocus(list)

	form := tview.NewForm().
		AddInputField("License key", "", 20, nil, nil).
		AddCheckbox("Staging", false, nil).
		AddButton("Save", nil).
		AddButton("Quit", func() {
			pages.SwitchToPage("menu")
		})
	form.Box.SetRect(0, 0, 100, 100)
	form.SetBorder(true).SetTitle("Enter some data").SetTitleAlign(tview.AlignLeft)

	pages.AddPage("self ins example", form, false, false)
	pages.AddPage("menu", grid, true, true)

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func diff(a, b time.Time) (year, month, day, hour, min, sec int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	year = int(y2 - y1)
	month = int(M2 - M1)
	day = int(d2 - d1)
	hour = int(h2 - h1)
	min = int(m2 - m1)
	sec = int(s2 - s1)

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}

	return
}

func startMemoryProfiler(interval int, filepath string) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	_, err := client.Post(
		"http://localhost:8083/profiler-start",
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"mem_profile":"%s","mem_profile_interval":%d}`, filepath, interval)),
	)
	if err != nil {
		log.Fatalln(err)
	}
}

func stopMemoryProfiler() {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	_, err := client.Post(
		"http://localhost:8083/profiler-stop",
		"application/json",
		strings.NewReader("{}"),
	)
	if err != nil {
		log.Fatalln(err)
	}
}

// getClient returns an agent notification client.
func getClient(pid int) (sender.Client, error) {
	if runtime.GOOS == "windows" || pid != 0 {
		return sender.NewClient(pid)
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
