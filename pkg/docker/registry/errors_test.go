package registry

import "testing"

func Test_errorResponseToString(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		want       string
	}{
		{
			name:     "quay.io",
			response: `{"errors":[{"code":"UNAUTHORIZED","detail":{},"message":"Invalid Username or Password"}]}`,
			want:     "Invalid Username or Password",
		},
		{
			name:     "docker.io",
			response: `{"details":"incorrect username or password"}`,
			want:     "incorrect username or password",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errorResponseToString(tt.statusCode, []byte(tt.response)); got != tt.want {
				t.Errorf("errorResponseToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
