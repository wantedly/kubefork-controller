package middleware

import (
	"context"
	"os"

	"github.com/honeybadger-io/honeybadger-go"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	apiKey := os.Getenv("HONEYBADGER_API_KEY")
	if apiKey == "" {
		return
	}

	appEnv := os.Getenv("APP_ENV")

	honeybadger.Configure(honeybadger.Configuration{
		APIKey: apiKey,
		Env:    appEnv,
	})
}

type hb struct {
	orig reconcile.Reconciler
}

func Honeybadger(r reconcile.Reconciler) reconcile.Reconciler {
	if r == nil {
		return nil
	}
	return &hb{orig: r}
}

func (r *hb) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	res, err := r.orig.Reconcile(ctx, req)
	if err != nil {
		honeybadger.Notify(err)
	}
	return res, err
}
