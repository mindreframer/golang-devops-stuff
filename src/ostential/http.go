package ostential
import (
	"ostential/types"
	"ostential/view"

	"fmt"
	"sort"
	"sync"
	"os/user"
	"net/url"
	"net/http"

 	"html/template"

	"github.com/rzab/gosigar"
	"github.com/codegangsta/martini"
)

func(s state) InterfacesDelta() []types.DeltaInterface {
	ifs := make([]types.DeltaInterface, len(s.InterfacesTotal))
	// copy(ifs, s.Interfaces)
	/* for i := range ifs {
		ifs[i].InterfaceTotal = s.InterfacesTotal[i]
	} // */

	prevtotals := s.PrevInterfacesTotal
	if len(prevtotals) == 0 {
		for i := range ifs {
			ifs[i] = types.DeltaInterface{
				Name: s.InterfacesTotal[i].Name,
				In:   humanB(uint64(s.InterfacesTotal[i].In)),
				Out:  humanB(uint64(s.InterfacesTotal[i].Out)),
			}
		}
		return ifs
	}
	bps := func(nowin, previn uint) string {
		if nowin < previn { // counters got reset
			return ""
		}
		n := (nowin - previn) * 8 // bits now
		return humanbits(uint64(n))
	}
	for i := range ifs {
		ifs[i] = types.DeltaInterface{
			Name:     s.InterfacesTotal[i].Name,
			In:       humanB(uint64(s.InterfacesTotal[i].In)),
			Out:      humanB(uint64(s.InterfacesTotal[i].Out)),
			DeltaIn:  bps(s.InterfacesTotal[i].In,  prevtotals[i].In),
			DeltaOut: bps(s.InterfacesTotal[i].Out, prevtotals[i].Out),
		}
	}
	sort.Sort(interfaceOrder(ifs))
	return ifs
}

func(s state) cpudelta() sigar.CpuList {
	prev := s.PREVCPU
	if len(prev.List) == 0 {
		return s.RAWCPU
	}
// 	cls := s.RAWCPU
	cls := sigar.CpuList{List: make([]sigar.Cpu, len(s.RAWCPU.List)) }
	copy(cls.List, s.RAWCPU.List)
	for i := range cls.List {
		cls.List[i].User -= prev.List[i].User
		cls.List[i].Nice -= prev.List[i].Nice
		cls.List[i].Sys  -= prev.List[i].Sys
		cls.List[i].Idle -= prev.List[i].Idle
	}
 	sort.Sort(cpuOrder(cls.List))
	return cls
}

func(s state) CPU() types.CPU {
	sum := sigar.Cpu{}
	cls := s.cpudelta()
	c := types.CPU{List: make([]types.CPU, len(cls.List))}
	for i, cp := range cls.List {

		total := cp.User + cp.Nice + cp.Sys + cp.Idle

		user := percent(cp.User, total)
		sys  := percent(cp.Sys,  total)

		idle := uint(0)
		if user + sys < 100 {
			idle = 100 - user - sys
		}

		c.List[i].N    = i
 		c.List[i].User, c.List[i].AttrUser = user, textAttr_colorPercent(user)
 		c.List[i].Sys,  c.List[i].AttrSys  = sys,  textAttr_colorPercent(sys)
		c.List[i].Idle, c.List[i].AttrIdle = idle, textAttr_colorPercent(100 - idle)

		sum.User += cp.User + cp.Nice
		sum.Sys  += cp.Sys
		sum.Idle += cp.Idle
	}
	total := sum.User + sum.Sys + sum.Idle // + sum.Nice

	user := percent(sum.User, total)
	sys  := percent(sum.Sys,  total)
	idle := uint(0)
	if user + sys < 100 {
		idle = 100 - user - sys
	}

	c.N    = len(cls.List)
 	c.User, c.AttrUser = user, textAttr_colorPercent(user)
 	c.Sys,  c.AttrSys  = sys,  textAttr_colorPercent(sys)
	c.Idle, c.AttrIdle = idle, textAttr_colorPercent(100 - idle)

	return c
}
func textAttr_colorPercent(p uint) template.HTMLAttr {
	return template.HTMLAttr(" class=\"text-" + colorPercent(p) + "\"")
}
func labelAttr_colorPercent(p uint) template.HTMLAttr {
	return template.HTMLAttr(" class=\"label label-" + colorPercent(p) + "\"")
}
func colorPercent(p uint) string {
	if p > 90 {
		return "danger"
	}
	if p > 80 {
		return "warning"
	}
	if p > 20 {
		return "info"
	}
	return "success"
}

