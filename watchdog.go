package main

import (
	"code.google.com/p/goconf/conf"
	"fmt"
	"github.com/cloudfoundry/gosigar"
	"labix.org/v2/mgo"
	"os"
	"path"
	"time"
)

type Application struct {
	Hostname    string
	Time        time.Time
	DF          *DF
	CPU         *CPUS
	LoadAverage *LoadAverage
	Uptime      *Uptime
	Mem         *Mem
	Swap        *Swap
	ProcList    *ProcList
}

type DF struct {
	Data []*FileSystem
}

type FileSystem struct {
	FileSystem string
	Size       uint64
	Used       uint64
	Avail      uint64
	Percent    float64
	Mounted    string
}

type CPUS struct {
	Data []*CpuStat
}

type CpuStat struct {
	ID      string
	User    uint64
	Nice    uint64
	Sys     uint64
	Idle    uint64
	Wait    uint64
	IRQ     uint64
	SoftIRQ uint64
	Stolen  uint64
}

type LoadAverage struct {
	One     float64
	Five    float64
	Fifteen float64
}

type Uptime struct {
	Length float64
}

type Mem struct {
	Total      uint64
	Free       uint64
	Used       uint64
	ActualFree uint64
	ActualUsed uint64
}

type Swap struct {
	Total uint64
	Free  uint64
	Used  uint64
}

type ProcList struct {
	List []*ProcState
}

type ProcState struct {
	Pid       int
	Name      string
	State     sigar.RunState
	Ppid      int
	Tty       int
	Priority  int
	Nice      int
	Processor int
	Memory    *ProcMem
	Time      *ProcTime
	Args      *ProcArgs
	Exe       *ProcExe
}

type ProcMem struct {
	Size        uint64
	Resident    uint64
	Share       uint64
	MinorFaults uint64
	MajorFaults uint64
	PageFaults  uint64
}

type ProcTime struct {
	User      uint64
	Sys       uint64
	Total     uint64
	StartTime uint64
}

type ProcArgs struct {
	Arguements []string
}

type ProcExe struct {
	Name string
	Cwd  string
	Root string
}

func main() {
	pid := sigar.ProcExe{}
	pid.Get(os.Getpid())
	base := path.Dir(pid.Name)
	config, err := conf.ReadConfigFile(base + "/watchdog.conf")
	hostname, err := config.GetString("", "hostname")
	dbuser, err := config.GetString("mongo", "username")
	dbpass, err := config.GetString("mongo", "password")
	dbhost, err := config.GetString("mongo", "host")
	dbname, err := config.GetString("mongo", "database")
	fslist := sigar.FileSystemList{}
	cpulist := sigar.CpuList{}
	load := sigar.LoadAverage{}
	uptime := sigar.Uptime{}
	mem := sigar.Mem{}
	swap := sigar.Swap{}
	plist := sigar.ProcList{}
	fslist.Get()
	cpulist.Get()
	load.Get()
	uptime.Get()
	mem.Get()
	swap.Get()
	plist.Get()
	h := &ProcList{}
	g := &Swap{
		Total: swap.Total,
		Free:  swap.Free,
		Used:  swap.Used,
	}
	f := &Mem{
		Used:       mem.Used,
		ActualFree: mem.ActualFree,
		ActualUsed: mem.ActualUsed,
		Total:      mem.Total,
		Free:       mem.Free,
	}
	e := &Uptime{
		Length: uptime.Length,
	}
	d := &LoadAverage{
		One:     load.One,
		Five:    load.Five,
		Fifteen: load.Fifteen,
	}
	c := &CPUS{}
	b := &DF{}
	a := &Application{
		Hostname:    hostname,
		Time:        time.Now().UTC(),
		DF:          b,
		CPU:         c,
		LoadAverage: d,
		Uptime:      e,
		Mem:         f,
		Swap:        g,
		ProcList:    h,
	}
	for _, fs := range fslist.List {
		dir_name := fs.DirName
		usage := sigar.FileSystemUsage{}
		usage.Get(dir_name)
		b.Data = append(b.Data, &FileSystem{
			FileSystem: fs.DevName,
			Size:       usage.Total,
			Used:       usage.Used,
			Avail:      usage.Avail,
			Percent:    usage.UsePercent(),
			Mounted:    dir_name,
		})
	}

	for _, cpu := range cpulist.List {
		c.Data = append(c.Data, &CpuStat{
			ID:      cpu.Name,
			User:    cpu.User,
			Nice:    cpu.Nice,
			Sys:     cpu.Sys,
			Idle:    cpu.Idle,
			Wait:    cpu.Wait,
			IRQ:     cpu.Irq,
			SoftIRQ: cpu.SoftIrq,
			Stolen:  cpu.Stolen,
		})
	}

	for _, proc := range plist.List {
		pr := sigar.ProcState{}
		pm := sigar.ProcMem{}
		pt := sigar.ProcTime{}
		pa := sigar.ProcArgs{}
		pe := sigar.ProcExe{}
		pr.Get(proc)
		pm.Get(proc)
		pt.Get(proc)
		pa.Get(proc)
		pe.Get(proc)
		procm := &ProcMem{
			Size:        pm.Size,
			Resident:    pm.Resident,
			Share:       pm.Share,
			MinorFaults: pm.MinorFaults,
			MajorFaults: pm.MajorFaults,
			PageFaults:  pm.PageFaults,
		}
		proct := &ProcTime{
			StartTime: pt.StartTime,
			Sys:       pt.Sys,
			Total:     pt.Total,
			User:      pt.User,
		}
		proca := &ProcArgs{
			Arguements: pa.List,
		}
		proce := &ProcExe{
			Name: pe.Name,
			Cwd:  pe.Cwd,
			Root: pe.Root,
		}

		h.List = append(h.List, &ProcState{
			Pid:       proc,
			Name:      pr.Name,
			State:     pr.State,
			Ppid:      pr.Ppid,
			Tty:       pr.Tty,
			Priority:  pr.Priority,
			Nice:      pr.Nice,
			Processor: pr.Processor,
			Memory:    procm,
			Time:      proct,
			Args:      proca,
			Exe:       proce,
		})
	}
	//fmt.Printf("%#v", v)
	session, err := mgo.Dial(dbuser + ":" + dbpass + "@" + dbhost + "/" + dbname)
	if err != nil {
		panic(err)
	}
	defer session.Close()
	db := session.DB(dbname).C(hostname)
	err = db.Insert(a)
	if err != nil {
		panic(err)
	}
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
