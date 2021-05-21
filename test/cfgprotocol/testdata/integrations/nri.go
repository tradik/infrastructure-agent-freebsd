// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
// +build ToAllowMultiMains
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/newrelic/infrastructure-agent/pkg/helpers"
)

func main() {
	payloadtype := helpers.GetEnv("PAYLOAD", "v3")
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
	            "inventory": {},
	            "events": []
	        }
	    ]
	}`

	v4 := `
    {
        "protocol_version": "4",
        "integration": {
            "name": "integration name",
            "version": "integration version"
        },
        "data": [
            {
                "common": {
                    "timestamp": 1531414060739,
                    "interval.ms": 10000,
                    "attributes": {}
                },
                "metrics": [
                    {
                        "name": "redis.metric1",
                        "type": "count",
                        "value": 93,
                        "attributes": {}
                    }
                ],
                "entity": {
                    "name": "uniqueName",
                    "type": "ENTITY_TYPE",
                    "displayName": "human readable name",
                    "metadata": {}
                },
                "inventory": { },
                "events": [
                    {
                        "summary": "foo",
                        "format": "event",
                        "attributes": {
                            "format": "attribute"
                        }
                    }
                ]
            }
        ]
    }
    `
	if payloadtype == "v4" {
		payload = v4
	}
	payload = strings.ReplaceAll(payload, "\n", "")

	for {
		fmt.Println(payload)
		time.Sleep(time.Second)
	}
}
