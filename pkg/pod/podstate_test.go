package pod

import (
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kube-stack.me/pkg/event"
)

func TestParseEvents(t *testing.T) {
	type args struct {
		old *corev1.Pod
		new *corev1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    []event.Type
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			args: args{
				old: &corev1.Pod{},
				new: &corev1.Pod{
					Spec: corev1.PodSpec{
						NodeName: "testNode",
					},
				},
			},
			want:    []event.Type{event.ScheduleEvent},
			wantErr: false,
		},
		{
			args: args{
				old: &corev1.Pod{},
				new: &corev1.Pod{
					ObjectMeta: v1.ObjectMeta{
						CreationTimestamp: v1.NewTime(time.Now()),
					},
					Spec: corev1.PodSpec{
						NodeName: "testNode",
					},
				},
			},
			want:    []event.Type{event.CreateEvent, event.ScheduleEvent},
			wantErr: false,
		},
		{
			args: args{
				old: &corev1.Pod{},
				new: &corev1.Pod{
					ObjectMeta: v1.ObjectMeta{
						CreationTimestamp: v1.NewTime(time.Now()),
					},
				},
			},
			want:    []event.Type{event.CreateEvent},
			wantErr: false,
		},
		{
			args: args{
				old: &corev1.Pod{
					ObjectMeta: v1.ObjectMeta{
						CreationTimestamp: v1.NewTime(time.Now()),
					},
				},
				new: &corev1.Pod{
					ObjectMeta: v1.ObjectMeta{
						CreationTimestamp: v1.NewTime(time.Now()),
					},
					Spec: corev1.PodSpec{
						NodeName: "testNode",
					},
				},
			},
			want:    []event.Type{event.ScheduleEvent},
			wantErr: false,
		},
		{
			args: args{
				old: &corev1.Pod{},
				new: nil,
			},
			want:    []event.Type{event.DeletedEvent},
			wantErr: false,
		},
		{
			args: args{
				old: &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
				new: &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			want:    []event.Type{event.ReadyEvent},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEvents(tt.args.old, tt.args.new)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseEvents() = %v, want %v", got, tt.want)
			}
		})
	}
}
