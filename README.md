# kube-stack
// TODO(user): Add simple overview of use/purpose

## Description
### CentralProbe
Update Pod Readiness Status when Node is Not ready;
example:
```
---
apiVersion: centralprobe.kube-stack.me/v1
kind: CentralProbe
metadata:
  name: centralprobe-sample
spec:
  selector:
    matchLabels:
      meta.k8s.alipay.com/biz-group: sigmaboss
---
apiVersion: v1
kind: Pod
metadata:
  name: readiness
  labels:
    meta.k8s.alipay.com/biz-group: sigmaboss
spec:
  restartPolicy: Never
  containers:
  - name: readiness
    image: busybox
    args:
    - /bin/sh
    - -c
    - touch /tmp/test; sleep 60; rm -rf /tmp/test; sleep 300
    readinessProbe:
      exec:
        command:
        - cat
        - /tmp/test
      initialDelaySeconds: 10
      periodSeconds: 5
```
### Pod limiter
limit pod create/update Operation
```
apiVersion: podlimiter.kube-stack.me/v1
kind: Podlimiter
metadata:
  name: podlimiter-sample
spec:
  rules:
  - name: pending # rule name
    labelSelector: ""
    fieldSelector: "spec.nodeName==" # available fields: "metadata.name", "metadata.namespace","spec.nodeName","spec.restartPolicy","spec.schedulerName","spec.serviceAccountName","status.phase","status.podIP","status.nominatedNodeName"
    threshhold: 10 # The maximum number of eligible pods
    enabled: true # true/false
```
webhook endpoint: /validating-pod
metrics:
podlimiter_rule_current_num, labels: podlimiter,rule



### apiserver性能探测
Probe watch latency and list all latency of apiserver.
```
$cd kube-stack
$go run main.go -slo-mode=true -kubeconfig ~/.kube/config -target-namespace=default -target-pod-name=kube-stack-target-pod
1.667539733941164e+09	INFO	watch.go	list all pods	{"time cost": "16.08025ms"}
1.66753973394863e+09	INFO	watch.go	updatePodPeriodically	{"rv": "1818", "ts": 1667539733925, "nack#": 1}
1.667539734009155e+09	INFO	watch.go	Watch Delay	{"ms": 84, "nack#": 0, "rv": "1818", "ts": "1667539733925"}
1.66753974395995e+09	INFO	watch.go	updatePodPeriodically	{"rv": "1831", "ts": 1667539743949, "nack#": 1}
1.6675397439599621e+09	INFO	watch.go	Watch Delay	{"ms": 10, "nack#": 0, "rv": "1831", "ts": "1667539743949"}
```
metrics:
* watch_event_delay_ms
* list_all_resource_duration_ms
* watch_event_recv_total

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/kube-stack:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/kube-stack:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
