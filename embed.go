//go:build prod
// +build prod

package main

import (
	"embed"
	_ "embed"
) // Blank import to register the embed package

//go:embed dist/*
var embeddedFS embed.FS

func init() {
	frontendFS = embeddedFS
}
