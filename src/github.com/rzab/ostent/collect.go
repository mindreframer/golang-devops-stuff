package ostent
import (
	"libostent/types"
	"share/templates.html"

	"os"
	"fmt"
	"strconv"
	"strings"

	"github.com/rzab/gosigar"
)

type generic struct {
	Hostname string
	IP       string
	LA       string
	Uptime   string
	LA1spark string
	la1 int
}

func getHostname() (string, error) {
	hostname, err := os.Hostname()
	if err == nil {
		hostname = strings.Split(hostname, ".")[0]
	}
	return hostname, err
}

func getGeneric(CH chan generic) {
	hostname, _ := getHostname()

	uptime := sigar.Uptime{};      uptime.Get()
	la     := sigar.LoadAverage{}; la    .Get()

	CH <- generic{
		Hostname: hostname,
		// IP: IP, // IP, _ := netinterface_ipaddr()
		LA: fmt.Sprintf("%.2f %.2f %.2f", la.One, la.Five, la.Fifteen),
		Uptime: formatUptime(uptime.Length),
		la1: int(float64(100) * la.One),
	}
}

func _getmem(kind string, in sigar.Swap) types.Memory {
	total, approxtotal := humanBandback(in.Total)
	used,  approxused  := humanBandback(in.Used)
	usepercent := percent(approxused, approxtotal)

	UPhtml, _ := view.UsePercentTemplate.Execute(struct{
		Class, Value, CLASSNAME string
	}{
		Value: strconv.Itoa(int(usepercent)), // without "%"
		Class: labelClass_colorPercent(usepercent),
	})

	return types.Memory{
		Kind:  kind,
		Total: total,
		Used:  used,
		Free:  humanB(in.Free),
		UsePercentHTML: UPhtml,
	}
}
func getRAM(CH chan types.Memory) {
	got := sigar.Mem{}; got.Get()

	inactive := got.ActualFree - got.Free // == got.Used - got.ActualUsed // "kern"
	_ = inactive

	// Used = .Total - .Free
	// | Free |           Used +%         | Total
	// | Free | Inactive | Active | Wired | Total

	// TODO active := vm_data.active_count << 12 (pagesize)
	// TODO wired  := vm_data.wire_count   << 12 (pagesoze)

	CH <- _getmem("RAM", sigar.Swap{
		Total: got.Total,
		Free:  got.Free,

		Used:  got.Used, // == .Total - .Free
	})
}

func getSwap(CH chan types.Memory) {
	got := sigar.Swap{}; got.Get()
	CH <- _getmem("swap", got)
}

func read_disks(CH chan []diskInfo) {
	var disks []diskInfo
	fls := sigar.FileSystemList{}
	fls.Get()

// 	devnames := map[string]bool{}
	dirnames := map[string]bool{}

	for _, fs := range fls.List {

		usage := sigar.FileSystemUsage{}
		usage.Get(fs.DirName)

		if  fs.DevName == "shm"    ||
			fs.DevName == "none"   ||
			fs.DevName == "proc"   ||
			fs.DevName == "udev"   ||
			fs.DevName == "devfs"  ||
			fs.DevName == "sysfs"  ||
			fs.DevName == "tmpfs"  ||
			fs.DevName == "devpts" ||
			fs.DevName == "cgroup" ||
			fs.DevName == "rootfs" ||
			fs.DevName == "rpc_pipefs" ||

			fs.DirName == "/dev" ||
			strings.HasPrefix(fs.DevName, "map ") {
			continue
		}
	// 	if _, ok := devnames[fs.DevName]; ok { continue }
		if _, ok := dirnames[fs.DirName]; ok { continue }
	// 	devnames[fs.DevName] = true
		dirnames[fs.DirName] = true

		iusePercent := 0.0
		if usage.Files != 0 {
			iusePercent = float64(100) * float64(usage.Files - usage.FreeFiles) / float64(usage.Files)
		}
		disks = append(disks, diskInfo{
			DevName:     fs.DevName,

			Total:       usage.Total << 10, // * 1024
			Used:        usage.Used  << 10, // == Total - Free
			Avail:       usage.Avail << 10,
			UsePercent:  usage.UsePercent(),

			Inodes:      usage.Files,
			Iused:       usage.Files - usage.FreeFiles,
			Ifree:       usage.FreeFiles,
			IusePercent: iusePercent,

			DirName:     fs.DirName,
		})
	}
	CH <- disks
}

func read_procs(CH chan []types.ProcInfo) {
	var procs []types.ProcInfo
	pls := sigar.ProcList{}
	pls.Get()

	for _, pid := range pls.List {

		state := sigar.ProcState{}
		// args  := sigar.ProcArgs{}
		time  := sigar.ProcTime{}
		mem   := sigar.ProcMem{}

		if err := state.Get(pid); err != nil { continue }
		// if err :=  args.Get(pid); err != nil { continue }
		if err :=  time.Get(pid); err != nil { continue }
		if err :=   mem.Get(pid); err != nil { continue }

		procs = append(procs, types.ProcInfo{
			PID:      uint(pid),
			Priority: state.Priority,
			Nice:     state.Nice,
			Time:     time.Total,
			// `procname' defined proc_{darwin,linux}.go
			Name:     procname(pid, state.Name),
			// Name:     strings.Join(append([]string{procname(pid, state.Name)}, args.List[1:]...), " "),
			Uid:      state.Uid,
			Size:     mem.Size,
			Resident: mem.Resident,
		})
	}
	CH <- procs
}
