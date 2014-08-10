package instrumentation

type Metric struct {
	Name  string                 `json:"name"`
	Value interface{}            `json:"value"`
	Tags  map[string]interface{} `json:"tags,omitempty"`
}
