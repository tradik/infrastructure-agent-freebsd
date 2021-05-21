// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build generator1

package main

import (
	"fmt"
	"strings"
	"time"
)

func main() {
	payload := `
  {
    "name": "com.newrelic.shelltest",
    "protocol_version": "3",
    "integration_version": "0.0.0",
    "data": [
        {
            "entity": {
                "name": "some-entity",
                "type": "shell-test",
                "id_attributes": []
            },
            "metrics": [
                {
                    "event_type": "ShellTestSample",
                    "some-metric": 1
                }
            ],
            "inventory": {
                "foo": {
                    "name": "bar"
                }
            },
            "events": []
        }
    ]
}`
	payload = strings.ReplaceAll(payload, "\n", "")

	for {
		fmt.Println(payload)
		time.Sleep(time.Second)
	}
}
