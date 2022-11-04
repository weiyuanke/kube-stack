package utils

import (
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type NonLeaderController struct {
	controller.Controller
}

func (*NonLeaderController) NeedLeaderElection() bool {
	return false
}

func NewNonLeaderController(name string, mgr manager.Manager, options controller.Options) (*NonLeaderController, error) {
	if c, err := controller.NewUnmanaged(name, mgr, options); err != nil {
		return nil, err
	} else {
		return &NonLeaderController{
			c,
		}, nil
	}
}
