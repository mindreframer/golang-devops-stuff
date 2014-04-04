package ostential
import (
	"fmt"
	"time"
	"math"
	"strconv"
)

func formatUptime(seconds float64) string { // "seconds" is expected to be sigar.Uptime.Length
	d := time.Duration(seconds) * time.Second
	s := ""
	if d > time.Duration(24) * time.Hour {
		days := d / time.Hour / 24
		end := ""
		if days > 1 {
			end = "s"
		}
		s += fmt.Sprintf("%d day%s, ", days, end)
	}
	t := time.Unix(int64(seconds), 0).UTC()
	tf := t.Format("15:04")
	if tf[0] == '0' {
		tf = " " + tf[1:]
	}
	s += tf
	return s
}

func humanBmany(n uint64, bits ...bool) (string, string, float64, float64) { // almost humanize.IBytes
	sizes    := []string{"B", "K", "M", "G", "T", "P", "E"}
	if len(bits) > 0 && bits[0] { // bits instead of bytes
		sizes = []string{"b", "k", "m", "g", "t", "p", "e"}
	}
	if n < 10 {
		return fmt.Sprintf("%d%s", n, sizes[0]) /* "%dB" */, "%.0f", float64(n), float64(1)
	}
	base := float64(1024)
	e := math.Floor(math.Log(float64(n)) / math.Log(base))
	pow := math.Pow(base, math.Floor(e))
	val := float64(n) / pow
	f := "%.0f"
	if val < 10 {
		f = "%.1f"
	}
	s := fmt.Sprintf(f+"%s", val, sizes[int(e)])
	if s[0] == ' ' {
		panic(fmt.Errorf("UNEXPECTED: starts with a space: \"%v\"", s))
	}
	return s, f, val, pow
}
func humanbits(n uint64) string {
	s, _, _, _ := humanBmany(n, true); return s
}
func humanB(n uint64, bits ...bool) string {
	s, _, _, _ := humanBmany(n, bits...); return s
}
func humanBandback(n uint64, bits ...bool) (string, uint64) {
	s, f, val, pow := humanBmany(n, bits...)
	d, err := strconv.ParseFloat(fmt.Sprintf(f, val), 64)
	if err != nil {
		panic(err)
	}
	return s, uint64(d * pow)
}

func percent(used, total uint64) uint {
	if total == 0 {
		return 0
	}
	used *= 100
	pct := uint64(used / total)
	if pct != 99 && used % total != 0 {
		pct += 1
	}
	return uint(pct)
}

func formatPercent(used, total uint64) string {
	return strconv.Itoa(int(percent(used, total))) // without "%"
}

func formatTime(T uint64) string {
// 	ms := T % 60
	t  := T / 1000
	ss := t % 60; t /= 60
	mm := t % 60; t /= 60
	hh := t % 24
	if hh > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hh, mm, ss)
	}
	return fmt.Sprintf("   %02d:%02d", mm, ss)
}
