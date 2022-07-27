package krmfn

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"path/filepath"
	"reflect"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/runfn"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
	"strings"
)

type executeFn struct {
	input         *bytes.Buffer
	functionNames []string
	functions     []*kyaml.RNode
	execDir       string
}

func (e *executeFn) Execute() (ResourceList, error) {
	var rl ResourceList
	out := bytes.Buffer{}

	if e.execDir == "" {
		err := e.setExecWorkingDir("")
		if err != nil {
			return rl, err
		}
	}

	resultDir, err := ioutil.TempDir("", "Result")
	defer os.RemoveAll(resultDir) // clean up
	if err != nil {
		return rl, errors.Wrap(err)
	}

	input := io.Reader(e.input)
	err = runfn.RunFns{
		Input:      input,
		Output:     &out,
		Functions:  e.functions,
		EnableExec: true,
		ResultsDir: resultDir,
		WorkingDir: e.execDir,
	}.Execute()

	if err != nil {
		return rl, errors.Wrap(err)
	}
	rl, err = GetResourceList(out.String(), resultDir, e.functionNames)
	return rl, err
}

func (e *executeFn) addInput(resource []byte) error {
	resource, err := yaml.JSONToYAML(resource)
	if err != nil {
		return errors.Wrap(err)
	}
	if e.input == nil {
		e.input = bytes.NewBuffer(resource)
	} else {
		oldInput := e.input.String() + itemSeparator + string(resource)
		e.input = bytes.NewBufferString(oldInput)
	}
	return errors.Wrap(err)
}

func (e *executeFn) addInputs(inputs ...runtime.Object) error {
	for _, input := range inputs {
		if strings.Contains(reflect.TypeOf(input).String(), "List") {
			return ErrUnsupportedInputList
		} else {
			value, err := yaml.Marshal(input)
			if err = e.addInput(value); err != nil {
				return errors.Wrap(err)
			}
		}
	}
	return nil
}

func (e *executeFn) addFunctions(functions ...Function) error {
	functionConfig, err := e.getFunctionConfig(functions)
	if err != nil {
		return errors.Wrap(err)
	}
	e.functions = append(e.functions, functionConfig...)
	return nil
}

func (e *executeFn) setExecWorkingDir(dir string) error {
	if dir == "" {
		wd, err := ioutil.TempDir("", "ExecWorkingDir")
		if err != nil {
			return errors.Wrap(err)
		}
		dir = wd
	} else {
		// check if the dir exists
		wd, err := filepath.Abs(dir)
		if err != nil {
			return errors.Wrap(err)
		}
		if _, err := os.Stat(wd); os.IsNotExist(err) {
			return errors.Errorf("%s does not exist", wd)
		}
		dir = wd
	}
	e.execDir = dir
	return nil
}

// getFunctionsToExecute parses the explicit functions to run.
func (e *executeFn) getFunctionConfig(functions []Function) ([]*kyaml.RNode, error) {
	var functionConfig []*kyaml.RNode
	for _, fn := range functions {
		if fn.Name == "" {
			return nil, ErrFunctionNameRequired
		}
		e.functionNames = append(e.functionNames, fn.Name)
		res, err := buildFnConfigResource(fn)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		// create the function spec to set as an annotation
		var fnAnnotation *kyaml.RNode
		if fn.Image != "" {
			fnAnnotation, err = getFnAnnotationForImage(fn)
		} else {
			fnAnnotation, err = getFnAnnotationForExec(fn)
		}

		if err != nil {
			return nil, errors.Wrap(err)
		}

		// set the function annotation on the function config, so that it is parsed by RunFns
		value, err := fnAnnotation.String()
		if err != nil {
			return nil, errors.Wrap(err)
		}

		if err = res.PipeE(
			kyaml.LookupCreate(kyaml.MappingNode, "metadata", "annotations"),
			kyaml.SetField(runtimeutil.FunctionAnnotationKey, kyaml.NewScalarRNode(value))); err != nil {
			return nil, errors.Wrap(err)
		}
		functionConfig = append(functionConfig, res)
	}
	return functionConfig, nil
}

func buildFnConfigResource(function Function) (*kyaml.RNode, error) {
	if function.Image == "" && function.Exec == "" {
		return nil, fmt.Errorf("function must have either image or exec, none specified")
	}
	// create the function config
	rc, err := kyaml.Parse(`
metadata:
  name: function-input
data: {}
`)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	// default the function config kind to ConfigMap, this may be overridden
	var kind = "ConfigMap"
	var version = "v1"

	// populate the function config with data.
	dataField, err := rc.Pipe(kyaml.Lookup("data"))
	for key, value := range function.ConfigMap {
		err := dataField.PipeE(
			kyaml.FieldSetter{Name: key, Value: kyaml.NewStringRNode(value), OverrideStyle: true})
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	if err = rc.PipeE(kyaml.SetField("kind", kyaml.NewScalarRNode(kind))); err != nil {
		return nil, errors.Wrap(err)
	}
	if err = rc.PipeE(kyaml.SetField("apiVersion", kyaml.NewScalarRNode(version))); err != nil {
		return nil, errors.Wrap(err)
	}
	return rc, nil
}

func getFnAnnotationForExec(function Function) (*kyaml.RNode, error) {
	fn, err := kyaml.Parse(`exec: {}`)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	path, err := filepath.Abs(function.Exec)
	if err = fn.PipeE(
		kyaml.Lookup("exec"),
		kyaml.SetField("path", kyaml.NewScalarRNode(path))); err != nil {
		return nil, errors.Wrap(err)
	}
	return fn, nil
}

func getFnAnnotationForImage(function Function) (*kyaml.RNode, error) {
	if err := ValidateFunctionImageURL(function.Image); err != nil {
		return nil, errors.Wrap(err)
	}

	fn, err := kyaml.Parse(`container: {}`)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if err = fn.PipeE(
		kyaml.Lookup("container"),
		kyaml.SetField("image", kyaml.NewScalarRNode(function.Image))); err != nil {
		return nil, errors.Wrap(err)
	}
	return fn, nil
}
