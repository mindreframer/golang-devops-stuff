package common

type CloneRequest struct {
	Origin   string
	Clone    string
	Role     string
	Reconfig bool
	Promote  bool
}

type FailoverRequest struct {
	Podname   string
	ReturnNew bool
}

type AddSlaveRequest struct {
	Podname      string
	SlaveAddress string
	SlavePort    int
	SlaveAuth    string
}

type MonitorRequest struct {
	Podname       string
	MasterAddress string
	AuthToken     string
	MasterPort    int
	Quorum        int
}
