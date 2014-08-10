package ostent
import (
	"testing"
)

func TestParseArgs(t *testing.T) {
	const defport = "9050"
	for _, v := range []struct{
		a string
		cmp string
	}{
		{   "8001", ":8001"},
		{  ":8001", ":8001"},
		{ "*:8001", ":8001"},
		{ "127.1:8001",     "127.1:8001"},
		{ "127.0.0.1:8001", "127.0.0.1:8001"},
		{ "127.0.0.1",      "127.0.0.1:"+ defport},
		{ "127",            "127.0.0.1:"+ defport},
		{ "127.1",          "127.1:"    + defport},
	} {
		bv := newBind(v.a, defport) // double Set, should be ok
		if err := bv.Set(v.a); err != nil {
			t.Error(err)
		}
		if bv.string != v.cmp {
			t.Errorf("Mismatch: bindFlag %v == %v != %v\n", v.a, v.cmp, bv.string)
		}
	}
}
