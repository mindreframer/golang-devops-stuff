package bandwidth_manager_test

import (
	"errors"
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/backend/linux_backend/bandwidth_manager"
	"github.com/vito/garden/command_runner/fake_command_runner"
	. "github.com/vito/garden/command_runner/fake_command_runner/matchers"
)

var fakeRunner *fake_command_runner.FakeCommandRunner
var bandwidthManager *bandwidth_manager.ContainerBandwidthManager

var _ = Describe("setting rate limits", func() {
	BeforeEach(func() {
		fakeRunner = fake_command_runner.New()
		bandwidthManager = bandwidth_manager.New("/depot/some-id", "some-id", fakeRunner)
	})

	It("executes net_rate.sh with the appropriate environment", func() {
		limits := backend.BandwidthLimits{
			RateInBytesPerSecond:      128,
			BurstRateInBytesPerSecond: 256,
		}

		err := bandwidthManager.SetLimits(limits)
		Expect(err).ToNot(HaveOccured())

		Expect(fakeRunner).To(HaveExecutedSerially(
			fake_command_runner.CommandSpec{
				Path: "/depot/some-id/net_rate.sh",
				Env: []string{
					"BURST=256",
					fmt.Sprintf("RATE=%d", 128*8),
				},
			},
		))
	})

	Context("when net_rate.sh fails", func() {
		nastyError := errors.New("oh no!")

		BeforeEach(func() {
			fakeRunner.WhenRunning(
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/net_rate.sh",
				}, func(*exec.Cmd) error {
					return nastyError
				},
			)
		})

		It("returns the error", func() {
			err := bandwidthManager.SetLimits(backend.BandwidthLimits{
				RateInBytesPerSecond:      128,
				BurstRateInBytesPerSecond: 256,
			})
			Expect(err).To(Equal(nastyError))
		})
	})
})

var _ = Describe("getting bandwidth limits", func() {
	BeforeEach(func() {
		fakeRunner = fake_command_runner.New()
		bandwidthManager = bandwidth_manager.New("/depot/some-id", "some-id", fakeRunner)
	})

	It("executes net.sh get_egress_info and get_ingress_info", func() {
		fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
			Path: "/depot/some-id/net.sh",
			Args: []string{"get_egress_info"},
			Env:  []string{"ID=some-id"},
		}, func(cmd *exec.Cmd) error {
			cmd.Stdout.Write([]byte(`qdisc tbf 8010: root refcnt 2 rate 8192bit burst 64Kb lat 24.4ms
qdisc ingress ffff: parent ffff:fff1 ----------------
`))
			return nil
		})

		fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
			Path: "/depot/some-id/net.sh",
			Args: []string{"get_ingress_info"},
			Env:  []string{"ID=some-id"},
		}, func(cmd *exec.Cmd) error {
			cmd.Stdout.Write([]byte(`filter protocol ip pref 1 u32
filter protocol ip pref 1 u32 fh 800: ht divisor 1
filter protocol ip pref 1 u32 fh 800::800 order 2048 key ht 800 bkt 0 flowid :1
  match 00000000/00000000 at 12
 police 0x10 rate 8192bit burst 64Kb mtu 2Kb action drop overhead 0b
ref 1 bind 1
`))
			return nil
		})

		usage, err := bandwidthManager.GetLimits()
		Expect(err).ToNot(HaveOccured())

		Expect(usage.InRate).To(Equal(uint64(1024)))
		Expect(usage.InBurst).To(Equal(uint64(65536)))

		Expect(usage.OutRate).To(Equal(uint64(1024)))
		Expect(usage.OutBurst).To(Equal(uint64(65536)))
	})

	Context("when net.sh get_egress_info fails", func() {
		disaster := errors.New("oh no!")

		BeforeEach(func() {
			fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
				Path: "/depot/some-id/net.sh",
				Args: []string{"get_egress_info"},
				Env:  []string{"ID=some-id"},
			}, func(*exec.Cmd) error {
				return disaster
			})
		})

		It("returns the error", func() {
			_, err := bandwidthManager.GetLimits()
			Expect(err).To(Equal(disaster))
		})
	})

	Context("when net.sh get_egress_info output doesn't match", func() {
		It("returns 0 limits and does not error", func() {
			fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
				Path: "/depot/some-id/net.sh",
				Args: []string{"get_egress_info"},
				Env:  []string{"ID=some-id"},
			}, func(cmd *exec.Cmd) error {
				cmd.Stdout.Write([]byte(`qdisc pfifo_fast 0: root refcnt 2 bands 3 priomap  1 2 2 2 1 2 0 0 1 1 1 1 1 1 1 1
`))
				return nil
			})

			fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
				Path: "/depot/some-id/net.sh",
				Args: []string{"get_ingress_info"},
				Env:  []string{"ID=some-id"},
			}, func(cmd *exec.Cmd) error {
				cmd.Stdout.Write([]byte(`filter protocol ip pref 1 u32
filter protocol ip pref 1 u32 fh 800: ht divisor 1
filter protocol ip pref 1 u32 fh 800::800 order 2048 key ht 800 bkt 0 flowid :1
  match 00000000/00000000 at 12
 police 0x10 rate 8192bit burst 64Kb mtu 2Kb action drop overhead 0b
ref 1 bind 1
`))
				return nil
			})

			usage, err := bandwidthManager.GetLimits()
			Expect(err).ToNot(HaveOccured())

			Expect(usage.InRate).To(Equal(uint64(0)))
			Expect(usage.InBurst).To(Equal(uint64(0)))

			Expect(usage.OutRate).To(Equal(uint64(1024)))
			Expect(usage.OutBurst).To(Equal(uint64(65536)))
		})
	})

	Context("when net.sh get_ingress_info fails", func() {
		disaster := errors.New("oh no!")

		BeforeEach(func() {
			fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
				Path: "/depot/some-id/net.sh",
				Args: []string{"get_ingress_info"},
				Env:  []string{"ID=some-id"},
			}, func(*exec.Cmd) error {
				return disaster
			})
		})

		It("returns the error", func() {
			_, err := bandwidthManager.GetLimits()
			Expect(err).To(Equal(disaster))
		})
	})

	Context("when net.sh get_ingress_info output doesn't match", func() {
		It("returns 0 limits and does not error", func() {
			fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
				Path: "/depot/some-id/net.sh",
				Args: []string{"get_egress_info"},
				Env:  []string{"ID=some-id"},
			}, func(cmd *exec.Cmd) error {
				cmd.Stdout.Write([]byte(`qdisc tbf 8010: root refcnt 2 rate 8192bit burst 64Kb lat 24.4ms
qdisc ingress ffff: parent ffff:fff1 ----------------
`))
				return nil
			})

			fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
				Path: "/depot/some-id/net.sh",
				Args: []string{"get_ingress_info"},
				Env:  []string{"ID=some-id"},
			}, func(cmd *exec.Cmd) error {
				cmd.Stdout.Write([]byte(``))
				return nil
			})

			usage, err := bandwidthManager.GetLimits()
			Expect(err).ToNot(HaveOccured())

			Expect(usage.InRate).To(Equal(uint64(1024)))
			Expect(usage.InBurst).To(Equal(uint64(65536)))

			Expect(usage.OutRate).To(Equal(uint64(0)))
			Expect(usage.OutBurst).To(Equal(uint64(0)))
		})
	})
})
