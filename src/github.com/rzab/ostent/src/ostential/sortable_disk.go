package ostential
import (
	"ostential/types"

 	"html/template"
	"encoding/json"
)

type diskOrder struct {
	disks []diskInfo
	seq types.SEQ
	reverse bool
}
func(do diskOrder) Len() int {
	return len(do.disks)
}
func(do diskOrder) Swap(i, j int) {
	do.disks[i], do.disks[j] = do.disks[j], do.disks[i]
}
func(do diskOrder) Less(i, j int) bool {
	t := false
	switch do.seq {
	case DFFS,    -DFFS:    t = do.seq.Sign(do.disks[i].DevName < do.disks[j].DevName)
	case DFSIZE,  -DFSIZE:  t = do.seq.Sign(do.disks[i].Total   < do.disks[j].Total)
	case DFUSED,  -DFUSED:  t = do.seq.Sign(do.disks[i].Used    < do.disks[j].Used)
	case DFAVAIL, -DFAVAIL: t = do.seq.Sign(do.disks[i].Avail   < do.disks[j].Avail)
	case DFMP,    -DFMP:    t = do.seq.Sign(do.disks[i].DirName < do.disks[j].DirName)
	}
	if do.reverse {
		return !t
	}
	return t
}
const (
____DFIOTA		types.SEQ = iota
	DFFS
	DFSIZE
	DFUSED
	DFAVAIL
	DFMP
)

type DiskLinkattrs types.Linkattrs
func(la DiskLinkattrs) DiskName()    template.HTMLAttr { return types.Linkattrs(la).Attrs(DFFS);    }
func(la DiskLinkattrs) Total()       template.HTMLAttr { return types.Linkattrs(la).Attrs(DFSIZE);  }
func(la DiskLinkattrs) Used()        template.HTMLAttr { return types.Linkattrs(la).Attrs(DFUSED);  }
func(la DiskLinkattrs) Avail()       template.HTMLAttr { return types.Linkattrs(la).Attrs(DFAVAIL); }
func(la DiskLinkattrs) DirName()     template.HTMLAttr { return types.Linkattrs(la).Attrs(DFMP);    }

func(la DiskLinkattrs) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]types.Attr{
		"DiskName": types.Linkattrs(la).Attr(DFFS),
		"Total":    types.Linkattrs(la).Attr(DFSIZE),
		"Used":     types.Linkattrs(la).Attr(DFUSED),
		"Avail":    types.Linkattrs(la).Attr(DFAVAIL),
		"DirName":  types.Linkattrs(la).Attr(DFMP),
	})
}
