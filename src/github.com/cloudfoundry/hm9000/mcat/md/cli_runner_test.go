package md_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/hm9000/config"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type CLIRunner struct {
	configPath           string
	listenerCmd          *exec.Cmd
	listenerStdoutBuffer *bytes.Buffer
	metricsServerCmd     *exec.Cmd
	apiServerCmd         *exec.Cmd
	verbose              bool
}

func NewCLIRunner(storeURLs []string, ccBaseURL string, natsPort int, metricsServerPort int, verbose bool) *CLIRunner {
	runner := &CLIRunner{
		verbose: verbose,
	}
	runner.generateConfig(storeURLs, ccBaseURL, natsPort, metricsServerPort)
	return runner
}

func (runner *CLIRunner) generateConfig(storeURLs []string, ccBaseURL string, natsPort int, metricsServerPort int) {
	tmpFile, err := ioutil.TempFile("/tmp", "hm9000_clirunner")
	defer tmpFile.Close()
	Ω(err).ShouldNot(HaveOccured())

	runner.configPath = tmpFile.Name()

	conf, err := config.DefaultConfig()
	Ω(err).ShouldNot(HaveOccured())
	conf.StoreURLs = storeURLs
	conf.CCBaseURL = ccBaseURL
	conf.NATS.Port = natsPort
	conf.SenderMessageLimit = 8
	conf.MaximumBackoffDelayInHeartbeats = 6
	conf.MetricsServerPort = metricsServerPort
	conf.MetricsServerUser = "bob"
	conf.MetricsServerPassword = "password"

	err = json.NewEncoder(tmpFile).Encode(conf)
	Ω(err).ShouldNot(HaveOccured())
}

func (runner *CLIRunner) StartListener(timestamp int) {
	runner.listenerCmd, runner.listenerStdoutBuffer = runner.start("listen", timestamp)
}

func (runner *CLIRunner) StopListener() {
	runner.listenerCmd.Process.Kill()
}

func (runner *CLIRunner) StartMetricsServer(timestamp int) {
	runner.metricsServerCmd, _ = runner.start("serve_metrics", timestamp)
}

func (runner *CLIRunner) StopMetricsServer() {
	runner.metricsServerCmd.Process.Kill()
}

func (runner *CLIRunner) StartAPIServer(timestamp int) {
	runner.apiServerCmd, _ = runner.start("serve_api", timestamp)
}

func (runner *CLIRunner) StopAPIServer() {
	runner.apiServerCmd.Process.Kill()
}

func (runner *CLIRunner) Cleanup() {
	os.Remove(runner.configPath)
}

func (runner *CLIRunner) start(command string, timestamp int) (*exec.Cmd, *bytes.Buffer) {
	cmd := exec.Command("hm9000", command, fmt.Sprintf("--config=%s", runner.configPath))
	cmd.Env = append(os.Environ(), fmt.Sprintf("HM9000_FAKE_TIME=%d", timestamp))
	buffer := bytes.NewBuffer([]byte{})
	cmd.Stdout = buffer
	cmd.Start()
	Eventually(func() int {
		return buffer.Len()
	}).ShouldNot(BeZero())

	return cmd, buffer
}

func (runner *CLIRunner) WaitForHeartbeats(num int) {
	Eventually(func() int {
		var validHeartbeat = regexp.MustCompile(`Received dea.heartbeat`)
		heartbeats := validHeartbeat.FindAll(runner.listenerStdoutBuffer.Bytes(), -1)
		return len(heartbeats)
	}).Should(BeNumerically("==", num))
}

func (runner *CLIRunner) Run(command string, timestamp int) {
	cmd := exec.Command("hm9000", command, fmt.Sprintf("--config=%s", runner.configPath))
	cmd.Env = append(os.Environ(), fmt.Sprintf("HM9000_FAKE_TIME=%d", timestamp))
	out, _ := cmd.CombinedOutput()
	if runner.verbose {
		fmt.Printf(command + "\n")
		fmt.Printf(strings.Repeat("~", len(command)) + "\n")
		fmt.Printf(string(out))

		fmt.Printf("\n")
	}
	// Ω(err).ShouldNot(HaveOccured(), "%s command failed", command)
	time.Sleep(50 * time.Millisecond) //give NATS a chance to send messages around, if necessary
}
