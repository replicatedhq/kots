package archives

import (
	"encoding/base64"
	"testing"
)

func TestIsTGZ(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "empty",
			input: "",
			want:  false,
		},
		{
			name:  "not a tgz",
			input: "bm90IGEgdGd6Cg==",
			want:  false,
		},
		{
			name:  "tgz",
			input: "H4sIAE0QXGQAA+2RQQ6DMAwE8xS/oGyCQ94ToaggcQqG9vmNSjmUA1Kl+kTmsrJsyWtvzP0wrqkxigAI3tNbu00Lu26FZXYBFvCBYF1wzpDXNLWzzBJzsdLnON5P5h5DStNJ//so+rNLNeInf0mz3OQpGjvKPzrmX/JnbmEIGmaOXDz/SqVyXV53bklCAAgAAA==",
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			b, err := base64.StdEncoding.DecodeString(tt.input)
			if err != nil {
				t.Errorf("failed to decode input: %v", err)
			}

			if got := IsTGZ(b); got != tt.want {
				t.Errorf("IsTGZ() = %v, want %v", got, tt.want)
			}
		})
	}
}
