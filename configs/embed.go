// Package configs embeds the example configuration written on first run.
package configs

import _ "embed"

// Example is the commented games.yaml template.
//
//go:embed games.example.yaml
var Example []byte
