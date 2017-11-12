package humio

import (
	"math/big"
	"strings"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/humio/cloudfoundry2humio/caching"
)

type Tags struct {
	OrgName   string `json:"orgname,omitempty"`
	OrgID     string `json:"orgid,omitempty"`
	SpaceName string `json:"spacename,omitempty"`
	SpaceID   string `json:"spaceid,omitempty"`
	AppName   string `json:"appname,omitempty"`
	AppID     string `json:"appid,omitempty"`
}

type OrganizationAttribute struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type SpaceAttribute struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type ApplicationAttribute struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type LogAttribute struct {
	Message        string `json:"message"`
	MessageType    string `json:"messagetype"`
	Timestamp      string `json:"timestamp"`
	SourceType     string `json:"sourcetype"`
	SourceInstance string `json:"sourceinst"`
	SourceTypeKey  string `json:"sourcetypekey"`
}

type HTTPAttribute struct {
	StartTimestamp string `json:"starttimestamp"`
	StopTimestamp  string `json:"stoptimestamp"`
	RequestID      string `json:"requestid"`
	PeerType       string `json:"peertype"`
	Method         string `json:"method"`
	URI            string `json:"uri"`
	RemoteAddress  string `json:"remoteaddr"`
	UserAgent      string `json:"ua"`
	StatusCode     int32  `json:"statuscode"`
	ContentLength  int64  `json:"contentlength"`
	InstanceIndex  int32  `json:"instanceindex"`
	InstanceID     string `json:"instanceid"`
	Forwarded      string `json:"forwarded"`
}

type Attributes struct {
	EventType      string                `json:"eventtype"`
	EventTime      string                `json:"timestamp"`
	Deployment     string                `json:"deployment"`
	Environment    string                `json:"env"`
	Job            string                `json:"job"`
	Index          string                `json:"index"`
	IP             string                `json:"ip,omitempty"`
	Tags           map[string]string     `json:"tags,omitempty"`
	NozzleInstance string                `json:"instance"`
	Org            OrganizationAttribute `json:"org,omitempty"`
	Space          SpaceAttribute        `json:"space,omitempty"`
	App            ApplicationAttribute  `json:"app,omitempty"`
	HTTP           HTTPAttribute         `json:"http,omitempty"`
	Log            LogAttribute          `json:"log,omitempty"`
}

type Event struct {
	Timestamp  string     `json:"timestamp"`
	Attributes Attributes `json:"attributes"`
}

type Events struct {
	Tags   Tags    `json:"tags"`
	Events []Event `json:"events"`
}

func NewEvent(e *events.Envelope, c caching.CachingClient) *Event {
	var timestamp = formatTimestamp(e.GetTimestamp())
	var eventType = e.GetEventType()

	var a = Attributes{
		EventType:      eventType.String(),
		EventTime:      timestamp,
		Deployment:     e.GetDeployment(),
		Environment:    c.GetEnvironmentName(),
		Job:            e.GetJob(),
		Index:          e.GetIndex(),
		IP:             e.GetIp(),
		Tags:           e.GetTags(),
		NozzleInstance: c.GetInstanceName(),
	}

	switch t := eventType; t {
	case events.Envelope_LogMessage:
		AddLogMessageAttributes(&a, e, c)
	case events.Envelope_HttpStartStop:
		AddHTTPStartStopAttributes(&a, e, c)
	default:
		return nil
	}

	var ev = Event{
		Timestamp:  timestamp,
		Attributes: a,
	}

	return &ev
}

func AddLogMessageAttributes(a *Attributes, e *events.Envelope, c caching.CachingClient) {
	var m = e.GetLogMessage()

	var l = LogAttribute{
		Timestamp:      formatTimestamp(m.GetTimestamp()),
		SourceType:     m.GetSourceType(),
		SourceInstance: m.GetSourceInstance(),
	}

	if m.Message != nil {
		l.Message = string(m.GetMessage())
	}

	if m.MessageType != nil {
		l.MessageType = m.MessageType.String()
		l.SourceTypeKey = l.SourceType + "-" + l.MessageType
	}

	if m.AppId != nil {
		var appInfo = c.GetAppInfo(*m.AppId)

		var org = OrganizationAttribute{
			ID:   appInfo.OrgID,
			Name: appInfo.Org,
		}
		a.Org = org

		var space = SpaceAttribute{
			ID:   appInfo.SpaceID,
			Name: appInfo.Space,
		}
		a.Space = space

		var app = ApplicationAttribute{
			ID:   *m.AppId,
			Name: appInfo.Name,
		}
		a.App = app
	}

	a.Log = l
}

func AddHTTPStartStopAttributes(a *Attributes, e *events.Envelope, c caching.CachingClient) {
	var m = e.GetHttpStartStop()

	var h = HTTPAttribute{
		StartTimestamp: formatTimestamp(m.GetStartTimestamp()),
		StopTimestamp:  formatTimestamp(m.GetStopTimestamp()),
		PeerType:       m.GetPeerType().String(),
		Method:         m.GetMethod().String(),
		URI:            m.GetUri(),
		RemoteAddress:  m.GetRemoteAddress(),
		UserAgent:      m.GetUserAgent(),
		StatusCode:     m.GetStatusCode(),
		ContentLength:  m.GetContentLength(),
		InstanceIndex:  m.GetInstanceIndex(),
		InstanceID:     m.GetInstanceId(),
	}

	if m.RequestId != nil {
		h.RequestID = cfUUIDToString(m.RequestId)
	}

	if e.HttpStartStop.GetForwarded() != nil {
		h.Forwarded = strings.Join(e.GetHttpStartStop().GetForwarded(), ",")
	}

	if m.ApplicationId != nil {
		id := cfUUIDToString(m.ApplicationId)
		var appInfo = c.GetAppInfo(id)

		var org = OrganizationAttribute{
			ID:   appInfo.OrgID,
			Name: appInfo.Org,
		}
		a.Org = org

		var space = SpaceAttribute{
			ID:   appInfo.SpaceID,
			Name: appInfo.Space,
		}
		a.Space = space

		var app = ApplicationAttribute{
			ID:   id,
			Name: appInfo.Name,
		}
		a.App = app
	}

	a.HTTP = h
}

func formatTimestamp(ts int64) string {
	q, r := new(big.Int).DivMod(big.NewInt(ts), big.NewInt(1000000000), new(big.Int))
	t := time.Unix(q.Int64(), r.Int64())
	return t.Format(time.RFC3339)
}
