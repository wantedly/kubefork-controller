package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type forkWatcher struct {
	Client   client.Reader
	channel  chan event.GenericEvent
	Log      logr.Logger
	TickTime time.Duration
}

func (c forkWatcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(c.TickTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			now := metav1.NewTime(time.Now())
			forkList := &forkv1beta1.ForkList{}
			if err := c.Client.List(ctx, forkList); err != nil {
				return errors.Wrap(err, "failed to get forkList")
			}

			for _, fork := range forkList.Items {
				// skip fork which doesn't have deadline
				if fork.Spec.Deadline == nil {
					continue
				}
				// skip fork which is not outdated
				if fork.Spec.Deadline.After(now.Time) {
					continue
				}
				// notify Reconciler with outdated fork
				c.channel <- event.GenericEvent{
					Object: &fork,
				}
			}

		}
	}
}
