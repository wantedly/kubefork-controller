package client

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client interface {
	GetAllServiceLabelsByServiceName(ctx context.Context, namespace, serviceName string) (map[string]string, error)
	GetAllDeploymentLabelsByDeploymentName(ctx context.Context, namespace, deploymentName string) (map[string]string, error)
	GetAllContainersByImageNameAndServiceAndDeploymentSelector(ctx context.Context, namespace, imageName string,
		serviceSelector, deploymentSelector *metav1.LabelSelector) ([]string, error)
}
