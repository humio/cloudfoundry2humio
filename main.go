package main

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/humio/cloudfoundry2humio/caching"
	"github.com/humio/cloudfoundry2humio/humio"
	"github.com/humio/cloudfoundry2humio/nozzle"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	firehoseSubscriptionID = "humio-nozzle"
	version                = "0.1.0"
)

var (
	apiAddress     = kingpin.Flag("api-addr", "Api URL").OverrideDefaultFromEnvar("API_ADDR").Required().String()
	dopplerAddress = kingpin.Flag("doppler-addr", "Traffic controller URL").OverrideDefaultFromEnvar("DOPPLER_ADDR").Required().String()
	cfUser         = kingpin.Flag("firehose-user", "CF user with admin and firehose access").OverrideDefaultFromEnvar("FIREHOSE_USER").Required().String()
	cfPassword     = kingpin.Flag("firehose-user-password", "Password of the CF user").OverrideDefaultFromEnvar("FIREHOSE_USER_PASSWORD").Required().String()
	environment    = kingpin.Flag("cf-environment", "CF environment name").OverrideDefaultFromEnvar("CF_ENVIRONMENT").Default("cf").String()

	// comma separated list of types to exclude.  For now use metric,log,http and revisit later
	eventFilter       = kingpin.Flag("eventFilter", "Comma separated list of types to exclude").Default("").OverrideDefaultFromEnvar("EVENT_FILTER").String()
	skipSslValidation = kingpin.Flag("skip-ssl-validation", "Skip SSL validation").Default("false").OverrideDefaultFromEnvar("SKIP_SSL_VALIDATION").Bool()
	idleTimeout       = kingpin.Flag("idle-timeout", "Keep Alive duration for the firehose consumer").Default("25s").OverrideDefaultFromEnvar("IDLE_TIMEOUT").Duration()
	logLevel          = kingpin.Flag("log-level", "Log level: DEBUG, INFO, ERROR").Default("INFO").OverrideDefaultFromEnvar("LOG_LEVEL").String()

	// Humio endpoint info
	humioHost        = kingpin.Flag("humio-host", "Humio host endpoint").OverrideDefaultFromEnvar("HUMIO_HOST").Required().String()
	humioDataspace   = kingpin.Flag("humio-dataspace", "Humio dataspace to push logs to").OverrideDefaultFromEnvar("HUMIO_DATASPACE").Required().String()
	humioIngestToken = kingpin.Flag("humio-ingest-token", "Humio ingest token").OverrideDefaultFromEnvar("HUMIO_INGEST_TOKEN").Required().String()
)

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	logger := lager.NewLogger("humio-nozzle")
	level := lager.INFO
	switch strings.ToUpper(*logLevel) {
	case "DEBUG":
		level = lager.DEBUG
	case "ERROR":
		level = lager.ERROR
	}
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, level))

	// enable thread dump
	threadDumpChan := registerGoRoutineDumpSignalChannel()
	defer close(threadDumpChan)
	go dumpGoRoutine(threadDumpChan)

	cachingCFClientConfig := &cfclient.Config{
		ApiAddress:        *apiAddress,
		Username:          *cfUser,
		Password:          *cfPassword,
		SkipSslValidation: *skipSslValidation,
	}

	cachingClient := caching.NewCaching(cachingCFClientConfig, logger, *environment)

	firehoseCFClientConfig := &cfclient.Config{
		ApiAddress:        *apiAddress,
		Username:          *cfUser,
		Password:          *cfPassword,
		SkipSslValidation: *skipSslValidation,
	}

	firehoseConfig := &nozzle.FirehoseConfig{
		SubscriptionId:       firehoseSubscriptionID,
		TrafficControllerUrl: *dopplerAddress,
		IdleTimeout:          *idleTimeout,
	}

	firehoseClient := nozzle.NewFirehoseClient(firehoseCFClientConfig, firehoseConfig, logger)

	humioConfig := &humio.HumioConfig{
		Host:      *humioHost,
		Dataspace: *humioDataspace,
		Token:     *humioIngestToken,
	}

	humioClient := humio.NewHumioClient(humioConfig, logger)

	nozzleConfig := &nozzle.NozzleConfig{
		HumioBatchTime:         5 * time.Second,
		HumioMaxMsgNumPerBatch: 500,
	}

	nozzleApp := nozzle.NewHumioNozzle(logger, firehoseClient, nozzleConfig, humioClient, cachingClient)
	nozzleApp.Start()
}

func registerGoRoutineDumpSignalChannel() chan os.Signal {
	threadDumpChan := make(chan os.Signal, 1)
	signal.Notify(threadDumpChan, syscall.SIGUSR1)

	return threadDumpChan
}

func dumpGoRoutine(dumpChan chan os.Signal) {
	for range dumpChan {
		goRoutineProfiles := pprof.Lookup("goroutine")
		if goRoutineProfiles != nil {
			goRoutineProfiles.WriteTo(os.Stdout, 2)
		}
	}
}
