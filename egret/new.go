package main

import (
	"bytes"
	"fmt"
	"go/build"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	egretExtraImportPath = "github.com/kenorld/egret-extra"
)

var cmdNew = &Command{
	UsageLine: "new [path] [skeleton]",
	Short:     "create a skeleton Egret application",
	Long: `
New creates a few files to get a new Egret application running quickly.

It puts all of the files in the given import path, taking the final element in
the path to be the app name.

Skeleton is an optional argument, provided as an import path

For example:

    egret new import/path/helloworld

    egret new import/path/helloworld import/path/skeleton
`,
}

func init() {
	cmdNew.Run = newApp
}

var (

	// go related paths
	gopath  string
	gocmd   string
	srcRoot string

	// egret related paths
	egretExtraPkg *build.Package
	appPath       string
	appName       string
	basePath      string
	importPath    string
	skeletonPath  string
)

func newApp(args []string) {
	// check for proper args by count
	if len(args) > 2 {
		errorf("Too many arguments provided.\nRun 'egret help new' for usage.\n")
	}

	// checking and setting go paths
	initGoPaths()

	// checking and setting application
	setApplicationPath(args)

	// checking and setting skeleton
	setSkeletonPath(args)

	// copy files to new app directory
	copyNewAppFiles()

	// goodbye world
	fmt.Fprintln(os.Stdout, "Your application is ready:\n  ", appPath)
	fmt.Fprintln(os.Stdout, "\nYou can run it with:\n   egret run", importPath)
}

const alphaNumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func generateSecret() string {
	chars := make([]byte, 64)
	for i := 0; i < 64; i++ {
		chars[i] = alphaNumeric[rand.Intn(len(alphaNumeric))]
	}
	return string(chars)
}

// lookup and set Go related variables
func initGoPaths() {
	// lookup go path
	gopath = build.Default.GOPATH
	if gopath == "" {
		errorf("Abort: GOPATH environment variable is not set. " +
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.")
	}

	// check for go executable
	var err error
	gocmd, err = exec.LookPath("go")
	if err != nil {
		errorf("Go executable not found in PATH.")
	}

	// egret/egret#1004 choose go path relative to current working directory
	workingDir, _ := os.Getwd()
	goPathList := filepath.SplitList(gopath)
	for _, path := range goPathList {
		if strings.HasPrefix(strings.ToLower(workingDir), strings.ToLower(path)) {
			srcRoot = path
			break
		}

		path, _ = filepath.EvalSymlinks(path)
		if len(path) > 0 && strings.HasPrefix(strings.ToLower(workingDir), strings.ToLower(path)) {
			srcRoot = path
			break
		}
	}

	if len(srcRoot) == 0 {
		logger.Fatal("Abort: could not create a Egret application outside of GOPATH.")
	}

	// set go src path
	srcRoot = filepath.Join(srcRoot, "src")
}

func setApplicationPath(args []string) {
	var err error
	importPath = args[0]

	// egret/egret#1014 validate relative path, we cannot use built-in functions
	// since Go import path is valid relative path too.
	// so check basic part of the path, which is "."
	if filepath.IsAbs(importPath) || strings.HasPrefix(importPath, ".") {
		errorf("Abort: '%s' looks like a directory.  Please provide a Go import path instead.",
			importPath)
	}

	_, err = build.Import(importPath, "", build.FindOnly)
	if err == nil {
		errorf("Abort: Import path %s already exists.\n", importPath)
	}

	egretExtraPkg, err = build.Import(egretExtraImportPath, "", build.FindOnly)
	if err != nil {
		errorf("Abort: Could not find Egret source code: %s\n", err)
	}

	appPath = filepath.Join(srcRoot, filepath.FromSlash(importPath))
	appName = filepath.Base(appPath)
	basePath = filepath.ToSlash(filepath.Dir(importPath))

	if basePath == "." {
		// we need to remove the a single '.' when
		// the app is in the $GOROOT/src directory
		basePath = ""
	} else {
		// we need to append a '/' when the app is
		// is a subdirectory such as $GOROOT/src/path/to/egretapp
		basePath += "/"
	}
}

func setSkeletonPath(args []string) {
	var err error
	if len(args) == 2 { // user specified
		skeletonName := args[1]
		_, err = build.Import(skeletonName, "", build.FindOnly)
		if err != nil {
			// Execute "go get <pkg>"
			getCmd := exec.Command(gocmd, "get", "-d", skeletonName)
			fmt.Println("Execute:", getCmd.Args)
			getOutput, err := getCmd.CombinedOutput()

			// check getOutput for no buildible string
			bpos := bytes.Index(getOutput, []byte("no buildable Go source files in"))
			if err != nil && bpos == -1 {
				errorf("Abort: Could not find or 'go get' Skeleton  source code: %s\n%s\n", getOutput, skeletonName)
			}
		}
		// use the
		skeletonPath = filepath.Join(srcRoot, skeletonName)

	} else {
		// use the egret default
		skeletonPath = filepath.Join(egretExtraPkg.Dir, "skeletons/default")
	}
}

func copyNewAppFiles() {
	var err error
	err = os.MkdirAll(appPath, 0777)
	panicOnError(err, "Failed to create directory "+appPath)

	mustCopyDir(appPath, skeletonPath, false, map[string]interface{}{
		// app.yaml
		"AppName":  appName,
		"BasePath": basePath,
		"Secret":   generateSecret(),
	})

	// Dotfiles are skipped by mustCopyDir, so we have to explicitly copy the .gitignore.
	gitignore := ".gitignore"
	mustCopyFile(filepath.Join(appPath, gitignore), filepath.Join(skeletonPath, gitignore))

}
