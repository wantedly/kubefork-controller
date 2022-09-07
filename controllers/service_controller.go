/*
Copyright 2020 Wantedly, Inc..

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
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/pkg/errors"
	"github.com/wantedly/kubefork-controller/domain/updater"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=services,verbs=gverbs=get;list;watch;

func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	if err := r.Get(ctx, req.NamespacedName, &corev1.Service{}); err != nil {
		// since ownerref is correctly set, we don't have to do anything when notfound
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	up := updater.NewVirtualServiceUpdater(r.Client, log.Log, r.Scheme)
	return ctrl.Result{}, errors.WithStack(up.Update(ctx, req.NamespacedName))
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(middleware.Honeybadger(r))
}
