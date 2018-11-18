// The command line tool for running Egret apps.
package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/agtorre/gocolorize"
	"go.uber.org/zap"
)

// Command structure cribbed from the genius organization of the "go" command.
type Command struct {
	Run                    func(args []string)
	UsageLine, Short, Long string
}

func (cmd *Command) Name() string {
	name := cmd.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

var commands = []*Command{
	cmdNew,
	cmdRun,
	cmdBuild,
	cmdPackage,
	cmdTest,
	cmdVersion,
}
var logger *zap.Logger

func main() {
	lg, _ := zap.NewDevelopment()
	logger = lg
	defer logger.Sync()
	if runtime.GOOS == "windows" {
		gocolorize.SetPlain(true)
	}
	fmt.Fprintf(os.Stdout, gocolorize.NewColor("blue").Paint(header))
	flag.Usage = func() { usage(1) }
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 || args[0] == "help" {
		if len(args) == 1 {
			usage(0)
		}
		if len(args) > 1 {
			for _, cmd := range commands {
				if cmd.Name() == args[1] {
					tmpl(os.Stdout, helpTemplate, cmd)
					return
				}
			}
		}
		usage(2)
	}

	// Commands use panic to abort execution when something goes wrong.
	// Panics are logged at the point of error.  Ignore those.
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		if _, ok := err.(LoggedError); !ok {
	// 			// This panic was not expected / logged.
	// 			panic(err)
	// 		}
	// 		os.Exit(1)
	// 	}
	// }()
	for _, cmd := range commands {
		if cmd.Name() == args[0] {
			cmd.Run(args[1:])
			return
		}
	}

	errorf("unknown command %q\nRun 'egret help' for usage.\n", args[0])
}

func errorf(format string, args ...interface{}) {
	// Ensure the user's command prompt starts on the next line.
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	panic(LoggedError{}) // Panic instead of os.Exit so that deferred will run.
}

//http://patorjk.com/software/taag/#p=testall&h=1&c=c&f=Graffiti&t=Egret
const header = `
    U _____ u   ____     ____    U _____ u  _____   
    \| ___"|/U /"___|uU |  _"\ u \| ___"|/ |_ " _|  
     |  _|"  \| |  _ / \| |_) |/  |  _|"     | |    
     | |___   | |_| |   |  _ <    | |___    /| |\   
     |_____|   \____|   |_| \_\   |_____|  u |_|U   
     <<   >>   _)(|_    //   \\_  <<   >>  _// \\_  
    (__) (__) (__)__)  (__)  (__)(__) (__)(__) (__) 
		
		                                             
`

const usageTemplate = `usage: egret command [arguments]

The commands are:
{{range .}}
    {{.Name | printf "%-11s"}} {{.Short}}{{end}}

Use "egret help [command]" for more information.
`

var helpTemplate = `usage: egret {{.UsageLine}}
{{.Long}}
`

func usage(exitCode int) {
	tmpl(os.Stderr, usageTemplate, commands)
	os.Exit(exitCode)
}

func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	template.Must(t.Parse(text))
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
