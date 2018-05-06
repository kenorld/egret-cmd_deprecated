package harness

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kenorld/egret-core"
)

var importErrorPattern = regexp.MustCompile("cannot find package \"([^\"]+)\"")

// Build the app:
// 1. Generate the the main.go file.
// 2. Run the appropriate "go build" command.
// Requires that egret.Init has been called previously.
// Returns the path to the built binary, and an error if there was a problem building it.
func Build(buildFlags ...string) (app *App, compileError *egret.Error) {
	if compileError != nil {
		return nil, compileError
	}

	// Read build config.
	buildTags := egret.Config.GetStringDefault("build.tags", "")

	// Build the user program (all code under app).
	// It relies on the user having "go" installed.
	goPath, err := exec.LookPath("go")
	if err != nil {
		logrus.Fatal("Go executable not found in PATH.")
	}

	pkg, err := build.Default.Import(egret.ImportPath, "", build.FindOnly)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ImportPath": egret.ImportPath,
		}).Fatal("Failure importing.")
	}

	// Binary path is a combination of $GOBIN/egret.d directory, app's import path and its name.
	binName := filepath.Join(pkg.BinDir, "egret.d", egret.ImportPath, filepath.Base(egret.BasePath))

	// Change binary path for Windows build
	goos := runtime.GOOS
	if goosEnv := os.Getenv("GOOS"); goosEnv != "" {
		goos = goosEnv
	}
	if goos == "windows" {
		binName += ".exe"
	}

	gotten := make(map[string]struct{})
	for {
		appVersion := getAppVersion()
		buildTime := time.Now().UTC().Format(time.RFC3339)
		versionLinkerFlags := fmt.Sprintf("-X %s/app.AppVersion=%s -X %s/app.BuildTime=%s",
			egret.ImportPath, appVersion, egret.ImportPath, buildTime)

		flags := []string{
			"build",
			"-i",
			"-ldflags", versionLinkerFlags,
			"-tags", buildTags,
			"-o", binName}

		// Add in build flags
		flags = append(flags, buildFlags...)

		// The main path
		flags = append(flags, path.Join(egret.ImportPath))

		buildCmd := exec.Command(goPath, flags...)
		logrus.Info("Exec:", buildCmd.Args)
		output, err := buildCmd.CombinedOutput()

		// If the build succeeded, we're done.
		if err == nil {
			return NewApp(binName), nil
		}
		logrus.Error(string(output))

		// See if it was an import error that we can go get.
		matches := importErrorPattern.FindStringSubmatch(string(output))
		if matches == nil {
			return nil, newCompileError(output)
		}

		// Ensure we haven't already tried to go get it.
		pkgName := matches[1]
		if _, alreadyTried := gotten[pkgName]; alreadyTried {
			return nil, newCompileError(output)
		}
		gotten[pkgName] = struct{}{}

		// Execute "go get <pkg>"
		getCmd := exec.Command(goPath, "get", pkgName)
		logrus.Info("Exec:", getCmd.Args)
		getOutput, err := getCmd.CombinedOutput()
		if err != nil {
			logrus.Error(string(getOutput))
			return nil, newCompileError(output)
		}

		// Success getting the import, attempt to build again.
	}
	logrus.Fatal("Not reachable")
	return nil, nil
}

// Try to define a version string for the compiled app
// The following is tried (first match returns):
// - Read a version explicitly specified in the APP_VERSION environment
//   variable
// - Read the output of "git describe" if the source is in a git repository
// If no version can be determined, an empty string is returned.
func getAppVersion() string {
	if version := os.Getenv("APP_VERSION"); version != "" {
		return version
	}

	// Check for the git binary
	if gitPath, err := exec.LookPath("git"); err == nil {
		// Check for the .git directory
		gitDir := path.Join(egret.BasePath, ".git")
		info, err := os.Stat(gitDir)
		if (err != nil && os.IsNotExist(err)) || !info.IsDir() {
			return ""
		}
		gitCmd := exec.Command(gitPath, "--git-dir="+gitDir, "describe", "--always", "--dirty")
		logrus.Info("Exec:", gitCmd.Args)
		output, err := gitCmd.Output()

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Warn("Cannot determine git repository version.")
			return ""
		}

		return "git-" + strings.TrimSpace(string(output))
	}

	return ""
}

func containsValue(m map[string]string, val string) bool {
	for _, v := range m {
		if v == val {
			return true
		}
	}
	return false
}

// Parse the output of the "go build" command.
// Return a detailed Error.
func newCompileError(output []byte) *egret.Error {
	errorMatch := regexp.MustCompile(`(?m)^([^:#]+):(\d+):(\d+:)? (.*)$`).
		FindSubmatch(output)
	if errorMatch == nil {
		errorMatch = regexp.MustCompile(`(?m)^(.*?)\:(\d+)\:\s(.*?)$`).FindSubmatch(output)

		if errorMatch == nil {
			logrus.WithFields(logrus.Fields{
				"error": string(output),
			}).Error("Failed to parse build errors.")
			return &egret.Error{
				Status:     500,
				Name:       "compilation_error",
				SourceType: "Go code",
				Title:      "Go Compilation Error",
				Summary:    "See console for build error.",
			}
		}

		errorMatch = append(errorMatch, errorMatch[3])

		logrus.WithFields(logrus.Fields{
			"error": string(output),
		}).Error("Build errors.")
	}

	// Read the source for the offending file.
	var (
		relFilename    = string(errorMatch[1]) // e.g. "src/egret/sample/core/routes/app.go"
		absFilename, _ = filepath.Abs(relFilename)
		line, _        = strconv.Atoi(string(errorMatch[2]))
		summary        = string(errorMatch[4])
		compileError   = &egret.Error{
			SourceType: "Go code",
			Name:       "compilation_error",
			Title:      "Go Compilation Error",
			Path:       relFilename,
			Summary:    summary,
			Line:       line,
		}
	)

	errorLink := egret.Config.GetStringDefault("error.link", "")

	if errorLink != "" {
		compileError.SetLink(errorLink)
	}

	fileStr, err := egret.ReadLines(absFilename)
	if err != nil {
		compileError.MetaError = absFilename + ": " + err.Error()
		logrus.WithFields(logrus.Fields{
			"error": compileError.MetaError,
		}).Error("Build compile error.")
		return compileError
	}

	compileError.SourceLines = fileStr
	return compileError
}
