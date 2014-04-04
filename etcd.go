package helixdns

import (
  "github.com/coreos/go-etcd/etcd"
)

type Response interface {
  Value() string
}

type Client interface {
  Get(path string) (Response, error)
}

type EtcdClient struct {
  Client *etcd.Client
}

type EtcdResponse struct {
  Response *etcd.Response
}

func NewEtcdClient(instanceUrl string) Client {
  return &EtcdClient{ Client: etcd.NewClient([]string{instanceUrl}) }
}

func (r EtcdResponse) Value() string {
  return r.Response.Node.Value;
}

func (c EtcdClient) Get(path string) (Response, error) {
  resp, err := c.Client.Get(path, false, false)

  if err != nil {
    return nil, err
  }

  return &EtcdResponse{ Response: resp }, nil
}
