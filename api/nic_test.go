package handler

import (
	"bytes"
	"testing"
)

func Test_writeStringsToB(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			"empty string",
			[]string{},
			"",
		},
		{
			"1 strings",
			[]string{"1string"},
			"1string",
		},
		{
			"2 strings",
			[]string{"1string", "2string"},
			"1string 2string",
		},
		{
			"3 strings",
			[]string{"1string", "2string", "3string"},
			"1string 2string 3string",
		},
	}
	buf := bytes.NewBuffer(nil)
	for _, tt := range tests {
		buf.Reset()
		t.Run(tt.name, func(t *testing.T) {
			writeStringsToB(buf, tt.args...)
		})
		if buf.String() != tt.want {
			t.Errorf("writeStringsToB() got: %q, wanted: %q", buf.String(), tt.want)
		}
	}
}
