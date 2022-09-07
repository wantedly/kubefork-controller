package updater

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	"github.com/wantedly/kubefork-controller/domain/lister"
	"github.com/wantedly/kubefork-controller/pkg/refresh"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewMicroserviceUpdater returns a Updater that reconciles a Microservice (a set of DeploymentCopies and Services)
func NewMicroserviceUpdater(client client.Client, log logr.Logger, scheme *runtime.Scheme) Updater {
	return &microserviceUpdater{
		client: client,
		log:    log,
		scheme: scheme,
	}
}

type microserviceUpdater struct {
	client client.Client
	log    logr.Logger
	scheme *runtime.Scheme
}

func (r microserviceUpdater) Update(ctx context.Context, forkSlug types.NamespacedName) error {
	// fork is the top of the reference tree
	fork := forkv1beta1.Fork{}
	if err := r.client.Get(ctx, forkSlug, &fork); err != nil {
		// When not found, all child resources should be cleaned with ownerf refs
		return errors.WithStack(client.IgnoreNotFound(err))
	}

	app, err := lister.NewAppBuilder(r.client, fork).Build(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	resourceLists := app.GenerateLists()

	ref := refresh.New(r.client, r.scheme)
	for _, m := range resourceLists {
		if err := ref.Refresh(ctx, &fork, m); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (r microserviceUpdater) UpdateAll(ctx context.Context, opts ...client.ListOption) error {
	forks := &forkv1beta1.ForkList{}
	err := r.client.List(ctx, forks, opts...)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, fork := range forks.Items {
		err := r.Update(ctx, types.NamespacedName{Name: fork.Name, Namespace: fork.Namespace})
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
