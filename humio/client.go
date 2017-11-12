package humio

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/parnurzeal/gorequest"
)

type HumioClient interface {
	PushEvents(*Events) error
	SingleLog(string) error
}

type client struct {
	config          HumioConfig
	httpPostTimeout time.Duration
	logger          lager.Logger
}

type HumioConfig struct {
	Host      string
	Dataspace string
	Token     string
}

func NewHumioClient(humioConfig *HumioConfig, logger lager.Logger) HumioClient {
	return &client{
		config:          *humioConfig,
		httpPostTimeout: 10,
		logger:          logger,
	}
}

func (c *client) PushEvents(events *Events) error {
	url := c.config.Host + "/api/v1/dataspaces/" + c.config.Dataspace + "/ingest"
	request := gorequest.New() // .Timeout(2*time.Millisecond)
	resp, body, errs := request.Post(url).
		Set("Authorization", "Bearer "+c.config.Token).
		Send([]Events{*events}).
		End()

	if errs != nil {
		c.logger.Fatal("failed pushing events to Humio", errs[0])
		return errs[0]
	}

	if resp.StatusCode != 200 {
		c.logger.Error("Humio returned an unexpected response: "+body, nil)
	}

	return nil
}

func (c *client) SingleLog(log string) error {
	url := c.config.Host + "/api/v1/dataspaces/" + c.config.Dataspace + "/ingest"
	request := gorequest.New()
	resp, body, errs := request.Post(url).
		Set("Authorization", "Bearer "+c.config.Token).
		Send(`[{"tags": {"source": "humio-nozzle", "job": "nozzle"}, "events": [{"attributes": {"log": "` + log + `"}, "timestamp": "` + time.Now().Format(time.RFC3339) + `"}]}]`).
		End()

	if errs != nil {
		c.logger.Fatal("failed sending single log event to Humio", errs[0])
		return errs[0]
	}

	if resp.StatusCode != 200 {
		c.logger.Error("Humio returned an unexpected response: "+body, nil)
	}

	return nil
}
