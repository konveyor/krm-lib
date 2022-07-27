package krmfn

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
)

var runnerErr error

type Runner struct {
	executeFn executeFn
}

func NewRunner() RunnerBuilder {
	return &Runner{
		executeFn: executeFn{},
	}
}

func (r Runner) WithInput(bytes []byte) RunnerBuilder {
	err := r.executeFn.addInput(bytes)
	appendError(err)
	return r
}

func (r Runner) WithFunctions(function ...Function) RunnerBuilder {
	err := r.executeFn.addFunctions(function...)
	appendError(err)
	return r
}

func (r Runner) WithInputs(objects ...runtime.Object) RunnerBuilder {
	if objects == nil {
		return r
	}
	err := r.executeFn.addInputs(objects...)
	appendError(err)
	return r
}

func (r Runner) WhereExecWorkingDir(dir string) RunnerBuilder {
	err := r.executeFn.setExecWorkingDir(dir)
	appendError(err)
	return r
}

func (r Runner) Build() (FunctionRunner, error) {
	if runnerErr != nil {
		return nil, runnerErr
	}
	if r.executeFn.input == nil {
		return nil, ErrInputRequired
	}
	if len(r.executeFn.functions) == 0 {
		return nil, ErrFunctionRequired
	}
	return &r.executeFn, runnerErr
}

func appendError(err error) {
	if err != nil {
		if runnerErr == nil {
			runnerErr = err
		} else {
			runnerErr = fmt.Errorf("%v\n%v", runnerErr, err)
		}
	}
}
