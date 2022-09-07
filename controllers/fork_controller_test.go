package controllers_test

import (
	"context"
	"testing"
	"time"

	ambassador "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	ddv1beta1 "github.com/wantedly/deployment-duplicator/api/v1beta1"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clock "k8s.io/utils/clock/testing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	util "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	"github.com/wantedly/kubefork-controller/controllers"
	ut "github.com/wantedly/kubefork-controller/pkg/testing"
)

type testcase struct {
	name         string
	explanation  string
	initialState []client.Object
}

func TestForkReconciler(t *testing.T) {
	scheme := runtime.NewScheme()

	regs := []func(*runtime.Scheme) error{
		clientgoscheme.AddToScheme,
		forkv1beta1.AddToScheme,
		istio.AddToScheme,
		ambassador.AddToScheme,
		ddv1beta1.AddToScheme,
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

	// This test works as the time is 23:00.
	// pastDate indicates that the deadline is 10 minutes pastDate.
	pastDate, _ := time.Parse(time.RFC3339, "2009-11-10T22:50:00Z")
	// futureDate indicates that the deadline is 10 minutes from now
	futureDate, _ := time.Parse(time.RFC3339, "2009-11-10T23:10:00Z")
	testcases := []testcase{
		{
			name:        "one fork without deadline",
			explanation: "fork without deadline won't be deleted by Reconcile",
			initialState: []client.Object{
				ut.GenFork("some-identifier", nil),
				ut.GenForkManager(),
			},
		},
		{
			name: "one fork",
			initialState: []client.Object{
				ut.GenFork("some-identifier", nil, ut.AddForkDeadline(metav1.NewTime(futureDate))),
				ut.GenForkManager(),
			},
		},
		{
			name: "one fork with HostRewrite",
			initialState: []client.Object{
				ut.GenFork("some-identifier", nil, ut.AddForkDeadline(metav1.NewTime(futureDate))),
				ut.GenForkManagerWithHostRewrite(),
			},
		},
		{
			name:        "old fork",
			explanation: "when a deadline is exceed, Updater must delete Fork resource",
			initialState: []client.Object{
				ut.GenService("some-service-name"),
				ut.GenVSConfig("some-service-name", "some-identifier"),
				ut.GenFork("some-identifier", nil, ut.AddForkDeadline(metav1.NewTime(pastDate))),
				ut.GenForkManager(),
			},
		},
		{
			name:        "old mapping",
			explanation: "when an outdated mapping (another-identifier) resource found, it should be deleted",
			initialState: []client.Object{
				ut.GenFork("some-identifier", nil, ut.AddForkDeadline(metav1.NewTime(futureDate))),
				setOwner(ut.GenForkManager(), ut.GenMapping("another-identifier", "another-identifier.example.com")),
				ut.GenForkManager(),
			},
		},
		{
			name:        "additional headers",
			explanation: "additional headers should be reflected to mappings",
			initialState: []client.Object{
				ut.GenFork("some-identifier", map[string]string{
					"x-some-header-key": "some-header-value", "x-another-header-key": "another-header-value",
				}, ut.AddForkDeadline(metav1.NewTime(futureDate))),
				setOwner(ut.GenForkManager(), ut.GenMapping("another-identifier", "another-identifier.example.com")),
				ut.GenForkManager(),
			},
		},
		{
			name:        "no fork resource",
			explanation: "this means ForkReconciler detected deletion event and it should delete the old mapping",
			initialState: []client.Object{
				ut.GenForkManager(),
			},
		},
		{
			name:        "multiple options merge",
			explanation: "when multiple resources found for an identifer, the options must be merged",
			initialState: []client.Object{
				ut.GenFork("some-identifier", map[string]string{
					"x-header-from-first-fork": "some-header-value",
				}, ut.SetForUpgrades("spdy/3.1")),
				ut.GenFork("some-identifier", map[string]string{
					"x-header-from-second-fork": "another-header-value",
				}, ut.SetForkName("some-identifier-2"), ut.SetForUpgrades("websocket")),
				ut.GenForkManager(),
			},
		},
	}

	crossInitialState := []testcase{
		{
			name: "with no additional resources",
		},
		{
			name: "with a mapping that is not related to fork",
			initialState: []client.Object{
				&ambassador.Mapping{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "getambassador.io",
						Kind:       "Mapping",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mapping-not-related-to-fork",
						Namespace: "ambassador",
					},
					Spec: ambassador.MappingSpec{
						AmbassadorID: []string{"some-other-ambassador-id"},
						Host:         "www.example.com",
						Prefix:       "/",
						Rewrite:      nil,
						Service:      "backend.example.com",
						TimeoutMs:    90000,
					},
				},
			},
		},
		{
			name: "with proper deployment and service",
			explanation: `
			In this case, a deploymencopy should be made because
			 - pods managed by "some-deployment" is routable from "service-for-some-deployment"
			 - "service-for-some-deployment" is selected by service selector app=some-app
			`,
			initialState: []client.Object{
				ut.GenDeployment("some-deployment", map[string]string{"app": "some-app", "role": "web"}),
				ut.GenService("service-for-some-deployment", ut.AddSVCLabel("app", "some-app")),
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

					// this time is just golang's birthday
					now, _ := time.Parse(time.RFC3339, "2009-11-10T23:00:00Z")
					fakeClock := clock.NewFakeClock(now)

					rec := controllers.ForkReconciler{Client: fakeClient, Scheme: scheme, Clock: fakeClock}

					ctx := context.Background()

					nn := types.NamespacedName{Name: "some-identifier", Namespace: "some-namespace"}
					req := ctrl.Request{NamespacedName: nn}
					if _, err := rec.Reconcile(ctx, req); err != nil {
						t.Fatalf("%+v", err)
					}

					{
						lists := []client.ObjectList{
							&ambassador.MappingList{},
							&ddv1beta1.DeploymentCopyList{},
							&corev1.ServiceList{},
							&forkv1beta1.VSConfigList{},
							&forkv1beta1.ForkList{},
						}

						for _, ls := range lists {
							if err := fakeClient.List(ctx, ls); err != nil {
								t.Fatalf("%+v", err)
							}
						}

						ifs := make([]interface{}, len(lists))
						for i, ls := range lists {
							ifs[i] = ls
						}
						ut.SnapshotYaml(t, ifs...)
					}
				})
			}
		})
	}
}
