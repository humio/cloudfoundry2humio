package nozzle

import (
	"crypto/tls"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/noaa/consumer"
	events "github.com/cloudfoundry/sonde-go/events"
)

type FirehoseClient interface {
	Connect() (<-chan *events.Envelope, <-chan error)
	CloseConsumer() error
}

type client struct {
	cfClientConfig *cfclient.Config
	firehoseConfig *FirehoseConfig
	logger         lager.Logger
	consumer       *consumer.Consumer
}

type FirehoseConfig struct {
	SubscriptionId       string
	TrafficControllerUrl string
	IdleTimeout          time.Duration
}

type CfClientTokenRefresh struct {
	cfClient *cfclient.Client
	logger   lager.Logger
}

func (ct *CfClientTokenRefresh) RefreshAuthToken() (string, error) {
	token, err := ct.cfClient.GetToken()
	if err != nil {
		ct.logger.Error("cannot retrieve CF token", err)
		return "", err
	}
	return token, err
}

func NewFirehoseClient(cfClientConfig *cfclient.Config, firehoseConfig *FirehoseConfig, logger lager.Logger) FirehoseClient {
	return &client{
		cfClientConfig: cfClientConfig,
		firehoseConfig: firehoseConfig,
		logger:         logger,
	}
}

func (c *client) Connect() (<-chan *events.Envelope, <-chan error) {
	c.logger.Debug("connect", lager.Data{"dopplerAddress": c.firehoseConfig.TrafficControllerUrl})
	cfClient, err := cfclient.NewClient(c.cfClientConfig)
	if err != nil {
		c.logger.Fatal("error creating cfclient", err)
	}

	c.consumer = consumer.New(
		c.firehoseConfig.TrafficControllerUrl,
		&tls.Config{InsecureSkipVerify: c.cfClientConfig.SkipSslValidation},
		nil)

	refresher := CfClientTokenRefresh{cfClient: cfClient, logger: c.logger}
	c.consumer.RefreshTokenFrom(&refresher)
	c.consumer.SetIdleTimeout(c.firehoseConfig.IdleTimeout)
	return c.consumer.FilteredFirehose(c.firehoseConfig.SubscriptionId, "", consumer.LogMessages)
}

func (c *client) CloseConsumer() error {
	return c.consumer.Close()
}
