package assets
import (
	"fmt"
	"sort"
	"strings"
	"path/filepath"
)

type sortassets struct {
	names []string
	substr_indexfrom []string
}

func (sa sortassets) Len() int {
	return len(sa.names)
}

func (sa sortassets) Less(i, j int) bool {
	ii, jj := sa.Len(), sa.Len()
	for w, v := range sa.substr_indexfrom {
		if strings.Contains(sa.names[i], v) {
			ii = w
		}
		if strings.Contains(sa.names[j], v) {
			jj = w
		}
	}
	// fmt.Printf("Less %t ((%d) %d vs %d) %s vs %s\n", ii < jj, sa.Len(), ii, jj, sa.names[i], sa.names[j])
	return ii < jj
}

func (sa sortassets) Swap(i, j int) {
	// fmt.Printf("Swap %d and %d; %s and %s\n", i, j, sa.names[i], sa.names[j])
	sa.names[i], sa.names[j] = sa.names[j], sa.names[i]
}

func JsAssetNames() []string {
	sa := sortassets{
		substr_indexfrom: []string{
			"jquery",
			"bootstrap",
			"react",
			"headroom",

			"gen", "jsript", // either /gen/ or /jscript/
			"milk", // from coffee script
		},
	}
	develreact := false

	for _, name := range AssetNames() {
		const dotjs = ".js"
		if !strings.HasSuffix(name, dotjs) {
			continue
		}
		src := "/"+name
		if develreact && strings.Contains(src, "react") {
			ver  := filepath.Base(filepath.Dir(src))
			base := filepath.Base(src)

			cutlen := len(dotjs) // asserted strings.HasSuffix(base, dotjs)
			cutlen += map[bool]int{true:len(".min")}[strings.HasSuffix(base[:len(base) - cutlen], ".min")]
			src = fmt.Sprintf("//fb.me/%s-%s%s", base[:len(base) - cutlen], ver, dotjs)
		}
		sa.names = append(sa.names, src)
	}

	sort.Stable(sa)
	return sa.names
}
