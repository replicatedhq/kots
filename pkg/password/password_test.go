package password

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
)

// getMockStore - will return a mock kots store
func getMockStore(t *testing.T) *mock_store.MockStore {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	return mock_store.NewMockStore(ctrl)
}

func TestValidateNewPassword(t *testing.T) {
	type args struct {
		currentPassword string
		newPassword     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// expect error when new password is empty
		{
			name: "expect error when new password is empty",
			args: args{
				newPassword: "",
			},
			wantErr: true,
		},
		// expect error when new password is less than 6 characters
		{
			name: "expect error when new password is less than 6 characters",
			args: args{
				newPassword: "12345",
			},
			wantErr: true,
		},
		// expect no error when new password is 6 characters
		{
			name: "expect no error when new password is 6 characters",
			args: args{
				newPassword: "123456",
			},
			wantErr: false,
		},
		// expect no error when new password is more than 6 characters
		{
			name: "expect no error when new password is more than 6 characters",
			args: args{
				newPassword: "1234567",
			},
			wantErr: false,
		},
		// expect error when current password is same as new password
		{
			name: "expect error when current password is same as new password",
			args: args{
				currentPassword: "123456",
				newPassword:     "123456",
			},
			wantErr: true,
		},
		// expect no error when current password is different from new password
		{
			name: "expect no error when current password is different from new password",
			args: args{
				currentPassword: "123456",
				newPassword:     "1234567",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePasswordInput(tt.args.currentPassword, tt.args.newPassword); (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validatePassword_userStoreError(t *testing.T) {
	wrongPwdShaBytes, err := bcrypt.GenerateFromPassword([]byte("wrong-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	currentPwdShaBytes, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		currentPassword                        string
		mockGetSharedPasswordBcryptResponse    []byte
		mockGetSharedPasswordBcryptErrResponse error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// expect error when kots userStore returns an error
		{
			name: "expect error when userStore returns an db error",
			args: args{
				currentPassword:                        "current-password",
				mockGetSharedPasswordBcryptResponse:    nil,
				mockGetSharedPasswordBcryptErrResponse: fmt.Errorf("failed to fetch password"),
			},
			wantErr: true,
		},
		{
			name: "expect error when password mismatch",
			args: args{
				currentPassword:                        "current-password",
				mockGetSharedPasswordBcryptResponse:    wrongPwdShaBytes,
				mockGetSharedPasswordBcryptErrResponse: nil,
			},
			wantErr: true,
		},
		{
			name: "expect error when password hash is invalid",
			args: args{
				currentPassword:                        "current-password",
				mockGetSharedPasswordBcryptResponse:    []byte("not-a-hash"),
				mockGetSharedPasswordBcryptErrResponse: nil,
			},
			wantErr: true,
		},
		{
			name: "expect no error when password match",
			args: args{
				currentPassword:                        "current-password",
				mockGetSharedPasswordBcryptResponse:    currentPwdShaBytes,
				mockGetSharedPasswordBcryptErrResponse: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := getMockStore(t)
			mockStore.EXPECT().GetSharedPasswordBcrypt().Return(tt.args.mockGetSharedPasswordBcryptResponse, tt.args.mockGetSharedPasswordBcryptErrResponse)
			if err := ValidateCurrentPassword(mockStore, tt.args.currentPassword); (err != nil) != tt.wantErr {
				t.Errorf("ValidateCurrentPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func newMockClientForExistingSecretGetErr() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, &kuberneteserrors.UnexpectedObjectError{}
	})
	return &mockClient
}

func newMockClientForSecretCreateFailErr() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, kuberneteserrors.NewNotFound(corev1.Resource("secret"), "kots-password")
	})
	mockClient.AddReactor("create", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, kuberneteserrors.NewTimeoutError("too slow", 100)
	})
	return &mockClient
}

func newMockClientForSecretCreateSuccess() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, kuberneteserrors.NewNotFound(corev1.Resource("secret"), "kots-password")
	})
	mockClient.AddReactor("create", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	return &mockClient
}

func newMockClientForExistingSecretUpdateFail() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		result := &corev1.Secret{
			Data: map[string][]byte{},
		}
		return true, result, nil
	})
	mockClient.AddReactor("update", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, kuberneteserrors.NewTimeoutError("too slow", 100)
	})
	return &mockClient
}

func newMockClientForExistingSecretUpdateSuccess() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		result := &corev1.Secret{
			Data: map[string][]byte{},
		}
		return true, result, nil
	})
	mockClient.AddReactor("update", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	return &mockClient
}

func newMockClientForDeleteSessionsErr() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, kuberneteserrors.NewNotFound(corev1.Resource("secret"), "kots-password")
	})
	mockClient.AddReactor("create", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	mockClient.AddReactor("update", "secrets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, kuberneteserrors.NewTimeoutError("too slow", 100)
	})
	return &mockClient
}

func Test_setSharedPasswordBcrypt(t *testing.T) {
	type args struct {
		clientset      kubernetes.Interface
		namespace      string
		bcryptPassword []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "expect error when getting k8s secret fails",
			args: args{
				clientset:      newMockClientForExistingSecretGetErr(),
				namespace:      "test",
				bcryptPassword: []byte("password"),
			},
			wantErr: true,
		},
		{
			name: "expect error when secret not found and failed to create secret",
			args: args{
				clientset:      newMockClientForSecretCreateFailErr(),
				namespace:      "test",
				bcryptPassword: []byte("password"),
			},
			wantErr: true,
		},
		{
			name: "expect no error when secret not found and secret created successfully",
			args: args{
				clientset:      newMockClientForSecretCreateSuccess(),
				namespace:      "test",
				bcryptPassword: []byte("password"),
			},
			wantErr: false,
		},
		{
			name: "expect error when existing secret update fails",
			args: args{
				clientset:      newMockClientForExistingSecretUpdateFail(),
				namespace:      "test",
				bcryptPassword: []byte("password"),
			},
			wantErr: true,
		},
		{
			name: "expect no error when existing secret updated successfully",
			args: args{
				clientset:      newMockClientForExistingSecretUpdateSuccess(),
				namespace:      "test",
				bcryptPassword: []byte("password"),
			},
			wantErr: false,
		},
		{
			name: "expect error when existing secret updated successfully and sessions delete err",
			args: args{
				clientset:      newMockClientForDeleteSessionsErr(),
				namespace:      "test",
				bcryptPassword: []byte("password"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := setSharedPasswordBcrypt(tt.args.clientset, tt.args.namespace, tt.args.bcryptPassword); (err != nil) != tt.wantErr {
				t.Errorf("setSharedPasswordBcrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
