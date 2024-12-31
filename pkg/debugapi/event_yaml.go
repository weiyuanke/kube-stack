package debugapi

import (
	"net/http"

	corecontroller "kube-stack.me/controllers/core"
	"kube-stack.me/pkg/utils"
)

type eventYamlProcessor struct {
}

func (p *eventYamlProcessor) Process(r *http.Request) (string, error) {
	eventUID := r.URL.Query().Get("uid")

	if eventUID != "" {
		eventYaml, err := utils.Get(corecontroller.TABLE_EVENTUID_EVENTYAML, eventUID)
		if err != nil {
			return "{}", err
		}
		return eventYaml, nil
	}

	return "{}", nil
}
