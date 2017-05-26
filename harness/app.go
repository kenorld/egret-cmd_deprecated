package harness

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/Sirupsen/logrus"
)

// App contains the configuration for running a Revel app.  (Not for the app itself)
// Its only purpose is constructing the command to execute.
type App struct {
	BinaryPath string // Path to the app executable
	Port       int    // Port to pass as a command line argument.
	cmd        AppCmd // The last cmd returned.
}

func NewApp(binPath string) *App {
	return &App{BinaryPath: binPath}
}

// Return a command to run the app server using the current configuration.
func (a *App) Cmd() AppCmd {
	a.cmd = NewAppCmd(a.BinaryPath, a.Port)
	return a.cmd
}

// Kill the last app command returned.
func (a *App) Kill() {
	a.cmd.Kill()
}

// AppCmd manages the running of a Revel app server.
// It requires eject.Init to have been called previously.
type AppCmd struct {
	*exec.Cmd
}

func NewAppCmd(binPath string, port int) AppCmd {
	cmd := exec.Command(binPath,
		fmt.Sprintf("-port=%d", port),
		fmt.Sprintf("-importPath=%s", eject.ImportPath),
		fmt.Sprintf("-runMode=%s", eject.RunMode))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return AppCmd{cmd}
}

// Start the app server, and wait until it is ready to serve requests.
func (cmd AppCmd) Start() error {
	listeningWriter := startupListeningWriter{os.Stdout, make(chan bool)}
	cmd.Stdout = listeningWriter
	logrus.WithFields(logrus.Fields{
		"Path": cmd.Path,
		"Args": cmd.Args,
	}).Info("Exec app.")
	if err := cmd.Cmd.Start(); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error running.")
	}

	select {
	case <-cmd.waitChan():
		logrus.Error("eject/harness: app died")
		return errors.New("eject/harness: app died")

	case <-time.After(30 * time.Second):
		cmd.Kill()
		logrus.Error("eject/harness: app timed out")
		return errors.New("eject/harness: app timed out")

	case <-listeningWriter.notifyReady:
		return nil
	}
	// panic("Impossible")
}

// Run the app server inline.  Never returns.
func (cmd AppCmd) Run() {
	logrus.WithFields(logrus.Fields{
		"Path": cmd.Path,
		"Args": cmd.Args,
	}).Info("Exec app.")
	if err := cmd.Cmd.Run(); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error running.")
	}
}

// Terminate the app server if it's running.
func (cmd AppCmd) Kill() {
	if cmd.Cmd != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) {
		logrus.Info("Killing eject server pid: ", cmd.Process.Pid)
		err := cmd.Process.Kill()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("Failed to kill eject server.")
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
// in the stream.  (Which tells us when the eject server has finished starting up)
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
