package nozzle_test

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/Humio/cf-firehose-to-humio/mocks"
	"github.com/Humio/cf-firehose-to-humio/nozzle"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	humioNozzle    *nozzle.HumioNozzle
	nozzleConfig   *nozzle.NozzleConfig
	firehoseClient *mocks.MockFirehoseClient
	cachingClient  *mocks.MockCaching
	logger         *mocks.MockLogger
)

var _ = Describe("Humio nozzle", func() {

	BeforeEach(func() {
		firehoseClient = mocks.NewMockFirehoseClient()
		cachingClient = &mocks.MockCaching{
			EnvironmentName: "dev",
			InstanceName:    "nozzle0",
		}
		logger = mocks.NewMockLogger()
		nozzleConfig = &nozzle.NozzleConfig{
			ExcludeMetricEvents: false,
			ExcludeLogEvents:    false,
			ExcludeHttpEvents:   false,
		}

		humioNozzle = nozzle.NewHumioNozzle(logger, firehoseClient, nozzleConfig, cachingClient)
		go humioNozzle.Start()
	})

	It("routes a LogMessage", func() {
		eventType := events.Envelope_LogMessage
		messageType := events.LogMessage_OUT

		logMessage := events.LogMessage{
			MessageType: &messageType,
		}

		envelope := &events.Envelope{
			EventType:  &eventType,
			LogMessage: &logMessage,
		}

		firehoseClient.MessageChan <- envelope

		msgJson := "[{\"EventType\":\"LogMessage\",\"Deployment\":\"\",\"Environment\":\"dev\",\"EventTime\":\"0001-01-01T00:00:00Z\",\"Job\":\"\",\"Index\":\"\",\"IP\":\"\",\"Tags\":\"\",\"NozzleInstance\":\"nozzle0\",\"MessageHash\":\"d396528c711f0053685aac71a95a9637\",\"Origin\":\"\",\"Message\":\"\",\"MessageType\":\"OUT\",\"Timestamp\":0,\"AppID\":\"\",\"ApplicationName\":\"\",\"SourceType\":\"\",\"SourceInstance\":\"\",\"SourceTypeKey\":\"-OUT\"}]"

	})

	It("routes a HttpStartStop", func() {
		eventType := events.Envelope_HttpStartStop
		peerType := events.PeerType_Client

		httpStartStop := events.HttpStartStop{
			PeerType: &peerType,
		}

		envelope := &events.Envelope{
			EventType:     &eventType,
			HttpStartStop: &httpStartStop,
		}

		firehoseClient.MessageChan <- envelope

		msgJson := "[{\"EventType\":\"HttpStartStop\",\"Deployment\":\"\",\"Environment\":\"dev\",\"EventTime\":\"0001-01-01T00:00:00Z\",\"Job\":\"\",\"Index\":\"\",\"IP\":\"\",\"Tags\":\"\",\"NozzleInstance\":\"nozzle0\",\"MessageHash\":\"b7338b4f4c40613986590b7e4ec508a9\",\"SourceInstance\":\"\",\"Origin\":\"\",\"StartTimestamp\":0,\"StopTimestamp\":0,\"RequestID\":\"\",\"PeerType\":\"Client\",\"Method\":\"GET\",\"URI\":\"\",\"RemoteAddress\":\"\",\"UserAgent\":\"\",\"StatusCode\":0,\"ContentLength\":0,\"ApplicationID\":\"\",\"ApplicationName\":\"\",\"InstanceIndex\":0,\"InstanceID\":\"\",\"Forwarded\":\"\"}]"
	})

	It("routes an Error", func() {
		eventType := events.Envelope_Error

		envelope := &events.Envelope{
			EventType: &eventType,
			Error:     &events.Error{},
		}

		firehoseClient.MessageChan <- envelope

		msgJson := "[{\"EventType\":\"Error\",\"Deployment\":\"\",\"Environment\":\"dev\",\"EventTime\":\"0001-01-01T00:00:00Z\",\"Job\":\"\",\"Index\":\"\",\"IP\":\"\",\"Tags\":\"\",\"NozzleInstance\":\"nozzle0\",\"MessageHash\":\"1aeb0d10b3411300c1ad275c668c581a\",\"SourceInstance\":\"\",\"Origin\":\"\",\"Source\":\"\",\"Code\":0,\"Message\":\"\"}]"

	})

	It("routes a ContainerMetric", func() {
		eventType := events.Envelope_ContainerMetric

		envelope := &events.Envelope{
			EventType:       &eventType,
			ContainerMetric: &events.ContainerMetric{},
		}

		firehoseClient.MessageChan <- envelope

		msgJson := "[{\"EventType\":\"ContainerMetric\",\"Deployment\":\"\",\"Environment\":\"dev\",\"EventTime\":\"0001-01-01T00:00:00Z\",\"Job\":\"\",\"Index\":\"\",\"IP\":\"\",\"Tags\":\"\",\"NozzleInstance\":\"nozzle0\",\"MessageHash\":\"7a2415d07f1304f829a5b1fc1390aa1e\",\"SourceInstance\":\"\",\"Origin\":\"\",\"ApplicationID\":\"\",\"ApplicationName\":\"\",\"InstanceIndex\":0}]"

	})

	It("routes a CounterEvent", func() {
		eventType := events.Envelope_CounterEvent

		envelope := &events.Envelope{
			EventType:    &eventType,
			CounterEvent: &events.CounterEvent{},
		}

		firehoseClient.MessageChan <- envelope

		msgJson := "[{\"EventType\":\"CounterEvent\",\"Deployment\":\"\",\"Environment\":\"dev\",\"EventTime\":\"0001-01-01T00:00:00Z\",\"Job\":\"\",\"Index\":\"\",\"IP\":\"\",\"Tags\":\"\",\"NozzleInstance\":\"nozzle0\",\"MessageHash\":\"5e28ff227b28d842fd7e08c0a764cf53\",\"SourceInstance\":\"\",\"Origin\":\"\",\"Name\":\"\",\"Delta\":0,\"Total\":0,\"CounterKey\":\"..\"}]"

	})

	It("routes a ValueMetric", func() {
		eventType := events.Envelope_ValueMetric

		envelope := &events.Envelope{
			EventType:   &eventType,
			ValueMetric: &events.ValueMetric{},
		}

		firehoseClient.MessageChan <- envelope

		msgJson := "[{\"EventType\":\"ValueMetric\",\"Deployment\":\"\",\"Environment\":\"dev\",\"EventTime\":\"0001-01-01T00:00:00Z\",\"Job\":\"\",\"Index\":\"\",\"IP\":\"\",\"Tags\":\"\",\"NozzleInstance\":\"nozzle0\",\"MessageHash\":\"cc4acf2df16fb78148a274ddc04800ca\",\"SourceInstance\":\"\",\"Origin\":\"\",\"Name\":\"\",\"Value\":0,\"Unit\":\"\",\"MetricKey\":\"..\"}]"

	})

	It("logs for unrecognized events", func() {
		eventType := events.Envelope_EventType(10)
		envelope := &events.Envelope{
			EventType: &eventType,
		}

		firehoseClient.MessageChan <- envelope

		Eventually(func() []mocks.Log {
			return logger.GetLogs(lager.INFO)
		}).Should(Equal([]mocks.Log{mocks.Log{
			Action: "uncategorized message",
			Data:   []lager.Data{{"message": "eventType:10 "}},
		}}))
	})
})

