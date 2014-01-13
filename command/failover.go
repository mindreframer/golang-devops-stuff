package command

import (
	"fmt"
)

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

func NewFailoverFromObj(in interface{}) (*Failover, error) {
	switch val := in.(type) {
	case bool:
		return &Failover{Active: val}, nil
	case map[string]interface{}:
		return NewFailoverFromDict(val)
	}
	return nil, nil
}

func NewFailoverFromDict(in map[string]interface{}) (*Failover, error) {
	activeI, exists := in["active"]
	active := true
	var ok bool
	if exists {
		active, ok = activeI.(bool)
		if !ok {
			return nil, fmt.Errorf("Failover: 'active' should be boolean")
		}
	}
	codesI, exists := in["codes"]
	var codes []int
	if exists {
		vals, ok := codesI.([]interface{})
		if !ok {
			return nil, fmt.Errorf("Codes should be an array")
		}
		codes = make([]int, len(vals))
		for i, iVal := range vals {
			val, ok := iVal.(float64)
			if !ok || float64(val) != float64(int(val)) {
				return nil, fmt.Errorf("Failover: code should be an integer")
			}
			codes[i] = int(val)
		}

	}
	return &Failover{Active: active, Codes: codes}, nil
}
