// Package main is the entry point for the jack CLI.
package main

import (
	"fmt"
	"os"

	"github.com/zoobzio/jack/core"
	"github.com/zoobzio/jack/handler"
)

func main() {
	app, err := core.NewApp()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	handler.Init(app)
	handler.Clone(app)
	handler.Status(app)
	handler.In(app)
	handler.Out(app)
	handler.Refresh(app)
	handler.Kill(app)

	if err := app.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
