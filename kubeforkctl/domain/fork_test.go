package domain

import (
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNewSelector(t *testing.T) {
	type args struct {
		labelFromOption []string
		labelFromName   map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    *metav1.LabelSelector
		wantErr bool
	}{
		{
			name: "[Normal] When option form is '<key>=<value>'",
			args: args{
				labelFromOption: []string{"key1=value1"},
				labelFromName:   map[string]string{"key2": "value2"},
			},
			want: &metav1.LabelSelector{
				MatchLabels:      map[string]string{"key1": "value1", "key2": "value2"},
				MatchExpressions: nil,
			},
			wantErr: false,
		},
		{
			name: "[Normal] When option form is '<key>=<value>' and key duplicates",
			args: args{
				labelFromOption: []string{"key1=value1"},
				labelFromName:   map[string]string{"key1": "value3", "key2": "value2"},
			},
			want: &metav1.LabelSelector{
				MatchLabels:      map[string]string{"key1": "value1", "key2": "value2"},
				MatchExpressions: nil,
			},
			wantErr: false,
		},
		{
			name: "[Noraml] When option form is '<key>!=<value>'",
			args: args{
				labelFromOption: []string{"key1!=value1"},
				labelFromName:   map[string]string{"key2": "value2"},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"key2": "value2"},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "key1",
						Operator: "NotIn",
						Values:   []string{"value1"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "[Noraml] When option form is '<key>'",
			args: args{
				labelFromOption: []string{"key1"},
				labelFromName:   map[string]string{"key2": "value2"},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"key2": "value2"},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "key1",
						Operator: "Exists",
						Values:   nil,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "[Noraml] When option form is '!<key>'",
			args: args{
				labelFromOption: []string{"!key1"},
				labelFromName:   map[string]string{"key2": "value2"},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"key2": "value2"},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "key1",
						Operator: "DoesNotExist",
						Values:   nil,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSelector(tt.args.labelFromOption, tt.args.labelFromName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSelector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Return value is mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
