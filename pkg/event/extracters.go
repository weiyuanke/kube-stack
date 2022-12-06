package event

import (
	"encoding/json"

	"github.com/oliveagle/jsonpath"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
)

type CreateExtracter struct {
}

func (e *CreateExtracter) ExtractEvent(data string) (EventType, error) {
	var json_data interface{}
	if err := json.Unmarshal([]byte(data), &json_data); err != nil {
		return NoEvent, err
	}

	res, err := jsonpath.JsonPathLookup(json_data, "$.metadata.creationTimestamp")
	if err != nil {
		return NoEvent, err
	}
	if v, ok := res.(string); ok && v != "" {
		return CreateEvent, nil
	}
	return NoEvent, nil
}

type ScheduleExtracter struct {
}

func (e *ScheduleExtracter) ExtractEvent(data string) (EventType, error) {
	var json_data interface{}
	if err := json.Unmarshal([]byte(data), &json_data); err != nil {
		return NoEvent, err
	}

	res, err := jsonpath.JsonPathLookup(json_data, "$.spec.nodeName")
	if err != nil {
		return NoEvent, err
	}
	if v, ok := res.(string); ok && v != "" {
		return ScheduleEvent, nil
	}
	return NoEvent, nil
}

type DeletedExtracter struct {
}

func (e *DeletedExtracter) ExtractEvent(data string) (EventType, error) {
	var json_data interface{}
	if err := json.Unmarshal([]byte(data), &json_data); err != nil {
		return NoEvent, err
	}

	spec, err := jsonpath.JsonPathLookup(json_data, "$.spec")
	if err != nil {
		return NoEvent, err
	}
	status, err := jsonpath.JsonPathLookup(json_data, "$.status")
	if spec == nil && status == nil {
		return DeletedEvent, nil
	}
	return NoEvent, nil
}

type IpAllocatedExtracter struct {
}

func (e *IpAllocatedExtracter) ExtractEvent(data string) (EventType, error) {
	var json_data interface{}
	if err := json.Unmarshal([]byte(data), &json_data); err != nil {
		return NoEvent, err
	}

	res, err := jsonpath.JsonPathLookup(json_data, "$.status.podIP")
	if err != nil {
		return NoEvent, err
	}
	if v, ok := res.(string); ok && v != "" {
		return IpAllocatedEvent, nil
	}
	return NoEvent, nil
}

type ReadyExtractor struct {
}

func (e *ReadyExtractor) ExtractEvent(data string) (EventType, error) {
	v := gjson.Get(data, `status.conditions.#(type==Ready).status`)
	if v.Type != gjson.String || v.String() != string(corev1.ConditionTrue) {
		return NoEvent, nil
	}

	return ReadyEvent, nil
}
