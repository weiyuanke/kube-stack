package slo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/looplab/fsm"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	slov1beta1 "kube-stack.me/apis/slo/v1beta1"
)

const (
	beginState       = "BeginState"
	endState         = "EndState"
	resourceStateKey = "ResourceStateKey"
)

type ResourceState struct {
	ctx                     context.Context
	CreateTime              time.Time
	NamespacedName          types.NamespacedName
	Resource                *unstructured.Unstructured
	FSM                     *fsm.FSM
	incoming                chan *unstructured.Unstructured
	Stopped                 bool
	stopCh                  chan struct{}
	resourceStateTransition *slov1beta1.ResourceStateTransition
	gvk                     schema.GroupVersionKind
	obj                     runtime.Object
}

// NewResourceState return resourceState of a resource
func NewResourceState(ctx context.Context, name types.NamespacedName, config *slov1beta1.ResourceStateTransition) (*ResourceState, error) {
	// group version kind resource
	gv, err := schema.ParseGroupVersion(config.Spec.Selector.APIVersion)
	if err != nil {
		return nil, err
	}

	ps := &ResourceState{
		ctx:                     ctx,
		CreateTime:              time.Now(),
		NamespacedName:          name,
		Resource:                nil,
		incoming:                make(chan *unstructured.Unstructured, 1000),
		Stopped:                 false,
		stopCh:                  make(chan struct{}),
		resourceStateTransition: config,
		gvk:                     schema.GroupKind{Group: gv.Group, Kind: config.Spec.Selector.Kind}.WithVersion(gv.Version),
	}

	obj, err := scheme.Scheme.New(ps.gvk)
	if err != nil {
		return nil, err
	}
	ps.obj = obj

	events := make([]fsm.EventDesc, 0)
	for _, tran := range config.Spec.Transitions {
		src := make([]string, 0)
		for _, s := range tran.Source {
			src = append(src, s)
		}

		events = append(events, fsm.EventDesc{
			Name: tran.Event,
			Src:  src,
			Dst:  tran.Target,
		})
	}

	enterStateFunc := func(e *fsm.Event) {
		metaVal, _ := e.FSM.Metadata(resourceStateKey)
		rs := metaVal.(*ResourceState)
		tranName := client.ObjectKeyFromObject(rs.resourceStateTransition).String()
		enterStateCounter.WithLabelValues(tranName, e.Dst).Inc()
		currentStateNum.WithLabelValues(tranName, e.Dst).Inc()
		currentStateNum.WithLabelValues(tranName, e.Src).Dec()

		if e.Dst == string(endState) {
			if !rs.Stopped {
				rs.Stopped = true
				close(rs.stopCh)
			}
		}
	}

	f := fsm.NewFSM(beginState, events, fsm.Callbacks{"enter_state": enterStateFunc})
	f.SetMetadata(resourceStateKey, ps)
	ps.FSM = f

	return ps, nil
}

// StartDispatching dispatch event
func (rs *ResourceState) StartDispatching() {
	for {
		select {
		case newResource, ok := <-rs.incoming:
			if !ok {
				return
			}

			events, _ := rs.parseEvents(rs.Resource, newResource)
			if newResource != nil {
				rs.Resource = newResource
			}

			for i := range events {
				rs.FSM.Event(string(events[i]))
			}
		case <-rs.stopCh:
			return
		}
	}
}

func (rs *ResourceState) StopDispatching() {
	if !rs.Stopped {
		rs.Stopped = true
		close(rs.stopCh)
	}
}

func (rs *ResourceState) parseEvents(old, new *unstructured.Unstructured) ([]string, error) {
	oldBytes, err := json.Marshal(old)
	if err != nil {
		return nil, err
	}

	newBytes, err := json.Marshal(new)
	if err != nil {
		return nil, err
	}

	diff, err := strategicpatch.CreateTwoWayMergePatch(oldBytes, newBytes, rs.obj)
	if err != nil {
		log.FromContext(rs.ctx).Error(err, "CreateTwoWayMergePatch")
		return nil, err
	}

	result := make([]string, 0)
	for _, event := range rs.resourceStateTransition.Spec.Events {
		if len(event.Requirements) <= 0 {
			continue
		}

		allMatch := true
		for _, match := range event.Requirements {
			if !match.Matches(string(diff)) {
				allMatch = false
			}
		}
		if allMatch {
			result = append(result, event.Name)
		}
	}
	log.FromContext(rs.ctx).Info("diff: "+string(diff), "events", result)

	return result, nil
}

// EnqueueEvent inqueue event
func (rs *ResourceState) EnqueueEvent(resource *unstructured.Unstructured) {
	if rs.Stopped {
		return
	}
	rs.incoming <- resource
}
