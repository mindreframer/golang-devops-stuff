package instructions

// Failover instructions enable/disable failover
// and provide more control over the process
type Failover struct {
	// Whether to activate the failover for the given request
	Active bool
	// Whenever vulcan encounters one of the http error codes from the upstream,
	// it will initiate a failover instead of proxying the response back to the
	// client, that allows for graceful restarts of the service without timeouts/service disruptions
	Codes []int
}
