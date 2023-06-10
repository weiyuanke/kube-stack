package event

import (
	"encoding/json"

	"github.com/oliveagle/jsonpath"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
)

// CreateExtracter to extract create event
type CreateExtracter struct {
}

// ExtractEvent implement ExtractEvent interface
func (e *CreateExtracter) ExtractEvent(data string) (Type, error) {
	var jsonData interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return NoEvent, err
	}

	res, err := jsonpath.JsonPathLookup(jsonData, "$.metadata.creationTimestamp")
	if err != nil {
		return NoEvent, err
	}
	if v, ok := res.(string); ok && v != "" {
		return CreateEvent, nil
	}
	return NoEvent, nil
}

// ScheduleExtracter to extract schedule event
type ScheduleExtracter struct {
}

// ExtractEvent implement interface
func (e *ScheduleExtracter) ExtractEvent(data string) (Type, error) {
	var jsonData interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return NoEvent, err
	}

	res, err := jsonpath.JsonPathLookup(jsonData, "$.spec.nodeName")
	if err != nil {
		return NoEvent, err
	}
	if v, ok := res.(string); ok && v != "" {
		return ScheduleEvent, nil
	}
	return NoEvent, nil
}

// DeletedExtracter extract delete event
type DeletedExtracter struct {
}

// ExtractEvent implement interface
func (e *DeletedExtracter) ExtractEvent(data string) (Type, error) {
	var jsonData interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return NoEvent, err
	}

	spec, err := jsonpath.JsonPathLookup(jsonData, "$.spec")
	if err != nil {
		return NoEvent, err
	}
	status, err := jsonpath.JsonPathLookup(jsonData, "$.status")
	if spec == nil && status == nil {
		return DeletedEvent, nil
	}
	return NoEvent, nil
}

// IPAllocatedExtracter extract ip allocation event
type IPAllocatedExtracter struct {
}

// ExtractEvent implement interface
func (e *IPAllocatedExtracter) ExtractEvent(data string) (Type, error) {
	var jsonData interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return NoEvent, err
	}

	res, err := jsonpath.JsonPathLookup(jsonData, "$.status.podIP")
	if err != nil {
		return NoEvent, err
	}
	if v, ok := res.(string); ok && v != "" {
		return IPAllocatedEvent, nil
	}
	return NoEvent, nil
}

type ReadyExtractor struct {
}

// ExtractEvent implement interface
func (e *ReadyExtractor) ExtractEvent(data string) (Type, error) {
	v := gjson.Get(data, `status.conditions.#(type==Ready).status`)
	if v.Type != gjson.String || v.String() != string(corev1.ConditionTrue) {
		return NoEvent, nil
	}

	return ReadyEvent, nil
}
