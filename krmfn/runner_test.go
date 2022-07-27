package krmfn

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestRunFnRunner(t *testing.T) {
	functions := getFns()

	temp := unstructured.Unstructured{}
	jsonValue, err := yaml.YAMLToJSON([]byte(exampleService))
	err = temp.UnmarshalJSON(jsonValue)
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}

	runner := NewRunner().
		WithInputs(&temp).
		WithFunctions(functions...).
		WhereExecWorkingDir("/usr")

	fnRunner, err := runner.Build()
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}

	OutRl, err := fnRunner.Execute()
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}

	expectedLabels := map[string]string{
		"app-name": "my-app",
		"env":      "dev",
		"tier":     "frontend",
		"app":      "guestbook",
	}
	assert.EqualValues(t, expectedLabels, OutRl.Items[0].(*unstructured.Unstructured).GetLabels())
}

func getFns() []Function {
	functions := []Function{
		{
			Name:  "Set Labels",
			Image: "gcr.io/kpt-fn/set-labels:v0.1",
			ConfigMap: map[string]string{
				"env":      "dev",
				"app-name": "my-app",
			},
		},
		{
			Name: "Clean Metadata",
			Exec: "../testdata/clean-metadata",
		},
	}
	return functions
}

func TestRunner_WithInputs(t *testing.T) {
	// test with empty input
	runner := NewRunner().WithFunctions(getFns()...)
	_, err := runner.Build()
	assert.EqualValues(t, ErrInputRequired, err)

	// expected input and function to be executed
	in := unstructured.Unstructured{}
	json, err := yaml.YAMLToJSON([]byte(exampleDeployment))
	err = in.UnmarshalJSON(json)
	if err != nil {
		t.Errorf("Unexpected Error: %v", err)
	}

	fnRunner, err := runner.WithInput([]byte(exampleService)).WithInputs(&in).Build()
	if err != nil {
		assert.Fail(t, "Runner failed while build", err)
	}
	_, err = fnRunner.Execute()
	if err != nil {
		assert.Fail(t, "Unexpected Error: %v", err)
	}

	// Provide a list object that implements runtime.Objects
	lst := unstructured.UnstructuredList{}
	lst.Items = append(lst.Items, in)
	fnRunner, err = runner.WithInputs(&lst).Build()
	assert.EqualValues(t, ErrUnsupportedInputList, err)
}