var _ = Describe("LogEventCount", func() {

	BeforeEach(func() {
		firehoseClient = mocks.NewMockFirehoseClient()
		cachingClient = &mocks.MockCaching{}
		logger = mocks.NewMockLogger()
		nozzleConfig = &nozzle.NozzleConfig{
			ExcludeMetricEvents:   false,
			ExcludeLogEvents:      false,
			ExcludeHttpEvents:     false,
			LogEventCount:         true,
			LogEventCountInterval: time.Duration(10) * time.Millisecond,
		}

		humioNozzle = nozzle.NewHumioNozzle(logger, firehoseClient, nozzleConfig, cachingClient)
		go humioNozzle.Start()
	})

	It("logs event count correctlty", func() {
		eventType := events.Envelope_ValueMetric

		envelope := &events.Envelope{
			EventType:   &eventType,
			ValueMetric: &events.ValueMetric{},
		}

		firehoseClient.MessageChan <- envelope

		eventType = events.Envelope_LogMessage
		messageType := events.LogMessage_OUT

		logMessage := events.LogMessage{
			MessageType: &messageType,
		}

		envelope = &events.Envelope{
			EventType:  &eventType,
			LogMessage: &logMessage,
		}

		firehoseClient.MessageChan <- envelope

		regExp := "\"Total\":2,\"CounterKey\":\"nozzle.stats.eventsReceived\".*\"Total\":2,\"CounterKey\":\"nozzle.stats.eventsSent\""

	})
})
