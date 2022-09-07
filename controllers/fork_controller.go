/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/pkg/errors"
	"github.com/wantedly/kubefork-controller/domain/updater"
	"github.com/wantedly/kubefork-controller/pkg/middleware"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
)

// ForkReconciler reconciles a Fork object
type ForkReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Clock  clock.WithTickerAndDelayedExecution
}

// Input resources
// +kubebuilder:rbac:groups=fork.k8s.wantedly.com,resources=forks;forkmanagers,verbs=get;list;watch;delete
//
// Output resources
// +kubebuilder:rbac:groups=getambassador.io,resources=mappings,verbs=get;list;watch;create;update;patch;delete;deletecollection;
// +kubebuilder:rbac:groups=duplication.k8s.wantedly.com,resources=deploymentcopies,verbs=get;list;watch;create;update;patch;delete;deletecollection;
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;
// +kubebuilder:rbac:groups="",resources=services,verbs=create;update;delete;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Fork object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ForkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	up := updater.NewMappingUpdater(r.Client, log.Log, r.Scheme)

	frk := &forkv1beta1.Fork{}
	forkSlug := types.NamespacedName{Namespace: req.Namespace, Name: req.Name}
	{
		if err := r.Get(ctx, forkSlug, frk); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, errors.WithStack(up.UpdateAll(ctx))
			}
			return ctrl.Result{}, errors.WithStack(err)
		}
	}

	now := v1.NewTime(r.Clock.Now())
	// Remove fork resources that exceed the deadline
	if frk.Spec.Deadline != nil && frk.Spec.Deadline.Before(&now) {
		return ctrl.Result{}, errors.WithStack(r.Delete(ctx, frk))
	}

	{ // update mapping
		slugParts := strings.Split(frk.Spec.Manager, "/")
		if len(slugParts) != 2 {
			return ctrl.Result{}, errors.New("malformed field `manager`")
		}

		nn := types.NamespacedName{Namespace: slugParts[0], Name: slugParts[1]}
		if err := up.Update(ctx, nn); err != nil {
			return ctrl.Result{}, errors.WithStack(err)
		}
	}

	{ // update deployment and service
		mup := updater.NewMicroserviceUpdater(r.Client, log.Log, r.Scheme)
		if err := mup.Update(ctx, forkSlug); err != nil {
			return ctrl.Result{}, errors.WithStack(err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *ForkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	watcher, err := r.SetupForkWatcher(mgr)
	if err != nil {
		return errors.WithStack(err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&forkv1beta1.Fork{}).
		Watches(watcher, &handler.EnqueueRequestForObject{}).
		Complete(middleware.Honeybadger(r))
}

func (r *ForkReconciler) SetupForkWatcher(mgr ctrl.Manager) (*source.Channel, error) {
	if r.Clock == nil {
		r.Clock = clock.RealClock{}
	}
	ch := make(chan event.GenericEvent)
	watcher := forkWatcher{
		Client:   r.Client,
		channel:  ch,
		Log:      log.Log,
		TickTime: 1 * time.Minute,
	}
	if err := mgr.Add(watcher); err != nil {
		return nil, errors.Wrap(err, "failed to add fork watcher")
	}
	src := source.Channel{
		Source: watcher.channel,
	}
	return &src, nil
}
