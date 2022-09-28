package k8sClientGo

import (
	"context"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/wantedly/kubefork-controller/kubeforkctl/client"
	"github.com/wantedly/kubefork-controller/kubeforkctl/lib"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Implementation of client by k8s.io/client-go

type k8sClientGo struct {
	clientSet *kubernetes.Clientset
}

func NewClientSet(kubeConfigPath string, isConfigOptionChanged bool) (client.Client, error) {
	if !isConfigOptionChanged && kubeConfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		kubeConfigPath = home + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return k8sClientGo{clientSet}, nil
}

func (k k8sClientGo) GetAllServiceLabelsByServiceName(ctx context.Context, namespace, serviceName string) (map[string]string, error) {
	s, err := k.clientSet.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return s.Labels, nil
}

func (k k8sClientGo) GetAllDeploymentLabelsByDeploymentName(ctx context.Context, namespace, deploymentName string) (map[string]string, error) {
	d, err := k.clientSet.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return d.Labels, nil
}

func (k k8sClientGo) GetAllContainersByImageNameAndServiceAndDeploymentSelector(ctx context.Context, namespace, imageName string,
	serviceSelector, deploymentSelector *metav1.LabelSelector) ([]string, error) {
	ss, err := k.clientSet.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(serviceSelector),
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	containers := map[string]v1.Container{}
	for _, s := range ss.Items {
		ds, err := k.clientSet.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
			/* Merging a selector of selected services and that of a selected deployment
			   the selector specified in deployment takes priority.*/
			LabelSelector: metav1.FormatLabelSelector(
				&metav1.LabelSelector{
					/* NOTE: Currently, only supported if the label of the pod that the service selects and the label
					   of the deployment that the service corresponds to are the same.*/
					MatchLabels:      lib.Merge(s.Spec.Selector, deploymentSelector.MatchLabels),
					MatchExpressions: deploymentSelector.MatchExpressions,
				}),
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// Collect containers matching selector
		for _, d := range ds.Items {
			for _, c := range d.Spec.Template.Spec.Containers {
				// Select only containers to switch an image
				if strings.Split(c.Image, ":")[0] == strings.Split(imageName, ":")[0] {
					containers[c.Name] = c
				}
			}
		}
	}

	containerNames := make([]string, 0)
	for key := range containers {
		containerNames = append(containerNames, key)
	}

	return containerNames, nil
}
