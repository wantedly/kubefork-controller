package lister

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	"github.com/wantedly/kubefork-controller/pkg/refresh"
	networkingv1beta1 "istio.io/api/networking/v1beta1"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	labelKeyForVS = "fork.k8s.wantedly.com/service"
)

func NewVirtualServiceBuilder(r client.Reader, serviceSlug types.NamespacedName) refresh.Builder {
	return &builder{
		r,
		serviceSlug,
	}
}

type builder struct {
	r           client.Reader
	serviceSlug types.NamespacedName
}

type vsLister struct {
	sortedConfigs []forkv1beta1.VSConfig
	service       corev1.Service
}

func (b builder) Build(ctx context.Context) (refresh.Lister, error) {
	var sortedConfigs []forkv1beta1.VSConfig
	{
		configs := &forkv1beta1.VSConfigList{}
		err := b.r.List(ctx, configs, &client.ListOptions{Namespace: b.serviceSlug.Namespace})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		sort.Slice(configs.Items, func(i, j int) bool { return configs.Items[i].ObjectMeta.Name < configs.Items[j].ObjectMeta.Name })
		sortedConfigs = configs.Items
	}
	service := corev1.Service{}
	if err := b.r.Get(ctx, b.serviceSlug, &service); err != nil {
		return nil, errors.WithStack(err)
	}

	return &vsLister{sortedConfigs: sortedConfigs, service: service}, nil
}

func (a vsLister) GenerateLists() []refresh.ObjectList {
	generators := []func() refresh.ObjectList{
		a.buildVirtualService,
	}
	res := make([]refresh.ObjectList, len(generators))
	for i, gen := range generators {
		res[i] = gen()
	}
	return res
}

func (a vsLister) buildHTTPRoutes() []*networkingv1beta1.HTTPRoute {
	var routes []*networkingv1beta1.HTTPRoute
	for _, config := range a.sortedConfigs {
		if config.Spec.Host != a.service.Name || config.Spec.HeaderValue == "" {
			continue
		}
		routes = append(
			routes,
			&networkingv1beta1.HTTPRoute{
				Match: []*networkingv1beta1.HTTPMatchRequest{
					{
						Headers: map[string]*networkingv1beta1.StringMatch{
							config.Spec.HeaderName: {
								MatchType: &networkingv1beta1.StringMatch_Exact{
									Exact: config.Spec.HeaderValue,
								},
							},
						},
					},
				},
				Route: []*networkingv1beta1.HTTPRouteDestination{
					{
						Destination: &networkingv1beta1.Destination{
							Host: config.Spec.Service,
						},
					},
				},
			},
		)
	}

	return append(
		routes,
		&networkingv1beta1.HTTPRoute{
			// DefaultはMatchが空
			Route: []*networkingv1beta1.HTTPRouteDestination{
				{
					Destination: &networkingv1beta1.Destination{
						Host: a.service.Name,
					},
				},
			},
		},
	)
}

func (a vsLister) buildVirtualService() refresh.ObjectList {
	var list []client.Object
	if routes := a.buildHTTPRoutes(); len(routes) > 1 {
		vs := istio.VirtualService{
			TypeMeta: v1.TypeMeta{
				Kind:       "VirtualService",
				APIVersion: "networking.istio.io/v1beta1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      a.service.Name,
				Namespace: a.service.Namespace,
				Labels: map[string]string{
					labelKeyForVS: a.service.Name,
				},
			},
			Spec: networkingv1beta1.VirtualService{
				Hosts: []string{a.service.Name},
				Http:  routes,
			},
		}
		vs.ObjectMeta.SetResourceVersion(vs.GetResourceVersion())
		list = append(list, &vs)
	}

	return refresh.ObjectList{
		Items:            list,
		GroupVersionKind: istio.SchemeGroupVersion.WithKind("VirtualServiceList"),
		Identity: func(obj client.Object) (string, error) {
			switch vs := obj.(type) {
			case *istio.VirtualService:
				hosts := vs.Spec.GetHosts()
				if len(hosts) == 0 {
					// when hosts is empty and owned by the target service, we see them broken and it's ok to delete
					return "", nil
				}
				return hosts[len(hosts)-1], nil
			case *unstructured.Unstructured:
				to := &istio.VirtualService{}
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(vs.Object, to); err != nil {
					return "", errors.WithStack(err)
				}
				hosts := to.Spec.GetHosts()
				if len(hosts) == 0 {
					// when hosts is empty and owned by the target service, we see them broken and it's ok to delete
					return "", nil
				}
				return hosts[len(hosts)-1], nil
			}

			return "", errors.Errorf("Unsupported type %t", obj)
		},
	}
}
