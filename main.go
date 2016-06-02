package main

import (
	"errors"
	//"fmt"
	//"github.com/Financial-Times/http-handlers-go/httphandlers"
	log "github.com/Sirupsen/logrus"
	//"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	//"github.com/rcrowley/go-metrics"
	"fmt"
	"github.com/sethgrid/pester"
	"net"
	"os"
	"net/http"
	"time"
)

type concorder interface {
	v2tov1(string) (map[string]struct{}, bool, error)
	v2Uuids() map[string]struct{}
	load() error
}

type httpClient interface {
	Get(url string) (resp *http.Response, err error)
}

func init() {
	log.SetFormatter(new(log.JSONFormatter))
	log.SetLevel(log.DebugLevel)
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

	if errs := orgService.compare(); errs != nil && len(errs) > 0 {
		for err := range errs {
			log.Errorf("ERROR by comparing content: %v", err)
		}
		return fmt.Errorf("Errors were encountered")
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
