package password

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"golang.org/x/crypto/bcrypt"
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

func TestChangePassword(t *testing.T) {
	currentPwdShaBytes, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		currentPassword                        string
		newPassword                            string
		mockGetSharedPasswordBcryptResponse    []byte
		mockGetSharedPasswordBcryptErrResponse error
		setSharedPasswordBcryptErrResponse     error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// expect error when kots userStore GetSharedPassword() returns an error
		{
			name: "expect error when userStore GetSharedPassword() returns an error",
			args: args{
				currentPassword:                        "current-password",
				newPassword:                            "new-password",
				mockGetSharedPasswordBcryptResponse:    nil,
				mockGetSharedPasswordBcryptErrResponse: fmt.Errorf("failed to fetch password"),
				setSharedPasswordBcryptErrResponse:     nil,
			},
			wantErr: true,
		},
		// expect error when kots userStore SetSharedPassword() returns an error
		{
			name: "expect error when userStore SetSharedPassword() returns an error",
			args: args{
				currentPassword:                        "current-password",
				newPassword:                            "new-password",
				mockGetSharedPasswordBcryptResponse:    currentPwdShaBytes,
				mockGetSharedPasswordBcryptErrResponse: nil,
				setSharedPasswordBcryptErrResponse:     fmt.Errorf("failed to set password"),
			},
			wantErr: true,
		},
		// expect no error when password match
		{
			name: "expect no error when password match",
			args: args{
				currentPassword:                        "current-password",
				newPassword:                            "new-password",
				mockGetSharedPasswordBcryptResponse:    currentPwdShaBytes,
				mockGetSharedPasswordBcryptErrResponse: nil,
				setSharedPasswordBcryptErrResponse:     nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := getMockStore(t)
			mockStore.EXPECT().GetSharedPasswordBcrypt().Return(tt.args.mockGetSharedPasswordBcryptResponse, tt.args.mockGetSharedPasswordBcryptErrResponse)
			mockStore.EXPECT().SetSharedPasswordBcrypt(gomock.Any()).Return(tt.args.setSharedPasswordBcryptErrResponse)
			if err := ChangePassword(mockStore, tt.args.currentPassword, tt.args.newPassword); (err != nil) != tt.wantErr {
				t.Errorf("ChangePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
