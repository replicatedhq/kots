package embeddedcluster

import (
	"context"
	"testing"
)

func x_Test_pullArtifact(t *testing.T) {
	type args struct {
		ctx     context.Context
		srcRepo string
		opts    pullArtifactOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				ctx:     context.Background(),
				srcRepo: "ttl.sh/ethan/embedded-cluster-operator-bin:24h",
				opts:    pullArtifactOptions{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			t.Log(tmpdir)
			if err := pullArtifact(tt.args.ctx, tt.args.srcRepo, tmpdir, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("pullArtifact() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
