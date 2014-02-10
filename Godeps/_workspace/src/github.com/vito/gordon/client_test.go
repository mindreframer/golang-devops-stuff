package gordon_test

import (
	"bytes"
	"errors"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/gordon"

	"code.google.com/p/gogoprotobuf/proto"
	"github.com/vito/gordon/warden"
)

var _ = Describe("Client", func() {
	var (
		client      Client
		writeBuffer *bytes.Buffer
		provider    *FakeConnectionProvider
	)

	BeforeEach(func() {
		writeBuffer = new(bytes.Buffer)

	})

	Describe("Connect", func() {
		Context("with a successful provider", func() {
			BeforeEach(func() {
				client = NewClient(NewFakeConnectionProvider(new(bytes.Buffer), new(bytes.Buffer)))
			})

			It("should connect", func() {
				err := client.Connect()
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("with a failing provider", func() {
			BeforeEach(func() {
				client = NewClient(&FailingConnectionProvider{})
			})

			It("should fail to connect", func() {
				err := client.Connect()
				Ω(err).Should(Equal(errors.New("nope!")))
			})
		})
	})

	Describe("The container lifecycle", func() {
		BeforeEach(func() {
			provider = NewFakeConnectionProvider(
				warden.Messages(
					&warden.CreateResponse{Handle: proto.String("foo")},
					&warden.StopResponse{},
					&warden.DestroyResponse{},
				),
				writeBuffer,
			)

			client = NewClient(provider)
			err := client.Connect()
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should be able to create, stop and destroy a container", func() {
			res, err := client.Create()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(res.GetHandle()).Should(Equal("foo"))

			_, err = client.Stop("foo", true, true)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = client.Destroy("foo")
			Ω(err).ShouldNot(HaveOccurred())

			expectedWriteBufferContents := string(warden.Messages(
				&warden.CreateRequest{},
				&warden.StopRequest{
					Handle:     proto.String("foo"),
					Background: proto.Bool(true),
					Kill:       proto.Bool(true),
				},
				&warden.DestroyRequest{Handle: proto.String("foo")},
			).Bytes())

			Ω(string(writeBuffer.Bytes())).Should(Equal(expectedWriteBufferContents))
		})
	})

	Describe("Spawning and streaming", func() {
		BeforeEach(func() {
			provider = NewFakeConnectionProvider(
				warden.Messages(
					&warden.SpawnResponse{
						JobId: proto.Uint32(42),
					},
					&warden.StreamResponse{
						Name: proto.String("stdout"),
						Data: proto.String("some data for stdout"),
					},
				),
				writeBuffer,
			)

			client = NewClient(provider)
			err := client.Connect()
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should spawn and stream succesfully", func(done Done) {
			spawned, err := client.Spawn("foo", "echo some data for stdout", true)
			Ω(err).ShouldNot(HaveOccurred())

			responses, err := client.Stream("foo", spawned.GetJobId())
			Ω(err).ShouldNot(HaveOccurred())

			expectedWriteBufferContents := string(warden.Messages(
				&warden.SpawnRequest{
					Handle:        proto.String("foo"),
					Script:        proto.String("echo some data for stdout"),
					DiscardOutput: proto.Bool(true),
				},
				&warden.StreamRequest{Handle: proto.String("foo"), JobId: proto.Uint32(42)},
			).Bytes())

			Ω(string(writeBuffer.Bytes())).Should(Equal(expectedWriteBufferContents))

			res := <-responses
			Ω(res.GetName()).Should(Equal("stdout"))
			Ω(res.GetData()).Should(Equal("some data for stdout"))

			close(done)
		})
	})

	Describe("Spawning and linking", func() {
		BeforeEach(func() {
			provider = NewFakeConnectionProvider(
				warden.Messages(
					&warden.SpawnResponse{
						JobId: proto.Uint32(42),
					},
					&warden.LinkResponse{
						Stdout:     proto.String("some data for stdout"),
						Stderr:     proto.String("some data for stderr"),
						ExitStatus: proto.Uint32(137),
					},
				),
				writeBuffer,
			)

			client = NewClient(provider)
			err := client.Connect()
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should spawn and link succesfully", func() {
			spawned, err := client.Spawn("foo", "echo some data for stdout", true)
			Ω(err).ShouldNot(HaveOccurred())

			res, err := client.Link("foo", spawned.GetJobId())
			Ω(err).ShouldNot(HaveOccurred())

			expectedWriteBufferContents := string(warden.Messages(
				&warden.SpawnRequest{
					Handle:        proto.String("foo"),
					Script:        proto.String("echo some data for stdout"),
					DiscardOutput: proto.Bool(true),
				},
				&warden.LinkRequest{Handle: proto.String("foo"), JobId: proto.Uint32(42)},
			).Bytes())

			Ω(string(writeBuffer.Bytes())).Should(Equal(expectedWriteBufferContents))

			Ω(res.GetStdout()).Should(Equal("some data for stdout"))
			Ω(res.GetStderr()).Should(Equal("some data for stderr"))
			Ω(res.GetExitStatus()).Should(Equal(uint32(137)))
		})
	})

	Describe("Querying containers", func() {
		Describe("Listing containers", func() {
			BeforeEach(func() {
				provider = NewFakeConnectionProvider(
					warden.Messages(
						&warden.ListResponse{
							Handles: []string{"container1", "container6"},
						},
					),
					writeBuffer,
				)

				client = NewClient(provider)
				err := client.Connect()
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should list the containers", func() {
				res, err := client.List()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(res.GetHandles()).Should(Equal([]string{"container1", "container6"}))

				expectedWriteBufferContents := string(warden.Messages(
					&warden.ListRequest{},
				).Bytes())

				Ω(string(writeBuffer.Bytes())).Should(Equal(expectedWriteBufferContents))
			})
		})

		Describe("Getting info for a specific container", func() {
			BeforeEach(func() {
				provider = NewFakeConnectionProvider(
					warden.Messages(
						&warden.InfoResponse{
							State: proto.String("stopped"),
						},
					),
					writeBuffer,
				)

				client = NewClient(provider)
				err := client.Connect()
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should return info for the requested handle", func() {
				res, err := client.Info("handle")

				Ω(err).ShouldNot(HaveOccurred())
				Ω(res.GetState()).Should(Equal("stopped"))

				expectedWriteBufferContents := string(warden.Messages(
					&warden.InfoRequest{
						Handle: proto.String("handle"),
					},
				).Bytes())

				Ω(string(writeBuffer.Bytes())).Should(Equal(expectedWriteBufferContents))
			})
		})

		Describe("Reconnecting", func() {
			var (
				firstWriteBuffer  *bytes.Buffer
				secondWriteBuffer *bytes.Buffer
			)

			BeforeEach(func() {
				firstWriteBuffer = bytes.NewBuffer([]byte{})
				secondWriteBuffer = bytes.NewBuffer([]byte{})

				mcp := &ManyConnectionProvider{
					ConnectionProviders: []ConnectionProvider{
						NewFakeConnectionProvider(
							warden.Messages(
								&warden.CreateResponse{Handle: proto.String("handle a")},
								// disconnect
							),
							firstWriteBuffer,
						),
						NewFakeConnectionProvider(
							warden.Messages(
								&warden.CreateResponse{Handle: proto.String("handle b")},
								&warden.DestroyResponse{},
								&warden.DestroyResponse{},
							),
							secondWriteBuffer,
						),
					},
				}

				client = NewClient(mcp)
				err := client.Connect()
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should attempt to reconnect when a disconnect occurs", func() {
				c1, err := client.Create()
				Ω(err).ShouldNot(HaveOccurred())

				// let client notice disconnect
				runtime.Gosched()

				c2, err := client.Create()
				Ω(err).ShouldNot(HaveOccurred())

				_, err = client.Destroy(c1.GetHandle())
				Ω(err).ShouldNot(HaveOccurred())

				_, err = client.Destroy(c2.GetHandle())
				Ω(err).ShouldNot(HaveOccurred())

				expectedWriteBufferContents := string(warden.Messages(
					&warden.CreateRequest{},
				).Bytes())

				Ω(string(firstWriteBuffer.Bytes())).Should(Equal(expectedWriteBufferContents))

				expectedWriteBufferContents = string(warden.Messages(
					&warden.CreateRequest{},
					&warden.DestroyRequest{
						Handle: proto.String("handle a"),
					},
					&warden.DestroyRequest{
						Handle: proto.String("handle b"),
					},
				).Bytes())

				Ω(string(secondWriteBuffer.Bytes())).Should(Equal(expectedWriteBufferContents))
			})
		})
	})
})