type memory struct {
	Total       string
	Used        string
	Free        string
	UsePercent  string

	AttrUsePercent template.HTMLAttr `json:"-"`
}

type diskInfo struct {
	DevName     string
	Total       uint64
	Used        uint64
	Avail       uint64
	UsePercent  float64
	Inodes      uint64
	Iused       uint64
	Ifree       uint64
	IusePercent float64
	DirName     string
}

func valuesSet(req *http.Request, base url.Values, pname string, bimap types.Biseqmap) types.SEQ {
	if params, ok := req.Form[pname]; ok && len(params) > 0 {
		if seq, ok := bimap.STRING2SEQ[params[0]]; ok {
			base.Set(pname, params[0])
			return seq
		}
	}
	return bimap.Default_seq
}

func orderDisk(disks []diskInfo, seq types.SEQ) []types.DiskData {
	sort.Stable(diskOrder{
		disks: disks,
		seq: seq,
		reverse: _DFBIMAP.SEQ2REVERSE[seq],
	})

	var dd []types.DiskData
	for _, disk := range disks {
		total,  approxtotal  := humanBandback(disk.Total)
		used,   approxused   := humanBandback(disk.Used)
		itotal, approxitotal := humanBandback(disk.Inodes)
		iused,  approxiused  := humanBandback(disk.Iused)

		short := ""
		if len(disk.DevName) > 10 {
			short = disk.DevName[:10]
		}
		dd = append(dd, types.DiskData{
			DiskName:    disk.DevName,
			ShortDiskName: short,

			Total:       total,
			Used:        used,
			Avail:       humanB(disk.Avail),
			UsePercent:  formatPercent(approxused, approxtotal),

			Inodes:      itotal,
			Iused:       iused,
			Ifree:       humanB(disk.Ifree),
			IusePercent: formatPercent(approxiused, approxitotal),

			DirName:     disk.DirName,

			AttrUsePercent:  labelAttr_colorPercent(percent(approxused,  approxtotal)),
			AttrIusePercent: labelAttr_colorPercent(percent(approxiused, approxitotal)),
		})
	}
	return dd
}

var _DFBIMAP = types.Seq2bimap(DFFS, // the default seq for ordering
	types.Seq2string{
		DFFS:      "fs",
		DFSIZE:    "size",
		DFUSED:    "used",
		DFAVAIL:   "avail",
		DFMP:      "mp",
	}, []types.SEQ{
		DFFS, DFMP,
	})

var _PSBIMAP = types.Seq2bimap(PSPID, // the default seq for ordering
	types.Seq2string{
		PSPID:   "pid",
		PSPRI:   "pri",
		PSNICE:  "nice",
		PSSIZE:  "size",
		PSRES:   "res",
		PSTIME:  "time",
		PSNAME:  "name",
		PSUID:   "user",
	}, []types.SEQ{
		PSNAME, PSUID,
	})

func username(uids map[uint]string, uid uint) string {
	if s, ok := uids[uid]; ok {
		return s
	}
	s := fmt.Sprintf("%d", uid)
	if usr, err := user.LookupId(s); err == nil {
		s = usr.Username
	}
	uids[uid] = s
	return s
}

func orderProc(procs []types.ProcInfo, seq types.SEQ) []types.ProcData {
	sort.Sort(procOrder{ // not sort.Stable
		procs: procs,
		seq: seq,
		reverse: _PSBIMAP.SEQ2REVERSE[seq],
	})

	if len(procs) > 20 {
		procs = procs[:20]
	}

	uids := map[uint]string{}
	var list []types.ProcData
	for _, proc := range procs {
		list = append(list, types.ProcData{
			PID:        proc.PID,
			Priority:   proc.Priority,
			Nice:       proc.Nice,
			Time:       formatTime(proc.Time),
			Name:       proc.Name,
			User:       username(uids, proc.Uid),
			Size:       humanB(proc.Size),
			Resident:   humanB(proc.Resident),
		})
	}
	return list
}

type state struct {
    About    about
    System   system
	RAWCPU   sigar.CpuList
	PREVCPU  sigar.CpuList
	RAM      memory
	Swap     memory
	DiskList []diskInfo
	ProcList []types.ProcInfo

	InterfacesTotal     []InterfaceTotal
	PrevInterfacesTotal []InterfaceTotal
}

