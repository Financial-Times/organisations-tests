package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type orgsRepo interface {
	orgFromURL(u string) (combinedOrg, error)
}

type httpOrgsRepo struct {
	client httpClient
}

func (r *httpOrgsRepo) orgFromURL(u string) (combinedOrg, error) {
	resp, err := r.client.Get(u)
	if err != nil {
		return combinedOrg{}, fmt.Errorf("Could not connect to %v. Error %v", u, err)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return combinedOrg{}, fmt.Errorf("Could not get orgs from %v. Returned %v", u, resp.StatusCode)
	}
	var org combinedOrg
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&org); err != nil {
		return combinedOrg{}, fmt.Errorf("Error decoding response from %v, error: %+v", u, err)
	}
	return org, nil
}
