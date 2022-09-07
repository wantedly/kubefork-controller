package application_test

import (
	"testing"

	application "github.com/wantedly/kubefork-controller/domain/lister/internal"
	ut "github.com/wantedly/kubefork-controller/pkg/testing"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestVSConfigIdentityWithUnstructured(t *testing.T) {
	vsc := ut.GenVSConfig("some-host", "some-identifier")
	unst, err := runtime.DefaultUnstructuredConverter.ToUnstructured(vsc)
	if err != nil {
		t.Error(err)
		return
	}

	obj := unstructured.Unstructured{
		Object: unst,
	}

	str, err := application.VSConfigIdentity(&obj)
	if err != nil {
		t.Error(err)
		return
	}

	expected := "some-host"
	if str != expected {
		t.Errorf("expected: %q, got: %q", expected, str)
	}
}