type Page struct {
    About     about
    System    system
	CPU       types.CPU
	RAM       memory
	Swap      memory
	DiskTable DiskTable
	ProcTable ProcTable

	Interfaces types.Interfaces

	DISTRIB   string
	HTTP_HOST string
}
type pageUpdate struct {
    About    about
    System   system
	CPU      types.CPU
	RAM      memory
	Swap     memory

	DiskTable DiskTable
	ProcTable ProcTable

	Interfaces []types.DeltaInterface
}

var stateLock sync.Mutex
var lastState state
func reset_prev() {
	stateLock.Lock()
	defer stateLock.Unlock()
	lastState.PrevInterfacesTotal = []InterfaceTotal{}
	lastState.PREVCPU.List        = []sigar.Cpu{}
}
func collect() { // state
	stateLock.Lock()
	defer stateLock.Unlock()

	prev_ifstotal := lastState.InterfacesTotal
	prev_cpu      := lastState.RAWCPU

	ifs, ip := NewInterfaces()
	about := getAbout()
	about.IP = ip

	lastState = state{
		About:    about,
		System:   getSystem(),
		RAM:      getRAM(),
		Swap:     getSwap(),
		DiskList: read_disks(),
		ProcList: read_procs(),
	}
	cl := sigar.CpuList{}; cl.Get()
	lastState.PREVCPU = prev_cpu
	lastState.RAWCPU  = cl

	ifstotal := filterInterfaces(ifs)
	lastState.PrevInterfacesTotal = prev_ifstotal
	lastState.InterfacesTotal     = ifstotal

//	return lastState
}

func linkattrs(req *http.Request, base url.Values, pname string, bimap types.Biseqmap) types.Linkattrs {
	return types.Linkattrs{
		Base:  base,
		Pname: pname,
		Bimap: bimap,
		Seq:   valuesSet(req, base, pname, bimap),
	}
}

func updates(req *http.Request, new_search bool) (pageUpdate, url.Values, types.SEQ, types.SEQ) {
	req.ParseForm()
	base := url.Values{}

	dflinks := DiskLinkattrs(linkattrs(req, base, "df", _DFBIMAP))
	pslinks := ProcLinkattrs(linkattrs(req, base, "ps", _PSBIMAP))

	var pu pageUpdate
	var disks_copy []diskInfo
	var procs_copy []types.ProcInfo
	func() {
		stateLock.Lock()
		defer stateLock.Unlock()

		disks_copy = make([]diskInfo, len(lastState.DiskList))
		procs_copy = make([]types.ProcInfo, len(lastState.ProcList))
		copy(disks_copy, lastState.DiskList)
		copy(procs_copy, lastState.ProcList)

		pu = pageUpdate{
			About:    lastState.About,
			System:   lastState.System,
			CPU:      lastState.CPU(),
			RAM:      lastState.RAM,
			Swap:     lastState.Swap,
			Interfaces: lastState.InterfacesDelta(),
		}
	}()
	pu.DiskTable.List = orderDisk(disks_copy, dflinks.Seq)
	pu.ProcTable.List = orderProc(procs_copy, pslinks.Seq)
	if new_search {
		pu.ProcTable.Links = &pslinks
		pu.DiskTable.Links = &dflinks
	}
	return pu, base, dflinks.Seq, pslinks.Seq
}

var DISTRIB string // set with init from init_*.go
func collected(req *http.Request) Page {
	latest, base, dfseq, psseq := updates(req, false)
	return Page{
		About:   latest.About,
		System:  latest.System,
		CPU:     latest.CPU,
		RAM:     latest.RAM,
		Swap:    latest.Swap,
		DiskTable: DiskTable{
			List: latest.DiskTable.List,
			Links: &DiskLinkattrs{
				Base: base,
				Pname: "df",
				Bimap: _DFBIMAP,
				Seq: dfseq,
			},
		},
		ProcTable: ProcTable{
			List: latest.ProcTable.List,
			Links: &ProcLinkattrs{
				Base: base,
				Pname: "ps",
				Bimap: _PSBIMAP,
				Seq: psseq,
			},
		},
		Interfaces: types.Interfaces{List: latest.Interfaces },
		DISTRIB: DISTRIB, // from init.go
		HTTP_HOST: req.Host,
	}
}

func index(req *http.Request, r view.Render) {
	r.HTML(200, "index.html", struct{Data interface{}}{collected(req)})
}

type Modern struct {
	*martini.Martini
	 martini.Router // the router functions for convenience
}
