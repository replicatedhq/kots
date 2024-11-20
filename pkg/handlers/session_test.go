package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/kots/pkg/session"
	"github.com/replicatedhq/kots/pkg/session/types"
	"github.com/replicatedhq/kots/pkg/store"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/stretchr/testify/require"
)

var (
	want = new(types.Session)
)

func signJWT(t *testing.T, sess *types.Session) string {
	signedJWT, err := session.SignJWT(sess)
	if err != nil {
		t.Error(err)
	}
	return signedJWT
}

func Test_requireValidSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	sess := &types.Session{
		ID:        "session-id",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(12 * time.Hour),
	}
	sessionJWT := signJWT(t, sess)

	tokenCookie := http.Cookie{
		Name:     "signed-token",
		Value:    fmt.Sprintf("Bearer %v", sessionJWT),
		HttpOnly: true,
	}

	mockStore.EXPECT().GetSession(sess.ID).Return(sess, nil).MaxTimes(2)
	mockStore.EXPECT().GetPasswordUpdatedAt().Return(nil, nil).MaxTimes(2)
	type args struct {
		kotsStore store.Store
		w         http.ResponseWriter
		r         *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    *types.Session
		wantErr bool
	}{
		{
			name: "HTTP Method OPTIONS should return session: nil, error: false",
			args: args{
				r: httptest.NewRequest("OPTIONS", "http://test.com", nil),
				w: httptest.NewRecorder(),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "invalid session should return session: nil, error: true",
			args: args{
				r: httptest.NewRequest("GET", "http://test.com", nil),
				w: httptest.NewRecorder(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid auth header session should return invalid token error - session: nil, error: true",
			args: args{
				r: &http.Request{
					Header: http.Header{
						"Authorization": []string{"Bearer invalid"},
					},
				},
				w: httptest.NewRecorder(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid token cookie session should return invalid token error - session: nil, error: true",
			args: args{
				r: &http.Request{
					Header: http.Header{
						"Cookie": []string{fmt.Sprintf("%s=%s", tokenCookie.Name, "Bearer invalid")},
					},
				},
				w: httptest.NewRecorder(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid session should return session: session, error: false",
			args: args{
				kotsStore: mockStore,
				r: &http.Request{
					Header: http.Header{
						"Authorization": []string{fmt.Sprintf("Bearer %v", sessionJWT)},
					},
				},
				w: httptest.NewRecorder(),
			},
			want:    sess,
			wantErr: false,
		},
		{
			name: "valid session using cookie should return session: session, error: false",
			args: args{
				kotsStore: mockStore,
				r: &http.Request{
					Header: http.Header{
						"Cookie": []string{fmt.Sprintf("%s=%s", tokenCookie.Name, tokenCookie.Value)},
					},
				},
				w: httptest.NewRecorder(),
			},
			want:    sess,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := requireValidSession(tt.args.kotsStore, tt.args.w, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("requireValidSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("requireValidSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_requireValidSession_emptySession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	emptySession := &types.Session{
		ID:        "session-id",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(12 * time.Hour),
	}
	emptySessionJWT := signJWT(t, emptySession)
	tokenCookie := http.Cookie{
		Name:     "signed-token",
		Value:    fmt.Sprintf("Bearer %v", emptySessionJWT),
		HttpOnly: true,
	}

	mockStore.EXPECT().GetSession(emptySession.ID).Return(nil, nil).Times(2)
	type args struct {
		kotsStore store.Store
		w         *httptest.ResponseRecorder
		r         *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    *types.Session
		wantErr bool
	}{
		{
			name: "empty session from auth header should return session: nil, error: true",
			args: args{
				kotsStore: mockStore,
				w:         httptest.NewRecorder(),
				r: &http.Request{
					Header: http.Header{
						"Authorization": []string{fmt.Sprintf("Bearer %v", emptySessionJWT)},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty session from auth cookie should return session: nil, error: true",
			args: args{
				kotsStore: mockStore,
				w:         httptest.NewRecorder(),
				r: &http.Request{
					Header: http.Header{
						"Cookie": []string{fmt.Sprintf("%s=%s", tokenCookie.Name, tokenCookie.Value)},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := requireValidSession(tt.args.kotsStore, tt.args.w, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("requireValidSession() emptySession error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			req := require.New(t)
			req.Equal("no session in auth header", err.Error())
			req.Equal(tt.want, got)
			req.Equal(401, tt.args.w.Code)
		})
	}
}

func Test_requireValidSession_expiredSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	expiredSession := &types.Session{
		ID:        "session-id",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	expiredSessionJWT := signJWT(t, expiredSession)
	expiredTokenCookie := http.Cookie{
		Name:     "signed-token",
		Value:    fmt.Sprintf("Bearer %v", expiredSessionJWT),
		HttpOnly: true,
	}

	w := httptest.NewRecorder()
	r := &http.Request{
		Header: http.Header{
			"Cookie": []string{fmt.Sprintf("%s=%s", expiredTokenCookie.Name, expiredTokenCookie.Value)},
		},
	}

	mockStore.EXPECT().GetSession(expiredSession.ID).Return(expiredSession, nil)
	mockStore.EXPECT().DeleteSession(expiredSession.ID).Return(nil)
	want = nil
	req := require.New(t)
	got, err := requireValidSession(mockStore, w, r)
	req.Error(err)
	req.Equal("session expired", err.Error())
	req.Equal(want, got)
	req.Equal(401, w.Code)
}

func Test_requireValidSession_expiredSession_withDeleteErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	expiredSession := &types.Session{
		ID:        "session-id",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	expiredSessionJWT := signJWT(t, expiredSession)
	expiredTokenCookie := http.Cookie{
		Name:     "signed-token",
		Value:    fmt.Sprintf("Bearer %v", expiredSessionJWT),
		HttpOnly: true,
	}

	w := httptest.NewRecorder()
	r := &http.Request{
		Header: http.Header{
			"Cookie": []string{fmt.Sprintf("%s=%s", expiredTokenCookie.Name, expiredTokenCookie.Value)},
		},
	}

	mockStore.EXPECT().GetSession(expiredSession.ID).Return(expiredSession, nil)
	mockStore.EXPECT().DeleteSession(expiredSession.ID).Return(fmt.Errorf(`failed to delete session`))
	want = nil
	req := require.New(t)
	got, err := requireValidSession(mockStore, w, r)
	req.Error(err)
	req.Equal("session expired", err.Error())
	req.Equal(want, got)
	req.Equal(401, w.Code)
}

func Test_requireValidSession_extendSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	extendSession := &types.Session{
		ID:        "session-id",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(12 * time.Hour).Add(-1 * time.Hour), // simulate a scenario where user is still using the session after one hour
	}
	extendSessionJWT := signJWT(t, extendSession)
	extendedTokenCookie := http.Cookie{
		Name:     "signed-token",
		Value:    fmt.Sprintf("Bearer %v", extendSessionJWT),
		HttpOnly: true,
	}

	w := httptest.NewRecorder()
	r := &http.Request{
		Header: http.Header{
			"Cookie": []string{fmt.Sprintf("%s=%s", extendedTokenCookie.Name, extendedTokenCookie.Value)},
		},
	}

	mockStore.EXPECT().GetSession(extendSession.ID).Return(extendSession, nil)
	mockStore.EXPECT().GetPasswordUpdatedAt().Return(&time.Time{}, nil)
	mockStore.EXPECT().UpdateSessionExpiresAt(extendSession.ID, gomock.Any()).Return(nil)

	want = nil
	req := require.New(t)
	got, err := requireValidSession(mockStore, w, r)
	req.NoError(err)
	extendSession.ExpiresAt = extendSession.ExpiresAt.Add(time.Hour * 1)
	req.Equal(extendSession, got)
}

func Test_requireValidSession_extendSession_withUpdateErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	extendSession := &types.Session{
		ID:        "session-id",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(12 * time.Hour).Add(-1 * time.Hour), // simulate a scenario where user is still using the session after one hour
	}
	extendSessionJWT := signJWT(t, extendSession)
	extendedTokenCookie := http.Cookie{
		Name:     "signed-token",
		Value:    fmt.Sprintf("Bearer %v", extendSessionJWT),
		HttpOnly: true,
	}

	w := httptest.NewRecorder()
	r := &http.Request{
		Header: http.Header{
			"Cookie": []string{fmt.Sprintf("%s=%s", extendedTokenCookie.Name, extendedTokenCookie.Value)},
		},
	}

	mockStore.EXPECT().GetSession(extendSession.ID).Return(extendSession, nil).MaxTimes(2)
	mockStore.EXPECT().GetPasswordUpdatedAt().Return(nil, nil).MaxTimes(2)
	mockStore.EXPECT().UpdateSessionExpiresAt(extendSession.ID, gomock.Any()).Return(fmt.Errorf("error while updating secret"))

	want = nil
	req := require.New(t)
	got, err := requireValidSession(mockStore, w, r)
	req.NoError(err)
	extendSession.ExpiresAt = extendSession.ExpiresAt.Add(time.Hour * 1)
	req.Equal(extendSession, got)

	// test again and confirm session expiredAt isn't changed
	got, err = requireValidSession(mockStore, w, r)
	req.NoError(err)
	req.Equal(extendSession, got)
}

func Test_requireValidSession_FailedToFetchPasswordUpdated_AfterSessionIssuedErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	extendSession := &types.Session{
		ID:        "session-id",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(12 * time.Hour).Add(-1 * time.Hour), // simulate a scenario where user is still using the session after one hour
	}
	extendSessionJWT := signJWT(t, extendSession)
	extendedTokenCookie := http.Cookie{
		Name:     "signed-token",
		Value:    fmt.Sprintf("Bearer %v", extendSessionJWT),
		HttpOnly: true,
	}

	w := httptest.NewRecorder()
	r := &http.Request{
		Header: http.Header{
			"Cookie": []string{fmt.Sprintf("%s=%s", extendedTokenCookie.Name, extendedTokenCookie.Value)},
		},
	}

	mockStore.EXPECT().GetSession(extendSession.ID).Return(extendSession, nil).MaxTimes(2)
	mockStore.EXPECT().GetPasswordUpdatedAt().Return(nil, fmt.Errorf("failed to fetch password updatedAt"))

	want = nil
	req := require.New(t)
	got, err := requireValidSession(mockStore, w, r)
	req.Error(err)
	req.Equal("failed to fetch password updatedAt", err.Error())
	req.Equal(want, got)
	req.Equal(500, w.Code)
}

func Test_requireValidSession_PasswordUpdated_AfterSessionIssuedErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	session := &types.Session{
		ID:        "session-id",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(12 * time.Hour).Add(-1 * time.Hour), // simulate a scenario where user is still using the session after one hour
	}
	sessionJWT := signJWT(t, session)
	tokenCookie := http.Cookie{
		Name:     "signed-token",
		Value:    fmt.Sprintf("Bearer %v", sessionJWT),
		HttpOnly: true,
	}

	w := httptest.NewRecorder()
	r := &http.Request{
		Header: http.Header{
			"Cookie": []string{fmt.Sprintf("%s=%s", tokenCookie.Name, tokenCookie.Value)},
		},
	}

	passwordUpdatedAt := time.Now().Add(1 * time.Hour)
	mockStore.EXPECT().GetSession(session.ID).Return(session, nil).MaxTimes(2)
	mockStore.EXPECT().GetPasswordUpdatedAt().Return(&passwordUpdatedAt, nil)
	mockStore.EXPECT().DeleteSession(session.ID).Return(nil)

	want = nil
	req := require.New(t)
	got, err := requireValidSession(mockStore, w, r)
	req.Error(err)
	req.Equal("password changed, please login again", err.Error())
	req.Equal(want, got)
	req.Equal(401, w.Code)
}

func Test_requireValidSession_PasswordUpdated_AfterSessionIssuedErr_logDeleteSessionErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	session := &types.Session{
		ID:        "session-id",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(12 * time.Hour).Add(-1 * time.Hour), // simulate a scenario where user is still using the session after one hour
	}
	sessionJWT := signJWT(t, session)
	tokenCookie := http.Cookie{
		Name:     "signed-token",
		Value:    fmt.Sprintf("Bearer %v", sessionJWT),
		HttpOnly: true,
	}

	w := httptest.NewRecorder()
	r := &http.Request{
		Header: http.Header{
			"Cookie": []string{fmt.Sprintf("%s=%s", tokenCookie.Name, tokenCookie.Value)},
		},
	}

	passwordUpdatedAt := time.Now().Add(1 * time.Hour)
	mockStore.EXPECT().GetSession(session.ID).Return(session, nil).MaxTimes(2)
	mockStore.EXPECT().GetPasswordUpdatedAt().Return(&passwordUpdatedAt, nil)
	mockStore.EXPECT().DeleteSession(session.ID).Return(fmt.Errorf("failed to delete session"))

	want = nil
	req := require.New(t)
	got, err := requireValidSession(mockStore, w, r)
	req.Error(err)
	req.Equal("password changed, please login again", err.Error())
	req.Equal(want, got)
	req.Equal(401, w.Code)
}
