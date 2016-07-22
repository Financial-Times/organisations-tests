package main

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/jawher/mow.cli"
	"github.com/sethgrid/pester"
	"net"
	"net/http"
	"os"
	"time"
)

type concorder interface {
	v2tov1(string) (map[string]struct{}, bool, error)
	v2Uuids() map[string]struct{}
	v1UuidToTmeId(uuid string) string
	load() error
}

type httpClient interface {
	Get(url string) (resp *http.Response, err error)
}

func main() {
	app := cli.App("organisations-tests", "A RESTful API for transforming combined organisations")
	compositeOrgsURL := app.String(cli.StringOpt{
		Name:   "composite-orgs-url",
		Value:  "",
		Desc:   "URL for composite organisations transformer",
		EnvVar: "COMPOSITE_ORGS_URL",
	})
	fsURL := app.String(cli.StringOpt{
		Name:   "fs-transformer-url",
		Value:  "",
		Desc:   "URL for factset organisations transformer",
		EnvVar: "FS_TRANSFORMER_URL",
	})
	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "Port to listen on",
		EnvVar: "PORT",
	})

	app.Action = func() {
		log.Println("Starting app")
		if err := runApp(*compositeOrgsURL, *fsURL, *port); err != nil {
			log.Fatal(err)
		} else {
			log.Infof("Comparision was successful")
		}
		log.Println("Finished app")
	}
	log.SetOutput(os.Stdout)
	app.Run(os.Args)
}

func runApp(compositeOrgsURL, fsURL string, port int) error {
	if compositeOrgsURL == "" {
		return errors.New("Composite organisation transformer URL must be provided")
	}
	if fsURL == "" {
		return errors.New("Factset Organisation transformer URL must be provided")
	}

	httpClient := newResilientClient()

	con := &berthaConcorder{
		client:         httpClient,
		uuidV2:         make(map[string]struct{}),
		uuidV2toUUIDV1: make(map[string]map[string]struct{}),
		uuidToTmeId:    make(map[string]string),
	}

	repo := &httpOrgsRepo{client: httpClient}

	orgService := &orgServiceImpl{
		fsURL:            fsURL,
		compositeOrgsURL: compositeOrgsURL,
		concorder:        con,
		orgsRepo:         repo,
		combinedOrgCache: make(map[string]*combinedOrg),
		initialised:      false,
	}

	var err error
	for {
		err = orgService.concorder.load()
		if err != nil {
			log.Errorf("ERROR loading concordance data. Retrying. (%+v)", err)

			time.Sleep(60 * time.Second)
		} else {
			break
		}
	}

	if results := orgService.compare(); results != nil && len(results) > 0 {
		for _, result := range results {
			log.Errorf("ERROR by comparing content: %+v, %+v, %+v", result.uuid, result.fieldName, result.err.Error())
		}
	}
	return nil

}

func newResilientClient() *pester.Client {
	tr := &http.Transport{
		MaxIdleConnsPerHost: 128,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
	}
	c := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
	return pester.NewExtendedClient(c)
}
