package pod

import (
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"github.com/looplab/fsm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	ctrl "sigs.k8s.io/controller-runtime"

	"kube-stack.me/pkg/event"
)

type podStateType string

const (
	beginState       podStateType = "BeginState"
	createdState     podStateType = "CreatedState"
	scheduledState   podStateType = "ScheduledState"
	ipAllocatedState podStateType = "IpAllocatedState"
	readyState       podStateType = "ReadyState"
	deletedState     podStateType = "DeletedState"
)

var (
	llog       logr.Logger            = ctrl.Log.WithName("watch.go")
	extracters []event.EventExtracter = []event.EventExtracter{
		&event.CreateExtracter{},
		&event.ScheduleExtracter{},
		&event.IpAllocatedExtracter{},
		&event.ReadyExtractor{},
		&event.DeletedExtracter{},
	}
)

type PodState struct {
	CreateTime     time.Time
	NamespacedName types.NamespacedName
	Pod            *corev1.Pod
	FSM            *fsm.FSM
	incoming       chan *corev1.Pod
	Stopped        bool
	stopCh         chan struct{}
}

func NewPodState(name types.NamespacedName) *PodState {
	ps := &PodState{
		CreateTime:     time.Now(),
		NamespacedName: name,
		Pod:            nil,
		incoming:       make(chan *corev1.Pod, 1000),
		Stopped:        false,
		stopCh:         make(chan struct{}),
	}

	enterStateFunc := func(e *fsm.Event) {
		enterStateCounter.WithLabelValues(e.Dst, ps.NamespacedName.Namespace).Inc()
		currentStateNum.WithLabelValues(e.Dst, ps.NamespacedName.Namespace).Inc()
		currentStateNum.WithLabelValues(e.Src, ps.NamespacedName.Namespace).Dec()

		if e.Dst == string(deletedState) {
			if !ps.Stopped {
				ps.Stopped = true
				close(ps.stopCh)
			}
		}
	}

	f := fsm.NewFSM(
		string(beginState),
		fsm.Events{
			{
				Name: string(event.CreateEvent),
				Src:  []string{string(beginState)},
				Dst:  string(createdState),
			},
			{
				Name: string(event.ScheduleEvent),
				Src:  []string{string(beginState), string(createdState)},
				Dst:  string(scheduledState),
			},
			{
				Name: string(event.IpAllocatedEvent),
				Src:  []string{string(scheduledState)},
				Dst:  string(ipAllocatedState),
			},
			{
				Name: string(event.ReadyEvent),
				Src: []string{
					string(beginState),
					string(createdState),
					string(scheduledState),
					string(ipAllocatedState),
				},
				Dst: string(readyState),
			},
			{
				Name: string(event.DeletedEvent),
				Src: []string{
					string(beginState),
					string(createdState),
					string(scheduledState),
					string(ipAllocatedState),
					string(readyState),
				},
				Dst: string(deletedState),
			},
		},
		fsm.Callbacks{
			"enter_state": enterStateFunc,
		},
	)

	ps.FSM = f

	return ps
}

func (ps *PodState) StopDispatching() {
	if !ps.Stopped {
		ps.Stopped = true
		close(ps.stopCh)
	}
}

// StartDispatching dispatch pod event
func (ps *PodState) StartDispatching() {
	go func() {
		for {
			select {
			case pod, ok := <-ps.incoming:
				if !ok {
					return
				}

				events, err := parseEvents(ps.Pod, pod)
				if err != nil {
					llog.Error(err, "PraseEvents Error")
					return
				}
				if pod != nil {
					ps.Pod = pod
				}

				for i := range events {
					ps.FSM.Event(string(events[i]))
				}
			case <-ps.stopCh:
				return
			}
		}
	}()
}

// ProcessEvent inqueue pod event
func (ps *PodState) ProcessEvent(pod *corev1.Pod) {
	if ps.Stopped {
		return
	}
	ps.incoming <- pod
}

func parseEvents(old *corev1.Pod, new *corev1.Pod) ([]event.EventType, error) {
	oldBytes, err := json.Marshal(old)
	if err != nil {
		return nil, err
	}

	newBytes, err := json.Marshal(new)
	if err != nil {
		return nil, err
	}

	diff, err := strategicpatch.CreateTwoWayMergePatch(oldBytes, newBytes, corev1.Pod{})
	if err != nil {
		return nil, err
	}

	result := make([]event.EventType, 0)
	for i := range extracters {
		if e, err := extracters[i].ExtractEvent(string(diff)); err == nil && e != event.NoEvent {
			result = append(result, e)
		}
	}

	return result, nil
}
