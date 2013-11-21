package phd_aws

import (
	"encoding/csv"
	"fmt"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
	"os"
	"strconv"
	"time"
)

type StorePerformanceReport struct {
	NumApps      int
	Subject      string
	Average      float64
	StdDeviation float64
}

type DataReporter struct {
	Title                      string
	writePerformanceReports    []StorePerformanceReport
	readPerformanceReports     []StorePerformanceReport
	deletePerformanceReporters []StorePerformanceReport
	timestamp                  string
}

func (reporter *DataReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {
	reporter.timestamp = fmt.Sprintf("%d", time.Now().Unix())
	reporter.writePerformanceReports = make(
		[]StorePerformanceReport, 0)
	reporter.readPerformanceReports = make([]StorePerformanceReport, 0)
	reporter.deletePerformanceReporters = make([]StorePerformanceReport, 0)
}

func (reporter *DataReporter) ExampleWillRun(exampleSummary *types.ExampleSummary) {
}

func (reporter *DataReporter) ExampleDidComplete(exampleSummary *types.ExampleSummary) {
	for _, measurement := range exampleSummary.Measurements {
		if measurement.Info != nil {
			info := measurement.Info.(StorePerformanceReport)
			info.Average = measurement.Average
			info.StdDeviation = measurement.StdDeviation
			if info.Subject == "write" {
				reporter.writePerformanceReports = append(reporter.writePerformanceReports, info)
			} else if info.Subject == "read" {
				reporter.readPerformanceReports = append(reporter.readPerformanceReports, info)
			} else if info.Subject == "delete" {
				reporter.deletePerformanceReporters = append(reporter.deletePerformanceReporters, info)
			}
		}
	}

	reporter.generateCSV(false)
}

func (reporter *DataReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {
	reporter.generateCSV(true)
}

func (reporter *DataReporter) generateCSV(final bool) {
	if len(reporter.writePerformanceReports) > 0 {
		finalString := ""
		if final {
			finalString = "_final"
		}
		filename := fmt.Sprintf("CSV_%s_%s%s.csv", reporter.timestamp, reporter.Title, finalString)
		f, err := os.Create(filename)
		if err != nil {
			panic(err.Error())
		}
		defer f.Close()
		w := csv.NewWriter(f)
		w.Write([]string{
			"# Apps",
			"Write Time",
			"σ Write Time",
			"Read Time",
			"σ Read Time",
			"Delete Time",
			"σ Delete Time",
		})

		for i := 0; i < len(reporter.writePerformanceReports); i++ {
			writeReport := reporter.writePerformanceReports[i]
			readReport := reporter.readPerformanceReports[i]
			deleteReport := reporter.deletePerformanceReporters[i]

			w.Write([]string{
				strconv.Itoa(writeReport.NumApps),
				strconv.FormatFloat(writeReport.Average, 'f', 3, 64),
				strconv.FormatFloat(writeReport.StdDeviation, 'f', 3, 64),
				strconv.FormatFloat(readReport.Average, 'f', 3, 64),
				strconv.FormatFloat(readReport.StdDeviation, 'f', 3, 64),
				strconv.FormatFloat(deleteReport.Average, 'f', 3, 64),
				strconv.FormatFloat(deleteReport.StdDeviation, 'f', 3, 64),
			})
		}

		w.Flush()
	}
}
