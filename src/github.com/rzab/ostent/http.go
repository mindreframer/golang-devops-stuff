package ostent
import (
	"libostent/types"
	"share/assets"
	"share/templates.html"

	"io"
	"bytes"
	"strconv"
	"strings"

	"fmt"
	"sort"
	"sync"
	"os/user"
	"net/url"
	"net/http"
	"html/template"
	"container/ring"

	"github.com/rzab/gosigar"
)

func bps(factor int, current, previous uint) string {
	if current < previous { // counters got reset
		return ""
	}
	diff := (current - previous) * uint(factor) // bits now if the factor is 8
	return humanbits(uint64(diff))
}

func ps(current, previous uint) string {
	if current < previous { // counters got reset
		return ""
	}
	return humanUnitless(uint64(current - previous))
}

func interfaceMeta(ii InterfaceInfo) types.InterfaceMeta {
	return types.InterfaceMeta{
		NameKey:  ii.Name,
		NameHTML: tooltipable(12, ii.Name),
	}
}

type interfaceFormat interface {
	Current(*types.Interface, InterfaceInfo)
	Delta  (*types.Interface, InterfaceInfo, InterfaceInfo)
}
type interfaceInout interface {
	InOut(InterfaceInfo) (uint, uint)
}

type interfaceBytes struct{}
func (_ interfaceBytes) Current(id *types.Interface, ii InterfaceInfo) {
	id.In  = humanB(uint64(ii. InBytes))
	id.Out = humanB(uint64(ii.OutBytes))
}
func (_ interfaceBytes) Delta(id *types.Interface, ii, pi InterfaceInfo) {
	id.DeltaIn  = bps(8, ii. InBytes, pi. InBytes)
	id.DeltaOut = bps(8, ii.OutBytes, pi.OutBytes)
}

type interfaceInoutErrors struct{}
func (_ interfaceInoutErrors) InOut(ii InterfaceInfo) (uint, uint) {
	return ii.InErrors, ii.OutErrors
}
type interfaceInoutPackets struct{}
func (_ interfaceInoutPackets) InOut(ii InterfaceInfo) (uint, uint) {
	return ii.InPackets, ii.OutPackets
}

type interfaceNumericals struct{interfaceInout}
func (ie interfaceNumericals) Current(id *types.Interface, ii InterfaceInfo) {
	in, out := ie.InOut(ii)
	id.In  = humanUnitless(uint64(in))
	id.Out = humanUnitless(uint64(out))
}
func (ie interfaceNumericals) Delta(id *types.Interface, ii, previousi InterfaceInfo) {
	in, out                   := ie.InOut(ii)
	previous_in, previous_out := ie.InOut(previousi)
	id.DeltaIn  = ps(in,  previous_in)
	id.DeltaOut = ps(out, previous_out)
}

func InterfacesDelta(format interfaceFormat, current, previous []InterfaceInfo, client client) *types.Interfaces {
	ifs := make([]types.Interface, len(current))

	for i := range ifs {
		di := types.Interface{
			InterfaceMeta: interfaceMeta(current[i]),
		}
		format.Current(&di, current[i])

		if len(previous) > i {
			format.Delta(&di, current[i], previous[i])
		}

		ifs[i] = di
	}
	if len(ifs) > 1 {
		sort.Sort(interfaceOrder(ifs))
		if !*client.ExpandIF && len(ifs) > client.toprows {
			ifs = ifs[:client.toprows]
		}
	}
	ni := new(types.Interfaces)
	ni.List = ifs
	return ni
}

func(li lastinfo) MEM(client client) *types.MEM {
	mem := new(types.MEM)
	mem.List = append(mem.List, li.RAM)
	if !*client.HideSWAP {
		mem.List = append(mem.List, li.Swap)
	}
	return mem
}

func(li lastinfo) cpuListDelta() (sigar.CpuList, bool) {
	if li.Previous == nil || len(li.Previous.CPU.List) == 0 {
		return li.CPU, false
	}
	prev := li.Previous.CPU
	coreno := len(li.CPU.List)
	if coreno == 0 { // wait, what?
		return sigar.CpuList{}, false
	}
	cls := sigar.CpuList{List: make([]sigar.Cpu, coreno) }
	copy(cls.List, li.CPU.List)
	for i := range cls.List {
		cls.List[i].User -= prev.List[i].User
		cls.List[i].Nice -= prev.List[i].Nice
		cls.List[i].Sys  -= prev.List[i].Sys
		cls.List[i].Idle -= prev.List[i].Idle
	}
	return cls, true
}

