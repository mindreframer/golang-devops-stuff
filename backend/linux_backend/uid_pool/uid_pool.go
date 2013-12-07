package uid_pool

type UIDPool interface {
	Acquire() (uint32, error)
	Release(uint32)
}
