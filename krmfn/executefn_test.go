package krmfn

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
)

var exampleService = `apiVersion: v1
kind: Service
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","kind":"Service","metadata":{"annotations":{},"labels":{"app":"guestbook","tier":"frontend"},"name":"frontend","namespace":"guestbook"},"spec":{"ports":[{"port":80}],"selector":{"app":"guestbook","tier":"frontend"}}}
  creationTimestamp: "2022-06-14T16:49:17Z"
  labels:
    app: guestbook
    tier: frontend
  name: frontend
  namespace: guestbook
  resourceVersion: "479"
  uid: 0e19ac91-c96d-4e64-b443-c72733bf9734
spec:
  clusterIP: 10.109.22.148
  clusterIPs:
    - 10.109.22.148
  internalTrafficPolicy: Cluster
  ipFamilies:
    - IPv4
  ipFamilyPolicy: SingleStack
  ports:
    - port: 80
      protocol: TCP
      targetPort: 80
  selector:
    app: guestbook
    tier: frontend
  sessionAffinity: None
  type: ClusterIP
status:
  loadBalancer: {}`

var exampleDeployment = `apiVersion: apps/v1 
kind: Deployment
metadata:
  name: frontend
spec:
  selector:
    matchLabels:
      app: guestbook
      tier: frontend
  replicas: 3
  template:
    metadata:
      labels:
        app: guestbook
        tier: frontend
    spec:
      containers:
      - name: php-redis
        image: gcr.io/google-samples/gb-frontend:v4
        resources:
          requests:
            cpu: 100m
            memory: 100Mi
        env:
        - name: GET_HOSTS_FROM
          # value: dns
          # If your cluster config does not include a dns service, then to
          # instead access environment variables to find service host
          # info, comment out the 'value: dns' line above, and uncomment the
          # line below:
          value: env
        ports:
        - containerPort: 80`

func TestExecuteFn_AddFunction(t *testing.T) {
	executeFn := executeFn{}
	configMap := make(map[string]string)
	configMap["env"] = "dev"
	configMap["app-name"] = "my-app"

	function := Function{
		Name:      "Example Function",
		Image:     "example.com/my-image:v0.1",
		ConfigMap: configMap,
	}
	err := executeFn.addFunctions(function)
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}
	fnAnnotation := executeFn.functions[0].GetAnnotations()[runtimeutil.FunctionAnnotationKey]
	fnAnnotation = strings.TrimSpace(fnAnnotation)
	assert.EqualValues(t, fnAnnotation, fmt.Sprintf("container: {image: '%s'}", function.Image))
	assert.EqualValues(t, configMap, executeFn.functions[0].GetDataMap())
}

func TestExecuteFn_Execute(t *testing.T) {
	executeFn := executeFn{}
	err := executeFn.addInput([]byte(exampleService))
	err = executeFn.addInput([]byte(exampleDeployment))
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}

	functions := getFns()
	err = executeFn.addFunctions(functions...)
	err = executeFn.setExecWorkingDir("../testdata")
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}
	rl, err := executeFn.Execute()
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}
	expectedLabels := map[string]string{
		"app-name": "my-app",
		"env":      "dev",
		"tier":     "frontend",
		"app":      "guestbook",
	}
	assert.EqualValues(t, expectedLabels, rl.Items[1].(*unstructured.Unstructured).GetLabels())
}

func TestReadResultFromFile(t *testing.T) {
	content := `- field:
    path: spec
  message: Error message
  resourceRef:
    apiVersion: v1
    kind: Pod
    name: bar
    namespace: foo-ns
  severity: error`

	var result framework.Results
	err := yaml.Unmarshal([]byte(content), &result)
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}
	assert.EqualValues(t, "Error message", result[0].Message)
}
