package updater

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Updater does the following
//   create or updates a resource to desired state
//   delete outdated resources
type Updater interface {
	Update(ctx context.Context, slug types.NamespacedName) error
	UpdateAll(ctx context.Context, opts ...client.ListOption) error
}
