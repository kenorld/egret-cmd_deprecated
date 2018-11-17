package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"go.uber.org/zap"

	"github.com/kenorld/egret-cmd/harness"
	egret "github.com/kenorld/egret-core"
)

var cmdTest = &Command{
	UsageLine: "test [import path] [run mode] [suite.method]",
	Short:     "run all tests from the command-line",
	Long: `
Run all tests for the Egret app named by the given import path.

For example, to run the booking sample application's tests:

    egret test github.com/kenorld/egret-samples/booking dev

The run mode is used to select which set of app.yaml configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

You can run a specific suite (and function) by specifying a third parameter.
For example, to run all of UserTest:

    egret test outspoken test UserTest

or one of UserTest's methods:

    egret test outspoken test UserTest.Test1
`,
}

func init() {
	cmdTest.Run = testApp
}

func testApp(args []string) {
	var err error
	if len(args) == 0 {
		errorf("No import path given.\nRun 'egret help test' for usage.\n")
	}

	mode := "dev"
	if len(args) >= 2 {
		mode = args[1]
	}

	// Find and parse app.yaml
	egret.Init(mode, args[0], "")

	// Ensure that the testrunner is loaded in this mode.
	// 	testRunnerFound := false
	// 	for _, module := range egret.Modules {
	// 		if module.ImportPath == egret.Config.StringDefault("module.testrunner", "github.com/kenorld/egret-extra/modules/testrunner") {
	// 			testRunnerFound = true
	// 			break
	// 		}
	// 	}
	// 	if !testRunnerFound {
	// 		errorf(`Error: The testrunner module is not running.

	// You can add it to a run mode configuration with the following line:

	// 	module.testrunner = github.com/kenorld/egret-extra/modules/testrunner

	// `)
	// 	}

	// Create a directory to hold the test result files.
	resultPath := path.Join(egret.BasePath, "test-results")
	if err = os.RemoveAll(resultPath); err != nil {
		errorf("Failed to remove test result directory %s: %s", resultPath, err)
	}
	if err = os.Mkdir(resultPath, 0777); err != nil {
		errorf("Failed to create test result directory %s: %s", resultPath, err)
	}

	// Direct all the output into a file in the test-results directory.
	file, err := os.OpenFile(path.Join(resultPath, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		errorf("Failed to create log file: %s", err)
	}

	app, reverr := harness.Build(logger)
	if reverr != nil {
		errorf("Error building: %s", reverr)
	}
	cmd := app.Cmd()
	cmd.Stderr = io.MultiWriter(cmd.Stderr, file)
	cmd.Stdout = io.MultiWriter(cmd.Stderr, file)

	// Start the app...
	if err := cmd.Start(); err != nil {
		errorf("%s", err)
	}
	defer cmd.Kill()
	logger.Info("Testing...",
		zap.String("app_name", egret.AppName),
		zap.String("import_path", egret.ImportPath),
		zap.String("mode", mode),
	)

	// Get a list of tests.
	// Since this is the first request to the server, retry/sleep a couple times
	// in case it hasn't finished starting up yet.
	var (
		// testSuites []routes.TestSuiteDesc
		resp    *http.Response
		baseUrl = fmt.Sprintf("http://127.0.0.1:%d", egret.HttpPort)
	)
	for i := 0; ; i++ {
		if resp, err = http.Get(baseUrl + "/@tests.list"); err == nil {
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if i < 3 {
			time.Sleep(3 * time.Second)
			continue
		}
		if err != nil {
			errorf("Failed to request test list: %s", err)
		} else {
			errorf("Failed to request test list: non-200 response")
		}
	}
	defer resp.Body.Close()
	// json.NewDecoder(resp.Body).Decode(&testSuites)

	// If a specific TestSuite[.Method] is specified, only run that suite/test
	// if len(args) == 3 {
	// 	testSuites = filterTestSuites(testSuites, args[2])
	// }
	// fmt.Printf("\n%d test suite%s to run.\n", len(testSuites), pluralize(len(testSuites), "", "s"))
	// fmt.Println()

	// Load the result template, which we execute for each suite.
	// module, _ := egret.ModuleByName("testrunner")
	// TemplateLoader := egret.NewTemplateLoader([]string{path.Join(module.Path, "core", "views")})
	// if err := TemplateLoader.Refresh(); err != nil {
	// 	errorf("Failed to compile templates: %s", err)
	// }
	// resultTemplate, err := TemplateLoader.Template("TestRunner/SuiteResult.html")
	// if err != nil {
	// 	errorf("Failed to load suite result template: %s", err)
	// }

	// Run each suite.
	// var (
	// 	overallSuccess = true
	// failedResults  []routes.TestSuiteResult
	// )
	// for _, suite := range testSuites {
	// 	// Print the name of the suite we're running.
	// 	name := suite.Name
	// 	if len(name) > 22 {
	// 		name = name[:19] + "..."
	// 	}
	// 	fmt.Printf("%-22s", name)

	// 	// Run every test.
	// 	startTime := time.Now()
	// 	suiteResult := routes.TestSuiteResult{Name: suite.Name, Passed: true}
	// 	for _, test := range suite.Tests {
	// 		testUrl := baseUrl + "/@tests/" + suite.Name + "/" + test.Name
	// 		resp, err := http.Get(testUrl)
	// 		if err != nil {
	// 			errorf("Failed to fetch test result at url %s: %s", testUrl, err)
	// 		}
	// 		defer resp.Body.Close()

	// 		var testResult routes.TestResult
	// 		json.NewDecoder(resp.Body).Decode(&testResult)
	// 		if !testResult.Passed {
	// 			suiteResult.Passed = false
	// 		}
	// 		suiteResult.Results = append(suiteResult.Results, testResult)
	// 	}
	// 	overallSuccess = overallSuccess && suiteResult.Passed

	// 	// Print result.  (Just PASSED or FAILED, and the time taken)
	// 	suiteResultStr, suiteAlert := "PASSED", ""
	// 	if !suiteResult.Passed {
	// 		suiteResultStr, suiteAlert = "FAILED", "!"
	// 		failedResults = append(failedResults, suiteResult)
	// 	}
	// 	fmt.Printf("%8s%3s%6ds\n", suiteResultStr, suiteAlert, int(time.Since(startTime).Seconds()))
	// 	// Create the result HTML file.
	// 	suiteResultFilename := path.Join(resultPath,
	// 		fmt.Sprintf("%s.%s.html", suite.Name, strings.ToLower(suiteResultStr)))
	// 	suiteResultFile, err := os.Create(suiteResultFilename)
	// 	if err != nil {
	// 		errorf("Failed to create result file %s: %s", suiteResultFilename, err)
	// 	}
	// 	if err = resultTemplate.Render(suiteResultFile, suiteResult); err != nil {
	// 		errorf("Failed to render result template: %s", err)
	// 	}
	// }

	// fmt.Println()
	// if overallSuccess {
	// 	writeResultFile(resultPath, "result.passed", "passed")
	// 	fmt.Println("All Tests Passed.")
	// } else {
	// 	for _, failedResult := range failedResults {
	// 		fmt.Printf("Failures:\n")
	// 		for _, result := range failedResult.Results {
	// 			if !result.Passed {
	// 				fmt.Printf("%s.%s\n", failedResult.Name, result.Name)
	// 				fmt.Printf("%s\n\n", result.ErrorSummary)
	// 			}
	// 		}
	// 	}
	// 	writeResultFile(resultPath, "result.failed", "failed")
	// 	errorf("Some tests failed.  See file://%s for results.", resultPath)
	// }
}

func writeResultFile(resultPath, name, content string) {
	if err := ioutil.WriteFile(path.Join(resultPath, name), []byte(content), 0666); err != nil {
		errorf("Failed to write result file %s: %s", path.Join(resultPath, name), err)
	}
}

func pluralize(num int, singular, plural string) string {
	if num == 1 {
		return singular
	}
	return plural
}

// Handlers test suites and individual tests to match
// the parsed command line parameter
// func filterTestSuites(suites []routes.TestSuiteDesc, suiteArgument string) []routes.TestSuiteDesc {
// 	var suiteName, testName string
// 	argArray := strings.Split(suiteArgument, ".")
// 	suiteName = argArray[0]
// 	if suiteName == "" {
// 		return suites
// 	}
// 	if len(argArray) == 2 {
// 		testName = argArray[1]
// 	}
// 	for _, suite := range suites {
// 		if suite.Name != suiteName {
// 			continue
// 		}
// 		if testName == "" {
// 			return []routes.TestSuiteDesc{suite}
// 		}
// 		// Only run a particular test in a suite
// 		for _, test := range suite.Tests {
// 			if test.Name != testName {
// 				continue
// 			}
// 			return []routes.TestSuiteDesc{
// 				routes.TestSuiteDesc{
// 					Name:  suite.Name,
// 					Tests: []routes.TestDesc{test},
// 				},
// 			}
// 		}
// 		errorf("Couldn't find test %s in suite %s", testName, suiteName)
// 	}
// 	errorf("Couldn't find test suite %s", suiteName)
// 	return nil
// }
