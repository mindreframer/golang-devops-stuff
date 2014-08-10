package common

type Healthz struct {
}

func (v *Healthz) Value() string {
	return "ok"
}
