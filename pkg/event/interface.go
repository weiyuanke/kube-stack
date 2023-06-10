package event

import (
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	llog logr.Logger = ctrl.Log.WithName("event")
)

// Type define Event Type
type Type string

const (
	// CreateEvent Pod is created
	CreateEvent Type = "CreateEvent"
	// ScheduleEvent pod is scheduled
	ScheduleEvent Type = "ScheduleEvent"
	// DeletedEvent pod is deleted
	DeletedEvent Type = "DeletedEvent"
	// IPAllocatedEvent ip allocated for pod
	IPAllocatedEvent Type = "IpAllocatedEvent"
	// ReadyEvent pod is ready
	ReadyEvent Type = "ReadyEvent"
	// NoEvent empty event
	NoEvent Type = "NoEvent"
)

// Extracter extract event from the given string(json format)
type Extracter interface {
	ExtractEvent(data string) (Type, error)
}
