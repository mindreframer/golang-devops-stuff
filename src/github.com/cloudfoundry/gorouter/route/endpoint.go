package route

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Endpoint struct {
	sync.Mutex

	ApplicationId     string
	Host              string
	Port              uint16
	Tags              map[string]string
	PrivateInstanceId string
}

func (e *Endpoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.CanonicalAddr())
}

func (e *Endpoint) CanonicalAddr() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

func (e *Endpoint) ToLogData() interface{} {
	return struct {
		ApplicationId string
		Host          string
		Port          uint16
		Tags          map[string]string
	}{
		e.ApplicationId,
		e.Host,
		e.Port,
		e.Tags,
	}
}
