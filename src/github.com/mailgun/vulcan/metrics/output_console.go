package metrics

import (
	"github.com/golang/glog"
	gmetrics "github.com/rcrowley/go-metrics"
	"time"
)

func logForever(r gmetrics.Registry, d time.Duration) {
	for {
		r.Each(func(name string, i interface{}) {
			switch m := i.(type) {
			case gmetrics.Counter:
				glog.Infof("counter %s\n", name)
				glog.Infof("  count:       %9d\n", m.Count())
			case gmetrics.Gauge:
				glog.Infof("gauge %s\n", name)
				glog.Infof("  value:       %9d\n", m.Value())
			case gmetrics.Healthcheck:
				m.Check()
				glog.Infof("healthcheck %s\n", name)
				glog.Infof("  error:       %v\n", m.Error())
			case gmetrics.Histogram:
				ps := m.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				glog.Infof("histogram %s\n", name)
				glog.Infof("  count:       %9d\n", m.Count())
				glog.Infof("  min:         %9d\n", m.Min())
				glog.Infof("  max:         %9d\n", m.Max())
				glog.Infof("  mean:        %12.2f\n", m.Mean())
				glog.Infof("  stddev:      %12.2f\n", m.StdDev())
				glog.Infof("  median:      %12.2f\n", ps[0])
				glog.Infof("  75%%:         %12.2f\n", ps[1])
				glog.Infof("  95%%:         %12.2f\n", ps[2])
				glog.Infof("  99%%:         %12.2f\n", ps[3])
				glog.Infof("  99.9%%:       %12.2f\n", ps[4])
			case gmetrics.Meter:
				glog.Infof("meter %s\n", name)
				glog.Infof("  count:       %9d\n", m.Count())
				glog.Infof("  1-min rate:  %12.2f\n", m.Rate1())
				glog.Infof("  5-min rate:  %12.2f\n", m.Rate5())
				glog.Infof("  15-min rate: %12.2f\n", m.Rate15())
				glog.Infof("  mean rate:   %12.2f\n", m.RateMean())
			case gmetrics.Timer:
				ps := m.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				glog.Infof("timer %s\n", name)
				glog.Infof("  count:       %9d\n", m.Count())
				glog.Infof("  min:         %9d\n", m.Min())
				glog.Infof("  max:         %9d\n", m.Max())
				glog.Infof("  mean:        %12.2f\n", m.Mean())
				glog.Infof("  stddev:      %12.2f\n", m.StdDev())
				glog.Infof("  median:      %12.2f\n", ps[0])
				glog.Infof("  75%%:         %12.2f\n", ps[1])
				glog.Infof("  95%%:         %12.2f\n", ps[2])
				glog.Infof("  99%%:         %12.2f\n", ps[3])
				glog.Infof("  99.9%%:       %12.2f\n", ps[4])
				glog.Infof("  1-min rate:  %12.2f\n", m.Rate1())
				glog.Infof("  5-min rate:  %12.2f\n", m.Rate5())
				glog.Infof("  15-min rate: %12.2f\n", m.Rate15())
				glog.Infof("  mean rate:   %12.2f\n", m.RateMean())
			}
		})
		time.Sleep(d)
	}
}

func ConsoleOutput() {
	go logForever(gmetrics.DefaultRegistry, 10e9)
}
