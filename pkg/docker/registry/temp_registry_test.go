package registry

import (
	"reflect"
	"testing"
)

func TestTempRegistry_SrcRef(t *testing.T) {
	type fields struct {
		port string
	}
	type args struct {
		image string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "no tag or digest",
			fields: fields{
				port: "34567",
			},
			args: args{
				image: "alpine",
			},
			want:    "localhost:34567/alpine:latest",
			wantErr: false,
		},
		{
			name: "tag only",
			fields: fields{
				port: "34567",
			},
			args: args{
				image: "alpine:3.14",
			},
			want:    "localhost:34567/alpine:3.14",
			wantErr: false,
		},
		{
			name: "digest only",
			fields: fields{
				port: "34567",
			},
			args: args{
				image: "alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			want:    "localhost:34567/alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			wantErr: false,
		},
		{
			name: "tag and digest",
			fields: fields{
				port: "34567",
			},
			args: args{
				image: "alpine:3.14@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			want:    "localhost:34567/alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			wantErr: false,
		},
		{
			name: "private image - no tag or digest",
			fields: fields{
				port: "34567",
			},
			args: args{
				image: "quay.io/replicatedcom/alpine",
			},
			want:    "localhost:34567/alpine:latest",
			wantErr: false,
		},
		{
			name: "private image - tag only",
			fields: fields{
				port: "34567",
			},
			args: args{
				image: "quay.io/replicatedcom/alpine:3.14",
			},
			want:    "localhost:34567/alpine:3.14",
			wantErr: false,
		},
		{
			name: "private image - digest only",
			fields: fields{
				port: "34567",
			},
			args: args{
				image: "quay.io/replicatedcom/alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			want:    "localhost:34567/alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			wantErr: false,
		},
		{
			name: "private image - tag and digest",
			fields: fields{
				port: "34567",
			},
			args: args{
				image: "quay.io/replicatedcom/alpine:3.14@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			want:    "localhost:34567/alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TempRegistry{
				port: tt.fields.port,
			}
			gotRef, err := r.SrcRef(tt.args.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("TempRegistry.SrcRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got := gotRef.DockerReference().String()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TempRegistry.SrcRef() = %v, want %v", got, tt.want)
			}
		})
	}
}
