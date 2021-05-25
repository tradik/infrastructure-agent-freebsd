// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package cfgprotocol

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert"
)

func tempFiles(pathContents map[string]string) (directory string, err error) {
	dir, err := ioutil.TempDir("", "tempFiles")
	if err != nil {
		return "", err
	}
	for path, content := range pathContents {
		if err := ioutil.WriteFile(filepath.Join(dir, path), []byte(content), 0666); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func removeTempFiles(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return nil
}

func findAllProcessByCmd(cmd string) ([]*process.Process, error) {
	ps, err := process.Processes()
	if err != nil {
		return nil, err
	}
	return findProcessByCmd(cmd, ps), nil
}

func findChildrenProcessByCmdName(cmd string) ([]*process.Process, error) {
	pp, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, err
	}
	children, err := pp.Children()
	if err != nil {
		return nil, err
	}
	return findProcessByCmd(cmd, children), nil
}

func findProcessByCmd(cmd string, ps []*process.Process) []*process.Process {
	pFound := []*process.Process{}
	for _, p := range ps {
		c, err := p.Cmdline()
		if err != nil {
			continue
		}
		if strings.Contains(c, cmd) {
			pFound = append(pFound, p)
		}
	}
	return pFound
}

func assertMetrics(t *testing.T, expectedStr, actual string, ignoredEventAttributes []string) {
	var v []map[string]interface{}
	if err := json.Unmarshal([]byte(actual), &v); err != nil {
		t.Error(err)
		t.FailNow()
	}
	for i := range v {
		events := v[i]["Events"].([]interface{})
		for i := range events {
			event := events[i].(map[string]interface{})
			for _, attr := range ignoredEventAttributes {
				delete(event, attr)
			}
		}
	}
	var expected []map[string]interface{}
	json.Unmarshal([]byte(expectedStr), &expected)

	assert.Equal(t, expected, v)
}

func traceRequests(ch chan http.Request) {
	for {
		select {
		case req := <-ch:
			bodyBuffer, _ := ioutil.ReadAll(req.Body)
			fmt.Println(string(bodyBuffer))
		}
	}
}

func createFile(from, dest string, vars map[string]interface{}) error {
	outputFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	t, err := template.ParseFiles(from)
	if err != nil {
		return err

	}
	return t.Execute(outputFile, vars)
}
