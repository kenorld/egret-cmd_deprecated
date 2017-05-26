package main

import (
	"strconv"

	"github.com/kenorld/eject-cmd/harness"
	"github.com/kenorld/eject-core"
	"github.com/Sirupsen/logrus"
)

var cmdRun = &Command{
	UsageLine: "run [import path] [run mode] [port]",
	Short:     "run a Eject application",
	Long: `
Run the Eject web application named by the given import path.

For example, to run the chat room sample application:

    eject run github.com/kenorld/eject-samples/chat dev

The run mode is used to select which set of app.yaml configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

You can set a port as an optional third parameter.  For example:

    eject run github.com/kenorld/eject-samples/chat prod 8080`,
}

func init() {
	cmdRun.Run = runApp
}

func runApp(args []string) {
	if len(args) == 0 {
		errorf("No import path given.\nRun 'eject help run' for usage.\n")
	}

	// Determine the run mode.
	mode := "dev"
	if len(args) >= 2 {
		mode = args[1]
	}

	// Find and parse app.yaml
	eject.Init(mode, args[0], "")
	eject.LoadMimeConfig()

	// Determine the override port, if any.
	port := eject.HttpPort
	if len(args) == 3 {
		var err error
		if port, err = strconv.Atoi(args[2]); err != nil {
			errorf("Failed to parse port as integer: %s", args[2])
		}
	}

	logrus.WithFields(logrus.Fields{
		"AppName": eject.AppName, "ImportPath": eject.ImportPath, "Mode": mode, "BasePath": eject.BasePath,
	}).Info("Running...")

	// If the app is run in "watched" mode, use the harness to run it.
	if eject.Config.GetBoolDefault("watch.enabled", true) && eject.Config.GetBoolDefault("watch.code", true) {
		logrus.Info("Running in watched mode.")
		eject.HttpPort = port
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
