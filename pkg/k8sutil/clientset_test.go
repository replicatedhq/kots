package k8sutil

import (
	"testing"

	"k8s.io/apimachinery/pkg/version"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func mockClientsetK8sVersion(expectedMajor string, expectedMinor string) kubernetes.Interface {
	clientset := fake.NewSimpleClientset()
	clientset.Discovery().(*discoveryfake.FakeDiscovery).FakedServerVersion = &version.Info{
		Major: expectedMajor,
		Minor: expectedMinor,
	}
	return clientset
}

func TestGetK8sMinorVersion(t *testing.T) {
	type args struct {
		clientset kubernetes.Interface
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{"expect minor version 22", args{mockClientsetK8sVersion("1", "22")}, 22, false},
		{"expect minor version 21", args{mockClientsetK8sVersion("1", "21")}, 21, false},
		{"expect minor version conversion error", args{mockClientsetK8sVersion("1", "a")}, -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetK8sMinorVersion(tt.args.clientset)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetK8sMinorVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetK8sMinorVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
