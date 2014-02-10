package metricsserver

import (
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/metricsaccountant"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	"strconv"
)

type CollectorRegistrar interface {
	RegisterWithCollector(cfcomponent.Component) error
}

type MetricsServer struct {
	registrar         CollectorRegistrar
	steno             *gosteno.Logger
	store             store.Store
	logger            logger.Logger
	timeProvider      timeprovider.TimeProvider
	config            *config.Config
	metricsAccountant metricsaccountant.MetricsAccountant
}

func New(registrar CollectorRegistrar, steno *gosteno.Logger, metricsAccountant metricsaccountant.MetricsAccountant, logger logger.Logger, store store.Store, timeProvider timeprovider.TimeProvider, conf *config.Config) *MetricsServer {
	return &MetricsServer{
		registrar:         registrar,
		store:             store,
		timeProvider:      timeProvider,
		steno:             steno,
		logger:            logger,
		config:            conf,
		metricsAccountant: metricsAccountant,
	}
}

func (s *MetricsServer) Emit() (context instrumentation.Context) {
	context.Name = "HM9000"

	NumberOfAppsWithAllInstancesReporting := 0
	NumberOfAppsWithMissingInstances := 0
	NumberOfUndesiredRunningApps := 0
	NumberOfRunningInstances := 0
	NumberOfMissingIndices := 0
	NumberOfCrashedInstances := 0
	NumberOfCrashedIndices := 0
	NumberOfDesiredApps := 0
	NumberOfDesiredInstances := 0
	NumberOfDesiredAppsPendingStaging := 0

	defer func() {
		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "NumberOfAppsWithAllInstancesReporting",
			Value: NumberOfAppsWithAllInstancesReporting,
		})

		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "NumberOfAppsWithMissingInstances",
			Value: NumberOfAppsWithMissingInstances,
		})

		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "NumberOfUndesiredRunningApps",
			Value: NumberOfUndesiredRunningApps,
		})

		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "NumberOfRunningInstances",
			Value: NumberOfRunningInstances,
		})

		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "NumberOfMissingIndices",
			Value: NumberOfMissingIndices,
		})

		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "NumberOfCrashedInstances",
			Value: NumberOfCrashedInstances,
		})

		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "NumberOfCrashedIndices",
			Value: NumberOfCrashedIndices,
		})

		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "NumberOfDesiredApps",
			Value: NumberOfDesiredApps,
		})

		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "NumberOfDesiredInstances",
			Value: NumberOfDesiredInstances,
		})

		context.Metrics = append(context.Metrics, instrumentation.Metric{
			Name:  "NumberOfDesiredAppsPendingStaging",
			Value: NumberOfDesiredAppsPendingStaging,
		})
	}()

	messageMetrics, err := s.metricsAccountant.GetMetrics()
	if err == nil {
		for key, value := range messageMetrics {
			context.Metrics = append(context.Metrics, instrumentation.Metric{
				Name:  key,
				Value: value,
			})
		}
	}

	err = s.store.VerifyFreshness(s.timeProvider.Time())
	if err != nil {
		s.logger.Error("Failed to server metrics: store is not fresh", err)
		NumberOfAppsWithAllInstancesReporting = -1
		NumberOfAppsWithMissingInstances = -1
		NumberOfUndesiredRunningApps = -1
		NumberOfRunningInstances = -1
		NumberOfMissingIndices = -1
		NumberOfCrashedInstances = -1
		NumberOfCrashedIndices = -1
		NumberOfDesiredApps = -1
		NumberOfDesiredInstances = -1
		NumberOfDesiredAppsPendingStaging = -1
		return
	}

	apps, err := s.store.GetApps()
	if err != nil {
		s.logger.Error("Failed to fetch apps: store is not fresh", err)
		NumberOfAppsWithAllInstancesReporting = -1
		NumberOfAppsWithMissingInstances = -1
		NumberOfUndesiredRunningApps = -1
		NumberOfRunningInstances = -1
		NumberOfMissingIndices = -1
		NumberOfCrashedInstances = -1
		NumberOfCrashedIndices = -1
		NumberOfDesiredApps = -1
		NumberOfDesiredInstances = -1
		NumberOfDesiredAppsPendingStaging = -1
		return
	}

	for _, app := range apps {
		numberOfMissingIndicesForApp := app.NumberOfDesiredInstances() - app.NumberOfDesiredIndicesReporting()
		if app.IsDesired() {
			if app.Desired.PackageState == models.AppPackageStatePending {
				NumberOfDesiredAppsPendingStaging++
			} else {
				NumberOfDesiredApps += 1
				NumberOfDesiredInstances += app.NumberOfDesiredInstances()

				if numberOfMissingIndicesForApp == 0 {
					NumberOfAppsWithAllInstancesReporting++
				} else {
					NumberOfAppsWithMissingInstances++
				}
				NumberOfMissingIndices += numberOfMissingIndicesForApp
			}
		} else {
			if app.HasStartingOrRunningInstances() {
				NumberOfUndesiredRunningApps++
			}
		}

		NumberOfRunningInstances += app.NumberOfStartingOrRunningInstances()
		NumberOfCrashedInstances += app.NumberOfCrashedInstances()
		NumberOfCrashedIndices += app.NumberOfCrashedIndices()
	}

	return
}

func (s *MetricsServer) Ok() bool {
	return true
}

func (s *MetricsServer) Start() error {
	component, err := cfcomponent.NewComponent(
		s.steno,
		"HM9000",
		0,
		s,
		uint32(s.config.MetricsServerPort),
		[]string{s.config.MetricsServerUser, s.config.MetricsServerPassword},
		[]instrumentation.Instrumentable{s},
	)

	if err != nil {
		return err
	}

	s.logger.Info("Serving Metrics", map[string]string{
		"IP":       component.IpAddress,
		"Port":     strconv.Itoa(int(component.StatusPort)),
		"Username": component.StatusCredentials[0],
		"Password": component.StatusCredentials[1],
	})

	go component.StartMonitoringEndpoints()

	err = s.registrar.RegisterWithCollector(component)

	return err
}
