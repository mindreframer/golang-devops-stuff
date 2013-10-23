package storerunner

type StoreRunner interface {
	Start()
	Stop()
	NodeURLS() []string
	DiskUsage() (bytes int64, err error)
	FastForwardTime(seconds int)
	Reset()
}
