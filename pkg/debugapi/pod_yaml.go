package debugapi

import (
	"net/http"

	corecontroller "kube-stack.me/controllers/core"
	"kube-stack.me/pkg/utils"
)

type podYamlProcessor struct {
}

func (p *podYamlProcessor) Process(r *http.Request) (string, error) {
	podName := r.URL.Query().Get("name")
	podUID := r.URL.Query().Get("uid")

	if podUID != "" {
		podyaml, err := utils.Get(corecontroller.TABLEUIDPODYAML, podUID)
		if err != nil {
			return "{}", err
		}
		return podyaml, nil
	}

	if podName != "" {
		uid, err := utils.Get(corecontroller.TABLEPODNAMEUID, podName)
		if err != nil {
			return "{}", err
		}
		podyaml, err := utils.Get(corecontroller.TABLEUIDPODYAML, uid)
		if err != nil {
			return "", err
		}
		return podyaml, nil
	}

	return "{}", nil
}
