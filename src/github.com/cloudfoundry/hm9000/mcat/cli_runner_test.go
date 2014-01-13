package mcat_test

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/hm9000/config"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

func interruptSession(session *cmdtest.Session) {
	session.Cmd.Process.Signal(os.Interrupt)
	session.Wait(time.Second)
}

type CLIRunner struct {
	configPath       string
	listenerSession  *cmdtest.Session
	metricsSession   *cmdtest.Session
	apiServerSession *cmdtest.Session
	evacuatorSession *cmdtest.Session

	verbose bool
}

func NewCLIRunner(storeType string, storeURLs []string, ccBaseURL string, natsPort int, metricsServerPort int, verbose bool) *CLIRunner {
	runner := &CLIRunner{
		verbose: verbose,
	}
	runner.generateConfig(storeType, storeURLs, ccBaseURL, natsPort, metricsServerPort)
	return runner
}

func (runner *CLIRunner) generateConfig(storeType string, storeURLs []string, ccBaseURL string, natsPort int, metricsServerPort int) {
	tmpFile, err := ioutil.TempFile("/tmp", "hm9000_clirunner")
	defer tmpFile.Close()
	Ω(err).ShouldNot(HaveOccurred())

	runner.configPath = tmpFile.Name()

	conf, err := config.DefaultConfig()
	Ω(err).ShouldNot(HaveOccurred())
	conf.StoreType = storeType
	conf.StoreURLs = storeURLs
	conf.CCBaseURL = ccBaseURL
	conf.NATS[0].Port = natsPort
	conf.SenderMessageLimit = 8
	conf.MaximumBackoffDelayInHeartbeats = 6
	conf.MetricsServerPort = metricsServerPort
	conf.MetricsServerUser = "bob"
	conf.MetricsServerPassword = "password"
	conf.StoreMaxConcurrentRequests = 10
	conf.ListenerHeartbeatSyncIntervalInMilliseconds = 100

	err = json.NewEncoder(tmpFile).Encode(conf)
	Ω(err).ShouldNot(HaveOccurred())
}

func (runner *CLIRunner) StartListener(timestamp int) {
	runner.listenerSession = runner.start("listen", timestamp, "Listening for Actual State")
}

func (runner *CLIRunner) StopListener() {
	interruptSession(runner.listenerSession)
}

func (runner *CLIRunner) StartMetricsServer(timestamp int) {
	runner.metricsSession = runner.start("serve_metrics", timestamp, "Serving Metrics")
}

func (runner *CLIRunner) StopMetricsServer() {
	interruptSession(runner.metricsSession)
}

func (runner *CLIRunner) StartAPIServer(timestamp int) {
	runner.apiServerSession = runner.start("serve_api", timestamp, "Serving API")
}

func (runner *CLIRunner) StopAPIServer() {
	interruptSession(runner.apiServerSession)
}

func (runner *CLIRunner) StartEvacuator(timestamp int) {
	runner.evacuatorSession = runner.start("evacuator", timestamp, "Listening for DEA Evacuations")
}

func (runner *CLIRunner) StopEvacuator() {
	interruptSession(runner.evacuatorSession)
}

func (runner *CLIRunner) Cleanup() {
	os.Remove(runner.configPath)
}

func (runner *CLIRunner) start(command string, timestamp int, message string) *cmdtest.Session {
	cmd := exec.Command("hm9000", command, fmt.Sprintf("--config=%s", runner.configPath))
	cmd.Env = append(os.Environ(), fmt.Sprintf("HM9000_FAKE_TIME=%d", timestamp))

	var session *cmdtest.Session
	var err error
	if runner.verbose {
		session, err = cmdtest.StartWrapped(cmd, teeToStdout, teeToStdout)
	} else {
		session, err = cmdtest.Start(cmd)
	}

	Ω(err).ShouldNot(HaveOccurred())

	Ω(session).Should(SayWithTimeout(message, 5*time.Second))

	return session
}

func (runner *CLIRunner) Run(command string, timestamp int) {
	cmd := exec.Command("hm9000", command, fmt.Sprintf("--config=%s", runner.configPath))
	cmd.Env = append(os.Environ(), fmt.Sprintf("HM9000_FAKE_TIME=%d", timestamp))
	out, _ := cmd.CombinedOutput()
	if runner.verbose {
		fmt.Printf(command+" (%s) \n", time.Unix(int64(timestamp), 0))
		fmt.Printf(strings.Repeat("~", len(command)) + "\n")
		fmt.Printf(string(out))

		fmt.Printf("\n")
	}

	time.Sleep(50 * time.Millisecond)
}

func (runner *CLIRunner) StartSession(command string, timestamp int, extraArgs ...string) *cmdtest.Session {
	args := []string{command, fmt.Sprintf("--config=%s", runner.configPath)}
	args = append(args, extraArgs...)

	cmd := exec.Command("hm9000", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("HM9000_FAKE_TIME=%d", timestamp))

	var session *cmdtest.Session
	var err error

	if runner.verbose {
		session, err = cmdtest.StartWrapped(cmd, teeToStdout, teeToStdout)
	} else {
		session, err = cmdtest.Start(cmd)
	}

	Ω(err).ShouldNot(HaveOccurred())

	return session
}

func teeToStdout(out io.Writer) io.Writer {
	return io.MultiWriter(out, os.Stdout)
}
