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
	uuidToTmeId    map[string]string
}

//Concordance model
type Concordance struct {
	TMEID  string `json:"tmeid"`
	V2UUID string `json:"v2uuid"`
}

const berthaURL = "https://bertha.ig.ft.com/view/publish/gss/1k7GHf3311hyLBsNgoocRRkHs7pIhJit0wQVReFfD_6w/orgs"

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
		if isIncomplete(con) {
			continue
		}
		v1uuid := v1ToUUID(con.TMEID)
		b.uuidToTmeId[v1uuid] = con.TMEID
		v2uuid := con.V2UUID
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

func isIncomplete(con Concordance) bool {
	return con.TMEID == "" || con.V2UUID == ""
}

func (b *berthaConcorder) v2Uuids() map[string]struct{} {
	return b.uuidV2
}

func (b *berthaConcorder) v1UuidToTmeId(uuid string) string {
	return b.uuidToTmeId[uuid]
}
