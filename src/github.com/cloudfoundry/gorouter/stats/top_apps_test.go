package stats_test

import (
	. "github.com/cloudfoundry/gorouter/stats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"time"
)

var _ = Describe("TopApps", func() {

	var topApps *TopApps

	BeforeEach(func() {
		topApps = NewTopApps()
	})

	It("marks application ids", func() {
		topApps.Mark("a", time.Unix(1, 0))
		topApps.Mark("b", time.Unix(1, 0))
		apps := topApps.TopSince(time.Unix(0, 0), 5)
		Ω(apps).To(HaveLen(2))
	})

	It("mark updates existing application ids", func() {
		topApps.Mark("b", time.Unix(1, 0))
		topApps.Mark("b", time.Unix(1, 0))

		apps := topApps.TopSince(time.Unix(0, 0), 5)
		Ω(apps).To(HaveLen(1))
	})

	It("trims aging application ids", func() {
		for i, x := range []string{"a", "b", "c"} {
			topApps.Mark(x, time.Unix(int64(i+1), 0))
		}

		apps := topApps.TopSince(time.Unix(0, 0), 5)
		Ω(apps).To(HaveLen(3))

		topApps.Trim(time.Unix(1, 0))
		apps = topApps.TopSince(time.Unix(1, 0), 5)
		Ω(apps).To(HaveLen(2))

		topApps.Trim(time.Unix(2, 0))
		apps = topApps.TopSince(time.Unix(2, 0), 5)
		Ω(apps).To(HaveLen(1))

		topApps.Trim(time.Unix(3, 0))
		apps = topApps.TopSince(time.Unix(3, 0), 5)
		Ω(apps).To(HaveLen(0))
	})

	It("reports top application ids", func() {
		f := func(x ...TopAppsTopEntry) []TopAppsTopEntry {
			if x == nil {
				x = make([]TopAppsTopEntry, 0)
			}
			return x
		}

		g := func(x string, y int64) TopAppsTopEntry {
			return TopAppsTopEntry{x, y}
		}

		x := []string{"a", "b", "c"}
		for i, y := range x {
			for j := 0; j < len(x); j++ {
				topApps.Mark(y, time.Unix(int64(i+j), 0))
			}
		}

		Ω(topApps.TopSince(time.Unix(2, 0), 3)).To(Equal(f(g("c", 3), g("b", 2), g("a", 1))))
		Ω(topApps.TopSince(time.Unix(3, 0), 3)).To(Equal(f(g("c", 2), g("b", 1))))
		Ω(topApps.TopSince(time.Unix(4, 0), 3)).To(Equal(f(g("c", 1))))
		Ω(topApps.TopSince(time.Unix(5, 0), 3)).To(Equal(f()))
	})
})
