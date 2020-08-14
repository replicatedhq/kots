package base

import (
	"testing"
)

func Test_isExcludedByAnnotation(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		want        bool
		wantErr     bool
	}{
		{
			name: "kots.io/exclude=true",
			annotations: map[string]string{
				"kots.io/exclude": "true",
			},
			want: true,
		},
		{
			name: "kots.io/exclude=false",
			annotations: map[string]string{
				"kots.io/exclude": "false",
			},
			want: false,
		},
		{
			name: "kots.io/when=true",
			annotations: map[string]string{
				"kots.io/when": "true",
			},
			want: false,
		},
		{
			name: "kots.io/when=false",
			annotations: map[string]string{
				"kots.io/when": "false",
			},
			want: true,
		},
		{
			name: "kots.io/exclude=error, kots.io/when=true",
			annotations: map[string]string{
				"kots.io/exclude": "error",
				"kots.io/when":    "true",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "kots.io/exclude=error, kots.io/when=false",
			annotations: map[string]string{
				"kots.io/exclude": "error",
				"kots.io/when":    "false",
			},
			want:    true,
			wantErr: true,
		},
		{
			name: "kots.io/exclude=true, kots.io/when=error",
			annotations: map[string]string{
				"kots.io/exclude": "true",
				"kots.io/when":    "error",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "kots.io/exclude=false, kots.io/when=error",
			annotations: map[string]string{
				"kots.io/exclude": "false",
				"kots.io/when":    "error",
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isExcludedByAnnotation(tt.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("isExcludedByAnnotation() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("isExcludedByAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isExcludedByAnnotationCompat(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]interface{}
		want        bool
		wantErr     bool
	}{
		{
			name: "kots.io/exclude=true",
			annotations: map[string]interface{}{
				"kots.io/exclude": true,
			},
			want: true,
		},
		{
			name: "kots.io/exclude=false",
			annotations: map[string]interface{}{
				"kots.io/exclude": false,
			},
			want: false,
		},
		{
			name: "kots.io/when=true",
			annotations: map[string]interface{}{
				"kots.io/when": true,
			},
			want: false,
		},
		{
			name: "kots.io/when=false",
			annotations: map[string]interface{}{
				"kots.io/when": false,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isExcludedByAnnotationCompat(tt.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("isExcludedByAnnotationCompat() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("isExcludedByAnnotationCompat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hasKotsHookEvents(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		want        bool
	}{
		{
			name: "single",
			annotations: map[string]string{
				"other-annotation": "true",
				"kots.io/hook":     "post-install",
			},
			want: true,
		},
		{
			name: "multiple",
			annotations: map[string]string{
				"other-annotation": "true",
				"kots.io/hook":     "pre-install,post-install",
			},
			want: true,
		},
		{
			name: "none",
			annotations: map[string]string{
				"other-annotation": "true",
			},
			want: false,
		},
		{
			name: "empty",
			annotations: map[string]string{
				"other-annotation": "true",
				"kots.io/hook":     "",
			},
			want: false,
		},
		{
			name: "unknown",
			annotations: map[string]string{
				"other-annotation": "true",
				"kots.io/hook":     "blah",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasKotsHookEvents(tt.annotations); got != tt.want {
				t.Errorf("hasKotsHookEvents() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseFile_ShouldBeIncludedInBaseKustomizationAnnotations(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
		wantErr bool
	}{
		{
			name: `kots.io/exclude="true"`,
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kotsadm
  annotations:
    kots.io/exclude: "true"`,
			want: false,
		},
		{
			name: `kots.io/exclude="false"`,
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kotsadm
  annotations:
    kots.io/exclude: "false"`,
			want: true,
		},
		{
			name: `kots.io/exclude=true`,
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kotsadm
  annotations:
    kots.io/exclude: true`,
			want: false,
		},
		{
			name: `kots.io/exclude=false`,
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kotsadm
  annotations:
    kots.io/exclude: false`,
			want:    false,
			wantErr: true, // nothing i can do about this one. it will fail eventually
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := BaseFile{
				Path:    "test.yaml",
				Content: []byte(tt.content),
			}
			got, err := f.ShouldBeIncludedInBaseKustomization(true, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("BaseFile.ShouldBeIncludedInBaseKustomization() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BaseFile.ShouldBeIncludedInBaseKustomization() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseFile_ShouldBeIncludedInBaseKustomizationKotskinds(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
		wantErr bool
	}{
		{
			name: `kots.io/exclude=true`,
			content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: myapp
  annotations:
    kots.io/exclude: true`,
			want: false,
		},
		{
			name: `kots.io/exclude=false`,
			content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: myapp
  annotations:
    kots.io/exclude: false`,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := BaseFile{
				Path:    "test.yaml",
				Content: []byte(tt.content),
			}
			got, err := f.ShouldBeIncludedInBaseKustomization(true, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("BaseFile.ShouldBeIncludedInBaseKustomization() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BaseFile.ShouldBeIncludedInBaseKustomization() = %v, want %v", got, tt.want)
			}
		})
	}
}
