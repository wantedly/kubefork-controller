package domain

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/wantedly/kubefork-controller/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Fork struct {
	f *v1beta1.Fork
}

func NewFork(identifier, namespace, forkManagerName string, replicaNum int32, validTime time.Duration, serviceSelector, deploymentSelector *metav1.LabelSelector,
	containers []v1.Container, deploymentAnnotation map[string]string) Fork {
	fork := v1beta1.Fork{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Fork",
			APIVersion: "fork.k8s.wantedly.com/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubefork-" + identifier,
			Namespace: namespace,
			Labels:    map[string]string{"fork.k8s.wantedly.com/identifier": identifier},
		},
		Spec: v1beta1.ForkSpec{
			Manager:    forkManagerName,
			Identifier: identifier,
			Deadline: &metav1.Time{
				Time: time.Now().Add(time.Hour * validTime),
			},
			Services: &v1beta1.ForkService{
				Selector: serviceSelector, // this field depends on '--service-label' and '--service-name' options
			},
			Deployments: &v1beta1.ForkDeployment{
				Selector: deploymentSelector, // this field depends on '--deployment-label' and '--deployment-name' options
				Template: &v1beta1.PodTemplateSpec{
					ObjectMeta: &metav1.ObjectMeta{
						Labels:      map[string]string{"app": identifier, "role": "fork"},
						Annotations: deploymentAnnotation, // if '--deployment-annotation' option is not used, this field is empty
					},
					Spec: v1beta1.PodSpec{
						Containers: containers, // if '--image' option is not used, this field is empty
					},
				},
				Replicas: &replicaNum,
			},
		},
		Status: v1beta1.ForkStatus{},
	}

	return Fork{&fork}
}

func (f Fork) OutputManifest(path string) error {
	y, err := yaml.Marshal(*(f.f))
	if err != nil {
		return errors.WithStack(err)
	}

	if path == "" {
		fmt.Println(string(y))
	} else {
		// TODO: Create a file if not exist
		if err := os.WriteFile(path, y, 0644); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Auxiliary functions to create Fork

func NewSelector(labelFromOption []string, labelFromName map[string]string) (*metav1.LabelSelector, error) {
	// NOTE: Labels are preferred to those directly selected in the options.
	matchLabels := labelFromName
	var matchExpressions []metav1.LabelSelectorRequirement

	for _, l := range labelFromOption {
		lArrNotEq := strings.Split(l, "!=")
		lArrEq := strings.Split(l, "=")
		lArrExc := strings.Split(l, "!")

		if len(lArrNotEq) == 2 {
			// If the label is of the form '<key>!=<value>'
			matchExpressions = append(matchExpressions,
				metav1.LabelSelectorRequirement{
					Key:      lArrNotEq[0],
					Operator: "NotIn",
					Values:   []string{lArrNotEq[1]},
				},
			)
		} else if len(lArrEq) == 1 && len(lArrExc) == 1 {
			// If the label is of the form '<key>'
			matchExpressions = append(matchExpressions,
				metav1.LabelSelectorRequirement{
					Key:      l,
					Operator: "Exists",
					Values:   nil,
				},
			)
		} else if len(lArrEq) == 1 && len(lArrExc) == 2 {
			// If the label is of the form '!<key>'
			matchExpressions = append(matchExpressions,
				metav1.LabelSelectorRequirement{
					Key:      lArrExc[1],
					Operator: "DoesNotExist",
					Values:   nil,
				},
			)
		} else if len(lArrEq) == 2 && len(lArrExc) == 1 {
			// If the label is of the form '<key>=<value>'
			matchLabels[lArrEq[0]] = lArrEq[1]
		} else {
			return nil, errors.New(fmt.Sprintf("%s is invalid. '<key>', '!<key>' '<key>=<value>' or '<key>!=<value>' label formats are supported", l))
		}
	}

	selector := &metav1.LabelSelector{
		MatchLabels:      matchLabels,
		MatchExpressions: matchExpressions,
	}

	return selector, nil
}

func NewContainers(image string, containerNames []string, env []v1.EnvVar) []v1.Container {
	containers := make([]v1.Container, len(containerNames))
	for i, name := range containerNames {
		containers[i] = v1.Container{
			Name:      name,
			Image:     image,
			Env:       env,
			Resources: v1.ResourceRequirements{},
		}
	}

	return containers
}

func NewEnv(identifier string, envMap map[string]string) []v1.EnvVar {
	env := []v1.EnvVar{{
		Name:  "FORK_IDENTIFIER",
		Value: identifier,
	}}

	for k, v := range envMap {
		env = append(env,
			v1.EnvVar{
				Name:  k,
				Value: v,
			})
	}

	return env
}
