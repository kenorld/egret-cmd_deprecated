package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/kenorld/egret-cmd/harness"
	"github.com/kenorld/egret-core"
)

var cmdBuild = &Command{
	UsageLine: "build [import path] [target path] [run mode]",
	Short:     "build a Egret application (e.g. for deployment)",
	Long: `
Build the Egret web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

The run mode is used to select which set of app.yaml configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

WARNING: The target path will be completely deleted, if it already exists!

For example:

    egret build github.com/kenorld/egret-samples/chat /tmp/chat
`,
}

func init() {
	cmdBuild.Run = buildApp
}

func buildApp(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "%s\n%s", cmdBuild.UsageLine, cmdBuild.Long)
		return
	}

	appImportPath, destPath, mode := args[0], args[1], "dev"
	if len(args) >= 3 {
		mode = args[2]
	}

	if !egret.Initialized {
		egret.Init(mode, appImportPath, "")
	}

	// First, verify that it is either already empty or looks like a previous
	// build (to avoid clobbering anything)
	if exists(destPath) && !empty(destPath) && !exists(path.Join(destPath, "run.sh")) {
		errorf("Abort: %s exists and does not look like a build directory.", destPath)
	}

	os.RemoveAll(destPath)
	srcPath := path.Join(destPath, "src")
	mustCopyDir(path.Join(srcPath, filepath.FromSlash(appImportPath)), egret.BasePath, false, nil)
	os.MkdirAll(destPath, 0777)

	app, eerr := harness.Build()
	panicOnError(eerr, "Failed to build")

	// Included are:
	// - run scripts
	// - binary
	// - egret
	// - app

	// Egret and the app are in a directory structure mirroring import path
	destBinaryPath := path.Join(destPath, filepath.Base(app.BinaryPath))
	tmpEgretPath := path.Join(srcPath, filepath.FromSlash(egret.EgretCoreImportPath))
	mustCopyFile(destBinaryPath, app.BinaryPath)
	mustChmod(destBinaryPath, 0755)
	mustCopyDir(path.Join(tmpEgretPath, "conf"), path.Join(egret.EgretPath, "conf"), false, nil)
	mustCopyDir(path.Join(tmpEgretPath, "views"), path.Join(egret.EgretPath, "views"), false, nil)

	fmt.Println("app.BinaryPath: ", app.BinaryPath, "  filepath.Base(app.BinaryPath):", filepath.Base(app.BinaryPath))
	tmplData, runShPath := map[string]interface{}{
		"BinName":    filepath.Base(app.BinaryPath),
		"ImportPath": appImportPath,
		"Mode":       mode,
	}, path.Join(destPath, "run.sh")

	mustRenderTemplate(
		runShPath,
		filepath.Join(egret.EgretPath, "..", "egret-cmd", "egret", "package_run.sh.template"),
		tmplData)

	mustChmod(runShPath, 0755)

	mustRenderTemplate(
		filepath.Join(destPath, "run.bat"),
		filepath.Join(egret.EgretPath, "..", "egret-cmd", "egret", "package_run.bat.template"),
		tmplData)
}
