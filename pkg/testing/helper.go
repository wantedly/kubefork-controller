package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	forkv1beta1 "github.com/wantedly/kubefork-controller/api/v1beta1"
	"log"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	ambassador "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
	"github.com/stuart-warren/yamlfmt"
	"gopkg.in/yaml.v2"
	networkingv1beta1 "istio.io/api/networking/v1beta1"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

type Resource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              interface{}
	Status            interface{}
}

type svcConfigOption func(*corev1.Service)
type forkConfigOption func(*forkv1beta1.Fork)

func AddSVCAnnotation(key, value string) svcConfigOption {
	return func(vsc *corev1.Service) {
		if vsc.Annotations == nil {
			vsc.Annotations = map[string]string{}
		}

		vsc.Annotations[key] = value
	}
}

func AddSVCLabel(key, value string) svcConfigOption {
	return func(svc *corev1.Service) {
		if svc.Labels == nil {
			svc.Labels = map[string]string{}
		}

		svc.Labels[key] = value
	}
}

func AddForkDeadline(time metav1.Time) forkConfigOption {
	return func(fork *forkv1beta1.Fork) {
		fork.Spec.Deadline = &time
	}
}

func SetForkName(name string) forkConfigOption {
	return func(fork *forkv1beta1.Fork) {
		fork.Name = name
	}
}

func SetForUpgrades(protocol string) forkConfigOption {
	return func(fork *forkv1beta1.Fork) {
		if fork.Spec.GatewayOptions == nil {
			fork.Spec.GatewayOptions = &forkv1beta1.GatewayOptions{}
		}
		fork.Spec.GatewayOptions.AllowUpgrade = append(fork.Spec.GatewayOptions.AllowUpgrade, protocol)
	}
}

func GenService(name string, opts ...svcConfigOption) *corev1.Service {
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "some-namespace",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					Name:       "http",
					Protocol:   "TCP",
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8081},
				},
			},
			Selector: map[string]string{
				"app":  "some-app",
				"role": "web",
			},
			Type: "ClusterIP",
		},
	}
	for _, opt := range opts {
		opt(svc)
	}

	return svc
}

func GenVSConfig(targetServiceName, identifier string) *forkv1beta1.VSConfig {
	vsc := &forkv1beta1.VSConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vsconfig.k8s.wantedly.com/v1beta1",
			Kind:       "VSConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", targetServiceName, identifier),
			Namespace: "some-namespace",
		},
		Spec: forkv1beta1.VSConfigSpec{
			Host:        targetServiceName,
			Service:     "custom-routing-service-name",
			HeaderName:  "some-header-name",
			HeaderValue: identifier,
		},
	}
	return vsc
}

func GenVS(name string, host string) *istio.VirtualService {
	vs := &istio.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1beta1",
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "some-namespace",
		},
		Spec: networkingv1beta1.VirtualService{
			Hosts: []string{host},
		},
	}
	return vs
}

func GenFork(name string, additionalHeaders map[string]string, opts ...forkConfigOption) *forkv1beta1.Fork {
	fork := &forkv1beta1.Fork{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vsconfig.k8s.wantedly.com/v1beta1",
			Kind:       "Fork",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "some-namespace",
		},
		Spec: forkv1beta1.ForkSpec{
			Identifier: name,
			Manager:    "ambassador/default",
		},
	}
	if len(additionalHeaders) != 0 {
		fork.Spec.GatewayOptions = &forkv1beta1.GatewayOptions{
			AddRequestHeaders: additionalHeaders,
		}
	}
	for _, opt := range opts {
		opt(fork)
	}
	return fork
}

func GenForkManager() *forkv1beta1.ForkManager {
	return &forkv1beta1.ForkManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "ambassador",
		},
		Spec: forkv1beta1.ForkManagerSpec{
			AmbassadorID: "ambassador",
			Upstreams: []forkv1beta1.Upstream{
				{Host: "sandbox.example.com"},
				{Host: "some-with-original.example.com", Original: "some-with-original.some-namespace:443"},
			},
			HeaderKey: "fork-identifier",
		},
	}
}

func GenForkManagerWithHostRewrite() *forkv1beta1.ForkManager {
	return &forkv1beta1.ForkManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "ambassador",
		},
		Spec: forkv1beta1.ForkManagerSpec{
			AmbassadorID: "ambassador",
			Upstreams: []forkv1beta1.Upstream{
				{Host: "sandbox.example.com"},
				{Host: "some-with-original.example.com", Original: "some-with-original.some-namespace:443", HostRewrite: "some-with-original.example.com"},
			},
			HeaderKey: "fork-identifier",
		},
	}
}

func GenMapping(name, host string) *ambassador.Mapping {
	ambassadorId := "ambassador"
	headerKey := "fork-identifier"

	return &ambassador.Mapping{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "getambassador.io",
			Kind:       "Mapping",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "ambassador",
			Labels: map[string]string{
				"fork.k8s.wantedly.com/manager": "default",
			},
		},
		Spec: ambassador.MappingSpec{
			AddRequestHeaders: map[string]ambassador.AddedHeader{
				headerKey: {String: &name},
			},
			AmbassadorID: []string{ambassadorId},
			Host:         host,
			Prefix:       "/",
			Rewrite:      pointer.StringPtr(""),
			Service:      fmt.Sprintf("https://%s", name),
			TimeoutMs:    90000,
		},
	}
}

func GenDeployment(name string, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "some-namespace",
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "some-deployment",
							Image: "some-deployment:some-commit-sha",
						},
					},
				},
			},
		},
	}
}

func SnapshotYaml(t *testing.T, objs ...interface{}) {
	t.Helper()

	manifests := make([]string, len(objs))

	for i, obj := range objs {
		// struct to map
		rs := make(map[string]interface{})
		{ // Marshal into json string to omit unused fields
			jsnBytes, err := json.Marshal(obj)
			if err != nil {
				t.Fatal(err)
			}

			err = json.Unmarshal(jsnBytes, &rs)
			if err != nil {
				t.Fatal(err)
			}
		}

		{
			query, err := gojq.Parse(`if has("items") and .items != null then .items = (.items | sort_by(.metadata.name) | reverse | map(del(.metadata.resourceVersion))) else . end`)
			if err != nil {
				log.Fatalln(err)
			}
			iter := query.Run(rs)
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				t.Error(err)
			}
			rs = v.(map[string]interface{})
		}

		// map to formatted yaml
		var formatted string
		{
			d, err := yaml.Marshal(&rs)
			if err != nil {
				t.Fatal(err)
			}

			formatted, err = format(d)
			if err != nil {
				t.Fatal(err)
			}
		}
		manifests[i] = formatted
	}

	recorder := cupaloy.New(cupaloy.SnapshotFileExtension(".yaml"))
	recorder.SnapshotT(t, strings.Join(manifests, "\n"))
}

func format(content []byte) (string, error) {
	bs, err := yamlfmt.Format(bytes.NewReader(content))

	if err != nil {
		return "", errors.Wrap(err, "failed to format yaml")
	}
	return string(bs), nil
}
