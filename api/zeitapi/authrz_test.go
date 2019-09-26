package zeitapi

import "testing"

func TestVerifyLogin(t *testing.T) {
	type args struct {
		domain   string
		user     string
		password string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyLogin(tt.args.domain, tt.args.user, tt.args.password); got != tt.want {
				t.Errorf("VerifyLogin() = %v, want %v", got, tt.want)
			}
		})
	}
}
