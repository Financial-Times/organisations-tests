package main

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

type berthaConcorder struct {
	client         httpClient
	loaded         bool
	lk             sync.Mutex
	uuidV2toUUIDV1 map[string]map[string]struct{}
	uuidV2         map[string]struct{}
}

//Concordance model
type Concordance struct {
	CompositeID string `json:"compositeid"`
	TmeID       string `json:"tmeid"`
	FsID        string `json:"entityid"`
}

const berthaURL = "https://bertha.ig.ft.com/view/publish/gss/1k7GHf3311hyLBsNgoocRRkHs7pIhJit0wQVReFfD_6w/concordances"

func (b *berthaConcorder) v2tov1(uuid string) (map[string]struct{}, bool, error) {
	b.lk.Lock()
	defer b.lk.Unlock()

	if !b.loaded {
		return nil, false, errors.New("concordance not loaded yet")
	}
	value, found := b.uuidV2toUUIDV1[uuid]
	return value, found, nil
}

func (b *berthaConcorder) load() error {
	b.lk.Lock()
	defer b.lk.Unlock()
	resp, err := b.client.Get(berthaURL)
	if err != nil {
		errMsg := fmt.Sprintf("Error while retrieving concordances: %v", err.Error())
		return errors.New(errMsg)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Bertha responded with status: %d", resp.StatusCode)
		return errors.New(errMsg)
	}

	var concordances []Concordance
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	json.Unmarshal(body, &concordances)
	var count = 0
	for _, con := range concordances {
		v1uuid := v1ToUUID(con.CompositeID)
		v2uuid := v2ToUUID(con.FsID)
		uuidSet, found := b.uuidV2toUUIDV1[v2uuid]
		if !found {
			uuidSet = make(map[string]struct{})
		}
		uuidSet[v1uuid] = struct{}{}
		b.uuidV2toUUIDV1[v2uuid] = uuidSet
		b.uuidV2[v2uuid] = struct{}{}
		count++
	}

	b.loaded = true
	log.Printf("Finished loading concordances: %v values", count)
	return nil
}

func (b *berthaConcorder) v2Uuids() map[string]struct{} {
	return b.uuidV2
}
