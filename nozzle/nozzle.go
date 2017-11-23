package nozzle

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/humio/cloudfoundry2humio/caching"
	"github.com/humio/cloudfoundry2humio/humio"
)

type HumioNozzle struct {
	logger         lager.Logger
	errChan        <-chan error
	msgChan        <-chan *events.Envelope
	signalChan     chan os.Signal
	firehoseClient FirehoseClient
	nozzleConfig   *NozzleConfig
	humioClient    humio.HumioClient
	cachingClient  caching.CachingClient
}

type NozzleConfig struct {
	HumioBatchTime         time.Duration
	HumioMaxMsgNumPerBatch int
}

func NewHumioNozzle(logger lager.Logger, firehoseClient FirehoseClient, nozzleConfig *NozzleConfig, humioClient humio.HumioClient, caching caching.CachingClient) *HumioNozzle {
	return &HumioNozzle{
		logger:         logger,
		errChan:        make(<-chan error),
		msgChan:        make(<-chan *events.Envelope),
		signalChan:     make(chan os.Signal, 2),
		firehoseClient: firehoseClient,
		nozzleConfig:   nozzleConfig,
		humioClient:    humioClient,
		cachingClient:  caching,
	}
}

func (o *HumioNozzle) Start() error {
	o.cachingClient.Initialize()

	// termination signal from CF for proper lifecycle
	signal.Notify(o.signalChan, syscall.SIGTERM, syscall.SIGINT)

	o.msgChan, o.errChan = o.firehoseClient.Connect()

	err := o.routeEvents()
	return err
}

func (o *HumioNozzle) routeEvents() error {
	pendingEvents := make([]humio.Events, 0)

	ticker := time.NewTicker(o.nozzleConfig.HumioBatchTime)
	for {
		select {
		case s := <-o.signalChan:
			o.logger.Info("exiting nozzle", lager.Data{"signal": s.String()})
			err := o.firehoseClient.CloseConsumer()
			if err != nil {
				o.logger.Error("error closing consumer", err)
			}
			os.Exit(1)
		case <-ticker.C:
			currentEvents := pendingEvents
			pendingEvents = make([]humio.Events, 0)
			go o.sendEvents(&currentEvents)
		case msg := <-o.msgChan:
			var humioEvent = humio.NewEvent(msg, o.cachingClient)
			if humioEvent != nil {
				var events = &humio.Events{
					Events: []humio.Event{*humioEvent},
					Tags: humio.Tags{
						AppID:     humioEvent.Attributes.App.ID,
						AppName:   humioEvent.Attributes.App.Name,
						SpaceID:   humioEvent.Attributes.Space.ID,
						SpaceName: humioEvent.Attributes.Space.Name,
						OrgID:     humioEvent.Attributes.Org.ID,
						OrgName:   humioEvent.Attributes.Org.Name,
					},
				}

				pendingEvents = append(pendingEvents, *events)
				doPost := false
				if len(pendingEvents) >= o.nozzleConfig.HumioMaxMsgNumPerBatch {
					doPost = true
					break
				}
				if doPost {
					currentEvents := pendingEvents
					pendingEvents = make([]humio.Events, 0)
					go o.sendEvents(&currentEvents)
				}
			}
		case err := <-o.errChan:
			o.logger.Error("Error while reading from the firehose", err)

			if strings.Contains(err.Error(), "close 1008 (policy violation)") {
				o.logger.Error("Disconnected because nozzle couldn't keep up. Please try scaling up the nozzle.", nil)
				o.logSlowConsumerAlert()
			}

			go o.sendEvents(&pendingEvents)

			o.logger.Error("Closing connection with traffic controller", nil)
			o.firehoseClient.CloseConsumer()
			return err
		}
	}
}

func (o *HumioNozzle) sendEvents(e *[]humio.Events) {
	if len(*e) == 0 {
		return
	}

	for _, ev := range *e {
		var err = o.humioClient.PushEvents(&ev)

		if err != nil {
			o.logger.Error("failed sending events to Humio", err)
		}
	}
}

func (o *HumioNozzle) logSlowConsumerAlert() {
	err := o.humioClient.SingleLog("Humio nozzle is too slow to consume events")

	if err != nil {
		o.logger.Error("failed sending single log to Humio", err)
	}
}
