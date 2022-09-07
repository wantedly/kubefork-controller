package updater

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/wantedly/kubefork-controller/domain/lister"
	"github.com/wantedly/kubefork-controller/pkg/refresh"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type virtualServiceUpdater struct {
	client client.Client
	log    logr.Logger
	scheme *runtime.Scheme
}

// NewVirtualServiceUpdater returns a Updater that reconciles VirtualService based on Service
func NewVirtualServiceUpdater(client client.Client, log logr.Logger, scheme *runtime.Scheme) Updater {
	return &virtualServiceUpdater{
		client: client,
		log:    log,
		scheme: scheme,
	}
}

func (r virtualServiceUpdater) Update(ctx context.Context, serviceSlug types.NamespacedName) error {
	// service is the top of the reference tree
	service := &corev1.Service{}
	if err := r.client.Get(ctx, serviceSlug, service); err != nil {
		// TODO: update status of vsconfig not to process them multiple times
		return errors.WithStack(err)
	}
	lstr, err := lister.NewVirtualServiceBuilder(r.client, serviceSlug).Build(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	resourceLists := lstr.GenerateLists()
	ref := refresh.New(r.client, r.scheme)
	for _, m := range resourceLists {
		if err := ref.Refresh(ctx, service, m); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (r virtualServiceUpdater) UpdateAll(ctx context.Context, opts ...client.ListOption) error {
	services := &corev1.ServiceList{}
	err := r.client.List(ctx, services, opts...)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, service := range services.Items {
		err := r.Update(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace})
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
