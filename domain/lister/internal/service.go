package application

import (
	"fmt"

	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type copyableService corev1.Service

func getLabelKeyforRoutingLabel(originalServiceName string) string {
	const routingLabelKeyPrefix = "fork.k8s.wantedly.com/routed-from"
	return fmt.Sprintf("%s-%s", routingLabelKeyPrefix, originalServiceName)
}

const forkIdentiferLabelKey = "fork.k8s.wantedly.com/identifier"

func (s copyableService) buildCopy(fork forkv1beta1.Fork, existingService map[string]corev1.Service) *corev1.Service {
	copiedSpec := s.Spec.DeepCopy()
	spec := corev1.ServiceSpec{}

	spec.ClusterIP = ""
	if existing, ok := existingService[s.serviceName(fork)]; ok {
		spec.ClusterIP = existing.Spec.ClusterIP
	}
	spec.Type = corev1.ServiceTypeClusterIP

	for _, port := range copiedSpec.Ports {
		spec.Ports = append(spec.Ports, corev1.ServicePort{
			Name:        port.Name,
			Protocol:    port.Protocol,
			AppProtocol: port.AppProtocol,
			Port:        port.Port,
			TargetPort:  port.TargetPort,
			// NodePort should be skipped because this is also immutable
		})
	}

	spec.Selector = map[string]string{
		getLabelKeyforRoutingLabel(s.Name): "true",
		forkIdentiferLabelKey:              fork.Spec.Identifier,
	}

	return &corev1.Service{
		ObjectMeta: v1.ObjectMeta{Name: s.serviceName(fork), Namespace: fork.Namespace},
		Spec:       spec,
	}
}

func (s copyableService) serviceName(fork forkv1beta1.Fork) string {
	// WARNING: changing name requires better gc algorithm
	//          deploymentcopies with older naming convention and correct ownerref and targetdeployment won't be deleted
	return fmt.Sprintf("%s-%s", s.Name, fork.Name)
}

func (s copyableService) buildVSConfig(fork forkv1beta1.Fork, headerName string) *forkv1beta1.VSConfig {
	name := s.serviceName(fork)

	return &forkv1beta1.VSConfig{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: fork.Namespace},
		Spec: forkv1beta1.VSConfigSpec{
			Host:        s.Name,
			Service:     s.serviceName(fork),
			HeaderName:  headerName,
			HeaderValue: fork.Spec.Identifier,
		},
	}
}
