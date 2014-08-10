package ostent
import (
	"libostent/types"
)

type cpuOrder []types.Core
func(co cpuOrder) Len() int {
	return len(co)
}
func(co cpuOrder) Less(i, j int) bool {
	return (co[j].User + co[j].Sys) < (co[i].User + co[i].Sys)
}
func(co cpuOrder) Swap(i, j int) {
	co[i], co[j] = co[j], co[i]
}

type interfaceOrder []types.Interface
func(io interfaceOrder) Len() int {
	return len(io)
}
func(io interfaceOrder) Swap(i, j int) {
	io[i], io[j] = io[j], io[i]
}
func(io interfaceOrder) Less(i, j int) bool {
	if rx_lo.Match([]byte(io[i].NameKey)) {
		return false
	}
	return io[i].NameKey < io[j].NameKey
}
