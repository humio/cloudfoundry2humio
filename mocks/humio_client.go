package mocks

import "github.com/humio/cloudfoundry2humio/humio"
import "encoding/json"

type MockHumioClient struct {
	lastEvent string
}

func NewMockHumioClient() *MockHumioClient {
	return &MockHumioClient{}
}

func (c *MockHumioClient) PushEvents(events *humio.Events) error {
	payload, err := json.Marshal(*events)
	if err == nil {
		c.lastEvent = string(payload)
	}
	return err
}

func (c *MockHumioClient) SingleLog(log string) error {
	return nil
}

func (c *MockHumioClient) GetLastPushedEvents() string {
	return c.lastEvent
}
