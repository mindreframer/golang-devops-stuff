package ostent
import (
	"libostent/types"

	"time"
)

type refresh struct {
	Duration
	tick int
}

func(r *refresh) refresh(forcerefresh bool) bool {
	if forcerefresh {
		return true
	}
	expires := r.expires()
	r.tick++
	if !expires {
		return false
	}
	r.tick = 0
	return true
}

func(r refresh) expires() bool {
	return r.tick + 1 >= int(time.Duration(r.Duration) / time.Second)
}

type internalClient struct {
	// NB lowercase fields only, NOT to be marshalled/exported

	psLimit int

	psSEQ types.SEQ
	dfSEQ types.SEQ

	toprows int
}

type title string
func (ti *title) merge(ns string, dt **title) {
	*dt = nil
	if string(*ti) == ns {
		return
	}
	*dt = newtitle(ns)
	*ti = **dt
}

type commonClient struct {
	HideMEM *bool `json:",omitempty"`
	HideIF  *bool `json:",omitempty"`
	HideCPU *bool `json:",omitempty"`
	HideDF  *bool `json:",omitempty"`
	HidePS  *bool `json:",omitempty"`
	HideVG  *bool `json:",omitempty"`

	HideSWAP  *bool `json:",omitempty"`

	ExpandIF  *bool `json:",omitempty"`
	ExpandCPU *bool `json:",omitempty"`
	ExpandDF  *bool `json:",omitempty"`

	TabIF *types.SEQ `json:",omitempty"`
	TabDF *types.SEQ `json:",omitempty"`
	TabTitleIF *title `json:",omitempty"`
	TabTitleDF *title `json:",omitempty"`

	// PSusers []string `json:omitempty`

	HideconfigMEM *bool `json:",omitempty"`
	HideconfigIF  *bool `json:",omitempty"`
	HideconfigCPU *bool `json:",omitempty"`
	HideconfigDF  *bool `json:",omitempty"`
	HideconfigPS  *bool `json:",omitempty"`
	HideconfigVG  *bool `json:",omitempty"`
}

// server side full client state
type client struct {
	internalClient `json:"-"` // NB not marshalled
	commonClient

	ExpandableIF  *bool   `json:",omitempty"`
	ExpandableCPU *bool   `json:",omitempty"`
	ExpandableDF  *bool   `json:",omitempty"`

	ExpandtextIF  *string `json:",omitempty"`
	ExpandtextCPU *string `json:",omitempty"`
	ExpandtextDF  *string `json:",omitempty"`

	RefreshGeneric *refresh `json:",omitempty"`
	RefreshMEM *refresh `json:",omitempty"`
	RefreshIF  *refresh `json:",omitempty"`
	RefreshCPU *refresh `json:",omitempty"`
	RefreshDF  *refresh `json:",omitempty"`
	RefreshPS  *refresh `json:",omitempty"`
	RefreshVG  *refresh `json:",omitempty"`

	PSplusText       *string `json:",omitempty"`
	PSnotExpandable  *bool   `json:",omitempty"`
	PSnotDecreasable *bool   `json:",omitempty"`
}

func (c *client) recalcrows() {
	c.toprows = map[bool]int{true: 1, false: 2}[bool(*c.HideSWAP)]
}

func setBool(b, b2 **bool, v bool) {
	if *b != nil && **b == v {
		return // unchanged
	}
	if *b == nil {
		*b = new(bool)
	}
	**b = v
	*b2 = *b
}

func setString(s, s2 **string, v string) {
	if *s != nil && **s == v {
		return // unchanged
	}
	if *s == nil {
		*s = new(string)
	}
	**s = v
	*s2 = *s
}

type sendClient struct {
	client

	RefreshErrorMEM *bool `json:",omitempty"`
	RefreshErrorIF  *bool `json:",omitempty"`
	RefreshErrorCPU *bool `json:",omitempty"`
	RefreshErrorDF  *bool `json:",omitempty"`
	RefreshErrorPS  *bool `json:",omitempty"`
	RefreshErrorVG  *bool `json:",omitempty"`

	DebugError *string  `json:",omitempty"`
}

func (c client) mergeBool(dst, src *bool, send **bool) {
	// c is unused
	if src == nil {
		return
	}
	*dst = *src
	*send = src
}

func (c client) mergeSEQ(dst, src *types.SEQ, send **types.SEQ) {
	// c is unused
	if src == nil {
		return
	}
	*dst = *src
	*send = src
}

