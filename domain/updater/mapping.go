package updater

import (
	"context"
	"fmt"
	"strings"

	ambassador "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	util "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// Mark ambassador mapping is managed by which ForkManager
	labelKey = "fork.k8s.wantedly.com/manager"
)

type mappingUpdater struct {
	client client.Client
	log    logr.Logger
	scheme *runtime.Scheme
}

// NewMappingUpdater returns a Updater that reconciles Mapping based on ForkManager
func NewMappingUpdater(client client.Client, log logr.Logger, scheme *runtime.Scheme) Updater {
	return &mappingUpdater{
		client: client,
		log:    log,
		scheme: scheme,
	}
}

func (r mappingUpdater) Update(ctx context.Context, managerSlug types.NamespacedName) error {
	fm := &forkv1beta1.ForkManager{}
	if err := r.client.Get(ctx, managerSlug, fm); err != nil {
		return errors.WithStack(err)
	}

	frks := &forkv1beta1.ForkList{}
	if err := r.client.List(ctx, frks); err != nil {
		return errors.WithStack(err)
	}

	// key:   Mapping name
	// value: true if should be deleted
	mappingDeleteCandidate := map[types.NamespacedName]struct{}{}
	{
		mps := &ambassador.MappingList{}
		req, err := labels.NewRequirement(labelKey, selection.Equals, []string{managerSlug.Name})
		if err != nil {
			return errors.WithStack(err)
		}
		ls := labels.Everything().Add(*req)
		if err := r.client.List(ctx, mps, &client.ListOptions{LabelSelector: ls}); err != nil {
			return errors.WithStack(err)
		}
		for _, mp := range mps.Items {
			mappingDeleteCandidate[types.NamespacedName{Namespace: mp.Namespace, Name: mp.Name}] = struct{}{}
		}
	}

	forkMap := groupForksByIdentifier(frks.Items)
	for identifier, forks := range forkMap {
		for _, upstream := range fm.Spec.Upstreams {
			key := strings.ReplaceAll(upstream.Host+"-"+identifier, ".", "-")
			// Reaching here means this mapping should not be deleted
			delete(mappingDeleteCandidate, types.NamespacedName{Namespace: fm.Namespace, Name: key})

			mp := &ambassador.Mapping{ObjectMeta: v1.ObjectMeta{Name: key, Namespace: fm.Namespace}}

			service := upstream.Original
			if service == "" {
				service = fmt.Sprintf("https://%s", upstream.Host)
			}

			if _, err := util.CreateOrUpdate(ctx, r.client, mp, func() error {
				mp.Spec = ambassador.MappingSpec{
					AddRequestHeaders: map[string]ambassador.AddedHeader{
						fm.Spec.HeaderKey: {String: pointer.StringPtr(identifier)},

						// see. https://github.com/wantedly/visit-ambassador-v2/pull/105
						"x-forwarded-host": {String: pointer.StringPtr("%REQ(:authority)%")},
					},
					AllowUpgrade: []string{"websocket"},
					AmbassadorID: []string{fm.Spec.AmbassadorID},
					Host:         fmt.Sprintf("%s.%s", identifier, upstream.Host),
					Prefix:       "/",
					Rewrite:      pointer.StringPtr(""),
					Service:      service,
					TimeoutMs:    90000,
				}

				// if the upstream has original(service name), then we use service name as Host
				// Priority is described bellow
				// 1. HostRewrite
				// 2. Original
				if upstream.Original != "" {
					// if the upstream has HostRewite, then we use it
					if upstream.HostRewrite != "" {
						mp.Spec.HostRewrite = upstream.HostRewrite
					} else {
						mp.Spec.HostRewrite = trimPort(upstream.Original)
					}
				}

				mp.Labels = map[string]string{
					labelKey: managerSlug.Name,
				}

				for _, fork := range forks {
					applyOptionsToMapping(mp, fork)
				}

				return errors.Wrap(util.SetControllerReference(fm, mp, r.scheme), "failed to set controller reference")
			}); err != nil {
				return errors.WithStack(err)
			}
		}
	}

	for mappingName := range mappingDeleteCandidate {
		mp := &ambassador.Mapping{ObjectMeta: v1.ObjectMeta{Name: mappingName.Name, Namespace: mappingName.Namespace}}
		if err := r.client.Delete(ctx, mp); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func groupForksByIdentifier(forks []forkv1beta1.Fork) map[string][]forkv1beta1.Fork {
	res := map[string][]forkv1beta1.Fork{}
	for _, f := range forks {
		if _, ok := res[f.Spec.Identifier]; !ok {
			res[f.Spec.Identifier] = []forkv1beta1.Fork{}
		}
		res[f.Spec.Identifier] = append(res[f.Spec.Identifier], f)
	}
	return res
}

func applyOptionsToMapping(mp *ambassador.Mapping, fork forkv1beta1.Fork) {
	if fork.Spec.GatewayOptions != nil {
		for k, v := range fork.Spec.GatewayOptions.AddRequestHeaders {
			mp.Spec.AddRequestHeaders[k] = ambassador.AddedHeader{
				String: pointer.StringPtr(v),
			}

			{
				upgradesSet := sets.NewString(mp.Spec.AllowUpgrade...)
				for _, upgrade := range fork.Spec.GatewayOptions.AllowUpgrade {
					upgradesSet.Insert(upgrade)
				}
				mp.Spec.AllowUpgrade = upgradesSet.List()
			}
		}
	}
}

func (r mappingUpdater) UpdateAll(ctx context.Context, opts ...client.ListOption) error {
	list := &forkv1beta1.ForkManagerList{}
	err := r.client.List(ctx, list, opts...)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, item := range list.Items {
		err := r.Update(ctx, types.NamespacedName{Name: item.Name, Namespace: item.Namespace})
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func trimPort(target string) string {
	return strings.Split(target, ":")[0]
}
