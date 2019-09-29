package zeitapi

import (
	"flag"
	"log"
	"os"
	"testing"
)

func Test_findRecordInResp(t *testing.T) {
	type args struct {
		resp     []byte
		recType  string
		recName  string
		recValue string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{
			"existing record",
			args{
				[]byte(`{"records":[{"id":"rec_fb8efab050a8d1338e374a2f","slug":"","name":"","type":"CAA","value":"0 issue \"letsencrypt.org\"","creator":"system","created":1569414163851,"updated":1569414163851},{"id":"rec_26013e44229d89bc74d1bdf8","slug":"","name":"","type":"ALIAS","value":"alias.zeit.co.","creator":"system","created":1569414163858,"updated":1569414163858},{"id":"rec_2f4285dff4789afdfc58fddb","slug":"","name":"*","type":"CNAME","value":"alias.zeit.co.","creator":"system","created":1569414163858,"updated":1569414163858},{"id":"rec_7db166980fdff0b72707d62f","slug":"to.denkruum.ch.-address-85.195.246.26","name":"to","type":"A","value":"85.195.246.26","creator":"EV25oKOhBp2mTTHZ2Sz0xGH8","created":1569356545249,"updated":1569356545249}]}`),
				"A",
				"to",
				"85.195.246.26",
			},
			"rec_7db166980fdff0b72707d62f",
			false,
		},
		{
			"update record",
			args{
				[]byte(`{"records":[{"id":"rec_fb8efab050a8d1338e374a2f","slug":"","name":"","type":"CAA","value":"0 issue \"letsencrypt.org\"","creator":"system","created":1569414163851,"updated":1569414163851},{"id":"rec_26013e44229d89bc74d1bdf8","slug":"","name":"","type":"ALIAS","value":"alias.zeit.co.","creator":"system","created":1569414163858,"updated":1569414163858},{"id":"rec_2f4285dff4789afdfc58fddb","slug":"","name":"*","type":"CNAME","value":"alias.zeit.co.","creator":"system","created":1569414163858,"updated":1569414163858},{"id":"rec_7db166980fdff0b72707d62f","slug":"to.denkruum.ch.-address-85.195.246.26","name":"to","type":"A","value":"85.195.246.26","creator":"EV25oKOhBp2mTTHZ2Sz0xGH8","created":1569356545249,"updated":1569356545249}]}`),
				"A",
				"to",
				"85.195.246.27",
			},
			"rec_7db166980fdff0b72707d62f",
			true,
		},
		{
			"new record",
			args{
				[]byte(`{"records":[{"id":"rec_fb8efab050a8d1338e374a2f","slug":"","name":"","type":"CAA","value":"0 issue \"letsencrypt.org\"","creator":"system","created":1569414163851,"updated":1569414163851},{"id":"rec_26013e44229d89bc74d1bdf8","slug":"","name":"","type":"ALIAS","value":"alias.zeit.co.","creator":"system","created":1569414163858,"updated":1569414163858},{"id":"rec_2f4285dff4789afdfc58fddb","slug":"","name":"*","type":"CNAME","value":"alias.zeit.co.","creator":"system","created":1569414163858,"updated":1569414163858}]}`),
				"A",
				"to",
				"85.195.246.27",
			},
			"",
			false,
		},
		{
			"garbadge",
			args{
				[]byte(`{"wrong":"json"}`),
				"A",
				"to",
				"85.195.246.27",
			},
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := findRecordInResp(tt.args.resp, tt.args.recType, tt.args.recName, tt.args.recValue)
			if got != tt.want {
				t.Errorf("findRecordInResp() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("findRecordInResp() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestZeitAPIBasics(t *testing.T) {

	t.Run("AddRecord", func(t *testing.T) {
		got := DNS.AddRecord(testDomain, "zeitTest", "TXT", "12345Test")
		if got != nil {
			t.Errorf("AddRecord() returned with error = %v", got)
		}
	})
	t.Run("FindRecord", func(t *testing.T) {
		b, got := DNS.GetRecordsFor(testDomain)
		if got != nil {
			t.Errorf("GetRecordsFor() returned with error => %v", got)
		}
		recID, _ := findRecordInResp(b, "TXT", "zeitTest", "12345Test")
		if recID == "" {
			t.Errorf("findRecordInResp() didn't find test record")
		}
	})

	var recToDeleteID string

	t.Run("UpdateRecord", func(t *testing.T) {
		// TODO: check the bool value
		_, got := DNS.UpdateRecord(testDomain, "TXT", "zeitTest", "12345TestNew")
		if got != nil {
			t.Errorf("UpdateRecord() coudn't update test record %v", got)
		}
		// now check, it was actually updated to the new value
		b, got := DNS.GetRecordsFor(testDomain)
		if got != nil {
			t.Errorf("GetRecordsFor() returned with error = %v", got)
		}
		// should return recID and false - we are searching for exiting falue
		recID, ok := findRecordInResp(b, "TXT", "zeitTest", "12345TestNew")
		switch {
		case recID == "" && ok:
			t.Errorf("findRecordInResp() returned bad response")
		case recID != "" && ok:
			t.Errorf("findRecordInResp() returned bad response, expected recID and false, got %q, %t", recID, ok)
		case recID == "" && !ok:
			t.Errorf("findRecordInResp() coudn't find updated record, got %q, %t", recID, ok)
		}
		recToDeleteID = recID
	})
	t.Run("DeleteRecord", func(t *testing.T) {
		got := DNS.DeleteRecord(testDomain, recToDeleteID)
		if got != nil {
			t.Errorf("DeleteRecord() returned with error = %v", got)
		}
	})
}

var (
	zeitAPIToken string
	testDomain   string
)

func TestMain(m *testing.M) {
	log.Println("Setting up tests ...")
	// call flag.Parse() here if TestMain uses flags
	flag.StringVar(&zeitAPIToken, "t", "", "Zeit token to use to test API")
	flag.StringVar(&testDomain, "d", "denkruum.ch", "Domain name to test with")
	flag.Parse()
	log.Printf("Token is: %q and domain is: %q \n", zeitAPIToken, testDomain)

	if zeitAPIToken == "" {
		log.Fatalf("API token not specified, exiting \n")
	}
	//base.go's init() executed first
	zeitAPI.Token = zeitAPIToken

	os.Exit(m.Run())
}
