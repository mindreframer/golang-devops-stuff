package server_test

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"time"

	"code.google.com/p/gogoprotobuf/proto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/garden/backend"
	"github.com/pivotal-cf-experimental/garden/backend/fake_backend"
	"github.com/pivotal-cf-experimental/garden/message_reader"
	protocol "github.com/pivotal-cf-experimental/garden/protocol"
	"github.com/pivotal-cf-experimental/garden/server"
)

var _ = Describe("The Warden server", func() {
	Context("when passed a socket", func() {
		It("listens on the given socket path and chmods it to 0777", func() {
			tmpdir, err := ioutil.TempDir(os.TempDir(), "warden-server-test")
			Expect(err).ToNot(HaveOccurred())

			socketPath := path.Join(tmpdir, "warden.sock")

			wardenServer := server.New("unix", socketPath, 0, fake_backend.New())

			err = wardenServer.Start()
			Expect(err).ToNot(HaveOccurred())

			Eventually(ErrorDialing("unix", socketPath)).ShouldNot(HaveOccurred())

			stat, err := os.Stat(socketPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(int(stat.Mode() & 0777)).To(Equal(0777))
		})

		It("deletes the socket file if it is already there", func() {
			tmpdir, err := ioutil.TempDir(os.TempDir(), "warden-server-test")
			Expect(err).ToNot(HaveOccurred())

			socketPath := path.Join(tmpdir, "warden.sock")

			socket, err := os.Create(socketPath)
			Expect(err).ToNot(HaveOccurred())
			socket.WriteString("oops")
			socket.Close()

			wardenServer := server.New("unix", socketPath, 0, fake_backend.New())

			err = wardenServer.Start()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when passed a tcp addr", func() {
		It("listens on the given addr", func() {
			wardenServer := server.New("tcp", ":60123", 0, fake_backend.New())

			err := wardenServer.Start()
			Expect(err).ToNot(HaveOccurred())

			Eventually(ErrorDialing("tcp", ":60123")).ShouldNot(HaveOccurred())
		})
	})

	It("starts the backend", func() {
		tmpdir, err := ioutil.TempDir(os.TempDir(), "warden-server-test")
		Expect(err).ToNot(HaveOccurred())

		socketPath := path.Join(tmpdir, "warden.sock")

		fakeBackend := fake_backend.New()

		wardenServer := server.New("unix", socketPath, 0, fakeBackend)

		err = wardenServer.Start()
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeBackend.Started).To(BeTrue())
	})

	It("destroys containers that have been idle for their grace time", func() {
		tmpdir, err := ioutil.TempDir(os.TempDir(), "warden-server-test")
		Expect(err).ToNot(HaveOccurred())

		socketPath := path.Join(tmpdir, "warden.sock")

		fakeBackend := fake_backend.New()

		_, err = fakeBackend.Create(backend.ContainerSpec{
			Handle:    "doomed",
			GraceTime: 100 * time.Millisecond,
		})
		Expect(err).ToNot(HaveOccurred())

		wardenServer := server.New("unix", socketPath, 0, fakeBackend)

		before := time.Now()

		err = wardenServer.Start()
		Expect(err).ToNot(HaveOccurred())

		_, err = fakeBackend.Lookup("doomed")
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			_, err := fakeBackend.Lookup("doomed")
			return err
		}).Should(HaveOccurred())

		Expect(time.Since(before)).To(BeNumerically(">", 100*time.Millisecond))
	})

	Context("when starting the backend fails", func() {
		disaster := errors.New("oh no!")

		It("fails to start", func() {
			tmpdir, err := ioutil.TempDir(os.TempDir(), "warden-server-test")
			Expect(err).ToNot(HaveOccurred())

			socketPath := path.Join(tmpdir, "warden.sock")

			fakeBackend := fake_backend.New()
			fakeBackend.StartError = disaster

			wardenServer := server.New("unix", socketPath, 0, fakeBackend)

			err = wardenServer.Start()
			Expect(err).To(Equal(disaster))
		})
	})

	Context("when listening on the socket fails", func() {
		It("fails to start", func() {
			tmpfile, err := ioutil.TempFile(os.TempDir(), "warden-server-test")
			Expect(err).ToNot(HaveOccurred())

			wardenServer := server.New(
				"unix",
				// weird scenario: /foo/X/warden.sock with X being a file
				path.Join(tmpfile.Name(), "warden.sock"),
				0,
				fake_backend.New(),
			)

			err = wardenServer.Start()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("shutting down", func() {
		var socketPath string

		var serverBackend backend.Backend
		var fakeBackend *fake_backend.FakeBackend

		var wardenServer *server.WardenServer

		var serverConnection net.Conn
		var responses *bufio.Reader

		BeforeEach(func() {
			tmpdir, err := ioutil.TempDir(os.TempDir(), "warden-server-test")
			Expect(err).ToNot(HaveOccurred())

			socketPath = path.Join(tmpdir, "warden.sock")
			fakeBackend = fake_backend.New()

			serverBackend = fakeBackend
		})

		JustBeforeEach(func() {
			wardenServer = server.New("unix", socketPath, 0, serverBackend)

			err := wardenServer.Start()
			Expect(err).ToNot(HaveOccurred())

			Eventually(ErrorDialing("unix", socketPath)).ShouldNot(HaveOccurred())

			serverConnection, err = net.Dial("unix", socketPath)
			Expect(err).ToNot(HaveOccurred())

			responses = bufio.NewReader(serverConnection)
		})

		writeMessages := func(message proto.Message) {
			num, err := protocol.Messages(message).WriteTo(serverConnection)
			Expect(err).ToNot(HaveOccurred())
			Expect(num).ToNot(Equal(0))
		}

		readResponse := func(response proto.Message) {
			err := message_reader.ReadMessage(responses, response)
			Expect(err).ToNot(HaveOccurred())
		}

		It("stops accepting new connections", func() {
			go wardenServer.Stop()
			Eventually(ErrorDialing("unix", socketPath)).Should(HaveOccurred())
		})

		It("stops handling requests on existing connections", func() {
			writeMessages(&protocol.PingRequest{})
			readResponse(&protocol.PingResponse{})

			go wardenServer.Stop()

			// server was already reading a request
			_, err := protocol.Messages(&protocol.PingRequest{}).WriteTo(serverConnection)
			Expect(err).ToNot(HaveOccurred())

			// server will not actually handle it
			err = message_reader.ReadMessage(responses, &protocol.PingResponse{})
			Expect(err).To(HaveOccurred())
		})

		It("stops the backend", func() {
			wardenServer.Stop()

			Expect(fakeBackend.Stopped).To(BeTrue())
		})

		Context("when a Create request is in-flight", func() {
			BeforeEach(func() {
				serverBackend = fake_backend.NewSlow(100 * time.Millisecond)
			})

			It("waits for it to complete and stops accepting requests", func() {
				writeMessages(&protocol.CreateRequest{})

				time.Sleep(10 * time.Millisecond)

				before := time.Now()

				wardenServer.Stop()

				Expect(time.Since(before)).To(BeNumerically(">", 50*time.Millisecond))

				readResponse(&protocol.CreateResponse{})

				_, err := protocol.Messages(&protocol.PingRequest{}).WriteTo(serverConnection)
				Expect(err).To(HaveOccurred())
			})
		})

		dontWaitRequests := []proto.Message{
			&protocol.LinkRequest{
				Handle: proto.String("some-handle"),
				JobId:  proto.Uint32(1),
			},
			&protocol.StreamRequest{
				Handle: proto.String("some-handle"),
				JobId:  proto.Uint32(1),
			},
			&protocol.RunRequest{
				Handle: proto.String("some-handle"),
				Script: proto.String("some-script"),
			},
		}

		for _, req := range dontWaitRequests {
			request := req

			Context(fmt.Sprintf("when a %T request is in-flight", request), func() {
				BeforeEach(func() {
					serverBackend = fake_backend.NewSlow(100 * time.Millisecond)

					container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
					Expect(err).ToNot(HaveOccurred())

					exitStatus := uint32(42)

					fakeContainer := container.(*fake_backend.FakeContainer)

					fakeContainer.StreamedJobChunks = []backend.JobStream{
						{
							ExitStatus: &exitStatus,
						},
					}
				})

				It("does not wait for it to complete", func() {
					writeMessages(request)

					time.Sleep(10 * time.Millisecond)

					before := time.Now()

					wardenServer.Stop()

					Expect(time.Since(before)).To(BeNumerically("<", 50*time.Millisecond))

					response := protocol.ResponseMessageForType(protocol.TypeForMessage(request))
					readResponse(response)

					_, err := protocol.Messages(&protocol.PingRequest{}).WriteTo(serverConnection)
					Expect(err).To(HaveOccurred())
				})
			})
		}
	})
})
