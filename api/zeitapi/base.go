package zeitapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
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

// zeitError captures errors returned by Zeit DNS API
// Zeit returns JSON messages that are structured as:
//   { "error" : { "code":"xxx", "message" : "msg..", "another api specific key":"vvv", "key2":"vvv", ..}}
type zeitError struct {
	Code string `json:"code"`
	Msg  string `json:"message"`
}

// Error handles ZeitAPI errors and implements Error Interface
// It doesn't wrap lower level errors
type Error struct {
	// ZeitError field will containe unmarshalled messaged returned by Zeit DNS API
	ZeitErr zeitError `json:"error"`
}

func (e Error) Error() string {
	if e.ZeitErr.Code == "" {
		return "Zeit API : No Error Code"
	}
	return fmt.Sprintf("Zeit API Error Code: %s , Message: %s", e.ZeitErr.Code, e.ZeitErr.Msg)

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
	// TODO: handle possible err here
	req, err := d.newDNSReq(method, domain, bytes.NewBuffer(body))
	req.r.Header.Add("Content-Type", "application/json")
	return req, nil
}

// Do wraps around lower level http.Do method. It'll return zeitapi.Error, if one is available
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
	var rj Error
	// TODO: consider using string based JSON processing to speed-up error handling
	err = json.Unmarshal(body, &rj)
	if err != nil {
		log.Fatalf("Zeit API Lib error: can't decode json from response %s , %s", body, err)
	}

	if rj.ZeitErr.Code != "" {
		return nil, rj
	}
	return body, nil

}

// findRecordInResp will return RecordID  of the record, if found
// it'll, however, return **false**, if the **recValue** is the same as the existing record
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

// UpdateRecord will add new record and return true.
//  An exiting record will be updated if the value is new, return true.
//  If the value is the same, it'll return false.
//  Err will either be an error or zeitapi.Error.
func (d *dnsAPIBase) UpdateRecord(tld, recType, recName, recValue string) (bool, error) {

	resp, err := d.GetRecordsFor(tld)
	if err != nil {
		return false, fmt.Errorf("Can't update domain: %w", err)
	}
	// find the record in the response
	recID, isFound := findRecordInResp(resp, recType, recName, recValue)
	if isFound { // exists and the value is new
		err = d.DeleteRecord(tld, recID)
		if err != nil {
			return false, fmt.Errorf("Can't update DNS record: %q : %w", recID, err)
		}
	} else if recID != "" {
		return false, nil // record found, but the value is the same
	}
	err = d.AddRecord(tld, recName, recType, recValue)
	if err != nil {
		return false, fmt.Errorf("Can't set record %q : %q for %q : %w", recType, recValue, tld, err)
	}
	// either completely new record or existing with a new value
	return true, nil
}

// SetAddressTo will set new IP Address, if it's changed and return true.
// New record will be created, if needed
func (d *dnsAPIBase) SetAddressTo(strFQDN string, ipAddr net.IP) (bool, error) {
	//get the lowest level domain
	i := strings.IndexByte(strFQDN, '.')
	if i < 0 {
		return false, fmt.Errorf("incorrect domain name supplied: %q", strFQDN)
	}
	lld := strFQDN[:i]
	tld := strFQDN[i+1:]

	// TODO: IPV6, "AAAA" records ...
	// ipAddr len will be 4 bytes for IPv4 and 16 bytes for Ipv6
	// will need to check for both?
	return d.UpdateRecord(tld, "A", lld, ipAddr.String())
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
