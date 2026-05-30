//go:build tools
// +build tools

// Package tools tracks build-time tool dependencies in go.mod so that
// `go run` resolves them with pinned versions. This file is never
// compiled into the provider binary.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
