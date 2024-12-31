package debugapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/northes/go-moonshot"
	corev1 "k8s.io/api/core/v1"
	corecontroller "kube-stack.me/controllers/core"
	"kube-stack.me/pkg/utils"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	llog = ctrl.Log.WithName("debugPodProcessor")
	mClient *moonshot.Client
)

func init() {
	key, _ := os.LookupEnv("MOONSHOT_KEY")
	if cli, err := moonshot.NewClient(key); err == nil {
		mClient = cli
	}
}

type debugPodProcessor struct {
}

func (p *debugPodProcessor) Process(r *http.Request) (string, error) {
	podUID := r.URL.Query().Get("uid")

	if podUID == "" {
		podName := r.URL.Query().Get("name")
		if uid, err := utils.Get(corecontroller.TABLEPODNAMEUID, podName); err == nil {
			podUID = uid
		}
	}

	if podUID != "" {
		// get podyaml
		podyaml, err := utils.Get(corecontroller.TABLEUIDPODYAML, podUID)
		if err != nil {
			return "{}", err
		}

		var pod corev1.Pod
		if err := json.Unmarshal([]byte(podyaml), &pod); err != nil {
			return "", err
		}

		// get related events
		eventUIDSet := make(map[string]struct{})
		if eventUIDs, err := utils.Get(corecontroller.TABLE_PODUID_EVENTUIDS, string(pod.UID)); err == nil {
			json.Unmarshal([]byte(eventUIDs), &eventUIDSet)
		}

		events := make([]corev1.Event, 0)
		for k := range eventUIDSet {
			if data, err := utils.Get(corecontroller.TABLE_EVENTUID_EVENTYAML, k); data != "" && err == nil {
				var e corev1.Event
				json.Unmarshal([]byte(data), &e)
				events = append(events, e)
			}
		}

		// get node info
		var node *corev1.Node
		if pod.Spec.NodeName != "" {
			node = &corev1.Node{}
			if uid, err := utils.Get(corecontroller.TABLENODENAMENODEUID, pod.Spec.NodeName); err == nil {
				if yaml, err := utils.Get(corecontroller.TABLENODEUIDNODEYAML, uid); err == nil {
					json.Unmarshal([]byte(yaml), node)
				}
			}
		}

		// get node events
		nodeEvents := make([]corev1.Event, 0)
		if node != nil {
			nodeEventUIDSet := make(map[string]struct{})
			if eventUIDs, err := utils.Get(corecontroller.TABLE_NODENAME_EVENTUIDS, string(node.Name)); err == nil {
				json.Unmarshal([]byte(eventUIDs), &nodeEventUIDSet)
			}
			for key := range nodeEventUIDSet {
				if data, err := utils.Get(corecontroller.TABLE_EVENTUID_EVENTYAML, key); data != "" && err == nil {
					var e corev1.Event
					json.Unmarshal([]byte(data), &e)
					nodeEvents = append(nodeEvents, e)
				}
			}
		}

		response := map[string]interface{}{
			"pod yaml":    pod,
			"pod events":  events,
			"node yaml":   node,
			"node events": nodeEvents,
		}

		// diagnose with llm
		builder := moonshot.NewChatCompletionsBuilder()
		systemPromt := `
			你是一个精通kubernetes的技术专家，负责帮用户排查各种各样的问题，比如Pod无法启动、创建Pod报错等;
			你的任务是帮助用户排查提交过来的各种问题，并且精准定位问题；

		`
		builder.AddSystemContent(systemPromt)
		builder.SetTemperature(0.3)
		bytes, _ := json.Marshal(response)
		preMsg := `
			基于以下提供的pod yaml, 以及对应的node yaml, pod events, node events，诊断该Pod.
			诊断结果尽可能简洁准确
		`
		builder.AddUserContent(preMsg + string(bytes))
		if resp, err := mClient.Chat().Completions(context.TODO(), builder.ToRequest()); err != nil {
			llog.Error(err, "llm failed", "resp", resp)
		} else {
			if len(resp.Choices) != 0 {
				choice := resp.Choices[0]
				response["llm result"] = strings.Split(choice.Message.Content, "\n")
			}
		}

		data, err := json.Marshal(response)
		if err != nil {
			return "", err
		}

		return string(data), nil
	}

	return "{}", nil
}
