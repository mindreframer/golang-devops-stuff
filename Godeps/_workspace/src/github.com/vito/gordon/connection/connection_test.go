package connection_test

import (
	"bytes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/gordon/connection"
	"math"

	"code.google.com/p/gogoprotobuf/proto"
	. "github.com/vito/gordon/test_helpers"
	"github.com/vito/gordon/warden"
)

var _ = Describe("Connection", func() {
	var (
		connection     *Connection
		writeBuffer    *bytes.Buffer
		wardenMessages []proto.Message
	)

	assertWriteBufferContains := func(messages ...proto.Message) {
		Ω(string(writeBuffer.Bytes())).Should(Equal(string(warden.Messages(messages...).Bytes())))
	}

	JustBeforeEach(func() {
		writeBuffer = bytes.NewBuffer([]byte{})

		fakeConn := &FakeConn{
			ReadBuffer:  warden.Messages(wardenMessages...),
			WriteBuffer: writeBuffer,
		}

		connection = New(fakeConn)
	})

	BeforeEach(func() {
		wardenMessages = []proto.Message{}
	})

	Describe("Creating", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.CreateResponse{
					Handle: proto.String("foohandle"),
				},
			)
		})

		It("should create a container", func() {
			resp, err := connection.Create()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(resp.GetHandle()).Should(Equal("foohandle"))

			assertWriteBufferContains(&warden.CreateRequest{})
		})
	})

	Describe("Stopping", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.StopResponse{},
			)
		})

		It("should stop the container", func() {
			_, err := connection.Stop("foo", true, true)
			Ω(err).ShouldNot(HaveOccurred())

			assertWriteBufferContains(&warden.StopRequest{
				Handle:     proto.String("foo"),
				Background: proto.Bool(true),
				Kill:       proto.Bool(true),
			})
		})
	})

	Describe("Destroying", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.DestroyResponse{},
			)
		})

		It("should stop the container", func() {
			_, err := connection.Destroy("foo")
			Ω(err).ShouldNot(HaveOccurred())

			assertWriteBufferContains(&warden.DestroyRequest{
				Handle: proto.String("foo"),
			})
		})
	})

	Describe("Limiting Memory", func() {
		Describe("Setting the memory limit", func() {
			BeforeEach(func() {
				wardenMessages = append(wardenMessages,
					&warden.LimitMemoryResponse{LimitInBytes: proto.Uint64(40)},
				)
			})

			It("should limit memory", func() {
				res, err := connection.LimitMemory("foo", 42)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(res.GetLimitInBytes()).Should(BeNumerically("==", 40))

				assertWriteBufferContains(&warden.LimitMemoryRequest{
					Handle:       proto.String("foo"),
					LimitInBytes: proto.Uint64(42),
				})
			})
		})

		Describe("Getting the memory limit", func() {
			Context("when the memory limit is well formatted", func() {
				BeforeEach(func() {
					wardenMessages = append(wardenMessages,
						&warden.LimitMemoryResponse{LimitInBytes: proto.Uint64(40)},
					)
				})

				It("should return the correct memory format", func() {
					memoryLimit, err := connection.GetMemoryLimit("foo")
					Ω(err).ShouldNot(HaveOccurred())
					Ω(memoryLimit).Should(BeNumerically("==", 40))

					assertWriteBufferContains(&warden.LimitMemoryRequest{
						Handle: proto.String("foo"),
					})
				})
			})

			Context("When the memory limit looks fishy", func() {
				BeforeEach(func() {
					wardenMessages = append(wardenMessages,
						&warden.LimitMemoryResponse{LimitInBytes: proto.Uint64(math.MaxInt64)},
					)
				})

				It("should return 0, without erroring", func() {
					memoryLimit, err := connection.GetMemoryLimit("foo")
					Ω(err).ShouldNot(HaveOccurred())
					Ω(memoryLimit).Should(BeNumerically("==", 0))

					assertWriteBufferContains(&warden.LimitMemoryRequest{
						Handle: proto.String("foo"),
					})
				})
			})
		})
	})

	Describe("Limiting Disk", func() {
		Describe("Setting the disk limit", func() {
			BeforeEach(func() {
				wardenMessages = append(wardenMessages,
					&warden.LimitDiskResponse{ByteLimit: proto.Uint64(40)},
				)
			})

			It("should limit disk", func() {
				res, err := connection.LimitDisk("foo", 42)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(res.GetByteLimit()).Should(BeNumerically("==", 40))

				assertWriteBufferContains(&warden.LimitDiskRequest{
					Handle:    proto.String("foo"),
					ByteLimit: proto.Uint64(42),
				})
			})
		})

		Describe("Getting the disk limit", func() {
			BeforeEach(func() {
				wardenMessages = append(wardenMessages,
					&warden.LimitDiskResponse{ByteLimit: proto.Uint64(40)},
				)
			})

			It("should return the correct memory format", func() {
				diskLimit, err := connection.GetDiskLimit("foo")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(diskLimit).Should(BeNumerically("==", 40))

				assertWriteBufferContains(&warden.LimitDiskRequest{
					Handle: proto.String("foo"),
				})
			})
		})
	})

	Describe("Spawning", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.SpawnResponse{JobId: proto.Uint32(42)},
				&warden.SpawnResponse{JobId: proto.Uint32(43)},
			)
		})

		It("should be able to spawn multiple jobs sequentially", func() {
			resp, err := connection.Spawn("foo-handle", "echo hi", true)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(resp.GetJobId()).Should(BeNumerically("==", 42))

			assertWriteBufferContains(&warden.SpawnRequest{
				Handle:        proto.String("foo-handle"),
				Script:        proto.String("echo hi"),
				DiscardOutput: proto.Bool(true),
			})

			writeBuffer.Reset()

			resp, err = connection.Spawn("foo-handle", "echo bye", false)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(resp.GetJobId()).Should(BeNumerically("==", 43))

			assertWriteBufferContains(&warden.SpawnRequest{
				Handle:        proto.String("foo-handle"),
				Script:        proto.String("echo bye"),
				DiscardOutput: proto.Bool(false),
			})
		})
	})

	Describe("NetIn", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.NetInResponse{
					HostPort:      proto.Uint32(7331),
					ContainerPort: proto.Uint32(7332),
				},
			)
		})

		It("should return the allocated ports", func() {
			resp, err := connection.NetIn("foo-handle")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(resp.GetHostPort()).Should(BeNumerically("==", 7331))
			Ω(resp.GetContainerPort()).Should(BeNumerically("==", 7332))

			assertWriteBufferContains(&warden.NetInRequest{
				Handle: proto.String("foo-handle"),
			})
		})
	})

	Describe("Listing containers", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.ListResponse{
					Handles: []string{"container1", "container2", "container3"},
				},
			)
		})

		It("should return the list of containers", func() {
			resp, err := connection.List()
			Ω(err).ShouldNot(HaveOccurred())

			Ω(resp.GetHandles()).Should(Equal([]string{"container1", "container2", "container3"}))

			assertWriteBufferContains(&warden.ListRequest{})
		})
	})

	Describe("Getting container info", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.InfoResponse{
					State: proto.String("active"),
				},
			)
		})

		It("should return the container's info", func() {
			resp, err := connection.Info("handle")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(resp.GetState()).Should(Equal("active"))

			assertWriteBufferContains(&warden.InfoRequest{
				Handle: proto.String("handle"),
			})
		})
	})

	Describe("Copying in", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.CopyInResponse{},
			)
		})

		It("should tell garden to copy", func() {
			_, err := connection.CopyIn("foo-handle", "/foo", "/bar")
			Ω(err).ShouldNot(HaveOccurred())

			assertWriteBufferContains(&warden.CopyInRequest{
				Handle:  proto.String("foo-handle"),
				SrcPath: proto.String("/foo"),
				DstPath: proto.String("/bar"),
			})
		})
	})

	Describe("Linking", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.LinkResponse{
					Stdout:     proto.String("some data for stdout"),
					Stderr:     proto.String("some data for stderr"),
					ExitStatus: proto.Uint32(137),
				},
			)
		})

		It("should link ot the process", func() {
			resp, err := connection.Link("foo-handle", 42)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(resp.GetExitStatus()).Should(BeNumerically("==", 137))
			Ω(resp.GetStdout()).Should(Equal("some data for stdout"))
			Ω(resp.GetStderr()).Should(Equal("some data for stderr"))

			assertWriteBufferContains(&warden.LinkRequest{
				Handle: proto.String("foo-handle"),
				JobId:  proto.Uint32(42),
			})
		})
	})

	Describe("Running", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.RunResponse{ExitStatus: proto.Uint32(137)},
			)
		})

		It("should start the process running", func() {
			resp, err := connection.Run("foo-handle", "echo hi")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(resp.GetExitStatus()).Should(BeNumerically("==", 137))

			assertWriteBufferContains(&warden.RunRequest{
				Handle: proto.String("foo-handle"),
				Script: proto.String("echo hi"),
			})
		})
	})

	Describe("When a connection error occurs", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.DestroyResponse{},
				//EOF
			)
		})

		It("should disconnect", func(done Done) {
			resp, err := connection.Destroy("foo-handle")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(resp).ShouldNot(BeNil())

			<-connection.Disconnected
			close(done)
		})
	})

	Describe("Disconnecting", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.ErrorResponse{Message: proto.String("boo")},
			)
		})

		It("should error", func() {
			resp, err := connection.Run("foo-handle", "echo hi")
			Ω(resp).Should(BeNil())
			Ω(err.Error()).Should(Equal("boo"))
		})
	})

	Describe("Round tripping", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.RunResponse{ExitStatus: proto.Uint32(137)},
			)
		})

		It("should do the round trip", func() {
			resp, err := connection.RoundTrip(
				&warden.RunRequest{
					Handle: proto.String("some-handle"),
					Script: proto.String("foo"),
				},
				&warden.RunResponse{},
			)

			Ω(err).ShouldNot(HaveOccurred())
			Ω(resp.(*warden.RunResponse).GetExitStatus()).Should(BeNumerically("==", 137))
		})
	})

	Describe("Streaming", func() {
		BeforeEach(func() {
			wardenMessages = append(wardenMessages,
				&warden.StreamResponse{Name: proto.String("stdout"), Data: proto.String("1")},
				&warden.StreamResponse{Name: proto.String("stderr"), Data: proto.String("2")},
				&warden.StreamResponse{ExitStatus: proto.Uint32(3)},
			)
		})

		It("should stream", func(done Done) {
			resp, finishedStreaming, err := connection.Stream("foo-handle", 42)
			Ω(err).ShouldNot(HaveOccurred())

			assertWriteBufferContains(&warden.StreamRequest{
				Handle: proto.String("foo-handle"),
				JobId:  proto.Uint32(42),
			})

			response1 := <-resp
			Ω(response1.GetName()).Should(Equal("stdout"))
			Ω(response1.GetData()).Should(Equal("1"))

			select {
			case <-finishedStreaming:
				Fail("should not have finished streaming")
			default:
			}

			response2 := <-resp
			Ω(response2.GetName()).Should(Equal("stderr"))
			Ω(response2.GetData()).Should(Equal("2"))

			select {
			case <-finishedStreaming:
				Fail("should not have finished streaming")
			default:
			}

			response3, ok := <-resp
			Ω(response3.GetExitStatus()).Should(BeNumerically("==", 3))
			Ω(ok).Should(BeTrue())

			_, ok = <-finishedStreaming
			Ω(ok).Should(BeFalse())

			close(done)
		})
	})
})
