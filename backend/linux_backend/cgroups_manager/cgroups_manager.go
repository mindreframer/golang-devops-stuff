package cgroups_manager

type CgroupsManager interface {
	Set(subsystem, name, value string) error
	Get(subsystem, name string) (string, error)
	SubsystemPath(subsystem string) string
}