func(li lastinfo) CPUDelta(client client) (*types.CPU, int) {
	cls, _ := li.cpuListDelta()
	coreno := len(cls.List)
	if coreno == 0 { // wait, what?
		return &types.CPU{}, coreno
	}

	sum := sigar.Cpu{}
	cores := make([]types.Core, coreno)
	for i, each := range cls.List {

		total := each.User + each.Nice + each.Sys + each.Idle

		user := percent(each.User, total)
		sys  := percent(each.Sys,  total)

		idle := uint(0)
		if user + sys < 100 {
			idle = 100 - user - sys
		}

		cores[i] = types.Core{
			N: fmt.Sprintf("#%d", i),
			User: user,
			Sys:  sys,
			Idle: idle,
			UserClass:  textClass_colorPercent(user),
			SysClass:   textClass_colorPercent(sys),
			IdleClass:  textClass_colorPercent(100 - idle),
			// UserSpark: li.fiveCPU[i].user.spark(),
			// SysSpark:  li.fiveCPU[i].sys .spark(),
			// IdleSpark: li.fiveCPU[i].idle.spark(),
		}

		sum.User += each.User + each.Nice
		sum.Sys  += each.Sys
		sum.Idle += each.Idle
	}

	cpu := new(types.CPU)

	if coreno == 1 {
		cores[0].N = "#0"
		cpu.List = cores
		return cpu, coreno
	}

	sort.Sort(cpuOrder(cores))

	if !*client.ExpandCPU {
		if coreno > client.toprows-1 {
			cores = cores[:client.toprows-1] // first core(s)
		}

		total := sum.User + sum.Sys + sum.Idle // + sum.Nice

		user := percent(sum.User, total)
		sys  := percent(sum.Sys,  total)
		idle := uint(0)
		if user + sys < 100 {
			idle = 100 - user - sys
		}
		cores = append([]types.Core{{ // "all N"
			N: fmt.Sprintf("all %d", coreno),
			User: user,
			Sys:  sys,
			Idle: idle,
			UserClass: textClass_colorPercent(user),
			SysClass:  textClass_colorPercent(sys),
			IdleClass: textClass_colorPercent(100 - idle),
			// UserSpark: .spark(),
			// SysSpark:  .spark(),
			// IdleSpark: .spark(),
		}}, cores...)
	}

	cpu.List = cores
	return cpu, coreno
}

func textClass_colorPercent(p uint) string {
	return "text-" + colorPercent(p)
}

func labelClass_colorPercent(p uint) string {
	return "label label-" + colorPercent(p)
}

