package server_test

import (
	"bufio"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"path"
	"time"

	"code.google.com/p/gogoprotobuf/proto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/backend/fake_backend"
	"github.com/vito/garden/message_reader"
	protocol "github.com/vito/garden/protocol"
	"github.com/vito/garden/server"
)

var _ = Describe("The Warden server", func() {
	It("listens on the given socket path and chmods it to 0777", func() {
		tmpdir, err := ioutil.TempDir(os.TempDir(), "warden-server-test")
		Expect(err).ToNot(HaveOccured())

		socketPath := path.Join(tmpdir, "warden.sock")

		wardenServer := server.New(socketPath, fake_backend.New())

		err = wardenServer.Start()
		Expect(err).ToNot(HaveOccured())

		Eventually(ErrorDialingUnix(socketPath)).ShouldNot(HaveOccured())

		stat, err := os.Stat(socketPath)
		Expect(err).ToNot(HaveOccured())

		Expect(int(stat.Mode() & 0777)).To(Equal(0777))
	})

	Context("when starting fails", func() {
		It("returns the error", func() {
			tmpfile, err := ioutil.TempFile(os.TempDir(), "warden-server-test")
			Expect(err).ToNot(HaveOccured())

			wardenServer := server.New(
				// weird scenario: /foo/X/warden.sock with X being a file
				path.Join(tmpfile.Name(), "warden.sock"),
				fake_backend.New(),
			)

			err = wardenServer.Start()
			Expect(err).To(HaveOccured())
		})
	})

	Context("when a client connects", func() {
		var socketPath string
		var serverBackend *fake_backend.FakeBackend

		var serverConnection net.Conn
		var responses *bufio.Reader

		BeforeEach(func() {
			tmpdir, err := ioutil.TempDir(os.TempDir(), "warden-server-test")
			Expect(err).ToNot(HaveOccured())

			socketPath = path.Join(tmpdir, "warden.sock")
			serverBackend = fake_backend.New()

			wardenServer := server.New(socketPath, serverBackend)

			err = wardenServer.Start()
			Expect(err).ToNot(HaveOccured())

			Eventually(ErrorDialingUnix(socketPath)).ShouldNot(HaveOccured())

			serverConnection, err = net.Dial("unix", socketPath)
			Expect(err).ToNot(HaveOccured())

			responses = bufio.NewReader(serverConnection)
		})

		writeMessages := func(message proto.Message) {
			num, err := protocol.Messages(message).WriteTo(serverConnection)
			Expect(err).ToNot(HaveOccured())
			Expect(num).ToNot(Equal(0))
		}

		readResponse := func(response proto.Message) {
			err := message_reader.ReadMessage(responses, response)
			Expect(err).ToNot(HaveOccured())
		}

		Context("and the client sends a PingRequest", func() {
			It("sends a PongResponse", func(done Done) {
				writeMessages(&protocol.PingRequest{})
				readResponse(&protocol.PingResponse{})
				close(done)
			}, 1.0)
		})

		Context("and the client sends a EchoRequest", func() {
			It("sends an EchoResponse with the same message", func(done Done) {
				message := proto.String("Hello, world!")

				writeMessages(&protocol.EchoRequest{Message: message})

				var response protocol.EchoResponse
				readResponse(&response)

				Expect(response.GetMessage()).To(Equal(*message))

				close(done)
			}, 1.0)
		})

		Context("and the client sends a CreateRequest", func() {
			It("sends a CreateResponse with the created handle", func(done Done) {
				writeMessages(&protocol.CreateRequest{
					Handle: proto.String("some-handle"),
				})

				var response protocol.CreateResponse
				readResponse(&response)

				Expect(response.GetHandle()).To(Equal("some-handle"))

				close(done)
			}, 1.0)

			It("creates the container with the spec from the request", func(done Done) {
				var bindMountMode protocol.CreateRequest_BindMount_Mode

				bindMountMode = protocol.CreateRequest_BindMount_RW

				writeMessages(&protocol.CreateRequest{
					Handle:    proto.String("some-handle"),
					GraceTime: proto.Uint32(42),
					Network:   proto.String("some-network"),
					Rootfs:    proto.String("/path/to/rootfs"),
					BindMounts: []*protocol.CreateRequest_BindMount{
						{
							SrcPath: proto.String("/bind/mount/src"),
							DstPath: proto.String("/bind/mount/dst"),
							Mode:    &bindMountMode,
						},
					},
				})

				var response protocol.CreateResponse
				readResponse(&response)

				container, found := serverBackend.CreatedContainers[response.GetHandle()]
				Expect(found).To(BeTrue())

				Expect(container.Spec).To(Equal(backend.ContainerSpec{
					Handle:     "some-handle",
					GraceTime:  time.Duration(42 * time.Second),
					Network:    "some-network",
					RootFSPath: "/path/to/rootfs",
					BindMounts: []backend.BindMount{
						{
							SrcPath: "/bind/mount/src",
							DstPath: "/bind/mount/dst",
							Mode:    backend.BindMountModeRW,
						},
					},
				}))

				close(done)
			}, 1.0)

			Context("when creating the container fails", func() {
				BeforeEach(func() {
					serverBackend.CreateError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.CreateRequest{
						Handle: proto.String("some-handle"),
					})

					var response protocol.CreateResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a DestroyRequest", func() {
			BeforeEach(func() {
				_, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())
			})

			It("destroys the container and sends a DestroyResponse", func(done Done) {
				writeMessages(&protocol.DestroyRequest{
					Handle: proto.String("some-handle"),
				})

				var response protocol.DestroyResponse
				readResponse(&response)

				Expect(serverBackend.CreatedContainers).ToNot(HaveKey("some-handle"))

				close(done)
			}, 1.0)

			Context("when destroying the container fails", func() {
				BeforeEach(func() {
					serverBackend.DestroyError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.DestroyRequest{
						Handle: proto.String("some-handle"),
					})

					var response protocol.DestroyResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a ListRequest", func() {
			BeforeEach(func() {
				_, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				_, err = serverBackend.Create(backend.ContainerSpec{Handle: "another-handle"})
				Expect(err).ToNot(HaveOccured())
			})

			It("sends a ListResponse containing the existing handles", func(done Done) {
				writeMessages(&protocol.ListRequest{})

				var response protocol.ListResponse
				readResponse(&response)

				Expect(response.GetHandles()).To(ContainElement("some-handle"))
				Expect(response.GetHandles()).To(ContainElement("another-handle"))

				close(done)
			}, 1.0)

			Context("when getting the containers fails", func() {
				BeforeEach(func() {
					serverBackend.ContainersError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.ListRequest{})

					var response protocol.ListResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a StopRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("stops the container and sends a StopResponse", func(done Done) {
				writeMessages(&protocol.StopRequest{
					Handle: proto.String(fakeContainer.Handle()),
					Kill:   proto.Bool(true),
				})

				var response protocol.StopResponse
				readResponse(&response)

				Expect(fakeContainer.Stopped).To(ContainElement(
					fake_backend.StopSpec{
						Killed: true,
					},
				))

				close(done)
			}, 1.0)

			Context("when background is true", func() {
				It("stops async and returns immediately", func(done Done) {
					fakeContainer.StopCallback = func() {
						time.Sleep(1 * time.Second)
					}

					writeMessages(&protocol.StopRequest{
						Handle:     proto.String(fakeContainer.Handle()),
						Kill:       proto.Bool(true),
						Background: proto.Bool(true),
					})

					var response protocol.StopResponse
					readResponse(&response)

					Expect(fakeContainer.Stopped).To(BeEmpty())

					close(done)
				}, 1.0)
			})

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.StopRequest{
						Handle: proto.String(fakeContainer.Handle()),
					})

					var response protocol.StopResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when stopping the container fails", func() {
				BeforeEach(func() {
					fakeContainer.StopError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.StopRequest{
						Handle: proto.String("some-handle"),
					})

					var response protocol.StopResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a CopyInRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("copies the file in and sends a CopyInResponse", func(done Done) {
				writeMessages(&protocol.CopyInRequest{
					Handle:  proto.String(fakeContainer.Handle()),
					SrcPath: proto.String("/src/path"),
					DstPath: proto.String("/dst/path"),
				})

				var response protocol.CopyInResponse
				readResponse(&response)

				Expect(fakeContainer.CopiedIn).To(ContainElement(
					[]string{"/src/path", "/dst/path"},
				))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.CopyInRequest{
						Handle:  proto.String(fakeContainer.Handle()),
						SrcPath: proto.String("/src/path"),
						DstPath: proto.String("/dst/path"),
					})

					var response protocol.CopyInResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when copying in to the container fails", func() {
				BeforeEach(func() {
					fakeContainer.CopyInError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.CopyInRequest{
						Handle:  proto.String(fakeContainer.Handle()),
						SrcPath: proto.String("/src/path"),
						DstPath: proto.String("/dst/path"),
					})

					var response protocol.CopyInResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a CopyOutRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("copies the file out and sends a CopyOutResponse", func(done Done) {
				writeMessages(&protocol.CopyOutRequest{
					Handle:  proto.String(fakeContainer.Handle()),
					SrcPath: proto.String("/src/path"),
					DstPath: proto.String("/dst/path"),
					Owner:   proto.String("someuser"),
				})

				var response protocol.CopyOutResponse
				readResponse(&response)

				Expect(fakeContainer.CopiedOut).To(ContainElement(
					[]string{"/src/path", "/dst/path", "someuser"},
				))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.CopyOutRequest{
						Handle:  proto.String(fakeContainer.Handle()),
						SrcPath: proto.String("/src/path"),
						DstPath: proto.String("/dst/path"),
						Owner:   proto.String("someuser"),
					})

					var response protocol.CopyOutResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when copying out of the container fails", func() {
				BeforeEach(func() {
					fakeContainer.CopyOutError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.CopyOutRequest{
						Handle:  proto.String(fakeContainer.Handle()),
						SrcPath: proto.String("/src/path"),
						DstPath: proto.String("/dst/path"),
						Owner:   proto.String("someuser"),
					})

					var response protocol.CopyOutResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a SpawnRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("spawns a job and sends a SpawnResponse", func(done Done) {
				fakeContainer.SpawnedJobID = 42

				writeMessages(&protocol.SpawnRequest{
					Handle:        proto.String(fakeContainer.Handle()),
					Script:        proto.String("/some/script"),
					Privileged:    proto.Bool(true),
					DiscardOutput: proto.Bool(true),
				})

				var response protocol.SpawnResponse
				readResponse(&response)

				Expect(response.GetJobId()).To(Equal(uint32(42)))

				Expect(fakeContainer.Spawned).To(ContainElement(
					backend.JobSpec{
						Script:        "/some/script",
						Privileged:    true,
						DiscardOutput: true,
					},
				))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.SpawnRequest{
						Handle: proto.String(fakeContainer.Handle()),
						Script: proto.String("/some/script"),
					})

					var response protocol.SpawnResponse

					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when spawning fails", func() {
				BeforeEach(func() {
					fakeContainer.SpawnError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.SpawnRequest{
						Handle: proto.String(fakeContainer.Handle()),
						Script: proto.String("/some/script"),
					})

					var response protocol.SpawnResponse

					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a LinkRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("links to the job and sends a LinkResponse", func(done Done) {
				fakeContainer.LinkedJobResult = backend.JobResult{
					ExitStatus: 42,
					Stdout:     []byte("job out\n"),
					Stderr:     []byte("job err\n"),
				}

				writeMessages(&protocol.LinkRequest{
					Handle: proto.String(fakeContainer.Handle()),
					JobId:  proto.Uint32(123),
				})

				var response protocol.LinkResponse
				readResponse(&response)

				Expect(response.GetExitStatus()).To(Equal(uint32(42)))
				Expect(response.GetStdout()).To(Equal("job out\n"))
				Expect(response.GetStderr()).To(Equal("job err\n"))

				Expect(fakeContainer.Linked).To(ContainElement(uint32(123)))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.LinkRequest{
						Handle: proto.String(fakeContainer.Handle()),
						JobId:  proto.Uint32(123),
					})

					var response protocol.LinkResponse

					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when linking fails", func() {
				BeforeEach(func() {
					fakeContainer.LinkError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.LinkRequest{
						Handle: proto.String(fakeContainer.Handle()),
						JobId:  proto.Uint32(123),
					})

					var response protocol.LinkResponse

					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a StreamRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("responds with a StreamResponse for every chunk", func(done Done) {
				exitStatus := uint32(42)

				fakeContainer.StreamedJobChunks = []backend.JobStream{
					{
						Name:       "stdout",
						Data:       []byte("job out\n"),
						ExitStatus: nil,
					},
					{
						Name:       "stderr",
						Data:       []byte("job err\n"),
						ExitStatus: nil,
					},
					{
						ExitStatus: &exitStatus,
					},
				}

				writeMessages(&protocol.StreamRequest{
					Handle: proto.String(fakeContainer.Handle()),
					JobId:  proto.Uint32(123),
				})

				var response1 protocol.StreamResponse
				readResponse(&response1)

				Expect(response1.GetName()).To(Equal("stdout"))
				Expect(response1.GetData()).To(Equal("job out\n"))
				Expect(response1.ExitStatus).To(BeNil())

				var response2 protocol.StreamResponse
				readResponse(&response2)

				Expect(response2.GetName()).To(Equal("stderr"))
				Expect(response2.GetData()).To(Equal("job err\n"))
				Expect(response2.ExitStatus).To(BeNil())

				var response3 protocol.StreamResponse
				readResponse(&response3)

				Expect(response3.GetName()).To(Equal(""))
				Expect(response3.GetData()).To(Equal(""))
				Expect(response3.GetExitStatus()).To(Equal(uint32(42)))

				Expect(fakeContainer.Streamed).To(ContainElement(uint32(123)))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.StreamRequest{
						Handle: proto.String(fakeContainer.Handle()),
						JobId:  proto.Uint32(123),
					})

					var response protocol.StreamResponse

					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when streaming fails", func() {
				BeforeEach(func() {
					fakeContainer.StreamError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.StreamRequest{
						Handle: proto.String(fakeContainer.Handle()),
						JobId:  proto.Uint32(123),
					})

					var response protocol.StreamResponse

					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a RunRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("spawns a job, links to it, and sends a RunResponse", func(done Done) {
				fakeContainer.SpawnedJobID = 123

				fakeContainer.LinkedJobResult = backend.JobResult{
					ExitStatus: 42,
					Stdout:     []byte("job out\n"),
					Stderr:     []byte("job err\n"),
				}

				writeMessages(&protocol.RunRequest{
					Handle:        proto.String(fakeContainer.Handle()),
					Script:        proto.String("/some/script"),
					Privileged:    proto.Bool(true),
					DiscardOutput: proto.Bool(true),
				})

				var response protocol.RunResponse
				readResponse(&response)

				Expect(fakeContainer.Spawned).To(ContainElement(
					backend.JobSpec{
						Script:        "/some/script",
						Privileged:    true,
						DiscardOutput: true,
					},
				))

				Expect(fakeContainer.Linked).To(ContainElement(uint32(123)))

				Expect(response.GetExitStatus()).To(Equal(uint32(42)))
				Expect(response.GetStdout()).To(Equal("job out\n"))
				Expect(response.GetStderr()).To(Equal("job err\n"))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.RunRequest{
						Handle: proto.String(fakeContainer.Handle()),
						Script: proto.String("/some/script"),
					})

					var response protocol.RunResponse

					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when spawning fails", func() {
				BeforeEach(func() {
					fakeContainer.SpawnError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.RunRequest{
						Handle: proto.String(fakeContainer.Handle()),
						Script: proto.String("/some/script"),
					})

					var response protocol.RunResponse

					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})

			Context("when linking fails", func() {
				BeforeEach(func() {
					fakeContainer.LinkError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.RunRequest{
						Handle: proto.String(fakeContainer.Handle()),
						Script: proto.String("/some/script"),
					})

					var response protocol.RunResponse

					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a LimitBandwidthRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("sets the container's bandwidth limits and returns them", func(done Done) {
				writeMessages(&protocol.LimitBandwidthRequest{
					Handle: proto.String(fakeContainer.Handle()),
					Rate:   proto.Uint64(123),
					Burst:  proto.Uint64(456),
				})

				var response protocol.LimitBandwidthResponse
				readResponse(&response)

				Expect(fakeContainer.LimitedBandwidth).To(Equal(
					backend.BandwidthLimits{
						RateInBytesPerSecond:      123,
						BurstRateInBytesPerSecond: 456,
					},
				))

				Expect(response.GetRate()).To(Equal(uint64(123)))
				Expect(response.GetBurst()).To(Equal(uint64(456)))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.LimitBandwidthRequest{
						Handle: proto.String(fakeContainer.Handle()),
						Rate:   proto.Uint64(123),
						Burst:  proto.Uint64(456),
					})

					var response protocol.LimitBandwidthResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when limiting the bandwidth fails", func() {
				BeforeEach(func() {
					fakeContainer.LimitBandwidthError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.LimitBandwidthRequest{
						Handle: proto.String(fakeContainer.Handle()),
						Rate:   proto.Uint64(123),
						Burst:  proto.Uint64(456),
					})

					var response protocol.LimitBandwidthResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a LimitMemoryRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("sets the container's memory limits and returns them", func(done Done) {
				writeMessages(&protocol.LimitMemoryRequest{
					Handle:       proto.String(fakeContainer.Handle()),
					LimitInBytes: proto.Uint64(123),
				})

				var response protocol.LimitMemoryResponse
				readResponse(&response)

				Expect(fakeContainer.LimitedMemory).To(Equal(
					backend.MemoryLimits{
						LimitInBytes: 123,
					},
				))

				Expect(response.GetLimitInBytes()).To(Equal(uint64(123)))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.LimitMemoryRequest{
						Handle:       proto.String(fakeContainer.Handle()),
						LimitInBytes: proto.Uint64(123),
					})

					var response protocol.LimitMemoryResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when limiting the memory fails", func() {
				BeforeEach(func() {
					fakeContainer.LimitMemoryError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.LimitMemoryRequest{
						Handle:       proto.String(fakeContainer.Handle()),
						LimitInBytes: proto.Uint64(123),
					})

					var response protocol.LimitMemoryResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a LimitDiskRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("sets the container's disk limits and returns them", func(done Done) {
				fakeContainer.LimitedDiskResult = backend.DiskLimits{
					BlockSoft: 1111,
					BlockHard: 2222,

					InodeSoft: 3333,
					InodeHard: 4444,

					ByteSoft: 5555,
					ByteHard: 6666,
				}

				writeMessages(&protocol.LimitDiskRequest{
					Handle:    proto.String(fakeContainer.Handle()),
					BlockSoft: proto.Uint64(111),
					BlockHard: proto.Uint64(222),
					InodeSoft: proto.Uint64(333),
					InodeHard: proto.Uint64(444),
					ByteSoft:  proto.Uint64(555),
					ByteHard:  proto.Uint64(666),
				})

				var response protocol.LimitDiskResponse
				readResponse(&response)

				Expect(fakeContainer.LimitedDisk).To(Equal(
					backend.DiskLimits{
						BlockSoft: 111,
						BlockHard: 222,

						InodeSoft: 333,
						InodeHard: 444,

						ByteSoft: 555,
						ByteHard: 666,
					},
				))

				Expect(response.GetBlockSoft()).To(Equal(uint64(1111)))
				Expect(response.GetBlockHard()).To(Equal(uint64(2222)))

				Expect(response.GetInodeSoft()).To(Equal(uint64(3333)))
				Expect(response.GetInodeHard()).To(Equal(uint64(4444)))

				Expect(response.GetByteSoft()).To(Equal(uint64(5555)))
				Expect(response.GetByteHard()).To(Equal(uint64(6666)))

				close(done)
			}, 1.0)

			Context("when Block is given", func() {
				It("passes it as BlockHard", func() {
					writeMessages(&protocol.LimitDiskRequest{
						Handle:    proto.String(fakeContainer.Handle()),
						BlockSoft: proto.Uint64(111),
						Block:     proto.Uint64(222),
						InodeSoft: proto.Uint64(333),
						InodeHard: proto.Uint64(444),
					})

					var response protocol.LimitDiskResponse
					readResponse(&response)

					Expect(fakeContainer.LimitedDisk).To(Equal(
						backend.DiskLimits{
							BlockSoft: 111,
							BlockHard: 222,

							InodeSoft: 333,
							InodeHard: 444,
						},
					))
				})
			})

			Context("when BlockLimit is given", func() {
				It("passes it as BlockHard", func() {
					writeMessages(&protocol.LimitDiskRequest{
						Handle:     proto.String(fakeContainer.Handle()),
						BlockSoft:  proto.Uint64(111),
						BlockLimit: proto.Uint64(222),
						InodeSoft:  proto.Uint64(333),
						InodeHard:  proto.Uint64(444),
					})

					var response protocol.LimitDiskResponse
					readResponse(&response)

					Expect(fakeContainer.LimitedDisk).To(Equal(
						backend.DiskLimits{
							BlockSoft: 111,
							BlockHard: 222,

							InodeSoft: 333,
							InodeHard: 444,
						},
					))
				})
			})

			Context("when Inode is given", func() {
				It("passes it as InodeHard", func() {
					writeMessages(&protocol.LimitDiskRequest{
						Handle:    proto.String(fakeContainer.Handle()),
						BlockSoft: proto.Uint64(111),
						BlockHard: proto.Uint64(222),
						InodeSoft: proto.Uint64(333),
						Inode:     proto.Uint64(444),
					})

					var response protocol.LimitDiskResponse
					readResponse(&response)

					Expect(fakeContainer.LimitedDisk).To(Equal(
						backend.DiskLimits{
							BlockSoft: 111,
							BlockHard: 222,

							InodeSoft: 333,
							InodeHard: 444,
						},
					))
				})
			})

			Context("when InodeLimit is given", func() {
				It("passes it as InodeHard", func() {
					writeMessages(&protocol.LimitDiskRequest{
						Handle:     proto.String(fakeContainer.Handle()),
						BlockSoft:  proto.Uint64(111),
						BlockHard:  proto.Uint64(222),
						InodeSoft:  proto.Uint64(333),
						InodeLimit: proto.Uint64(444),
					})

					var response protocol.LimitDiskResponse
					readResponse(&response)

					Expect(fakeContainer.LimitedDisk).To(Equal(
						backend.DiskLimits{
							BlockSoft: 111,
							BlockHard: 222,

							InodeSoft: 333,
							InodeHard: 444,
						},
					))
				})
			})

			Context("when Byte is given", func() {
				It("passes it as ByteHard", func() {
					writeMessages(&protocol.LimitDiskRequest{
						Handle:    proto.String(fakeContainer.Handle()),
						BlockSoft: proto.Uint64(111),
						BlockHard: proto.Uint64(222),
						InodeSoft: proto.Uint64(333),
						InodeHard: proto.Uint64(444),
						Byte:      proto.Uint64(555),
					})

					var response protocol.LimitDiskResponse
					readResponse(&response)

					Expect(fakeContainer.LimitedDisk).To(Equal(
						backend.DiskLimits{
							BlockSoft: 111,
							BlockHard: 222,

							InodeSoft: 333,
							InodeHard: 444,

							ByteHard: 555,
						},
					))
				})
			})

			Context("when ByteLimit is given", func() {
				It("passes it as ByteHard", func() {
					writeMessages(&protocol.LimitDiskRequest{
						Handle:    proto.String(fakeContainer.Handle()),
						BlockSoft: proto.Uint64(111),
						BlockHard: proto.Uint64(222),
						InodeSoft: proto.Uint64(333),
						InodeHard: proto.Uint64(444),
						ByteLimit: proto.Uint64(555),
					})

					var response protocol.LimitDiskResponse
					readResponse(&response)

					Expect(fakeContainer.LimitedDisk).To(Equal(
						backend.DiskLimits{
							BlockSoft: 111,
							BlockHard: 222,

							InodeSoft: 333,
							InodeHard: 444,

							ByteHard: 555,
						},
					))
				})
			})

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.LimitDiskRequest{
						Handle:    proto.String(fakeContainer.Handle()),
						BlockSoft: proto.Uint64(111),
						BlockHard: proto.Uint64(222),
						InodeSoft: proto.Uint64(333),
						InodeHard: proto.Uint64(444),
					})

					var response protocol.LimitDiskResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when limiting the disk fails", func() {
				BeforeEach(func() {
					fakeContainer.LimitDiskError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.LimitDiskRequest{
						Handle:    proto.String(fakeContainer.Handle()),
						BlockSoft: proto.Uint64(111),
						BlockHard: proto.Uint64(222),
						InodeSoft: proto.Uint64(333),
						InodeHard: proto.Uint64(444),
					})

					var response protocol.LimitDiskResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a NetInRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("maps the ports and returns them", func(done Done) {
				writeMessages(&protocol.NetInRequest{
					Handle:        proto.String(fakeContainer.Handle()),
					HostPort:      proto.Uint32(123),
					ContainerPort: proto.Uint32(456),
				})

				var response protocol.NetInResponse
				readResponse(&response)

				Expect(fakeContainer.MappedIn).To(ContainElement(
					[]uint32{123, 456},
				))

				Expect(response.GetHostPort()).To(Equal(uint32(123)))
				Expect(response.GetContainerPort()).To(Equal(uint32(456)))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.NetInRequest{
						Handle:        proto.String(fakeContainer.Handle()),
						HostPort:      proto.Uint32(123),
						ContainerPort: proto.Uint32(456),
					})

					var response protocol.NetInResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when mapping the port fails", func() {
				BeforeEach(func() {
					fakeContainer.NetInError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.NetInRequest{
						Handle:        proto.String(fakeContainer.Handle()),
						HostPort:      proto.Uint32(123),
						ContainerPort: proto.Uint32(456),
					})

					var response protocol.NetInResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a NetOutRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("permits traffic outside of the container", func(done Done) {
				writeMessages(&protocol.NetOutRequest{
					Handle:  proto.String(fakeContainer.Handle()),
					Network: proto.String("1.2.3.4/22"),
					Port:    proto.Uint32(456),
				})

				var response protocol.NetOutResponse
				readResponse(&response)

				Expect(fakeContainer.PermittedOut).To(ContainElement(
					fake_backend.NetOutSpec{"1.2.3.4/22", 456},
				))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.NetOutRequest{
						Handle:  proto.String(fakeContainer.Handle()),
						Network: proto.String("1.2.3.4/22"),
						Port:    proto.Uint32(456),
					})

					var response protocol.NetOutResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when permitting traffic fails", func() {
				BeforeEach(func() {
					fakeContainer.NetOutError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.NetOutRequest{
						Handle:  proto.String(fakeContainer.Handle()),
						Network: proto.String("1.2.3.4/22"),
						Port:    proto.Uint32(456),
					})

					var response protocol.NetOutResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})

		Context("and the client sends a InfoRequest", func() {
			var fakeContainer *fake_backend.FakeContainer

			BeforeEach(func() {
				container, err := serverBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
				Expect(err).ToNot(HaveOccured())

				fakeContainer = container.(*fake_backend.FakeContainer)
			})

			It("reports information about the container", func(done Done) {
				fakeContainer.ReportedInfo = backend.ContainerInfo{
					State:         "active",
					Events:        []string{"oom", "party"},
					HostIP:        "host-ip",
					ContainerIP:   "container-ip",
					ContainerPath: "/path/to/container",
					JobIDs:        []uint32{1, 2},
					MemoryStat: backend.ContainerMemoryStat{
						Cache:                   1,
						Rss:                     2,
						MappedFile:              3,
						Pgpgin:                  4,
						Pgpgout:                 5,
						Swap:                    6,
						Pgfault:                 7,
						Pgmajfault:              8,
						InactiveAnon:            9,
						ActiveAnon:              10,
						InactiveFile:            11,
						ActiveFile:              12,
						Unevictable:             13,
						HierarchicalMemoryLimit: 14,
						HierarchicalMemswLimit:  15,
						TotalCache:              16,
						TotalRss:                17,
						TotalMappedFile:         18,
						TotalPgpgin:             19,
						TotalPgpgout:            20,
						TotalSwap:               21,
						TotalPgfault:            22,
						TotalPgmajfault:         23,
						TotalInactiveAnon:       24,
						TotalActiveAnon:         25,
						TotalInactiveFile:       26,
						TotalActiveFile:         27,
						TotalUnevictable:        28,
					},
					CPUStat: backend.ContainerCPUStat{
						Usage:  1,
						User:   2,
						System: 3,
					},
					DiskStat: backend.ContainerDiskStat{
						BytesUsed:  1,
						InodesUsed: 2,
					},
					BandwidthStat: backend.ContainerBandwidthStat{
						InRate:   1,
						InBurst:  2,
						OutRate:  3,
						OutBurst: 4,
					},
				}

				writeMessages(&protocol.InfoRequest{
					Handle: proto.String(fakeContainer.Handle()),
				})

				var response protocol.InfoResponse
				readResponse(&response)

				Expect(response.GetState()).To(Equal("active"))
				Expect(response.GetEvents()).To(Equal([]string{"oom", "party"}))
				Expect(response.GetHostIp()).To(Equal("host-ip"))
				Expect(response.GetContainerIp()).To(Equal("container-ip"))
				Expect(response.GetContainerPath()).To(Equal("/path/to/container"))
				Expect(response.GetJobIds()).To(Equal([]uint64{1, 2}))

				Expect(response.GetMemoryStat().GetCache()).To(Equal(uint64(1)))
				Expect(response.GetMemoryStat().GetRss()).To(Equal(uint64(2)))
				Expect(response.GetMemoryStat().GetMappedFile()).To(Equal(uint64(3)))
				Expect(response.GetMemoryStat().GetPgpgin()).To(Equal(uint64(4)))
				Expect(response.GetMemoryStat().GetPgpgout()).To(Equal(uint64(5)))
				Expect(response.GetMemoryStat().GetSwap()).To(Equal(uint64(6)))
				Expect(response.GetMemoryStat().GetPgfault()).To(Equal(uint64(7)))
				Expect(response.GetMemoryStat().GetPgmajfault()).To(Equal(uint64(8)))
				Expect(response.GetMemoryStat().GetInactiveAnon()).To(Equal(uint64(9)))
				Expect(response.GetMemoryStat().GetActiveAnon()).To(Equal(uint64(10)))
				Expect(response.GetMemoryStat().GetInactiveFile()).To(Equal(uint64(11)))
				Expect(response.GetMemoryStat().GetActiveFile()).To(Equal(uint64(12)))
				Expect(response.GetMemoryStat().GetUnevictable()).To(Equal(uint64(13)))
				Expect(response.GetMemoryStat().GetHierarchicalMemoryLimit()).To(Equal(uint64(14)))
				Expect(response.GetMemoryStat().GetHierarchicalMemswLimit()).To(Equal(uint64(15)))
				Expect(response.GetMemoryStat().GetTotalCache()).To(Equal(uint64(16)))
				Expect(response.GetMemoryStat().GetTotalRss()).To(Equal(uint64(17)))
				Expect(response.GetMemoryStat().GetTotalMappedFile()).To(Equal(uint64(18)))
				Expect(response.GetMemoryStat().GetTotalPgpgin()).To(Equal(uint64(19)))
				Expect(response.GetMemoryStat().GetTotalPgpgout()).To(Equal(uint64(20)))
				Expect(response.GetMemoryStat().GetTotalSwap()).To(Equal(uint64(21)))
				Expect(response.GetMemoryStat().GetTotalPgfault()).To(Equal(uint64(22)))
				Expect(response.GetMemoryStat().GetTotalPgmajfault()).To(Equal(uint64(23)))
				Expect(response.GetMemoryStat().GetTotalInactiveAnon()).To(Equal(uint64(24)))
				Expect(response.GetMemoryStat().GetTotalActiveAnon()).To(Equal(uint64(25)))
				Expect(response.GetMemoryStat().GetTotalInactiveFile()).To(Equal(uint64(26)))
				Expect(response.GetMemoryStat().GetTotalActiveFile()).To(Equal(uint64(27)))
				Expect(response.GetMemoryStat().GetTotalUnevictable()).To(Equal(uint64(28)))

				Expect(response.GetCpuStat().GetUsage()).To(Equal(uint64(1)))
				Expect(response.GetCpuStat().GetUser()).To(Equal(uint64(2)))
				Expect(response.GetCpuStat().GetSystem()).To(Equal(uint64(3)))

				Expect(response.GetDiskStat().GetBytesUsed()).To(Equal(uint64(1)))
				Expect(response.GetDiskStat().GetInodesUsed()).To(Equal(uint64(2)))

				Expect(response.GetBandwidthStat().GetInRate()).To(Equal(uint64(1)))
				Expect(response.GetBandwidthStat().GetInBurst()).To(Equal(uint64(2)))
				Expect(response.GetBandwidthStat().GetOutRate()).To(Equal(uint64(3)))
				Expect(response.GetBandwidthStat().GetOutBurst()).To(Equal(uint64(4)))

				close(done)
			}, 1.0)

			Context("when the container is not found", func() {
				BeforeEach(func() {
					serverBackend.Destroy(fakeContainer.Handle())
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.InfoRequest{
						Handle: proto.String(fakeContainer.Handle()),
					})

					var response protocol.InfoResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{
						Message: "unknown handle: some-handle",
					}))

					close(done)
				}, 1.0)
			})

			Context("when getting container info fails", func() {
				BeforeEach(func() {
					fakeContainer.InfoError = errors.New("oh no!")
				})

				It("sends a WardenError response", func(done Done) {
					writeMessages(&protocol.InfoRequest{
						Handle: proto.String(fakeContainer.Handle()),
					})

					var response protocol.InfoResponse
					err := message_reader.ReadMessage(responses, &response)
					Expect(err).To(Equal(&message_reader.WardenError{Message: "oh no!"}))

					close(done)
				}, 1.0)
			})
		})
	})
})

func ErrorDialingUnix(socketPath string) func() error {
	return func() error {
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			conn.Close()
		}

		return err
	}
}
