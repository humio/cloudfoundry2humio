package nozzle_test

import (
	"time"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/humio/cloudfoundry2humio/mocks"
	"github.com/humio/cloudfoundry2humio/nozzle"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	humioNozzle    *nozzle.HumioNozzle
	nozzleConfig   *nozzle.NozzleConfig
	firehoseClient *mocks.MockFirehoseClient
	humioClient    *mocks.MockHumioClient
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
			HumioBatchTime:         5 * time.Millisecond,
			HumioMaxMsgNumPerBatch: 1,
		}
		humioClient = mocks.NewMockHumioClient()

		humioNozzle = nozzle.NewHumioNozzle(logger, firehoseClient, nozzleConfig, humioClient, cachingClient)
		go humioNozzle.Start()
	})

	It("routes a LogMessage", func() {
		eventType := events.Envelope_LogMessage
		messageType := events.LogMessage_OUT

		var t0 int64
		t0 = 1 * 1000000000

		logMessage := events.LogMessage{
			MessageType: &messageType,
			Timestamp:   &t0,
		}

		envelope := &events.Envelope{
			EventType:  &eventType,
			LogMessage: &logMessage,
		}

		firehoseClient.MessageChan <- envelope

		msgJson := `{"tags":{},"events":[{"timestamp":"1970-01-01T01:00:00+01:00","attributes":{"eventtype":"LogMessage","timestamp":"1970-01-01T01:00:00+01:00","deployment":"","env":"dev","job":"","index":"","instance":"nozzle0","org":{},"space":{},"app":{},"http":{"starttimestamp":"","stoptimestamp":"","requestid":"","peertype":"","method":"","uri":"","remoteaddr":"","ua":"","statuscode":0,"contentlength":0,"instanceindex":0,"instanceid":"","forwarded":""},"log":{"message":"","messagetype":"OUT","timestamp":"1970-01-01T01:00:01+01:00","sourcetype":"","sourceinst":"","sourcetypekey":"-OUT"}}}]}`
		Eventually(func() string {
			return humioClient.GetLastPushedEvents()
		}).Should(Equal(msgJson))
	})

	It("routes a HttpStartStop", func() {
		eventType := events.Envelope_HttpStartStop
		peerType := events.PeerType_Client

		var t0 int64
		t0 = 1 * 1000000000

		var t1 int64
		t1 = 2 * 1000000000

		httpStartStop := events.HttpStartStop{
			PeerType:       &peerType,
			StartTimestamp: &t0,
			StopTimestamp:  &t1,
		}

		envelope := &events.Envelope{
			EventType:     &eventType,
			HttpStartStop: &httpStartStop,
		}

		firehoseClient.MessageChan <- envelope

		msgJson := `{"tags":{},"events":[{"timestamp":"1970-01-01T01:00:00+01:00","attributes":{"eventtype":"HttpStartStop","timestamp":"1970-01-01T01:00:00+01:00","deployment":"","env":"dev","job":"","index":"","instance":"nozzle0","org":{},"space":{},"app":{},"http":{"starttimestamp":"1970-01-01T01:00:01+01:00","stoptimestamp":"1970-01-01T01:00:02+01:00","requestid":"","peertype":"Client","method":"GET","uri":"","remoteaddr":"","ua":"","statuscode":0,"contentlength":0,"instanceindex":0,"instanceid":"","forwarded":""},"log":{"message":"","messagetype":"","timestamp":"","sourcetype":"","sourceinst":"","sourcetypekey":""}}}]}`
		Eventually(func() string {
			return humioClient.GetLastPushedEvents()
		}).Should(Equal(msgJson))
	})
})
