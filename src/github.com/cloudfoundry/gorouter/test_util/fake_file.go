package test_util

type FakeFile struct {
	Payload []byte
}

func (f *FakeFile) Write(data []byte) (int, error) {
	f.Payload = data
	return 12, nil
}
