//go:build prod
// +build prod

package main

import (
	"embed"
)

//go:embed dist/*
var embeddedFS embed.FS

func init() {
	frontendFS = embeddedFS
}
