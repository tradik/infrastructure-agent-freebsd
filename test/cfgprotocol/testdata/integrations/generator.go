// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build ToAllowMultiMains
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/newrelic/infrastructure-agent/pkg/helpers"
	"github.com/newrelic/infrastructure-agent/test/cfgprotocol/testdata"
)

func main() {

	vars := helpers.GetEnv("VARS", "{}")
	tv := &testdata.TemplateVars{}
	if err := json.Unmarshal([]byte(vars), &tv); err != nil {
		log.Fatal("fail to parse vars")
	}
	payload := testdata.GeneratorPayload

	t := template.Must(template.New("template").Parse(payload))

	switch tv.Scenario {
	case testdata.OneConfig:
		t.Execute(os.Stdout, tv)
		log.Default().Println("integration Log")
		os.Exit(0)

	case testdata.TenConfigs:
		for i := 1; i < 11; i++ {
			tv.ConfigName = fmt.Sprintf("config%d", i)
			t.Execute(os.Stdout, tv)
		}
		os.Exit(0)
	}
}
