package updater_test

import (
	"context"
	"testing"

	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	"github.com/wantedly/kubefork-controller/domain/updater"
	ut "github.com/wantedly/kubefork-controller/pkg/testing"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	util "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type testcase struct {
	name         string
	explanation  string
	initialState []client.Object
}

func TestUpdate(t *testing.T) {
	scheme := runtime.NewScheme()

	regs := []func(*runtime.Scheme) error{
		clientgoscheme.AddToScheme,
		forkv1beta1.AddToScheme,
		istio.AddToScheme,
	}

	for _, add := range regs {
		if err := add(scheme); err != nil {
			t.Fatal(err)
		}
	}

	setOwner := func(parent, child client.Object) client.Object {
		if err := util.SetControllerReference(parent, child, scheme); err != nil {
			t.Fatal(err)
		}
		return child
	}

	testcases := []testcase{
		{
			name:        "only service",
			explanation: "no related vsconfig is provided, updater doesn't create vs",
			initialState: []client.Object{
				ut.GenService("some-service-name"),
			},
		},
		{
			name:        "only service and virtual service",
			explanation: "no related vsconfig is provided, updater delete vs which is the same name as service",
			initialState: []client.Object{
				ut.GenService("some-service-name"),
				// it has to be owned by the service to delete
				setOwner(ut.GenService("some-service-name"), ut.GenVS("some-service-name", "some-service-name")),
			},
		},
		{
			name:        "one vsconfig",
			explanation: "when a vsconfig exists Updater must reflect the configuration",
			initialState: []client.Object{
				ut.GenService("some-service-name"),
				ut.GenVSConfig("some-service-name", "some-identifier"),
			},
		},
		{
			name:        "multiple vsconfig",
			explanation: "when multiple vsconfigs exist Updater must reflect all of the configuration",
			initialState: []client.Object{
				ut.GenService("some-service-name"),
				ut.GenVSConfig("some-service-name", "some-identifier"),
				ut.GenVSConfig("some-service-name", "another-identifier"),
			},
		},
		{
			name:        "wrong hosts",
			explanation: `Since istio doesn't accept empty string as host the result shouldn't contain "". This test case ensure that updates succeeds even when updating a virtual service with malformed hosts`,
			initialState: []client.Object{
				setOwner(ut.GenService("some-service-name"), ut.GenVS("some-service-name", "")),
				ut.GenService("some-service-name"),
			},
		},
		{
			name:        "empty identifier",
			explanation: `When the headerValue is empty, match will evaluate based on whether or not a header is attached. This test case ensure that updates doesn't create virtual service with empty identifier`,
			initialState: []client.Object{
				ut.GenVSConfig("some-service-name", ""),
				ut.GenVSConfig("some-service-name", "some-identifier"),
				ut.GenService("some-service-name"),
			},
		},
	}

	crossInitialState := []testcase{
		{
			name: "with no additional resources",
		},
		{
			name:        "with empty virtual service",
			explanation: "Update deletes virtual service without vsConfig if its name is same as service",
			initialState: []client.Object{
				ut.GenVS("existing-virtual-service-name", "some-service-name"),
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for _, cr := range crossInitialState {
				cr := cr
				t.Run(cr.name, func(t *testing.T) {
					existingResources := append(tc.initialState, cr.initialState...)
					fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingResources...).Build()
					up := updater.NewVirtualServiceUpdater(fakeClient, ctrl.Log, scheme)

					ctx := context.Background()

					nn := types.NamespacedName{Name: "some-service-name", Namespace: "some-namespace"}
					if err := up.Update(ctx, nn); err != nil {
						t.Fatal(err)
					}

					{
						vsl := &istio.VirtualServiceList{}
						if err := fakeClient.List(ctx, vsl, &client.ListOptions{Namespace: "some-namespace"}); err != nil {
							t.Fatal(err)
						}
						ut.SnapshotYaml(t, vsl)
					}
				})
			}
		})
	}
}

func TestUpdateAll(t *testing.T) {
	scheme := runtime.NewScheme()

	regs := []func(*runtime.Scheme) error{
		clientgoscheme.AddToScheme,
		forkv1beta1.AddToScheme,
		istio.AddToScheme,
	}

	for _, add := range regs {
		if err := add(scheme); err != nil {
			t.Fatal(err)
		}
	}

	setOwner := func(parent, child client.Object) client.Object {
		if err := util.SetControllerReference(parent, child, scheme); err != nil {
			t.Fatal(err)
		}
		return child
	}

	testcases := []testcase{
		{
			name:         "nothing",
			explanation:  "UpdateAll can be performed without any resource.Do nothing for existing resources",
			initialState: []client.Object{},
		},
		{
			name:        "only service",
			explanation: "no related vsconfig is provided, updater doesn't create vs",
			initialState: []client.Object{
				ut.GenService("some-service-name"),
			},
		},
		{
			name:        "one vsconfig",
			explanation: "service and vsconfig existence is the minimal requirement to run create virtual service via Update",
			initialState: []client.Object{
				ut.GenService("some-service-name"),
				ut.GenVSConfig("some-service-name", "some-identifier"),
			},
		},
		{
			name:        "multiple vsconfig",
			explanation: "when multiple vsconfigs exist Updater must reflect all of the configurations",
			initialState: []client.Object{
				ut.GenService("some-service-name"),
				ut.GenVSConfig("some-service-name", "some-identifier"),
				ut.GenVSConfig("some-service-name", "another-identifier"),
			},
		},
	}

	crossInitialState := []testcase{
		{
			name: "with no additional resources",
		},
		{
			name:        "with empty virtual service",
			explanation: "UpdateAll idempotently overrides any existing virtual service or delete it if there is no vsconfig",
			initialState: []client.Object{
				setOwner(ut.GenService("some-service-name"), ut.GenVS("some-service-name", "some-service-name")),
			},
		},
		{
			// TODO: this should create virtual service
			name:        "with empty vsconfig",
			explanation: "When virtual service (`some-random-service` in this case) is missing it does nothing to the vs",
			initialState: []client.Object{
				ut.GenService("some-random-service"), // TODO: without this, UpdateAll returns an error
				ut.GenVSConfig("some-random-service", "some-random-identifire"),
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for _, cr := range crossInitialState {
				cr := cr
				t.Run(cr.name, func(t *testing.T) {
					existingResources := append(tc.initialState, cr.initialState...)

					fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingResources...).Build()
					up := updater.NewVirtualServiceUpdater(fakeClient, ctrl.Log, scheme)

					ctx := context.Background()

					if err := up.UpdateAll(ctx, &client.ListOptions{Namespace: "some-namespace"}); err != nil {
						t.Fatal(err)
					}

					{
						vsl := &istio.VirtualServiceList{}
						if err := fakeClient.List(ctx, vsl, &client.ListOptions{Namespace: "some-namespace"}); err != nil {
							t.Fatal(err)
						}
						ut.SnapshotYaml(t, vsl)
					}
				})
			}
		})
	}
}
