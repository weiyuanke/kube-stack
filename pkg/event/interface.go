package event

import (
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	llog logr.Logger = ctrl.Log.WithName("event")
)

type EventType string

const (
	CreateEvent      EventType = "CreateEvent"
	ScheduleEvent    EventType = "ScheduleEvent"
	DeletedEvent     EventType = "DeletedEvent"
	IpAllocatedEvent EventType = "IpAllocatedEvent"
	ReadyEvent       EventType = "ReadyEvent"
	NoEvent          EventType = "NoEvent"
)

type EventExtracter interface {
	ExtractEvent(data string) (EventType, error)
}
