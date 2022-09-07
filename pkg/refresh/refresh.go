package refresh

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	util "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Builder is responsible of creating on Application
//
// Builder
// - collect information
// - should not contain any logic regarging generating objects
type Builder interface {
	Build(ctx context.Context) (Lister, error)
}

// Lister is responsible of generating ObjectList for each resource
//
// Lister
// - generate objects
// - should not contain any logic accessing API servers
type Lister interface {
	GenerateLists() []ObjectList
}

// Refresher reconciles objects by creating, updating, and deleting
type Refresher interface {
	// Refresh syncs objects
	//
	// each object will be processed as described below
	// - when it doesn't exist, it will be created
	// - when the object exists, it will be updated
	//
	// Also an object that matches all of the conditions below will be deleted
	// - owned by parent
	// - is not present in list
	//
	// when return value from Identify of two objects are the same they are considered to be a same object
	Refresh(ctx context.Context, parent client.Object, list ObjectList) error
}

type refresher struct {
	client client.Client
	scheme *runtime.Scheme
}

type ObjectList struct {
	Items            []client.Object
	GroupVersionKind schema.GroupVersionKind
	Identity         func(client.Object) (string, error)
}

func New(client client.Client, scheme *runtime.Scheme) Refresher {
	return &refresher{client, scheme}
}

func (r refresher) Refresh(ctx context.Context, parent client.Object, list ObjectList) error {
	existingObjs, err := r.handleExisting(ctx, parent, list)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, obj := range list.Items {
		objKey, err := list.Identity(obj)
		if err != nil {
			return errors.WithStack(err)
		}
		emptyObj := reflect.New(reflect.ValueOf(obj).Elem().Type()).Interface().(client.Object)
		emptyObj.SetNamespace(parent.GetNamespace())
		if existing, ok := existingObjs[objKey]; ok {
			emptyObj.SetName(existing.GetName())
		} else {
			if name := obj.GetName(); name != "" {
				if 63 < len(name) {
					name = name[:63]
				}
				emptyObj.SetName(name)
			} else {
				name := createResourceName("%s-%s", objKey, parent.GetName())
				emptyObj.SetName(name)
			}
		}
		if _, err := util.CreateOrUpdate(ctx, r.client, emptyObj, func() error {
			{
				v := emptyObj.GetResourceVersion()
				ns := emptyObj.GetNamespace()
				n := emptyObj.GetName()

				defer func() {
					// to prevent resource version being nil
					emptyObj.SetResourceVersion(v)
					// because we cannot update namespace and name mutateFn
					// we set those back
					emptyObj.SetNamespace(ns)
					emptyObj.SetName(n)
				}()
			}

			// TODO: if the resource already exists and not owned by this, it should return error
			if err := r.scheme.Convert(obj, emptyObj, nil); err != nil {
				return errors.WithStack(err)
			}

			return errors.WithStack(util.SetControllerReference(parent, emptyObj, r.scheme))
		}); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// handleExisting is responsible of two things
// - collect information about existing objects to be updated
// - delete outdated objects
func (r refresher) handleExisting(ctx context.Context, parent client.Object, list ObjectList) (map[string]unstructured.Unstructured, error) {
	desiredIds := sets.String{}
	for _, obj := range list.Items {
		id, err := list.Identity(obj)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		desiredIds.Insert(id)
	}

	// k: identify
	// v: object
	existingMap := map[string]unstructured.Unstructured{}
	{ // Delete outdated resource
		currentObjs := &unstructured.UnstructuredList{}
		currentObjs.SetGroupVersionKind(list.GroupVersionKind)
		if err := r.client.List(ctx, currentObjs, &client.ListOptions{Namespace: parent.GetNamespace()}); err != nil {
			return nil, errors.WithStack(err)
		}
		for _, obj := range currentObjs.Items {
			if !r.ownedByParent(&obj, parent) {
				continue
			}
			id, err := list.Identity(&obj)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			if desiredIds.Has(id) {
				existingMap[id] = obj
				continue
			}

			// reaching here means the object is no longer required, because
			// - it is owned by the parent object we are reconciling
			// - and it's identity is not present in the given list
			if err := r.client.Delete(ctx, &obj); err != nil {
				return nil, errors.WithStack(err)
			}
		}
	}

	return existingMap, nil
}

func (r refresher) ownedByParent(dependent client.Object, parent client.Object) bool {
	parentGVK := parent.GetObjectKind().GroupVersionKind()
	for _, ref := range dependent.GetOwnerReferences() {
		if ref.Name != parent.GetName() {
			continue
		}
		if ref.APIVersion != parentGVK.GroupVersion().String() {
			continue
		}
		if ref.Kind != parentGVK.Kind {
			continue
		}

		return true
	}

	return false
}

func createResourceName(format, first, last string) string {
	name := fmt.Sprintf(format, first, last)
	length := float64(len(name))

	// Margin so as not to exceed 63 characters
	if length < 60 {
		return name
	}
	restLen := length - 60
	firstLen := float64(len(first))
	lastLen := float64(len(last))

	// Calculate the amount of text to be removed to get the same ratio.
	firstPos := int(firstLen - (firstLen / length * restLen))
	lastPos := int(lastLen - (lastLen / length * restLen))
	return fmt.Sprintf(format, first[:firstPos], last[:lastPos])
}
