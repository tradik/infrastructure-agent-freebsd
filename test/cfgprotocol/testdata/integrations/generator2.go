// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build generator2

package main

import (
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"
)

func main() {

	cfgprotocol := `
{
  "config_protocol_version": "1",
  "action": "register_config",
  "config_name": "{{ .name }}",
  "config": {
  "integrations": [
    {
    "name": "nri-child",
    "exec": [
      "go",
      "run",
      "testdata/integrations/integration1.go"
    ]
    }
  ]
  }
}`
	cfgprotocol = strings.ReplaceAll(cfgprotocol, "\n", "")

	t := template.Must(template.New("test").Parse(cfgprotocol))

	for i := 1; ; i++ {
		vars := map[string]interface{}{
			"name": fmt.Sprintf("config%d", i),
		}
		t.Execute(os.Stdout, vars)
		fmt.Print("\n")
		time.Sleep(1 * time.Second)
	}
}
