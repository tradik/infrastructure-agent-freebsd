// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package testdata

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"text/template"
)

var Path = filepath.Join("testData", "integrations") + string(filepath.Separator)

const (
	Generator = "generator.go"
	Nri       = "nri.go"
)

type Scenario int

const (
	OneConfig Scenario = iota
	TenConfigs
)

type TemplateVars struct {
	GeneratorGoFile string
	NriGoFile       string
	ConfigName      string
	NriPayload      string
	Scenario        Scenario
	AllTemplateVars string // Wraps all parameters in json
}

func GenerateConfigFile(tv TemplateVars) (string, error) {
	// Encodes the config into a json to send through env var.
	j, err := json.Marshal(&tv)
	if err != nil {
		return "", err
	}
	tv.AllTemplateVars = string(j)

	var buf bytes.Buffer
	t := template.Must(template.New("file").Parse(generator))
	if err := t.Execute(&buf, tv); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var generator = `---
integrations:
  - name: {{or .GeneratorGoFile "` + Generator + `"}}
    exec:
      - go
      - run
      - ` + Path + `{{or .GeneratorGoFile "` + Generator + `" }}
    env:
      VARS: '{{ .AllTemplateVars }}'
`

var GeneratorPayload = strings.ReplaceAll(`
{
  "config_protocol_version": "1",
  "action": "register_config",
  "config_name": "{{or .ConfigName "defaultName"}}",
  "config": {
  "integrations": [
    {
    "name": "{{or .NriGoFile "`+Nri+`"}}",
    "exec": [
      "go",
      "run",
      "`+Path+`{{or .NriGoFile "`+Nri+`"}}"
    ],
    "env": {
      "PAYLOAD": "{{or .NriPayload "v3"}}"
    }
    }
  ]
  }
}`, "\n", "") + "\n"
