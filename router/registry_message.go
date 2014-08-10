package router

import (
	"github.com/cloudfoundry/gorouter/route"
)

type registryMessage struct {
	Host string            `json:"host"`
	Port uint16            `json:"port"`
	Uris []route.Uri       `json:"uris"`
	Tags map[string]string `json:"tags"`
	App  string            `json:"app"`

	PrivateInstanceId string `json:"private_instance_id"`
}

func (rm *registryMessage) makeEndpoint() *route.Endpoint {
	return route.NewEndpoint(rm.App, rm.Host, rm.Port, rm.PrivateInstanceId, rm.Tags)
}
