package application

import (
	"fmt"

	ddv1beta1 "github.com/wantedly/deployment-duplicator/api/v1beta1"
	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type copyableDeployment appsv1.Deployment

func (d copyableDeployment) buildCopy(fork forkv1beta1.Fork, serviceNames []string) *ddv1beta1.DeploymentCopy {
	// WARNING: changing name requires better gc algorithm
	//          deploymentcopies with older naming convention and correct ownerref and targetdeployment won't be deleted
	name := fmt.Sprintf("%s-%s", d.Name, fork.Name)

	return &ddv1beta1.DeploymentCopy{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: fork.Namespace},
		Spec:       d.buildDeploymentCopySpec(fork, serviceNames),
	}
}

func (d copyableDeployment) buildDeploymentCopySpec(fork forkv1beta1.Fork, serviceNames []string) ddv1beta1.DeploymentCopySpec {
	labels := map[string]string{
		forkIdentiferLabelKey: fork.Spec.Identifier,
	}
	for _, s := range serviceNames {
		labels[getLabelKeyforRoutingLabel(s)] = "true"
	}

	var replicas int32
	// default is replicas: 1
	if fork.Spec.Deployments.Replicas == nil {
		replicas = 1
	} else {
		replicas = *fork.Spec.Deployments.Replicas
	}

	spec := ddv1beta1.DeploymentCopySpec{
		Replicas:             replicas,
		TargetDeploymentName: d.Name,
		NameSuffix:           fork.Name,
		CustomLabels:         labels,
	}

	tmpl := fork.Spec.Deployments.Template
	if tmpl == nil {
		return spec
	}

	spec.CustomLabels = mergeMap(spec.CustomLabels, tmpl.Labels)
	spec.CustomAnnotations = mergeMap(spec.CustomAnnotations, tmpl.Annotations)

	{
		var containers []ddv1beta1.Container
		for _, ctr := range tmpl.Spec.Containers {
			c := ddv1beta1.Container{
				Name:  ctr.Name,
				Image: ctr.Image,
				Env:   ctr.Env,
			}
			containers = append(containers, c)
		}
		spec.TargetContainers = containers
	}

	return spec
}

func mergeMap(ms ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, m := range ms {
		for k, v := range m {
			merged[k] = v
		}
	}

	return merged
}
