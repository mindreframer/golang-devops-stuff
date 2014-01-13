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

func (registryMessage *registryMessage) makeEndpoint() *route.Endpoint {
	return &route.Endpoint{
		Host: registryMessage.Host,
		Port: registryMessage.Port,
		ApplicationId: registryMessage.App,
		Tags:          registryMessage.Tags,
		PrivateInstanceId: registryMessage.PrivateInstanceId,
	}
}
