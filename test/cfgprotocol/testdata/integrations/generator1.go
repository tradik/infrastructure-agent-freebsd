// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build integration1

package main

import (
	"fmt"
	"strings"
	"time"
)

func main() {
	cfgprotocol := `
{
  "config_protocol_version": "1",
  "action": "register_config",
  "config_name": "myconfig",
  "config": {
  "integrations": [
    {
    "name": "nri-child",
    "exec": [
      "go",
      "run",
      "testdata/integrations/integration1.go"
    ],
    "interval": "15s",
    "env": {
      "PORT": "3306"
    }
    }
  ]
  }
}`
	cfgprotocol = strings.ReplaceAll(cfgprotocol, "\n", "")

	for {
		fmt.Println(cfgprotocol)
		time.Sleep(5 * time.Second)
	}
}
