package loadbalance

type Endpoint interface {
	Id() string
	IsActive() bool
}

type Balancer interface {
	NextEndpoint([]Endpoint) (Endpoint, error)
}
