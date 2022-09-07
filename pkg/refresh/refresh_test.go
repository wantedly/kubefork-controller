package refresh_test

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/wantedly/kubefork-controller/pkg/refresh"
	ut "github.com/wantedly/kubefork-controller/pkg/testing"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	util "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type testcase struct {
	name         string
	explanation  string
	initialState []client.Object
	objectList   refresh.ObjectList
}

func TestRefresh(t *testing.T) {
	scheme := runtime.NewScheme()

	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	parent := ut.GenService("some-svc")

	setOwner := func(child client.Object) client.Object {
		if err := util.SetControllerReference(parent, child, scheme); err != nil {
			t.Fatal(err)
		}
		return child
	}
	const labelKey = "some-label-key"

	labelGetter := func(obj client.Object) (string, error) {
		key, ok := obj.GetLabels()[labelKey]
		if !ok {
			return "", errors.Errorf("service doesn't have label %q", labelKey)
		}
		return key, nil
	}
	gvk, err := apiutil.GVKForObject(&appsv1.DeploymentList{}, scheme)
	if err != nil {
		t.Fatal(err)
	}

	testcases := []testcase{
		{
			name:         "empty state",
			explanation:  "When both desired and cluster stat are empty it does nothing",
			initialState: nil,
			objectList: refresh.ObjectList{
				Items:            nil,
				GroupVersionKind: gvk,
				Identity:         labelGetter,
			},
		},
		{
			name:        "one existing resource",
			explanation: "when objects with the same kind detected, it doesn't affect the existings",
			initialState: []client.Object{
				ut.GenDeployment("deploy-1", map[string]string{labelKey: "1"}),
			},
			objectList: refresh.ObjectList{
				Items: []client.Object{
					ut.GenDeployment("", map[string]string{labelKey: "2"}),
				},
				GroupVersionKind: gvk,
				Identity:         labelGetter,
			},
		},
		{
			name:        "one existing resource owned by parent",
			explanation: "when the resource is owned by the parent and not listed in the ObjectList, it deletes the object",
			initialState: []client.Object{
				setOwner(ut.GenDeployment("deploy-1", map[string]string{labelKey: "1"})),
			},
			objectList: refresh.ObjectList{
				Items: []client.Object{
					ut.GenDeployment("", map[string]string{labelKey: "2"}),
				},
				GroupVersionKind: gvk,
				Identity:         labelGetter,
			},
		},
		{
			name:        "one existing resource owned by parent to be updated",
			explanation: "when old object found with the same identity fonud, it updates respecting current object name",
			initialState: []client.Object{
				setOwner(ut.GenDeployment("deploy-1", map[string]string{
					labelKey:                     "1",
					"this-key-should-be-updated": "this-is-before-update",
				})),
			},
			objectList: refresh.ObjectList{
				Items: []client.Object{
					ut.GenDeployment("", map[string]string{labelKey: "2"}),
					ut.GenDeployment("", map[string]string{
						labelKey:                     "1", // because of this, deploy-1 will be updated
						"this-key-should-be-updated": "this-is-after-update",
					}),
				},
				GroupVersionKind: gvk,
				Identity:         labelGetter,
			},
		},
		{
			name:         "if object has long name",
			explanation:  "Names longer than 63 characters will need to be trimmed.",
			initialState: nil,
			objectList: refresh.ObjectList{
				Items: []client.Object{
					ut.GenDeployment("random-70-character-name-cmFuZG9tLTcwLWNoYXJhY3Rlci1uYW1lCci1uYW1lC==", map[string]string{labelKey: "1"}),
					ut.GenDeployment("", map[string]string{labelKey: "random-70-character-objKey-cmFuZG9tLTcwLWNoYXJhY3Rlci1uYW1lCci1uYW1lC"}),
				},
				GroupVersionKind: gvk,
				Identity:         labelGetter,
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			objs := append(tc.initialState, parent)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			ref := refresh.New(fakeClient, scheme)

			ctx := context.Background()

			if err := ref.Refresh(ctx, parent, tc.objectList); err != nil {
				t.Fatalf("%+v", err)
			}

			{
				dl := &appsv1.DeploymentList{}
				if err := fakeClient.List(ctx, dl); err != nil {
					t.Fatalf("%+v", err)
				}
				ut.SnapshotYaml(t, dl)
			}
		})
	}
}
