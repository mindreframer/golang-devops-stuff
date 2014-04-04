package ostential
import (
	"ostential/types"

	"os"
	"fmt"
	"strconv"
	"strings"
	"github.com/rzab/gosigar"
)

type about struct {
	Hostname string
	IP       string
}
func getAbout() about {
	hostname, _ := os.Hostname()
	// IP, _ := netinterface_ipaddr()
	return about{
		Hostname: hostname,
		// IP: IP,
	}
}

type system struct {
	Uptime string
	La1    string
	La5    string
	La15   string
	LA     string
}
func getSystem() system {
	uptime := sigar.Uptime{}
	uptime.Get()

	s := system{
		Uptime: formatUptime(uptime.Length),
	}

	la := sigar.LoadAverage{}
	la.Get()

	s.La1  = fmt.Sprintf("%.2f", la.One)
	s.La5  = fmt.Sprintf("%.2f", la.Five)
	s.La15 = fmt.Sprintf("%.2f", la.Fifteen)
	s.LA   = fmt.Sprintf("%.2f %.2f %.2f", la.One, la.Five, la.Fifteen)
	return s
}

func _getmem(in sigar.Swap) memory {
	total, approxtotal := humanBandback(in.Total)
	used,  approxused  := humanBandback(in.Used)
	usepercent := percent(approxused, approxtotal)
	return memory{
		Total: total,
		Used:  used,
		Free:  humanB(in.Free),
		UsePercent:     strconv.Itoa(int(usepercent)), // without "%"
		AttrUsePercent: labelAttr_colorPercent(usepercent),
	}
}
func getRAM() memory {
	got := sigar.Mem{}; got.Get()
	return _getmem(sigar.Swap{
		Total: got.Total,
		Used:  got.Used,
		Free:  got.Free,
	})
}
func getSwap() memory {
	got := sigar.Swap{}; got.Get()
	return _getmem(got)
}

func read_disks() (disks []diskInfo) {
	fls := sigar.FileSystemList{}
	fls.Get()

	devnames := map[string]bool{}
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
			fs.DevName == "rootfs" ||
			fs.DevName == "rpc_pipefs" ||

			fs.DirName == "/dev" ||
			strings.HasPrefix(fs.DevName, "map ") {
			continue
		}
		if _, ok := devnames[fs.DevName]; ok { continue }
		if _, ok := dirnames[fs.DirName]; ok { continue }
		devnames[fs.DevName] = true
		devnames[fs.DirName] = true

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
	if false { // testing
		usage := sigar.FileSystemUsage{
			Total: 1024 * 1024 * 256,
			Used:  1024 * 1024 * 64,
			Avail: 1024 * 1024 * (256 - 64),
			Files: 1024 * 1024 * 32,
			FreeFiles: 1024 * 1024 * 8,
		}
		iusePercent := 0.0
		if usage.Files != 0 {
			iusePercent = float64(100) * float64(usage.Files - usage.FreeFiles) / float64(usage.Files)
		}

		disks = append(disks, diskInfo{
			DevName:     "/dev/dummy",

			Total:       usage.Total << 10,
			Used:        usage.Used  << 10,
			Avail:       usage.Avail << 10,
			UsePercent:  usage.UsePercent(),

			Inodes:      usage.Files,
			Iused:       usage.Files - usage.FreeFiles,
			Ifree:       usage.FreeFiles,
			IusePercent: iusePercent,

			DirName:     "/dummy",
		})
	}
	return disks
}

func read_procs() (procs []types.ProcInfo) {
	pls := sigar.ProcList{}
	pls.Get()

	for _, pid := range pls.List {

		state := sigar.ProcState{}
		time  := sigar.ProcTime{}
		mem   := sigar.ProcMem{}

		if err := state.Get(pid); err != nil { continue }
		if err :=  time.Get(pid); err != nil { continue }
		if err :=   mem.Get(pid); err != nil { continue }

		procs = append(procs, types.ProcInfo{
			PID:      uint(pid),
			Priority: state.Priority,
			Nice:     state.Nice,
			Time:     time.Total,
			Name:     procname(pid, state.Name), // proc_{darwin,linux}.go
			Uid:      state.Uid,
			Size:       mem.Size,
			Resident:   mem.Resident,
		})
	}
	if false { // testing
		procs = append(procs, types.ProcInfo{
			PID:      uint(10000),
			Priority: 30,
			Nice:     0,
			Time:     uint64(10),
			Name:     "NOBOY",
			Uid:      4294967294,
			Size:       100000,
			Resident:   200000,
		})
	}
	return procs
}
