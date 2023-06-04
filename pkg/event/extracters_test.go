package event

import (
	"testing"
)

func TestScheduleExtracter_ExtractEvent(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		e       *ScheduleExtracter
		args    args
		want    EventType
		wantErr bool
	}{
		{
			e: &ScheduleExtracter{},
			args: args{
				data: `{"spec":{"nodeName":"testnode"}}`,
			},
			want:    ScheduleEvent,
			wantErr: false,
		},
		{
			e: &ScheduleExtracter{},
			args: args{
				data: `{"spec":{"nodeName":""}}`,
			},
			want:    NoEvent,
			wantErr: false,
		},
		{
			e: &ScheduleExtracter{},
			args: args{
				data: "{}",
			},
			want:    NoEvent,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ScheduleExtracter{}
			got, err := e.ExtractEvent(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScheduleExtracter.ExtractEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ScheduleExtracter.ExtractEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateExtracter_ExtractEvent(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		e       *CreateExtracter
		args    args
		want    EventType
		wantErr bool
	}{
		{
			e: &CreateExtracter{},
			args: args{
				data: `{}`,
			},
			want:    NoEvent,
			wantErr: true,
		},
		{
			e: &CreateExtracter{},
			args: args{
				data: `{"metadata":{"creationTimestamp":"2022-11-25T12:07:06Z"}}`,
			},
			want:    CreateEvent,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &CreateExtracter{}
			got, err := e.ExtractEvent(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateExtracter.ExtractEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CreateExtracter.ExtractEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeletedExtracter_ExtractEvent(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		e       *DeletedExtracter
		args    args
		want    EventType
		wantErr bool
	}{
		{
			e: &DeletedExtracter{},
			args: args{
				data: `{"spec":null}`,
			},
			want:    DeletedEvent,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &DeletedExtracter{}
			got, err := e.ExtractEvent(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeletedExtracter.ExtractEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeletedExtracter.ExtractEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIpAllocatedExtracter_ExtractEvent(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		e       *IpAllocatedExtracter
		args    args
		want    EventType
		wantErr bool
	}{
		{
			e: &IpAllocatedExtracter{},
			args: args{
				data: `{"status":{"podIP":"10.244.0.4"}}`,
			},
			want:    IpAllocatedEvent,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &IpAllocatedExtracter{}
			got, err := e.ExtractEvent(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("IpAllocatedExtracter.ExtractEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IpAllocatedExtracter.ExtractEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadyExtractor_ExtractEvent(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		e       *ReadyExtractor
		args    args
		want    EventType
		wantErr bool
	}{
		{
			e: &ReadyExtractor{},
			args: args{
				data: `{"status":{"$setElementOrder/conditions":[{"type":"Ready"}],"conditions":[{"status":"True","type":"Ready"}]}}`,
			},
			want:    ReadyEvent,
			wantErr: false,
		},
		{
			e: &ReadyExtractor{},
			args: args{
				data: `{"status":{"conditions":[{"status":"True","type":"Ready"}]}}`,
			},
			want:    ReadyEvent,
			wantErr: false,
		},
		{
			e: &ReadyExtractor{},
			args: args{
				data: `{}`,
			},
			want:    NoEvent,
			wantErr: false,
		},
		{
			e: &ReadyExtractor{},
			args: args{
				data: `{"status":{"$setElementOrder/conditions":[{"type":"Ready"}],"conditions":[{"status":"False","type":"Ready"}]}}`,
			},
			want:    NoEvent,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ReadyExtractor{}
			got, err := e.ExtractEvent(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadyExtractor.ExtractEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadyExtractor.ExtractEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}
