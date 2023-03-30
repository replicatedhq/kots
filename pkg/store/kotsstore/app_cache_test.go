package kotsstore

import (
	"reflect"
	"testing"
	"time"

	apptypes "github.com/replicatedhq/kots/pkg/app/types"
)

func TestSetCachedApp(t *testing.T) {
	tests := []struct {
		name    string
		app     *apptypes.App
		want    *apptypes.App
		wantErr bool
	}{
		{
			name: "set app to cache",
			app: &apptypes.App{
				ID: "test-app",
			},
			want: &apptypes.App{
				ID: "test-app",
			},
			wantErr: false,
		},
		{
			name:    "set nil app to cache",
			app:     nil,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kotsStore := StoreFromEnv()
			err := kotsStore.SetCachedApp(tt.app)

			if (err != nil) != tt.wantErr {
				t.Errorf("KOTSStore.SetCachedApp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.app == nil {
				return
			}

			got := kotsStore.GetAppFromCache(tt.app.ID)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KOTSStore.SetCachedApp() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestGetAppFromCache(t *testing.T) {
	kotsStore := &KOTSStore{
		cachedApp: map[string]*cachedApp{
			"expired-app": {
				app: &apptypes.App{
					ID: "expired-app",
				},
				expirationTime: time.Now().Add(-time.Minute),
			},
			"test-app": {
				app: &apptypes.App{
					ID: "test-app",
				},
				expirationTime: time.Now().Add(time.Minute),
			},
		},
	}

	tests := []struct {
		name string
		app  *apptypes.App
		want *apptypes.App
	}{
		{
			name: "app is not in cache",
			app: &apptypes.App{
				ID: "does-not-exist",
			},
			want: nil,
		},
		{
			name: "get app from cache",
			app: &apptypes.App{
				ID: "test-app",
			},
			want: &apptypes.App{
				ID: "test-app",
			},
		},
		{
			name: "get app from cache past the expiration time",
			app: &apptypes.App{
				ID: "expired-app",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := kotsStore.GetAppFromCache(tt.app.ID)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KOTSStore.GetAppFromCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
