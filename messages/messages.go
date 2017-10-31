package messages

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	hex "encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Humio/cf-firehose-to-humio/caching"
	events "github.com/cloudfoundry/sonde-go/events"
)

// BaseMessage contains common data elements
type BaseMessage struct {
	EventType      string
	Deployment     string
	Environment    string
	EventTime      time.Time
	Job            string
	Index          string
	IP             string
	Tags           string
	NozzleInstance string
	MessageHash    string
}

// NewBaseMessage Creates the common attributes of messages
func NewBaseMessage(e *events.Envelope, c caching.CachingClient) *BaseMessage {
	var b = BaseMessage{
		EventType:      e.GetEventType().String(),
		Deployment:     e.GetDeployment(),
		Environment:    c.GetEnvironmentName(),
		Job:            e.GetJob(),
		Index:          e.GetIndex(),
		IP:             e.GetIp(),
		NozzleInstance: c.GetInstanceName(),
	}
	if e.Timestamp != nil {
		b.EventTime = time.Unix(0, *e.Timestamp)
	}
	if e.Origin != nil {
		b.Origin = e.GetOrigin()
	}
	if e.Deployment != nil && e.Job != nil && e.Index != nil {
		b.SourceInstance = fmt.Sprintf("%s.%s.%s", e.GetDeployment(), e.GetJob(), e.GetIndex())
	}

	if e.GetTags() != nil {
		b.Tags = fmt.Sprintf("%v", e.GetTags())
	}
	// String() returns string from underlying protobuf message
	var hash = md5.Sum([]byte(e.String()))
	b.MessageHash = hex.EncodeToString(hash[:])

	return &b
}

// An HTTPStartStop event represents the whole lifecycle of an HTTP request.
type HTTPStartStop struct {
	BaseMessage
	StartTimestamp     int64
	StopTimestamp      int64
	RequestID          string
	PeerType           string // Client/Server
	Method             string // HTTP method
	URI                string
	RemoteAddress      string
	UserAgent          string
	StatusCode         int32
	ContentLength      int64
	ApplicationID      string
	ApplicationName    string
	ApplicationOrg     string
	ApplicationOrgID   string
	ApplicationSpace   string
	ApplicationSpaceID string
	InstanceIndex      int32
	InstanceID         string
	Forwarded          string
}

// NewHTTPStartStop creates a new NewHTTPStartStop
func NewHTTPStartStop(e *events.Envelope, c caching.CachingClient) *HTTPStartStop {
	var m = e.GetHttpStartStop()
	var r = HTTPStartStop{
		BaseMessage:    *NewBaseMessage(e, c),
		StartTimestamp: m.GetStartTimestamp(),
		StopTimestamp:  m.GetStopTimestamp(),
		PeerType:       m.GetPeerType().String(), // Client/Server
		Method:         m.GetMethod().String(),   // HTTP method
		URI:            m.GetUri(),
		RemoteAddress:  m.GetRemoteAddress(),
		UserAgent:      m.GetUserAgent(),
		StatusCode:     m.GetStatusCode(),
		ContentLength:  m.GetContentLength(),
		InstanceIndex:  m.GetInstanceIndex(),
		InstanceID:     m.GetInstanceId(),
	}
	if m.RequestId != nil {
		r.RequestID = cfUUIDToString(m.RequestId)
	}
	if m.ApplicationId != nil {
		id := cfUUIDToString(m.ApplicationId)
		r.ApplicationID = id
		var appInfo = c.GetAppInfo(id)
		r.ApplicationName = appInfo.Name
		r.ApplicationOrg = appInfo.Org
		r.ApplicationOrgID = appInfo.OrgID
		r.ApplicationSpace = appInfo.Space
		r.ApplicationSpaceID = appInfo.SpaceID
	}
	if e.HttpStartStop.GetForwarded() != nil {
		r.Forwarded = strings.Join(e.GetHttpStartStop().GetForwarded(), ",")
	}
	return &r
}

//A LogMessage contains a "log line" and associated metadata.
type LogMessage struct {
	BaseMessage
	Message            string
	MessageType        string // OUT or ERROR
	Timestamp          int64
	AppID              string
	ApplicationName    string
	ApplicationOrg     string
	ApplicationOrgID   string
	ApplicationSpace   string
	ApplicationSpaceID string
	SourceType         string // APP,RTR,DEA,STG,etc
	SourceInstance     string
	SourceTypeKey      string // Key for aggregation until multiple levels of grouping supported
}

// NewLogMessage creates a new NewLogMessage
func NewLogMessage(e *events.Envelope, c caching.CachingClient) *LogMessage {
	var m = e.GetLogMessage()
	var r = LogMessage{
		BaseMessage:    *NewBaseMessage(e, c),
		Timestamp:      m.GetTimestamp(),
		AppID:          m.GetAppId(),
		SourceType:     m.GetSourceType(),
		SourceInstance: m.GetSourceInstance(),
	}
	if m.Message != nil {
		r.Message = string(m.GetMessage())
	}
	if m.MessageType != nil {
		r.MessageType = m.MessageType.String()
		r.SourceTypeKey = r.SourceType + "-" + r.MessageType
	}
	if m.AppId != nil {
		var appInfo = c.GetAppInfo(*m.AppId)
		r.ApplicationName = appInfo.Name
		r.ApplicationOrg = appInfo.Org
		r.ApplicationOrgID = appInfo.OrgID
		r.ApplicationSpace = appInfo.Space
		r.ApplicationSpaceID = appInfo.SpaceID
	}
	return &r
}

// An Error event represents an error in the originating process.
type Error struct {
	BaseMessage
	Source  string
	Code    int32
	Message string
}

// NewError creates a new NewError
func NewError(e *events.Envelope, c caching.CachingClient) *Error {
	return &Error{
		BaseMessage: *NewBaseMessage(e, c),
		Source:      e.Error.GetSource(),
		Code:        e.Error.GetCode(),
		Message:     e.Error.GetMessage(),
	}
}

func cfUUIDToString(uuid *events.UUID) string {
	lowBytes := new(bytes.Buffer)
	binary.Write(lowBytes, binary.LittleEndian, uuid.Low)
	highBytes := new(bytes.Buffer)
	binary.Write(highBytes, binary.LittleEndian, uuid.High)
	return fmt.Sprintf("%x-%x-%x-%x-%x", lowBytes.Bytes()[0:4], lowBytes.Bytes()[4:6], lowBytes.Bytes()[6:8], highBytes.Bytes()[0:2], highBytes.Bytes()[2:])
}
