package ostent
import (
	"libostent/types"
)

var DFTABS = dftabs{
	DFinodes: DFINODES_TABID,
	DFbytes:  DFBYTES_TABID,

	DFinodesTitle: "Disks inodes",
	DFbytesTitle:  "Disks",
}

var IFTABS = iftabs{
	IFpackets: IFPACKETS_TABID,
	IFerrors:  IFERRORS_TABID,
	IFbytes:   IFBYTES_TABID,

	IFpacketsTitle: "Interfaces packets",
	IFerrorsTitle:  "Interfaces errors",
	IFbytesTitle:   "Interfaces",
}

type dftabs struct {
	DFinodes types.SEQ
	DFbytes  types.SEQ

	DFinodesTitle string
	DFbytesTitle  string
}

func (df dftabs) Title(s types.SEQ) string {
	switch {
	case s == df.DFinodes: return df.DFinodesTitle
	case s == df.DFbytes:  return df.DFbytesTitle
	}
	panic("SHOUND NOT HAPPEN")
	return ""
}

type iftabs struct {
	IFpackets types.SEQ
	IFerrors  types.SEQ
	IFbytes   types.SEQ

	IFpacketsTitle string
	IFerrorsTitle  string
	IFbytesTitle   string
}

func (fi iftabs) Title(s types.SEQ) string {
	switch {
	case s == fi.IFpackets:  return fi.IFpacketsTitle
	case s == fi.IFerrors:   return fi.IFerrorsTitle
	case s == fi.IFbytes:    return fi.IFbytesTitle
	}
	panic("SHOUND NOT HAPPEN")
	return ""
}

const (
	____IFTABID types.SEQ = iota
	IFPACKETS_TABID
	 IFERRORS_TABID
	  IFBYTES_TABID
)

const (
	____DFTABID types.SEQ = iota
	DFINODES_TABID
	 DFBYTES_TABID
)

/* UNUSED ?
var IF_TABS = []types.SEQ{
	IFPACKETS_TABID,
	 IFERRORS_TABID,
	  IFBYTES_TABID,
}

var DF_TABS = []types.SEQ{
	DFINODES_TABID,
	 DFBYTES_TABID,
}
*/
