package authlib

import (
	"log"
	"os"
	"testing"
)

//Base64 encoded token
const secretToken = "YXgugRPzoLcz80xGCXnnWZcUFRfZ9Cmo-Iin1MwpjsM"

func TestVerifyLogin(t *testing.T) {
	type args struct {
		domain   string
		user     string
		password string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"good login",
			args{
				"to.denkruum.ch",
				"vvv",
				"NVwJdKNeMuAP-irILTI7Q_xB_XbSFtQXcFwucnCloiY",
			},
			true,
			false,
		},
		{
			"bad login",
			args{
				"to.denkruum.ch",
				"vv",
				"NVwJdKNeMuAP-irILTI7Q_xB_XbSFtQXcFwucnCloiY",
			},
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VerifyLogin(tt.args.domain, tt.args.user, tt.args.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyLogin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VerifyLogin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMain(m *testing.M) {
	log.Println("Setting up tests ...")

	InitAuth(secretToken)

	os.Exit(m.Run())
}
