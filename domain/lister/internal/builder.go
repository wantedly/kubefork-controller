package application

import (
	"context"
	"sort"
	"strings"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	"github.com/wantedly/kubefork-controller/pkg/refresh"
)

func NewBuilder(reader client.Reader, fork forkv1beta1.Fork) refresh.Builder {
	return builder{
		reader,
		fork,
	}
}

type builder struct {
	reader client.Reader
	fork   forkv1beta1.Fork
}

// Build collects information to build Application
func (b builder) Build(ctx context.Context) (refresh.Lister, error) {
	forkHeader, err := b.getForkHeaderKey(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	services, err := b.forkTargetServices(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	existingCopiedServices, err := b.existingForkedServices(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	serviceNameToDeployName := map[string][]string{}
	deploySet := map[string]appsv1.Deployment{}
	for _, svc := range services {
		dplys, err := b.listDeploymentsRoutableFromSVC(ctx, svc)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		for _, d := range dplys {
			deploySet[d.Name] = d
			serviceNameToDeployName[svc.Name] = append(serviceNameToDeployName[svc.Name], d.Name)
		}
	}

	deploys := []appsv1.Deployment{}
	for _, dply := range deploySet {
		deploys = append(deploys, dply)
	}
	// for less flaky behavior
	sort.Slice(deploys, func(i, j int) bool { return deploys[i].Name < deploys[j].Name })

	return &app{services, deploys, existingCopiedServices, inverseMap(serviceNameToDeployName), forkHeader, b.fork}, nil
}

func (b builder) getForkHeaderKey(ctx context.Context) (string, error) {
	slugParts := strings.Split(b.fork.Spec.Manager, "/")
	if len(slugParts) != 2 {
		return "", errors.New("malformed field `manager`")
	}

	managerSlug := types.NamespacedName{Namespace: slugParts[0], Name: slugParts[1]}

	fm := &forkv1beta1.ForkManager{}
	if err := b.reader.Get(ctx, managerSlug, fm); err != nil {
		return "", errors.WithStack(err)
	}
	return fm.Spec.HeaderKey, nil
}

func (b builder) forkTargetServices(ctx context.Context) ([]corev1.Service, error) {
	services := b.fork.Spec.Services
	if services == nil {
		return nil, nil
	}
	serviceList := &corev1.ServiceList{}
	{ // list all services to be forked
		selector, err := metav1.LabelSelectorAsSelector(services.Selector)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if err := b.reader.List(ctx, serviceList, &client.ListOptions{LabelSelector: selector, Namespace: b.fork.Namespace}); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return serviceList.Items, nil
}

func (b builder) existingForkedServices(ctx context.Context) (map[string]corev1.Service, error) {
	serviceList := &corev1.ServiceList{}
	{ // list all services that are already forked
		labelKV := map[string][]string{
			forkIdentiferLabelKey: {b.fork.Spec.Identifier},
		}
		ls := labels.Everything()
		for k, v := range labelKV {
			req, err := labels.NewRequirement(k, selection.Equals, v)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			ls = ls.Add(*req)
		}

		if err := b.reader.List(ctx, serviceList, &client.ListOptions{LabelSelector: ls, Namespace: b.fork.Namespace}); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	ret := map[string]corev1.Service{}
	for _, svc := range serviceList.Items {
		ret[svc.Name] = svc
	}

	return ret, nil
}

func (b builder) listDeploymentsRoutableFromSVC(ctx context.Context, svc corev1.Service) ([]appsv1.Deployment, error) {
	deployList := &appsv1.DeploymentList{}

	// If no selector is specified, do not list any deployments
	if b.fork.Spec.Deployments == nil || b.fork.Spec.Deployments.Selector == nil {
		return nil, nil
	}

	{ // list deployments in the namespace
		// select all by default
		selector := labels.Everything()
		// when selector is specified that will be applied
		if deploys := b.fork.Spec.Deployments; deploys != nil && deploys.Selector != nil {
			var err error
			selector, err = metav1.LabelSelectorAsSelector(deploys.Selector)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}
		if err := b.reader.List(ctx, deployList, &client.ListOptions{LabelSelector: selector, Namespace: b.fork.Namespace}); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	routableDeployments := []appsv1.Deployment{}
	// select only routable from the service
	for _, dply := range deployList.Items {
		if !deploymentExtension(dply).routableFrom(svc) {
			continue
		}

		routableDeployments = append(routableDeployments, dply)
	}

	return routableDeployments, nil
}

type deploymentExtension appsv1.Deployment

func (d deploymentExtension) routableFrom(svc corev1.Service) bool {
	svcSelector := labels.Set(svc.Spec.Selector).AsSelector()
	return svcSelector.Matches(labels.Set(d.Spec.Template.Labels))
}

// {a: [x, y], b: [y, z] } => {x: [a], y: [a, b], z: [b] }
func inverseMap(in map[string][]string) map[string][]string {
	out := map[string][]string{}
	for oldKey, items := range in {
		for _, newKey := range items {
			if current, ok := out[newKey]; !ok {
				out[newKey] = []string{oldKey}
			} else {
				out[newKey] = append(current, oldKey)
			}
		}
	}
	return out
}
