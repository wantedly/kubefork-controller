package application_test

import (
	"context"
	"testing"

	ddv1beta1 "github.com/wantedly/deployment-duplicator/api/v1beta1"
	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	application "github.com/wantedly/kubefork-controller/domain/lister/internal"
	ut "github.com/wantedly/kubefork-controller/pkg/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type testcase struct {
	name         string
	explanation  string
	initialState []client.Object
	replicas     *int32
}

func TestBuild(t *testing.T) {
	scheme := runtime.NewScheme()

	{
		regs := []func(*runtime.Scheme) error{
			clientgoscheme.AddToScheme,
			forkv1beta1.AddToScheme,
			ddv1beta1.AddToScheme,
		}

		for _, add := range regs {
			if err := add(scheme); err != nil {
				t.Fatal(err)
			}
		}
	}

	routableLabel := map[string]string{"app": "some-app", "role": "web", "fork-target-in-this-test": "true"}
	testcases := []testcase{
		{
			name:         "empty state",
			explanation:  "When empty no resource will be created",
			initialState: nil,
			replicas:     nil,
		},
		{
			name:         "empty state with 1 replicas",
			explanation:  "When empty no resource will be created even replicas is specified",
			initialState: nil,
			replicas:     intPointer(1),
		},
		{
			name:        "one existing resource",
			explanation: "when objects with the same kind detected, it doesn't affect the existings",
			initialState: []client.Object{
				ut.GenService("service-not-forked"),
				ut.GenService("service-1", ut.AddSVCLabel("fork-target-in-this-test", "true")),
				ut.GenService("service-2", ut.AddSVCLabel("fork-target-in-this-test", "true")),
				ut.GenDeployment("deploy-1", routableLabel), // routable from service-1
				ut.GenDeployment("deploy-2", routableLabel), // routable from service-1
				ut.GenDeployment("deploy-not-forked", nil),  // not routable from service-1
			},
			replicas: nil,
		},
		{
			name:        "one existing resource with default replicas",
			explanation: "builder will create deploymentcopy with 1 replicas by default",
			initialState: []client.Object{
				ut.GenService("service-not-forked"),
				ut.GenService("service-1", ut.AddSVCLabel("fork-target-in-this-test", "true")),
				ut.GenService("service-2", ut.AddSVCLabel("fork-target-in-this-test", "true")),
				ut.GenDeployment("deploy-1", routableLabel), // routable from service-1
				ut.GenDeployment("deploy-2", routableLabel), // routable from service-1
				ut.GenDeployment("deploy-not-forked", nil),  // not routable from service-1
			},
			replicas: nil,
		},
		{
			name:        "one existing resource with one replicas",
			explanation: "builder should create deploymentcopy with 1 replicas by fork resource",
			initialState: []client.Object{
				ut.GenService("service-1", ut.AddSVCLabel("fork-target-in-this-test", "true")),
				ut.GenDeployment("deploy-1", routableLabel), // routable from service-1
			},
			replicas: intPointer(1),
		},
		{
			name:        "one existing resource with two replicas",
			explanation: "builder should create deploymentcopy with 2 replicas by fork resource",
			initialState: []client.Object{
				ut.GenService("service-1", ut.AddSVCLabel("fork-target-in-this-test", "true")),
				ut.GenDeployment("deploy-1", routableLabel), // routable from service-1
			},
			replicas: intPointer(2),
		},
	}

	fork := forkv1beta1.Fork{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vsconfig.k8s.wantedly.com/v1beta1",
			Kind:       "VSConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-fork",
			Namespace: "some-namespace",
		},
		Spec: forkv1beta1.ForkSpec{
			Identifier: "some-identifier",
			Manager:    "ambassador/default",
			GatewayOptions: &forkv1beta1.GatewayOptions{
				AddRequestHeaders: nil,
			},
			Services: &forkv1beta1.ForkService{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"fork-target-in-this-test": "true"},
				},
			},
			Deployments: &forkv1beta1.ForkDeployment{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"fork-target-in-this-test": "true"},
				},
				Template: &forkv1beta1.PodTemplateSpec{
					ObjectMeta: &metav1.ObjectMeta{
						Labels: map[string]string{
							"some-label-added-to-copied-deployment": "true",
						},
						Annotations: map[string]string{
							"some-annotation-added-to-copied-deployment": "true",
						},
					},
				},
			},
		},
	}

	forkManager := ut.GenForkManager()

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			objs := append(tc.initialState, forkManager)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()

			fork.Spec.Deployments.Replicas = tc.replicas
			builder := application.NewBuilder(fakeClient, fork)
			ctx := context.Background()
			app, err := builder.Build(ctx)
			if err != nil {
				t.Fatalf("%+v", err)
			}
			lists := app.GenerateLists()

			snapshotTargets := make([]interface{}, len(lists))
			for i, ls := range lists {
				snapshotTargets[i] = map[string]interface{}{
					"items":            ls.Items,
					"GroupVersionKind": ls.GroupVersionKind,
				}
			}

			ut.SnapshotYaml(t, snapshotTargets...)
		})
	}

}

func intPointer(i int32) *int32 {
	return &i
}
