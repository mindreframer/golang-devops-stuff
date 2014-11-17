// The actions package contains the code for interacting directly with Redis
// instances and taking actions against them. This includes higher level actions
// which apply to componets and to lower level actions which are taken against
// components directly.
package actions

import (
	"log"
	"strings"

	"github.com/therealbill/libredis/client"
	"github.com/therealbill/libredis/info"
)

// RedisPod is the construct used for holding data about a Redis Pod and taking
// action against it.
type RedisPod struct {
	Name                  string
	Info                  client.MasterInfo
	Slaves                []info.InfoSlaves
	Master                *RedisNode
	SentinelCount         int
	ActiveSentinelCount   int
	ReportedSentinelCount int
	AuthToken             string
	ValidAuth             bool
	NeededSentinels       int
	MissingSentinels      bool
	TooManySentinels      bool
	HasInfo               bool
	NeedsReset            bool
	HasValidSlaves        bool
}

// NewPod will return a RedisPod construct. It requires the nae, address, port,
// and authentication token.
func NewPod(name, address string, port int, auth string) (rp RedisPod, err error) {
	rp.Name = name
	rp.AuthToken = auth
	return

}

// NewMasterFromMasterInfo accepts a MasterInfo struct from libredis/client
// combined with an authentication token to use and returns a RedisPod
// instance.
func NewMasterFromMasterInfo(mi client.MasterInfo, authtoken string) (rp RedisPod, err error) {
	rp.Name = mi.Name
	rp.Info = mi
	rp.AuthToken = authtoken
	return rp, nil
}

// HasQuorum checks to see if the pod has Quorum.
func (rp *RedisPod) HasQuorum() bool {
	return rp.SentinelCount >= rp.Info.Quorum
}

// CanFailover tests failover conditions to determine if a failover call would
// succeed
func (rp *RedisPod) CanFailover() bool {
	promotable_slaves := 0
	if rp.Master == nil {
		master, err := LoadNodeFromHostPort(rp.Info.IP, rp.Info.Port, rp.AuthToken)
		if err != nil {
			log.Printf("Unable to load %s. Err: '%s'", rp.Name, err)
			if strings.Contains(err.Error(), "invalid password") {
				rp.ValidAuth = false
			} else {
				rp.ValidAuth = true
			}
			return false
		}
		rp.Master = master
	}
	if rp.Master.Slaves == nil {
		rp.HasInfo = false
	} else {
		rp.HasInfo = true
		for _, slave := range rp.Master.Slaves {
			if slave.Info.Replication.SlavePriority > 0 {
				rp.HasValidSlaves = true
				promotable_slaves++
			}
		}
	}
	if promotable_slaves == 0 {
		rp.HasValidSlaves = false
		return false
	} else {
		rp.HasValidSlaves = true
	}
	if !rp.HasQuorum() {
		return false
	}
	return true
}

// SlavesHaveEnoughMemory checks all slaves for their maximum memory to
// validate they match or beter the master
func (rp *RedisPod) SlavesHaveEnoughMemory() bool {
	ok := true
	// This should filter out slaves which have a slave priority of 0
	if rp.Master == nil {
		return false
	}
	for _, node := range rp.Master.Slaves {
		if node.MaxMemory < rp.Master.MaxMemory {
			node.HasEnoughMemoryForMaster = false
			ok = false
		}
	}
	return ok
}

// HasErrors checks various error conditions and returns t/f
// TODO: Some of these are better categorized as warnings and this should be
// split into a pair of functions: one for errors and one for warning.
// This will require additional work to incorporate the HasWarnings concept
// through the system.
func (rp *RedisPod) HasErrors() bool {
	rp.NeededSentinels = rp.Info.Quorum + 1
	rp.ReportedSentinelCount = rp.Info.NumOtherSentinels
	hasErrors := false
	if rp.Info.NumOtherSentinels > 0 {
		rp.ReportedSentinelCount++
	}
	if rp.NeededSentinels > rp.SentinelCount {
		rp.MissingSentinels = true
		hasErrors = true
	}
	if rp.Info.NumOtherSentinels+1 > rp.NeededSentinels {
		rp.NeedsReset = true
		hasErrors = true
	}
	if rp.ReportedSentinelCount >= (rp.Info.Quorum * 2) {
		rp.TooManySentinels = true
		hasErrors = true
	}
	if !rp.CanFailover() {
		hasErrors = true
	}
	if !rp.SlavesHaveEnoughMemory() {
		hasErrors = true
	}
	return hasErrors
}
