package harness

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/kenorld/egret-core"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

// App contains the configuration for running a Revel app.  (Not for the app itself)
// Its only purpose is constructing the command to execute.
type App struct {
	BinaryPath string // Path to the app executable
	Port       int    // Port to pass as a command line argument.
	cmd        AppCmd // The last cmd returned.
	logger     *zap.Logger
}

func NewApp(binPath string, logger *zap.Logger) *App {
	return &App{BinaryPath: binPath, logger: logger}
}

// Return a command to run the app server using the current configuration.
func (a *App) Cmd() AppCmd {
	a.cmd = NewAppCmd(a.BinaryPath, a.Port, a.logger)
	return a.cmd
}

// Kill the last app command returned.
func (a *App) Kill() {
	a.cmd.Kill()
}

// AppCmd manages the running of a Revel app server.
// It requires egret.Init to have been called previously.
type AppCmd struct {
	*exec.Cmd
	logger *zap.Logger
}

func NewAppCmd(binPath string, port int, logger *zap.Logger) AppCmd {
	cmd := exec.Command(binPath,
		fmt.Sprintf("-port=%d", port),
		fmt.Sprintf("-importPath=%s", egret.ImportPath),
		fmt.Sprintf("-runMode=%s", egret.RunMode))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return AppCmd{cmd, logger}
}

// Start the app server, and wait until it is ready to serve requests.
func (cmd AppCmd) Start() error {
	listeningWriter := startupListeningWriter{os.Stdout, make(chan bool)}
	cmd.Stdout = listeningWriter
	cmd.logger.Info("Exec app", zap.String("path", cmd.Path), zap.Strings("args", cmd.Args))
	if err := cmd.Cmd.Start(); err != nil {
		cmd.logger.Error("Error running", zap.Error(err))
	}

	select {
	case <-cmd.waitChan():
		cmd.logger.Error("egret/harness: app died")
		return errors.New("egret/harness: app died")

	case <-time.After(30 * time.Second):
		cmd.Kill()
		cmd.logger.Error("egret/harness: app timed out")
		return errors.New("egret/harness: app timed out")

	case <-listeningWriter.notifyReady:
		return nil
	}
	// panic("Impossible")
}

// Run the app server inline.  Never returns.
func (cmd AppCmd) Run() {
	cmd.logger.Info("Exec app", zap.String("path", cmd.Path), zap.Strings("args", cmd.Args))
	if err := cmd.Cmd.Run(); err != nil {
		cmd.logger.Error("Error running", zap.Error(err))
	}
}

// Terminate the app server if it's running.
func (cmd AppCmd) Kill() {
	if cmd.Cmd != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) {
		cmd.logger.Info("Killing egret server pid: " + cast.ToString(cmd.Process.Pid))
		err := cmd.Process.Kill()
		if err != nil {
			cmd.logger.Error("Failed to kill egret server", zap.Error(err))
		}
	}
}

// Return a channel that is notified when Wait() returns.
func (cmd AppCmd) waitChan() <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		cmd.Wait()
		ch <- struct{}{}
	}()
	return ch
}

// A io.Writer that copies to the destination, and listens for "Listening on.."
// in the stream.  (Which tells us when the egret server has finished starting up)
// This is super ghetto, but by far the simplest thing that should work.
type startupListeningWriter struct {
	dest        io.Writer
	notifyReady chan bool
}

func (w startupListeningWriter) Write(p []byte) (n int, err error) {
	if w.notifyReady != nil && bytes.Contains(p, []byte("listen")) {
		w.notifyReady <- true
		w.notifyReady = nil
	}
	return w.dest.Write(p)
}
