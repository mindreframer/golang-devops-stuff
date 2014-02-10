package gordon

import (
	"time"

	"github.com/vito/gordon/connection"
	"github.com/vito/gordon/warden"
)

type Client interface {
	Connect() error

	Create() (*warden.CreateResponse, error)
	Stop(handle string, background, kill bool) (*warden.StopResponse, error)
	Destroy(handle string) (*warden.DestroyResponse, error)
	Spawn(handle, script string, discardOutput bool) (*warden.SpawnResponse, error)
	Link(handle string, jobID uint32) (*warden.LinkResponse, error)
	NetIn(handle string) (*warden.NetInResponse, error)
	LimitMemory(handle string, limit uint64) (*warden.LimitMemoryResponse, error)
	GetMemoryLimit(handle string) (uint64, error)
	LimitDisk(handle string, limit uint64) (*warden.LimitDiskResponse, error)
	GetDiskLimit(handle string) (uint64, error)
	List() (*warden.ListResponse, error)
	Info(handle string) (*warden.InfoResponse, error)
	CopyIn(handle, src, dst string) (*warden.CopyInResponse, error)
	Stream(handle string, jobID uint32) (<-chan *warden.StreamResponse, error)
	Run(handle, script string) (*warden.RunResponse, error)
}

type client struct {
	connectionProvider ConnectionProvider
	connection         chan *connection.Connection
}

func NewClient(cp ConnectionProvider) Client {
	return &client{
		connectionProvider: cp,
		connection:         make(chan *connection.Connection),
	}
}

func (c *client) Connect() error {
	conn, err := c.connectionProvider.ProvideConnection()
	if err != nil {
		return err
	}

	go c.serveConnection(conn)

	return nil
}

func (c *client) Create() (*warden.CreateResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.Create()
}

func (c *client) Stop(handle string, background, kill bool) (*warden.StopResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.Stop(handle, background, kill)
}

func (c *client) Destroy(handle string) (*warden.DestroyResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.Destroy(handle)
}

func (c *client) Spawn(handle, script string, discardOutput bool) (*warden.SpawnResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.Spawn(handle, script, discardOutput)
}

func (c *client) Link(handle string, jobID uint32) (*warden.LinkResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.Link(handle, jobID)
}

func (c *client) NetIn(handle string) (*warden.NetInResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.NetIn(handle)
}

func (c *client) LimitMemory(handle string, limit uint64) (*warden.LimitMemoryResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.LimitMemory(handle, limit)
}

func (c *client) GetMemoryLimit(handle string) (uint64, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.GetMemoryLimit(handle)
}

func (c *client) LimitDisk(handle string, limit uint64) (*warden.LimitDiskResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.LimitDisk(handle, limit)
}

func (c *client) GetDiskLimit(handle string) (uint64, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.GetDiskLimit(handle)
}

func (c *client) List() (*warden.ListResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.List()
}

func (c *client) Info(handle string) (*warden.InfoResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.Info(handle)
}

func (c *client) CopyIn(handle, src, dst string) (*warden.CopyInResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.CopyIn(handle, src, dst)
}

func (c *client) Stream(handle string, jobID uint32) (<-chan *warden.StreamResponse, error) {
	conn := c.acquireConnection()

	responses, done, err := conn.Stream(handle, jobID)
	if err != nil {
		c.release(conn)
		return nil, err
	}

	go func() {
		<-done
		c.release(conn)
	}()

	return responses, nil
}

func (c *client) Run(handle, script string) (*warden.RunResponse, error) {
	conn := c.acquireConnection()
	defer c.release(conn)

	return conn.Run(handle, script)
}

func (c *client) serveConnection(conn *connection.Connection) {
	select {
	case <-conn.Disconnected:

	case c.connection <- conn:

	case <-time.After(5 * time.Second):
		conn.Close()
	}
}

func (c *client) release(conn *connection.Connection) {
	go c.serveConnection(conn)
}

func (c *client) acquireConnection() *connection.Connection {
	select {
	case conn := <-c.connection:
		return conn

	case <-time.After(1 * time.Second):
		return c.connect()
	}
}

func (c *client) connect() *connection.Connection {
	for {
		conn, err := c.connectionProvider.ProvideConnection()
		if err == nil {
			return conn
		}

		time.Sleep(500 * time.Millisecond)
	}
}
