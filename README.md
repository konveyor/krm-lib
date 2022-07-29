KRMLib
======

This is a repo to hold golang libraries specifically for handling KRM Functions.

- [KRM Fn Execution Lib](#krmfn)

## KRMFN
Execute multiple KRM functions as containerized images and executable binaries.

### Using the library
To use the krmfn library, you need to create a RunnerBuilder object and 

```
var runner RunnerBuilder
runner = krmfn.NewRunner()
```

Build the RunnerBuilder object with the following interface

```
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
```

KRM function is provided as an input to library in the following format
```
type Function struct {
	// Name is the name of the function.
	Name string
	// `Image` specifies the function container image.
	//	image: gcr.io/kpt-fn/set-labels
	Image string `yaml:"image,omitempty" json:"image,omitempty"`

	// Exec specifies the function binary executable.
	// 	 exec: /usr/local/bin/myfn
	Exec string `yaml:"exec,omitempty" json:"exec,omitempty"`

	// `ConfigMap` is a convenient way to specify a function config of kind ConfigMap.
	ConfigMap map[string]string `yaml:"configMap,omitempty" json:"configMap,omitempty"`
}
```


#### Code Example

This example assumes that you have a kubernetes resource as a [runtime.object](https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#Object).

```
var resource1 runtime.Object
var resource2 []byte
var err error

resource2, err := ioutil.ReadFile("testdata/resource.yaml")
functions := []krmfn.Function{
		{
			Name:  "Set Namespace",
			Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
			ConfigMap: map[string]string{
				"namespace": "LibDemo",
			},
		},
		{
			Name: "Set Labels",
			Exec: "testdata/set-labels",
			ConfigMap: map[string]string{
				"env": "dev",
			},
		},
	}
  
  runner := krmfn.NewRunner().
		WithInput(resource2).
    WithInputs(resource1).
		WithFunctions(functions...)
  
 fnRunner, err := runner.build()
 output, err := fnRunner.Execute()
```

***output*** is a ResourceList object with the following structure:

```
type ResourceList struct {
	// Items is the ResourceList.items output value - transformed resources.
	Items []runtime.Object

	// Results is a map of function name with its result,
      // where result is an optional meta resource emitted by the function for observability and debugging purposes.
	Results map[string]framework.Result
}
```
