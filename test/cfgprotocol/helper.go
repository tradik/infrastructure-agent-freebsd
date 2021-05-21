// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package cfgprotocol

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/process"
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
