package nozzle

import (
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/Humio/cf-firehose-to-humio/caching"
	"github.com/Humio/cf-firehose-to-humio/firehose"
	"github.com/Humio/cf-firehose-to-humio/messages"
	"github.com/cloudfoundry/sonde-go/events"
)

type HumioNozzle struct {
	logger              lager.Logger
	errChan             <-chan error
	msgChan             <-chan *events.Envelope
	signalChan          chan os.Signal
	firehoseClient      firehose.Client
	nozzleConfig        *NozzleConfig
	goroutineSem        chan int // to control the number of active post goroutines
	cachingClient       caching.CachingClient
	totalEventsReceived uint64
	totalEventsSent     uint64
	totalEventsLost     uint64
	mutex               *sync.Mutex
}

type NozzleConfig struct {
	HumioBatchTime         time.Duration
	HumioMaxMsgNumPerBatch int
	ExcludeMetricEvents    bool
	ExcludeLogEvents       bool
	ExcludeHttpEvents      bool
	LogEventCount          bool
	LogEventCountInterval  time.Duration
}

func NewHumioNozzle(logger lager.Logger, firehoseClient firehose.Client, nozzleConfig *NozzleConfig, caching caching.CachingClient) *HumioNozzle {
	return &HumioNozzle{
		logger:              logger,
		errChan:             make(<-chan error),
		msgChan:             make(<-chan *events.Envelope),
		signalChan:          make(chan os.Signal, 2),
		firehoseClient:      firehoseClient,
		nozzleConfig:        nozzleConfig,
		cachingClient:       caching,
		totalEventsReceived: uint64(0),
		totalEventsSent:     uint64(0),
		totalEventsLost:     uint64(0),
		mutex:               &sync.Mutex{},
	}
}

func (o *HumioNozzle) Start() error {
	o.cachingClient.Initialize()

	// setup for termination signal from CF
	signal.Notify(o.signalChan, syscall.SIGTERM, syscall.SIGINT)

	o.msgChan, o.errChan = o.firehoseClient.Connect()

	if o.nozzleConfig.LogEventCount {
		o.logTotalEvents(o.nozzleConfig.LogEventCountInterval)
	}
	err := o.routeEvents()
	return err
}

func (o *HumioNozzle) logTotalEvents(interval time.Duration) {
	logEventCountTicker := time.NewTicker(interval)
	lastReceivedCount := uint64(0)
	lastSentCount := uint64(0)
	lastLostCount := uint64(0)

	go func() {
		for range logEventCountTicker.C {
			timeStamp := time.Now().UnixNano()
			totalReceivedCount := o.totalEventsReceived
			totalSentCount := o.totalEventsSent
			totalLostCount := o.totalEventsLost
			currentEvents := make(map[string][]interface{})

			// Generate CounterEvent
			o.addEventCountEvent("eventsReceived", totalReceivedCount-lastReceivedCount, totalReceivedCount, &timeStamp, &currentEvents)
			o.addEventCountEvent("eventsSent", totalSentCount-lastSentCount, totalSentCount, &timeStamp, &currentEvents)
			o.addEventCountEvent("eventsLost", totalLostCount-lastLostCount, totalLostCount, &timeStamp, &currentEvents)

			o.goroutineSem <- 1
			o.postData(&currentEvents, false)

			lastReceivedCount = totalReceivedCount
			lastSentCount = totalSentCount
			lastLostCount = totalLostCount
		}
	}()
}

func (o *HumioNozzle) addEventCountEvent(name string, deltaCount uint64, count uint64, timeStamp *int64, currentEvents *map[string][]interface{}) {
}

func (o *HumioNozzle) postData(events *map[string][]interface{}, addCount bool) {
	for k, v := range *events {
		if len(v) > 0 {
			if msgAsJson, err := json.Marshal(&v); err != nil {
				o.logger.Error("error marshalling message to JSON", err,
					lager.Data{"event type": k},
					lager.Data{"event count": len(v)})
			} else {
				o.logger.Debug("Posting to Humio",
					lager.Data{"event type": k},
					lager.Data{"event count": len(v)},
					lager.Data{"total size": len(msgAsJson)})
				nRetries := 4
				if nRetries == 0 && addCount {
					o.mutex.Lock()
					o.totalEventsLost += uint64(len(v))
					o.mutex.Unlock()
				}
			}
		}
	}
	<-o.goroutineSem
}

func (o *HumioNozzle) routeEvents() error {
	pendingEvents := make(map[string][]interface{})
	// Firehose message processing loop
	ticker := time.NewTicker(o.nozzleConfig.HumioBatchTime)
	for {
		// loop over message and signal channel
		select {
		case s := <-o.signalChan:
			o.logger.Info("exiting", lager.Data{"signal caught": s.String()})
			err := o.firehoseClient.CloseConsumer()
			if err != nil {
				o.logger.Error("error closing consumer", err)
			}
			os.Exit(1)
		case <-ticker.C:
			// get the pending as current
			currentEvents := pendingEvents
			// reset the pending events
			pendingEvents = make(map[string][]interface{})
			o.goroutineSem <- 1
			go o.postData(&currentEvents, true)
		case msg := <-o.msgChan:
			o.totalEventsReceived++
			// process message
			var humioMessage HumioMessage
			var humioMessageType = msg.GetEventType().String()
			switch msg.GetEventType() {
			// Logs Errors
			case events.Envelope_LogMessage:
				if !o.nozzleConfig.ExcludeLogEvents {
					humioMessage = messages.NewLogMessage(msg, o.cachingClient)
					pendingEvents[humioMessageType] = append(pendingEvents[humioMessageType], humioMessage)
				}

			case events.Envelope_Error:
				if !o.nozzleConfig.ExcludeLogEvents {
					humioMessage = messages.NewError(msg, o.cachingClient)
					pendingEvents[humioMessageType] = append(pendingEvents[humioMessageType], humioMessage)
				}

			// HTTP Start/Stop
			case events.Envelope_HttpStartStop:
				if !o.nozzleConfig.ExcludeHttpEvents {
					humioMessage = messages.NewHTTPStartStop(msg, o.cachingClient)
					pendingEvents[humioMessageType] = append(pendingEvents[humioMessageType], humioMessage)
				}
			default:
				o.logger.Info("uncategorized message", lager.Data{"message": msg.String()})
				continue
			}
			// When the number of one type of events reaches the max per batch, trigger the post immediately
			doPost := false
			for _, v := range pendingEvents {
				if len(v) >= o.nozzleConfig.HumioMaxMsgNumPerBatch {
					doPost = true
					break
				}
			}
			if doPost {
				currentEvents := pendingEvents
				pendingEvents = make(map[string][]interface{})
				o.goroutineSem <- 1
				go o.postData(&currentEvents, true)
			}
		case err := <-o.errChan:
			o.logger.Error("Error while reading from the firehose", err)

			if strings.Contains(err.Error(), "close 1008 (policy violation)") {
				o.logger.Error("Disconnected because nozzle couldn't keep up. Please try scaling up the nozzle.", nil)
				o.logSlowConsumerAlert()
			}

			// post the buffered messages
			o.goroutineSem <- 1
			o.postData(&pendingEvents, true)

			o.logger.Error("Closing connection with traffic controller", nil)
			o.firehoseClient.CloseConsumer()
			return err
		}
	}
}

// Log slowConsumerAlert as a ValueMetric event to OMS
func (o *HumioNozzle) logSlowConsumerAlert() {
}

type HumioMessage interface{}
