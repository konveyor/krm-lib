package fn

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"regexp"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/yaml"
	"strings"
)

var itemSeparator = "---\n"

func GetResourceList(content string, resultDir string, names []string) (ResourceList, error) {
	rl := ResourceList{}
	rl.Results = make(map[string]framework.Result)

	items, err := cleanOutput(content)
	if err != nil {
		return rl, err
	}

	for _, item := range items {
		jsonItem, err := yaml.YAMLToJSON([]byte(item))
		if err != nil {
			return rl, errors.Wrap(err)
		}
		unstructuredItem := unstructured.Unstructured{}
		err = unstructuredItem.UnmarshalJSON(jsonItem)
		if err != nil {
			return rl, errors.Wrap(err)
		}
		rl.Items = append(rl.Items, &unstructuredItem)
	}

	resultFiles, err := ioutil.ReadDir(resultDir)
	if err != nil {
		return rl, errors.Wrap(err)
	}

	for _, resultFile := range resultFiles {
		fileName := strings.TrimSuffix(resultFile.Name(), ".yaml")
		fileNo := 0
		_, err := fmt.Sscan(strings.SplitN(fileName, "-", 2)[1], &fileNo)
		result, err := readFile(resultDir + "/" + resultFile.Name())
		if err != nil {
			return rl, errors.Wrap(err)
		}

		rl.Results[names[fileNo]] = *result[0]
	}

	return rl, nil
}

func readFile(file string) (framework.Results, error) {
	result := framework.Results{}
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return result, errors.Wrap(err)
	}

	err = yaml.Unmarshal(content, &result)
	if err != nil {
		return result, errors.Wrap(err)
	}
	return result, nil
}

func cleanOutput(content string) ([]string, error) {
	var items []string
	r := strings.NewReader(content)
	out := bytes.Buffer{}

	outputs := []kio.Writer{
		&kio.ByteWriter{
			Writer: &out,
			ClearAnnotations: []string{
				kioutil.IndexAnnotation, kioutil.PathAnnotation,
				kioutil.LegacyIndexAnnotation, kioutil.LegacyPathAnnotation},
		},
	}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: r, PreserveSeqIndent: true, WrapBareSeqNode: true}},
		Outputs: outputs,
	}.Execute()

	if err != nil {
		return items, err
	}
	items = strings.Split(out.String(), itemSeparator)
	return items, nil
}

// ValidateFunctionImageURL validates the function name.
// According to Docker implementation
// https://github.com/docker/distribution/blob/master/reference/reference.go. A valid
// name definition is:
//	name                            := [domain '/'] path-component ['/' path-component]*
//	domain                          := domain-component ['.' domain-component]* [':' port-number]
//	domain-component                := /([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])/
//	port-number                     := /[0-9]+/
//	path-component                  := alpha-numeric [separator alpha-numeric]*
// 	alpha-numeric                   := /[a-z0-9]+/
//	separator                       := /[_.]|__|[-]*/
// https://github.com/GoogleContainerTools/kpt/blob/b197de30601072d7b8668dd41150f398a7f415f5/pkg/api/kptfile/v1/validation.go#L120-L150
func ValidateFunctionImageURL(name string) error {
	pathComponentRegexp := `(?:[a-z0-9](?:(?:[_.]|__|[-]*)[a-z0-9]+)*)`
	domainComponentRegexp := `(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])`
	domainRegexp := fmt.Sprintf(`%s(?:\.%s)*(?:\:[0-9]+)?`, domainComponentRegexp, domainComponentRegexp)
	nameRegexp := fmt.Sprintf(`(?:%s\/)?%s(?:\/%s)*`, domainRegexp,
		pathComponentRegexp, pathComponentRegexp)
	tagRegexp := `(?:[\w][\w.-]{0,127})`
	shaRegexp := `(sha256:[a-zA-Z0-9]{64})`
	versionRegexp := fmt.Sprintf(`(%s|%s)`, tagRegexp, shaRegexp)
	t := fmt.Sprintf(`^(?:%s(?:(\:|@)%s)?)$`, nameRegexp, versionRegexp)

	matched, err := regexp.MatchString(t, name)
	if err != nil {
		return errors.Wrap(err)
	}
	if !matched {
		return fmt.Errorf("function name %q is invalid", name)
	}
	return nil
}
