package daemon_test

import (
	"encoding/gob"
	"io/ioutil"
	"net"
	"os"
	"path"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"

	"github.com/vito/garden/backend/linux_backend/wshd/daemon"
	"github.com/vito/garden/backend/linux_backend/wshd/protocol"
)

var _ = Describe("Handling command requests", func() {
	var socketPath string

	var serverConnection *net.UnixConn

	var stdin *os.File
	var outExpect *cmdtest.Expector
	var errExpect *cmdtest.Expector
	var status *os.File

	BeforeEach(func() {
		tmpdir, err := ioutil.TempDir(os.TempDir(), "warden-server-test")
		Expect(err).ToNot(HaveOccured())

		socketPath = path.Join(tmpdir, "warden.sock")

		listener, err := net.Listen("unix", socketPath)
		Expect(err).ToNot(HaveOccured())

		socketFile, err := listener.(*net.UnixListener).File()
		Expect(err).ToNot(HaveOccured())

		daemonServer := daemon.New(socketFile)

		err = daemonServer.Start()
		Expect(err).ToNot(HaveOccured())

		Eventually(ErrorDialingUnix(socketPath)).ShouldNot(HaveOccured())

		conn, err := net.Dial("unix", socketPath)
		Expect(err).ToNot(HaveOccured())

		conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		serverConnection = conn.(*net.UnixConn)
	})

	readExitStatus := func() int {
		decoder := gob.NewDecoder(status)

		var exitStatus protocol.ExitStatusMessage

		err := decoder.Decode(&exitStatus)
		Expect(err).ToNot(HaveOccured())

		return exitStatus.ExitStatus
	}

	Describe("running commands", func() {
		var request protocol.RequestMessage

		BeforeEach(func() {
			request = protocol.RequestMessage{
				Argv: []string{"/bin/bash"},
			}
		})

		JustBeforeEach(func() {
			encoder := gob.NewEncoder(serverConnection)

			err := encoder.Encode(request)
			Expect(err).ToNot(HaveOccured())

			fds := readFDs(serverConnection)

			stdin = os.NewFile(uintptr(fds[0]), "stdin")
			stdout := os.NewFile(uintptr(fds[1]), "stdout")
			stderr := os.NewFile(uintptr(fds[2]), "stderr")
			status = os.NewFile(uintptr(fds[3]), "status")

			outExpect = cmdtest.NewExpector(stdout, 1*time.Second)
			errExpect = cmdtest.NewExpector(stderr, 1*time.Second)
		})

		It("sends file descriptors for stdin, stdout, stderr, and exit status", func() {
			stdin.Write([]byte("echo hi out\n"))
			stdin.Write([]byte("echo hi err 1>&2\n"))
			stdin.Write([]byte("exit 42\n"))
			stdin.Close()

			err := outExpect.Expect("hi out\n")
			Expect(err).ToNot(HaveOccured())

			err = errExpect.Expect("hi err\n")
			Expect(err).ToNot(HaveOccured())

			Expect(readExitStatus()).To(Equal(42))
		})

		It("runs the command with $PATH set", func() {
			stdin.Write([]byte("echo PATH: $PATH\n"))
			stdin.Close()

			err := outExpect.Expect("PATH: /sbin:/bin:/usr/sbin:/usr/bin\n")
			Expect(err).ToNot(HaveOccured())
		})

		It("runs with $HOME and $USER set", func() {
			stdin.Write([]byte("echo $USER\n"))
			stdin.Write([]byte("echo $HOME\n"))
			stdin.Close()

			err := outExpect.Expect("root\n")
			Expect(err).ToNot(HaveOccured())

			err = outExpect.Expect("/root\n")
			Expect(err).ToNot(HaveOccured())
		})

		Context("when a user is given", func() {
			Context("but the user does not exist", func() {
				BeforeEach(func() {
					request.User = "bogus-user"
				})

				It("returns exit status 255", func() {
					Expect(readExitStatus()).To(Equal(255))
				})
			})
		})

		Context("when a command fails to start", func() {
			BeforeEach(func() {
				request.Argv = []string{"/bogus/path"}
			})

			It("returns exit status 255", func() {
				Expect(readExitStatus()).To(Equal(255))
			})
		})

		Context("when a command is given without an absolute path", func() {
			BeforeEach(func() {
				request.Argv = []string{"ifconfig"}
			})

			It("resolves the executable in the user's $PATH", func() {
				Expect(readExitStatus()).To(Equal(0))
			})
		})
	})
})

func readFDs(conn *net.UnixConn) []int {
	var b [2048]byte
	var oob [2048]byte

	_, oobn, _, _, err := conn.ReadMsgUnix(b[:], oob[:])
	Expect(err).ToNot(HaveOccured())

	scms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	Expect(err).ToNot(HaveOccured())

	Expect(len(scms)).To(Equal(1))

	scm := scms[0]

	fds, err := syscall.ParseUnixRights(&scm)
	Expect(err).ToNot(HaveOccured())

	Expect(len(fds)).To(Equal(4))

	return fds
}
func ErrorDialingUnix(socketPath string) func() error {
	return func() error {
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			conn.Close()
		}

		return err
	}
}
