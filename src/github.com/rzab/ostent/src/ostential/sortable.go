package ostential
import (
	"ostential/types"

 	"html/template"
	"github.com/rzab/gosigar"
)

type cpuOrder []sigar.Cpu
func(co cpuOrder) Len() int {
	return len(co)
}
func(co cpuOrder) Less(i, j int) bool {
	return (co[j].User + co[j].Sys) < (co[i].User + co[i].Sys)
}
func(co cpuOrder) Swap(i, j int) {
	co[i], co[j] = co[j], co[i]
}

type interfaceOrder []types.DeltaInterface
func(io interfaceOrder) Len() int {
	return len(io)
}
func(io interfaceOrder) Swap(i, j int) {
	io[i], io[j] = io[j], io[i]
}
func(io interfaceOrder) Less(i, j int) bool {
	if io[i].Name == "lo" || rx_lo.Match([]byte(io[i].Name)) {
		return true
	}
	return io[i].Name < io[j].Name
}

type ProcTable struct {
	List  []types.ProcData
	Links *ProcLinkattrs `json:",omitempty"`

	JS         string            `json:"-"`
	TRHTMLAttr template.HTMLAttr `json:"-"`
}

type DiskTable struct {
	List  []types.DiskData
	Links *DiskLinkattrs `json:",omitempty"`

	JS         string            `json:"-"`
 	TRHTMLAttr template.HTMLAttr `json:"-"`
	NOrow1     bool              `json:"-"`
	NOrow2     bool              `json:"-"`
}
