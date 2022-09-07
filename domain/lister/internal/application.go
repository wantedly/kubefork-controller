package application

import (
	"github.com/pkg/errors"
	ddv1beta1 "github.com/wantedly/deployment-duplicator/api/v1beta1"
	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	"github.com/wantedly/kubefork-controller/pkg/refresh"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// an app consists of
// - a list of Service
// - relation between those
type app struct {
	services    []corev1.Service
	deployments []appsv1.Deployment

	existingCopiedServices map[string]corev1.Service
	// key - deployment name
	// value - a list of services that routes to the deployment of the key
	deployNameToServiceNames map[string][]string
	forkHeader               string
	fork                     forkv1beta1.Fork
}

func (a app) GenerateLists() []refresh.ObjectList {
	generators := []func() refresh.ObjectList{
		a.generateDeploymentCopies,
		a.generateServices,
		a.generateVSConfigs,
	}
	res := make([]refresh.ObjectList, len(generators))
	for i, gen := range generators {
		res[i] = gen()
	}
	return res
}

func (a app) generateDeploymentCopies() refresh.ObjectList {
	copies := make([]client.Object, len(a.deployments))
	for i, dply := range a.deployments {
		copies[i] = copyableDeployment(dply).buildCopy(a.fork, a.deployNameToServiceNames[dply.Name])
	}

	return refresh.ObjectList{
		Items:            copies,
		GroupVersionKind: ddv1beta1.GroupVersion.WithKind("DeploymentCopyList"),
		Identity: func(obj client.Object) (string, error) {
			return obj.GetName(), nil
		},
	}
}

func (a app) generateServices() refresh.ObjectList {
	// Define the label key here so that we can keep consistency for the labels and id function
	const labelKey = "fork.k8s.wantedly.com/original-service-name"
	svcs := make([]client.Object, len(a.services))
	for i, svc := range a.services {
		obj := copyableService(svc).buildCopy(a.fork, a.existingCopiedServices)
		obj.Labels = mergeMap(obj.Labels, map[string]string{labelKey: svc.Name, forkIdentiferLabelKey: a.fork.Spec.Identifier})
		svcs[i] = obj
	}

	return refresh.ObjectList{
		Items:            svcs,
		GroupVersionKind: corev1.SchemeGroupVersion.WithKind("ServiceList"),
		Identity: func(obj client.Object) (string, error) {
			labels := obj.GetLabels()
			if labels == nil {
				// services not managed by fork may not have labels
				// this is why we cannot return error when labels is nil
				return "", nil
			}
			return labels[labelKey], nil
		},
	}
}

func (a app) generateVSConfigs() refresh.ObjectList {
	svcs := make([]client.Object, len(a.services))
	for i, svc := range a.services {
		svcs[i] = copyableService(svc).buildVSConfig(a.fork, a.forkHeader)
	}

	return refresh.ObjectList{
		Items:            svcs,
		GroupVersionKind: forkv1beta1.GroupVersion.WithKind("VSConfigList"),
		Identity:         VSConfigIdentity,
	}
}

// FIXME: expocted only because of test for short term fix, consider re-encapsulating after fix
func VSConfigIdentity(obj client.Object) (string, error) {
	switch vs := obj.(type) {
	case *forkv1beta1.VSConfig:
		return vs.Spec.Host, nil
	case *unstructured.Unstructured:
		to := &forkv1beta1.VSConfig{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(vs.Object, to); err != nil {
			return "", errors.WithStack(err)
		}
		return VSConfigIdentity(to)
	}

	return "", errors.Errorf("Unsupported type %t", obj)
}
