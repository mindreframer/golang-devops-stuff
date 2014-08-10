package ostent
// #include <sys/param.h>
import "C"
import (
	"path/filepath"
	"github.com/rzab/gosigar"
)

// ProcState returns chopped proc name, in which case
// get the ProcExe and return basename of executable path
func procname(pid int, pbi_comm string) string {
	if len(pbi_comm) + 1 < C.MAXCOMLEN {
		return pbi_comm
	}
	exe := sigar.ProcExe{}
	if err := exe.Get(pid); err != nil {
		return pbi_comm
	}
	return filepath.Base(exe.Name)
}
