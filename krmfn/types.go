package krmfn

import (
	"errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

var ErrInputRequired = errors.New("inputs are required")
var ErrFunctionRequired = errors.New("at least one function is required")
var ErrUnsupportedInputList = errors.New("unsupported input of type List")
var ErrFunctionNameRequired = errors.New("function must have a name")

//ResourceList is a Kubernetes list type used as the output data format in the Functions execution
type ResourceList struct {
	// Items is the ResourceList.items input and output value.
	//
	// e.g. given the function input:
	//
	//    items:
	//    - kind: Deployment
	//      ...
	//    - kind: Service
	//      ...
	//
	// Items will be a slice containing the Deployment and Service resources
	// Mutating functions will alter this field during processing.
	// This field is required.
	Items []runtime.Object

	// Results is ResourceList.results output value.
	// Validating functions can optionally use this field to communicate structured
	// validation error data to downstream functions.
	Results map[string]framework.Result
}

// Function specifies a KRM function to run.
type Function struct {
	// Name is the name of the function.
	Name string
	// `Image` specifies the function container image.
	//	image: gcr.io/kpt-fn/set-labels
	Image string `yaml:"image,omitempty" json:"image,omitempty"`

	// Exec specifies the function binary executable.
	// The executable can be fully qualified, or it must exist in the $PATH e.g:
	//
	// 	 exec: set-namespace
	// 	 exec: /usr/local/bin/my-custom-fn
	Exec string `yaml:"exec,omitempty" json:"exec,omitempty"`

	// `ConfigMap` is a convenient way to specify a function config of kind ConfigMap.
	ConfigMap map[string]string `yaml:"configMap,omitempty" json:"configMap,omitempty"`
}

// RunnerBuilder is a executeFn builder that can be used to build a FunctionRunner.
type RunnerBuilder interface {
	// WithInput adds raw input to the builder.
	WithInput([]byte) RunnerBuilder

	// WithInputs adds runtime.objects as the input resource.
	WithInputs(...runtime.Object) RunnerBuilder

	// WithFunctions provide a list of functions to run.
	WithFunctions(...Function) RunnerBuilder

	// WhereExecWorkingDir specifies which working directory an exec function should run in.
	WhereExecWorkingDir(string) RunnerBuilder

	// Build builds the runner with the provided options.
	Build() (FunctionRunner, error)
}

// FunctionRunner is a runner that can execute the functions.
type FunctionRunner interface {
	Execute() (ResourceList, error)
}