func colorPercent(p uint) string {
	if p > 90 { return "danger"  }
	if p > 80 { return "warning" }
	if p > 20 { return "info"    }
	return "success"
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

func tooltipable(limit int, full string) template.HTML {
	if len(full) > limit {
		short := full[:limit]
		if html, err := view.TooltipableTemplate.Execute(struct {
			Full, Short string
		}{
			Full: full,
			Short: short,
		}); err == nil {
			return html
		}
	}
	return template.HTML(template.HTMLEscapeString(full))
}

func orderDisks(disks []diskInfo, seq types.SEQ) []diskInfo {
	if len(disks) > 1 {
		sort.Stable(diskOrder{
			disks: disks,
			seq: seq,
			reverse: _DFBIMAP.SEQ2REVERSE[seq],
		})
	}
	return disks
}

func diskMeta(disk diskInfo) types.DiskMeta {
	return types.DiskMeta{
		DiskNameHTML: tooltipable(12, disk.DevName),
		DirNameHTML:  tooltipable(6, disk.DirName),
		DirNameKey:   disk.DirName,
	}
}

func dfbytes(diskinfos []diskInfo, client client) *types.DFbytes {
	var disks []types.DiskBytes
	for i, disk := range diskinfos {
		if !*client.ExpandDF && i > client.toprows-1 {
			break
		}
		total,  approxtotal  := humanBandback(disk.Total)
		used,   approxused   := humanBandback(disk.Used)
		disks = append(disks, types.DiskBytes{
			DiskMeta: diskMeta(disk),
			Total:       total,
			Used:        used,
			Avail:       humanB(disk.Avail),
			UsePercent:  formatPercent(approxused, approxtotal),
			UsePercentClass: labelClass_colorPercent(percent(approxused,  approxtotal)),
		})
	}
	dsb := new(types.DFbytes)
	dsb.List = disks
	return dsb
}

func dfinodes(diskinfos []diskInfo, client client) *types.DFinodes {
	var disks []types.DiskInodes
	for i, disk := range diskinfos {
		if !*client.ExpandDF && i > client.toprows-1 {
			break
		}
		itotal, approxitotal := humanBandback(disk.Inodes)
		iused,  approxiused  := humanBandback(disk.Iused)
		disks = append(disks, types.DiskInodes{
			DiskMeta: diskMeta(disk),
			Inodes:      itotal,
			Iused:       iused,
			Ifree:       humanB(disk.Ifree),
			IusePercent: formatPercent(approxiused, approxitotal),
			IusePercentClass: labelClass_colorPercent(percent(approxiused, approxitotal)),
		})
	}
	dsi := new(types.DFinodes)
	dsi.List = disks
	return dsi
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

func orderProc(procs []types.ProcInfo, client *client, send *sendClient) []types.ProcData {
	if len(procs) > 1 {
		sort.Sort(procOrder{ // not sort.Stable
			procs:   procs,
			seq:     client.psSEQ,
			reverse: _PSBIMAP.SEQ2REVERSE[client.psSEQ],
		})
	}

	limitPS := client.psLimit
	notdec := limitPS <= 1
	notexp := limitPS >= len(procs)

	if limitPS >= len(procs) { // notexp
		limitPS = len(procs) // NB modified limitPS
	} else {
		procs = procs[:limitPS]
	}

	setBool  (&client.PSnotDecreasable, &send.PSnotDecreasable, notdec)
	setBool  (&client.PSnotExpandable,  &send.PSnotExpandable,  notexp)
	setString(&client.PSplusText,       &send.PSplusText,       fmt.Sprintf("%d+", limitPS))

	uids := map[uint]string{}
	var list []types.ProcData
	for _, proc := range procs {
		list = append(list, types.ProcData{
			PID:        proc.PID,
			Priority:   proc.Priority,
			Nice:       proc.Nice,
			Time:       formatTime(proc.Time),
			NameHTML:   tooltipable(42, proc.Name),
			UserHTML:   tooltipable(12, username(uids, proc.Uid)),
			Size:       humanB(proc.Size),
			Resident:   humanB(proc.Resident),
		})
	}
	return list
}

type Previous struct {
	CPU        sigar.CpuList
	Interfaces []InterfaceInfo
}

type last struct {
	lastinfo
	mutex sync.Mutex
}

type lastinfo struct {
    Generic generic
	CPU     sigar.CpuList
	RAM     types.Memory
	Swap    types.Memory
	DiskList   []diskInfo
	ProcList   []types.ProcInfo
	Interfaces []InterfaceInfo
	Previous *Previous
	lastfive lastfive
}

type lastfive struct {
//	CPU []*fiveCPU
	LA1   *five
}

type fiveCPU struct {
	user, sys, idle *five
}

type five struct {
	*ring.Ring
	min, max int
}

func newFive() *five {
	return &five{Ring: ring.New(5), min: -1, max: -1}
}

func(f *five) push(v int) {
	push(&f, v)
}

func push(ff **five, v int) {
	if *ff == nil {
		*ff = newFive()
	}
	f := *ff
	setmin := f.min == -1 || v < f.min
	setmax := f.max == -1 || v > f.max
	if setmin {
		f.min = v
	}
	if setmax {
		f.max = v
	}

	r := f.Move(1)
	r.Move(4).Value = v
	f.Ring = r // gc please

	// recalc min, max of the remained values

	if !setmin {
		if f.Ring != nil && f.Ring.Value != nil {
			f.min = f.Ring.Value.(int)
		}
		f.Do(func(o interface{}) {
			if o == nil {
				return
			}
			v := o.(int)
			if f.min > v {
				f.min = v
			}
		})
	}
	if !setmax {
		if f.Ring != nil && f.Ring.Value != nil {
			f.max = f.Ring.Value.(int)
		}
		f.Do(func(o interface{}) {
			if o == nil {
				return
			}
			v := o.(int)
			if f.max < v {
				f.max = v
			}
		})
	}
}

func(f five) spark() string {
	if f.max == -1 || f.min == -1 { // || f.max == f.min {
		return ""
	}
	spread := f.max - f.min

	bars := []string{
		"▁",
		"▂",
		"▃",
// 		"▄", // looks bad in browsers
		"▅",
		"▆",
		"▇",
// 		"█", // looks bad in browsers
	}

	s := ""
	f.Do(func(o interface{}) {
		if o == nil {
			return
		}
		v := o.(int)
		fi := 0.0
		if spread != 0 {
			fi = float64(v - f.min) / float64(spread)
			if fi > 1.0 {
				// panic("impossible") // ??
				fi = 1.0
			}
		}
		i := int(fi * float64(len(bars) - 1))
		s += bars[ i ]
	})
	return s
}

type PageData struct {
    Generic generic
	CPU     types.CPU
	MEM     types.MEM

	PStable  PStable
	PSlinks *PSlinks        `json:",omitempty"`

	DFlinks *DFlinks        `json:",omitempty"`
	DFbytes  types.DFbytes  `json:",omitempty"`
	DFinodes types.DFinodes `json:",omitempty"`

	IFbytes   types.Interfaces
	IFerrors  types.Interfaces
	IFpackets types.Interfaces

	VagrantMachines *vagrantMachines
	VagrantError     string
	VagrantErrord    bool

	DISTRIB     string
	VERSION     string
	PeriodDuration Duration

    Client client

	IFTABS iftabs
	DFTABS dftabs
}

type pageUpdate struct {
    Generic  *generic        `json:",omitempty"`

	CPU      *types.CPU      `json:",omitempty"`
	MEM      *types.MEM      `json:",omitempty"`

	DFlinks  *DFlinks        `json:",omitempty"`
	DFbytes  *types.DFbytes  `json:",omitempty"`
	DFinodes *types.DFinodes `json:",omitempty"`

	PSlinks *PSlinks       `json:",omitempty"`
	PStable *PStable       `json:",omitempty"`

	IFbytes   *types.Interfaces `json:",omitempty"`
	IFerrors  *types.Interfaces `json:",omitempty"`
	IFpackets *types.Interfaces `json:",omitempty"`

	VagrantMachines *vagrantMachines `json:",omitempty"`
	VagrantError     string
	VagrantErrord    bool

	Client *sendClient `json:",omitempty"`
}

var lastInfo last

func (la *last) reset_prev() {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	if la.Previous == nil {
		return
	}
	la.Previous.CPU        = sigar.CpuList{}
	la.Previous.Interfaces = []InterfaceInfo{}
}

func (la *last) collect() {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	gch  := make(chan generic,          1)
	rch  := make(chan types.Memory,     1)
	sch  := make(chan types.Memory,     1)
	cch  := make(chan sigar.CpuList,    1)
	dch  := make(chan []diskInfo,       1)
	pch  := make(chan []types.ProcInfo, 1)
	ifch := make(chan InterfacesInfo,   1)

	go getRAM       (rch)
	go getSwap      (sch)
	go getGeneric   (gch)
	go read_disks   (dch)
	go read_procs   (pch)
	go NewInterfaces(ifch)
	go func(CH chan sigar.CpuList) {
		cl := sigar.CpuList{}; cl.Get()
		CH <- cl
	}(cch)

	// .mutex unchanged
	la.lastinfo = lastinfo{
		lastfive: la.lastfive,
		Previous: &Previous{
			CPU:        la.CPU,
			Interfaces: la.Interfaces,
		},
		Generic:  <-gch,
		RAM:      <-rch,
		Swap:     <-sch,
		CPU:      <-cch,
		DiskList: <-dch,
		ProcList: <-pch,
	}

	ii := <-ifch
	la.Generic.IP = ii.IP
	la.Interfaces = filterInterfaces(ii.List)

	// push(&la.lastfive.LA1, la.Generic.la1)
	// la.Generic.LA1spark = la.lastfive.LA1.spark()

	/* delta, isdelta := la.cpuListDelta()
	for i, core := range delta.List {
		var fcpu *fiveCPU
		if i >= len(la.lastfive.CPU) {
			fcpu = &fiveCPU{
				user: newFive(),
				sys:  newFive(),
				idle: newFive(),
			}
			la.lastfive.CPU = append(la.lastfive.CPU, fcpu)
		} else {
			fcpu = la.lastfive.CPU[i]
		}
		if isdelta {
			_ = core
			fcpu.user.push(int(core.User))
			fcpu.sys .push(int(core.Sys))
			fcpu.idle.push(int(core.Idle))
		}
	} // */
}

func linkattrs(req *http.Request, base url.Values, pname string, bimap types.Biseqmap, seq *types.SEQ) *types.Linkattrs {
	*seq = valuesSet(req, base, pname, bimap)
	return &types.Linkattrs{
		Base:  base,
		Pname: pname,
		Bimap: bimap,
	}
}

func getUpdates(req *http.Request, client *client, send sendClient, forcerefresh bool) pageUpdate {

	client.recalcrows() // before anything

	var (
		coreno      int
		df_copy     []diskInfo
		ps_copy     []types.ProcInfo
		if_copy     []InterfaceInfo
		previf_copy []InterfaceInfo
	)
	pu := pageUpdate{}
	func() {
		lastInfo.mutex.Lock()
		defer lastInfo.mutex.Unlock()

		df_copy = make([]diskInfo,       len(lastInfo.DiskList))
		ps_copy = make([]types.ProcInfo, len(lastInfo.ProcList))
		if_copy = make([]InterfaceInfo,  len(lastInfo.Interfaces))

		copy(df_copy, lastInfo.DiskList)
		copy(ps_copy, lastInfo.ProcList)
		copy(if_copy, lastInfo.Interfaces)

		if lastInfo.lastinfo.Previous != nil {
			previf_copy = make([]InterfaceInfo, len(lastInfo.Previous.Interfaces))
			copy(previf_copy, lastInfo.Previous.Interfaces)
		}

		if client.RefreshGeneric.refresh(forcerefresh) {
			g := lastInfo.Generic
			// g.LA = g.LA1spark + " " + g.LA
			pu.Generic = &g // &lastInfo.Generic
		}
		if !*client.HideMEM && client.RefreshMEM.refresh(forcerefresh) {
			pu.MEM = lastInfo.MEM(*client)
		}
		if !*client.HideCPU && client.RefreshCPU.refresh(forcerefresh) {
			pu.CPU, coreno = lastInfo.CPUDelta(*client)
		}
	}()

	if req != nil {
		req.ParseForm() // do ParseForm even if req.Form == nil, otherwise *links won't be set for page requests without parameters
		base := url.Values{}
		pu.PSlinks = (*PSlinks)(linkattrs(req, base, "ps", _PSBIMAP, &client.psSEQ))
		pu.DFlinks = (*DFlinks)(linkattrs(req, base, "df", _DFBIMAP, &client.dfSEQ))
	}

	if pu.CPU != nil { // TODO Is it ok to update the *client.Expand*CPU when the CPU is shown only?
		setBool  (&client.ExpandableCPU, &send.ExpandableCPU, coreno > client.toprows - 1) // one row reserved for "all N"
		setString(&client.ExpandtextCPU, &send.ExpandtextCPU, fmt.Sprintf("Expanded (%d)", coreno))
	}

	if true {
		setBool  (&client.ExpandableIF, &send.ExpandableIF, len(if_copy) > client.toprows)
		setString(&client.ExpandtextIF, &send.ExpandtextIF, fmt.Sprintf("Expanded (%d)", len(if_copy)))

		setBool  (&client.ExpandableDF, &send.ExpandableDF, len(df_copy) > client.toprows)
		setString(&client.ExpandtextDF, &send.ExpandtextDF, fmt.Sprintf("Expanded (%d)", len(df_copy)))
	}

	if !*client.HideDF && client.RefreshDF.refresh(forcerefresh) {
		orderedDisks := orderDisks(df_copy, client.dfSEQ)

		       if *client.TabDF == DFBYTES_TABID  { pu.DFbytes  = dfbytes (orderedDisks, *client)
		} else if *client.TabDF == DFINODES_TABID { pu.DFinodes = dfinodes(orderedDisks, *client)
		}
	}

	if !*client.HideIF && client.RefreshIF.refresh(forcerefresh) {
		switch *client.TabIF {
		case IFBYTES_TABID:   pu.IFbytes   = InterfacesDelta(interfaceBytes{},                             if_copy, previf_copy, *client)
		case IFERRORS_TABID:  pu.IFerrors  = InterfacesDelta(interfaceNumericals{interfaceInoutErrors{}},  if_copy, previf_copy, *client)
		case IFPACKETS_TABID: pu.IFpackets = InterfacesDelta(interfaceNumericals{interfaceInoutPackets{}}, if_copy, previf_copy, *client)
		}
	}

	if !*client.HidePS && client.RefreshPS.refresh(forcerefresh) {
		pu.PStable = new(PStable)
		pu.PStable.List = orderProc(ps_copy, client, &send)
	}

	if !*client.HideVG && client.RefreshVG.refresh(forcerefresh) {
		machines, err := vagrantmachines()
		if err != nil {
			pu.VagrantError = err.Error()
			pu.VagrantErrord = true
		} else {
			pu.VagrantMachines = machines
			pu.VagrantErrord = false
		}
	}

	if send != (sendClient{}) {
		pu.Client = &send
	}
	return pu
}

func pageData(req *http.Request) PageData {
	if Connections.Len() == 0 {
		// collect when there're no active connections, so Loop does not collect
		lastInfo.collect()
	}

	client := defaultClient()
	updates := getUpdates(req, &client, sendClient{}, true)

	data := PageData{
		Client:     client,
		Generic:   *updates.Generic,
		CPU:       *updates.CPU,
		MEM:       *updates.MEM,

		DFlinks:    updates.DFlinks,
		PSlinks:    updates.PSlinks,

		PStable:   *updates.PStable,

		DISTRIB:    DISTRIB, // value from init_*.go
		VERSION:    VERSION, // value from server.go
		PeriodDuration: periodFlag.Duration,
	}

	       if updates.DFbytes  != nil { data.DFbytes  = *updates.DFbytes
	} else if updates.DFinodes != nil { data.DFinodes = *updates.DFinodes
	}

	       if updates.IFbytes   != nil { data.IFbytes   = *updates.IFbytes
	} else if updates.IFerrors  != nil { data.IFerrors  = *updates.IFerrors
	} else if updates.IFpackets != nil { data.IFpackets = *updates.IFpackets
	}
	data.VagrantMachines = updates.VagrantMachines
	data.VagrantError    = updates.VagrantError
	data.VagrantErrord   = updates.VagrantErrord

	data.DFTABS = DFTABS // from tabs.go
	data.IFTABS = IFTABS // from tabs.go

	return data
}

func statusLine(status int) string {
	return fmt.Sprintf("%d %s", status, http.StatusText(status))
}

func init() {
	SCRIPTS = assets.JsAssetNames()
}

var SCRIPTS []string
var INDEXTEMPLATE = view.Bincompile()

func scripts(r *http.Request) (scripts []string) {
	for _, s := range SCRIPTS {
		if !strings.HasPrefix(string(s), "//") {
			s = "//"+r.Host+s
		}
		scripts = append(scripts, s)
	}
	return scripts
}

func index(w http.ResponseWriter, r *http.Request) {
	indexTemplate, err := INDEXTEMPLATE.Clone()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	if err := indexTemplate.ExecuteTemplate(buf, "index.html",
		struct{
			Data interface{}
			SCRIPTS []string
			CLASSNAME string
		}{
			Data: pageData(r),
			SCRIPTS: scripts(r),
		},
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
 	w.Header().Set("Content-Length", strconv.Itoa(buf.Len())) // len(buf.String())
	io.Copy(w, buf) // or w.Write(buf.Bytes())
}
