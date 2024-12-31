package debugapi

import (
	"net/http"

	corecontroller "kube-stack.me/controllers/core"
	"kube-stack.me/pkg/utils"
)

type nodeYamlProcessor struct {
}

func (p *nodeYamlProcessor) Process(r *http.Request) (string, error) {
	nodeUID := r.URL.Query().Get("uid")

	if nodeUID == "" {
		nodeName := r.URL.Query().Get("name")
		if uid, err := utils.Get(corecontroller.TABLENODENAMENODEUID, nodeName); err == nil {
			nodeUID = uid
		}
	}

	if nodeUID != "" {
		if data, err := utils.Get(corecontroller.TABLENODEUIDNODEYAML, nodeUID); err == nil {
			return data, nil
		}
	}

	return "{}", nil
}