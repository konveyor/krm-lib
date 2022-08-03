package krmfn

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
)

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
	exampleService, err := ioutil.ReadFile("../testdata/service.yaml")
	exampleDeployment, err := ioutil.ReadFile("../testdata/deployment.yaml")
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}

	executeFn := executeFn{}
	err = executeFn.addInput(exampleService)
	err = executeFn.addInput(exampleDeployment)
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
