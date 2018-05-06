package main

import (
	"strconv"

	"github.com/kenorld/egret-cmd/harness"
	"github.com/kenorld/egret-core"
	"github.com/sirupsen/logrus"
)

var cmdRun = &Command{
	UsageLine: "run [import path] [run mode] [port]",
	Short:     "run a Egret application",
	Long: `
Run the Egret web application named by the given import path.

For example, to run the chat room sample application:

    egret run github.com/kenorld/egret-samples/chat dev

The run mode is used to select which set of app.yaml configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

You can set a port as an optional third parameter.  For example:

    egret run github.com/kenorld/egret-samples/chat prod 8080`,
}

func init() {
	cmdRun.Run = runApp
}

func runApp(args []string) {
	if len(args) == 0 {
		errorf("No import path given.\nRun 'egret help run' for usage.\n")
	}

	// Determine the run mode.
	mode := "dev"
	if len(args) >= 2 {
		mode = args[1]
	}

	// Find and parse app.yaml
	egret.Init(mode, args[0], "")
	egret.LoadMimeConfig()

	// Determine the override port, if any.
	port := egret.HttpPort
	if len(args) == 3 {
		var err error
		if port, err = strconv.Atoi(args[2]); err != nil {
			errorf("Failed to parse port as integer: %s", args[2])
		}
	}

	logrus.WithFields(logrus.Fields{
		"AppName": egret.AppName, "ImportPath": egret.ImportPath, "Mode": mode, "BasePath": egret.BasePath,
	}).Info("Running...")

	// If the app is run in "watched" mode, use the harness to run it.
	if egret.Config.GetBoolDefault("watch.enabled", true) && egret.Config.GetBoolDefault("watch.code", true) {
		logrus.Info("Running in watched mode.")
		egret.HttpPort = port
		harness.NewHarness().Run() // Never returns.
	}

	// Else, just build and run the app.
	logrus.Info("Running in live build mode.")
	app, err := harness.Build()
	if err != nil {
		errorf("Failed to build app: %s", err)
	}
	app.Port = port

	app.Cmd().Run()
}
