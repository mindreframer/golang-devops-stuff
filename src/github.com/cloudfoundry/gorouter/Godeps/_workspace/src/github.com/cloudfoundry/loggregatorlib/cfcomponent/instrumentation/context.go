package instrumentation

type Context struct {
	Name    string   `json:"name"`
	Metrics []Metric `json:"metrics"`
}
