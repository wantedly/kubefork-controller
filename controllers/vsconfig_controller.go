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
	"github.com/wantedly/kubefork-controller/pkg/middleware"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	"github.com/wantedly/kubefork-controller/domain/updater"
)

// VSConfigReconciler reconciles a VSConfig object
type VSConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=fork.k8s.wantedly.com,resources=vsconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=fork.k8s.wantedly.com,resources=vsconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=fork.k8s.wantedly.com,resources=vsconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete;deletecollection;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VSConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *VSConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	ns := req.Namespace

	up := updater.NewVirtualServiceUpdater(r.Client, log.Log, r.Scheme)
	instance := forkv1beta1.VSConfig{}

	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, errors.WithStack(err)
		}

		// when notfound, we don't objcets of which resource to be updated, so calling UpdateAll
		return ctrl.Result{}, errors.WithStack(up.UpdateAll(ctx, &client.ListOptions{Namespace: ns}))
	}

	return ctrl.Result{}, errors.WithStack(up.Update(ctx, types.NamespacedName{Namespace: ns, Name: instance.Spec.Host}))
}

// SetupWithManager sets up the controller with the Manager.
func (r *VSConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&forkv1beta1.VSConfig{}).
		Complete(middleware.Honeybadger(r))
}
