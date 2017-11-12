package humio

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/cloudfoundry/sonde-go/events"
)

func cfUUIDToString(uuid *events.UUID) string {
	lowBytes := new(bytes.Buffer)
	binary.Write(lowBytes, binary.LittleEndian, uuid.Low)
	highBytes := new(bytes.Buffer)
	binary.Write(highBytes, binary.LittleEndian, uuid.High)
	return fmt.Sprintf("%x-%x-%x-%x-%x", lowBytes.Bytes()[0:4], lowBytes.Bytes()[4:6], lowBytes.Bytes()[6:8], highBytes.Bytes()[0:2], highBytes.Bytes()[2:])
}
