package zeitapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type zeitAPIBase struct {
	Token string
	URL   string
}
type dnsAPIBase struct {
	endpoint string
}

type zeitRequest struct {
	r *http.Request
}

// Zeit retruns JSON messages that are structured as:
//   { "error" : { "code":"xxx", "message" : "msg..", "another api specific key":"vvv", "key2":"vvv", ..}}
type zeitError struct {
	Error struct {
		Code string `json:"code"`
		Msg  string `json:"message"`
	} `json:"error"`
}

var (
	zeitAPI zeitAPIBase
	// DNS provides methods to work with Zeit's DNS records
	DNS dnsAPIBase
)

const nowAPITokenName = "NOW_API_TKN"

func init() {
	zeitAPI.Token = os.Getenv(nowAPITokenName)
	zeitAPI.URL = "https://api.zeit.co"
	DNS.endpoint = zeitAPI.URL + "/v2/domains/"
}
func (d *dnsAPIBase) newDNSReq(method, domain string, body io.Reader) (*zeitRequest, error) {

	var urlB strings.Builder
	parts := []string{d.endpoint, domain, "/records"}
	for _, s := range parts {
		urlB.WriteString(s)
	}

	// handle REST logic of the API, body will carry recID as io.Reader
	if method == http.MethodDelete {
		urlB.WriteByte('/')
		var b bytes.Buffer
		b.ReadFrom(body)
		urlB.Write(b.Bytes())
		body = nil
	}

	req, err := http.NewRequest(method, urlB.String(), body)
	if err != nil {
		return nil, err
	}
	return &zeitRequest{req}, nil
}

func (d *dnsAPIBase) newDNSReqJSON(method, domain string, payload interface{}) (*zeitRequest, error) {

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := d.newDNSReq(method, domain, bytes.NewBuffer(body))
	req.r.Header.Add("Content-Type", "application/json")
	return req, nil
}

func (zr *zeitRequest) Do() (body []byte, err error) {
	client := &http.Client{}
	zr.r.Header.Add("Authorization", "Bearer "+zeitAPI.Token)
	resp, err := client.Do(zr.r)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	//TODO: should we generate an error here?
	if resp.ContentLength < 0 {
		return nil, nil
	}
	body = make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	//handle error response from Zeit API
	var rj zeitError
	// TODO: consider using string based JSON processing to speed-up error handling
	err = json.Unmarshal(body, &rj)
	if err != nil {
		log.Fatalf("Zeit API request processing error: can't decode json from response %s , %s", body, err)
	}

	if rj.Error.Code != "" {
		return nil, fmt.Errorf("Zeit API error %s", rj.Error.Msg)
	}
	return body, nil

}

func findRecordInResp(resp []byte, recType, recName, recValue string) (string, bool) {
	var result struct {
		Records []struct {
			ID      string `json:"id"`
			RecType string `json:"type"`
			Name    string `json:"name"`
			Value   string `json:"value"`
		} `json:"records"`
	}

	if len(resp) == 0 {
		return "", false
	}
	err := json.Unmarshal(resp, &result)
	if err != nil {
		log.Printf("Zeit API lib error, can't decode JSON from response %q, %v", resp, err)
	}
	recID := ""
	found := false
	for _, rec := range result.Records {
		if rec.RecType == recType && rec.Name == recName {
			if rec.Value != recValue {
				//found is hint to update, true if the value is new
				found = true
			}
			recID = rec.ID
			break
		}
	}
	return recID, found
}

// GetRecordsFor returns records for a specified domain
func (d *dnsAPIBase) GetRecordsFor(domain string) ([]byte, error) {

	req, err := d.newDNSReq(http.MethodGet, domain, nil)
	if err != nil {
		return nil, fmt.Errorf("Can't initiate DNS records request: %w", err)
	}
	body, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("Can't get DNS records for domain %q : %w", domain, err)
	}
	return body, nil
}

func (d *dnsAPIBase) UpdateRecord(tld, recType, recName, recValue string) error {
	resp, err := d.GetRecordsFor(tld)
	if err != nil {
		return fmt.Errorf("Can't update domain: %w", err)
	}

	recID, ok := findRecordInResp(resp, recType, recName, recValue)
	//delete, if it does
	if ok {
		err = d.DeleteRecord(tld, recID)
		if err != nil {
			return fmt.Errorf("Can't update DNS record: %q : %w", recID, err)
		}
	}
	err = d.AddRecord(tld, recName, recType, recValue)
	if err != nil {
		return fmt.Errorf("Can't set record %q : %q for %q : %w", recType, recValue, tld, err)
	}

	return nil
}

func (d *dnsAPIBase) SetAddressTo(strFQDN string, ipAddr string) error {
	//get the lowest level domain
	i := strings.IndexByte(strFQDN, '.')
	if i < 0 {
		return fmt.Errorf("incorrect domain name supplied: %q", strFQDN)
	}
	lld := strFQDN[:i]
	//check if it exisits
	tld := strFQDN[i+1:]

	//TODO: IPV6, "AAAA" records ...
	return d.UpdateRecord(tld, "A", lld, ipAddr)
}

func (d *dnsAPIBase) AddRecord(domain, child, recType, value string) error {
	//init payload
	params := struct {
		Name  string `json:"name,omitempty"`
		Typ   string `json:"type,omitempty"`
		Value string `json:"value,omitempty"`
	}{child, recType, value}

	req, err := d.newDNSReqJSON(http.MethodPost, domain, params)
	if err != nil {
		return fmt.Errorf("Add Record, can't create http request %w", err)
	}
	_, err = req.Do()
	if err != nil {
		return fmt.Errorf("Can't add DNS record for domain %s : %w", domain, err)
	}
	return nil
}
func (d *dnsAPIBase) DeleteRecord(domain, recID string) error {
	//init payload

	req, err := d.newDNSReq(http.MethodDelete, domain, strings.NewReader(recID))
	if err != nil {
		return fmt.Errorf("DeleteRecord, can't create http request %w", err)
	}
	_, err = req.Do()
	if err != nil {
		return fmt.Errorf("Can't delete DNS record for domain %s : %w", domain, err)
	}
	return nil
}
