package ostential
import (
	"ostential/types"

 	"html/template"
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

type ProcLinkattrs types.Linkattrs
func(la ProcLinkattrs) PID()      template.HTMLAttr { return types.Linkattrs(la).Attrs(PSPID ); }
func(la ProcLinkattrs) Priority() template.HTMLAttr { return types.Linkattrs(la).Attrs(PSPRI ); }
func(la ProcLinkattrs) Nice()     template.HTMLAttr { return types.Linkattrs(la).Attrs(PSNICE); }
func(la ProcLinkattrs) Time()     template.HTMLAttr { return types.Linkattrs(la).Attrs(PSTIME); }
func(la ProcLinkattrs) Name()     template.HTMLAttr { return types.Linkattrs(la).Attrs(PSNAME); }
func(la ProcLinkattrs) User()     template.HTMLAttr { return types.Linkattrs(la).Attrs(PSUID ); }
func(la ProcLinkattrs) Size()     template.HTMLAttr { return types.Linkattrs(la).Attrs(PSSIZE); }
func(la ProcLinkattrs) Resident() template.HTMLAttr { return types.Linkattrs(la).Attrs(PSRES ); }

func(la ProcLinkattrs) MarshalJSON() ([]byte, error) {
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
