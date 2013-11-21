package md_test

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

type CLIRunner struct {
	configPath       string
	listenerCmd      *exec.Cmd
	listenerSession  *cmdtest.Session
	metricsServerCmd *exec.Cmd
	apiServerCmd     *exec.Cmd
	evacuatorCmd     *exec.Cmd

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
	Ω(err).ShouldNot(HaveOccured())

	runner.configPath = tmpFile.Name()

	conf, err := config.DefaultConfig()
	Ω(err).ShouldNot(HaveOccured())
	conf.StoreType = storeType
	conf.StoreURLs = storeURLs
	conf.CCBaseURL = ccBaseURL
	conf.NATS.Port = natsPort
	conf.SenderMessageLimit = 8
	conf.MaximumBackoffDelayInHeartbeats = 6
	conf.MetricsServerPort = metricsServerPort
	conf.MetricsServerUser = "bob"
	conf.MetricsServerPassword = "password"
	conf.StoreMaxConcurrentRequests = 10
	conf.ListenerHeartbeatSyncIntervalInMilliseconds = 1

	err = json.NewEncoder(tmpFile).Encode(conf)
	Ω(err).ShouldNot(HaveOccured())
}

func (runner *CLIRunner) StartListener(timestamp int) {
	runner.listenerCmd, runner.listenerSession = runner.start("listen", timestamp)
}

func (runner *CLIRunner) StopListener() {
	runner.listenerCmd.Process.Signal(os.Interrupt)
	runner.listenerCmd.Wait()
}

func (runner *CLIRunner) StartMetricsServer(timestamp int) {
	runner.metricsServerCmd, _ = runner.start("serve_metrics", timestamp)
}

func (runner *CLIRunner) StopMetricsServer() {
	runner.metricsServerCmd.Process.Signal(os.Interrupt)
	runner.metricsServerCmd.Wait()
}

func (runner *CLIRunner) StartAPIServer(timestamp int) {
	runner.apiServerCmd, _ = runner.start("serve_api", timestamp)
}

func (runner *CLIRunner) StopAPIServer() {
	runner.apiServerCmd.Process.Signal(os.Interrupt)
	runner.apiServerCmd.Wait()
}

func (runner *CLIRunner) StartEvacuator(timestamp int) {
	runner.evacuatorCmd, _ = runner.start("evacuator", timestamp)
}

func (runner *CLIRunner) StopEvacuator() {
	runner.evacuatorCmd.Process.Signal(os.Interrupt)
	runner.evacuatorCmd.Process.Wait()
}

func (runner *CLIRunner) Cleanup() {
	os.Remove(runner.configPath)
}

func (runner *CLIRunner) start(command string, timestamp int) (*exec.Cmd, *cmdtest.Session) {
	cmd := exec.Command("hm9000", command, fmt.Sprintf("--config=%s", runner.configPath))
	cmd.Env = append(os.Environ(), fmt.Sprintf("HM9000_FAKE_TIME=%d", timestamp))

	var session *cmdtest.Session
	var err error
	if runner.verbose {
		session, err = cmdtest.StartWrapped(cmd, teeToStdout, teeToStdout)
	} else {
		session, err = cmdtest.Start(cmd)
	}

	Ω(err).ShouldNot(HaveOccured())

	Ω(session).Should(SayWithTimeout(".", 5*time.Second))

	return cmd, session
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

func teeToStdout(out io.Reader) io.Reader {
	return io.TeeReader(out, os.Stdout)
}