func(c *client) Merge(r recvClient, s *sendClient) {
	c.mergeBool(c.HideMEM, r.HideMEM, &s.HideMEM)
	c.mergeBool(c.HideIF,  r.HideIF,  &s.HideIF)
	c.mergeBool(c.HideCPU, r.HideCPU, &s.HideCPU)
	c.mergeBool(c.HideDF,  r.HideDF,  &s.HideDF)
	c.mergeBool(c.HidePS,  r.HidePS,  &s.HidePS)
	c.mergeBool(c.HideVG,  r.HideVG,  &s.HideVG)

	c.mergeBool(c.HideSWAP,  r.HideSWAP,  &s.HideSWAP)
	c.mergeBool(c.ExpandIF,  r.ExpandIF,  &s.ExpandIF)
	c.mergeBool(c.ExpandCPU, r.ExpandCPU, &s.ExpandCPU)
	c.mergeBool(c.ExpandDF,  r.ExpandDF,  &s.ExpandDF)

	c.mergeBool(c.HideconfigMEM, r.HideconfigMEM, &s.HideconfigMEM)
	c.mergeBool(c.HideconfigIF,  r.HideconfigIF,  &s.HideconfigIF)
	c.mergeBool(c.HideconfigCPU, r.HideconfigCPU, &s.HideconfigCPU)
	c.mergeBool(c.HideconfigDF,  r.HideconfigDF,  &s.HideconfigDF)
	c.mergeBool(c.HideconfigPS,  r.HideconfigPS,  &s.HideconfigPS)
	c.mergeBool(c.HideconfigVG,  r.HideconfigVG,  &s.HideconfigVG)

	c.mergeSEQ (c.TabIF, r.TabIF, &s.TabIF)
	c.mergeSEQ (c.TabDF, r.TabDF, &s.TabDF)

	// merge NOT from the r
	c.TabTitleIF.merge(IFTABS.Title(*c.TabIF), &s.TabTitleIF)
	c.TabTitleDF.merge(DFTABS.Title(*c.TabDF), &s.TabTitleDF)
}

func newtitle(s string) *title {
	p := new(title)
	*p = title(s)
	return p
}

func newfalse()      *bool { return new(bool); }
func newtrue()       *bool { return newbool(true); }
func newbool(v bool) *bool { b := new(bool);  *b = v; return b }

func newseq(v types.SEQ) *types.SEQ {
	s := new(types.SEQ)
	*s = v
	return s
}

func newdefaultrefresh() *refresh {
	r := new(refresh)
	*r = refresh{Duration: periodFlag.Duration}
	return r
}

func defaultClient() client {
	cs := client{}

	cs.HideMEM = newfalse()
	cs.HideIF  = newfalse()
	cs.HideCPU = newfalse()
	cs.HideDF  = newfalse()
	cs.HidePS  = newfalse()
	cs.HideVG  = newfalse()

	cs.HideSWAP  = newfalse()
	cs.ExpandIF  = newfalse()
	cs.ExpandCPU = newfalse()
	cs.ExpandDF  = newfalse()

	cs.TabIF = newseq(IFBYTES_TABID)
	cs.TabDF = newseq(DFBYTES_TABID)
	cs.TabTitleIF = newtitle(IFTABS.Title(*cs.TabIF))
	cs.TabTitleDF = newtitle(DFTABS.Title(*cs.TabDF))

	hideconfig := true
	// hideconfig  = false // DEVELOPMENT

	cs.HideconfigMEM = newbool(hideconfig)
	cs.HideconfigIF  = newbool(hideconfig)
	cs.HideconfigCPU = newbool(hideconfig)
	cs.HideconfigDF  = newbool(hideconfig)
	cs.HideconfigPS  = newbool(hideconfig)
	cs.HideconfigVG  = newbool(hideconfig)

	cs.RefreshGeneric = newdefaultrefresh()
	cs.RefreshMEM = newdefaultrefresh()
	cs.RefreshIF  = newdefaultrefresh()
	cs.RefreshCPU = newdefaultrefresh()
	cs.RefreshDF  = newdefaultrefresh()
	cs.RefreshPS  = newdefaultrefresh()
	cs.RefreshVG  = newdefaultrefresh()

	cs.psLimit = 8

	cs.psSEQ = _PSBIMAP.Default_seq
	cs.dfSEQ = _DFBIMAP.Default_seq

	cs.recalcrows()

	return cs
}
