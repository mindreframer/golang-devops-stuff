package ostent
import (
	"libostent/types"

	"encoding/json"
)

type procOrder struct {
	procs []types.ProcInfo
	seq types.SEQ
	reverse bool
}
func(po procOrder) Len() int {
	return len(po.procs)
}
func(po procOrder) Swap(i, j int) {
	po.procs[i], po.procs[j] = po.procs[j], po.procs[i]
}
func(po procOrder) Less(i, j int) bool {
	t := false
	switch po.seq {
	case PSPID,  -PSPID:  t = po.seq.Sign(po.procs[i].PID      < po.procs[j].PID)
	case PSPRI,  -PSPRI:  t = po.seq.Sign(po.procs[i].Priority < po.procs[j].Priority)
	case PSNICE, -PSNICE: t = po.seq.Sign(po.procs[i].Nice     < po.procs[j].Nice)
	case PSSIZE, -PSSIZE: t = po.seq.Sign(po.procs[i].Size     < po.procs[j].Size)
	case PSRES,  -PSRES:  t = po.seq.Sign(po.procs[i].Resident < po.procs[j].Resident)
	case PSTIME, -PSTIME: t = po.seq.Sign(po.procs[i].Time     < po.procs[j].Time)
	case PSNAME, -PSNAME: t = po.seq.Sign(po.procs[i].Name     < po.procs[j].Name)
	case PSUID,  -PSUID:  t = po.seq.Sign(po.procs[i].Uid      < po.procs[j].Uid)
	}
	if po.reverse {
		return !t
	}
	return t
}
const (
____PSIOTA		types.SEQ = iota
	PSPID
    PSPRI
    PSNICE
    PSSIZE
    PSRES
	PSTIME
	PSNAME
	PSUID
)

type PSlinks types.Linkattrs
func(la PSlinks) PID()      types.Attr { return types.Linkattrs(la).Attr(PSPID ); }
func(la PSlinks) Priority() types.Attr { return types.Linkattrs(la).Attr(PSPRI ); }
func(la PSlinks) Nice()     types.Attr { return types.Linkattrs(la).Attr(PSNICE); }
func(la PSlinks) Time()     types.Attr { return types.Linkattrs(la).Attr(PSTIME); }
func(la PSlinks) Name()     types.Attr { return types.Linkattrs(la).Attr(PSNAME); }
func(la PSlinks) User()     types.Attr { return types.Linkattrs(la).Attr(PSUID ); }
func(la PSlinks) Size()     types.Attr { return types.Linkattrs(la).Attr(PSSIZE); }
func(la PSlinks) Resident() types.Attr { return types.Linkattrs(la).Attr(PSRES ); }

func(la PSlinks) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]types.Attr{
		"PID":      types.Linkattrs(la).Attr(PSPID),
		"Priority": types.Linkattrs(la).Attr(PSPRI),
		"Nice":     types.Linkattrs(la).Attr(PSNICE),
		"Time":     types.Linkattrs(la).Attr(PSTIME),
		"Name":     types.Linkattrs(la).Attr(PSNAME),
		"User":     types.Linkattrs(la).Attr(PSUID),
		"Size":     types.Linkattrs(la).Attr(PSSIZE),
		"Resident": types.Linkattrs(la).Attr(PSRES),
	})
}

type PStable struct {
	List []types.ProcData `json:",omitempty"`
}
