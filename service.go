package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
)

const (
	uppIdentifier       = "http://api.ft.com/system/FT-UPP"
)

type orgsService interface {
	getOrgs() ([]byte, error)
	getOrgByUUID(uuid string) (combinedOrg, bool, error)
	isInitialised() bool
	getBaseURI() string
	count() int
}

type orgServiceImpl struct {
	fsURL            string
	compositeOrgsURL string
	concorder        concorder
	orgsRepo         orgsRepo
	combinedOrgCache map[string]*combinedOrg
	list             []listEntry
	initialised      bool
	cacheFileName    string
	c                int
}

func (s *orgServiceImpl) compare() []error {
	var errors []error
	for v2Uuid := range s.concorder.v2Uuids() {
		canonicalUUID, err := s.getCanonicalUUIDForV2UUID(v2Uuid)
		if err != nil {
			errors = append(errors, err)
			log.Errorf("getCanonicalUUIDForV2UUID:" + err.Error())
			continue
		}
		compositeOrg, err := s.orgsRepo.orgFromURL(s.compositeOrgsURL + canonicalUUID)
		if err != nil {
			errors = append(errors, err)
			log.Errorf("orgFromURL(" + s.compositeOrgsURL + canonicalUUID + "):" + err.Error())
			continue
		}

		fsOrg, err := s.orgsRepo.orgFromURL(s.fsURL + v2Uuid)
		if err != nil {
			errors = append(errors, err)
			log.Errorf("orgFromURL(" + s.fsURL + v2Uuid + "):" + err.Error())
			continue
		}

		fieldsCompareMap :=s.checkFields(compositeOrg, fsOrg)

		for _, err := range fieldsCompareMap {
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	return errors
}

func (s *orgServiceImpl) getCanonicalUUIDForV2UUID(v2Uuid string) (string, error) {
	v1UUIDMap, _, err := s.concorder.v2tov1(v2Uuid)
	if err != nil {
		return "", err
	}
	var uuids []string
	for v1UUID,_ := range v1UUIDMap {
		uuids = append(uuids, v1UUID)
	}
	uuids = append(uuids, v2Uuid)
	return canonical(uuids...), nil
}

func (s *orgServiceImpl) isInitialised() bool {
	return s.initialised
}

func (s *orgServiceImpl) checkFields(compositeOrg combinedOrg, fsOrg combinedOrg) (map[string]error) {
	fieldsCompareMap := make(map[string]error, 12)
	log.Infof("Checking fields of composite org [%v] with fs org [%v]", compositeOrg.UUID, fsOrg.UUID)
	fieldsCompareMap["Type"] = compareStrings(compositeOrg.Type, fsOrg.Type)
	fieldsCompareMap["ProperName"] = compareStrings(compositeOrg.ProperName, fsOrg.ProperName)
	fieldsCompareMap["LegalName"] = compareStrings(compositeOrg.LegalName, fsOrg.LegalName)
	fieldsCompareMap["ShortName"] = compareStrings(compositeOrg.ShortName, fsOrg.ShortName)
	fieldsCompareMap["HiddenLabel"] = compareStrings(compositeOrg.HiddenLabel, fsOrg.HiddenLabel)
	fieldsCompareMap["IndustryClassification"] = compareStrings(compositeOrg.IndustryClassification, fsOrg.IndustryClassification)
	fieldsCompareMap["ParentOrganisation"] = compareStrings(compositeOrg.ParentOrganisation, fsOrg.ParentOrganisation)
	fieldsCompareMap["TradeNames"] = compareArrays(compositeOrg.TradeNames, fsOrg.TradeNames)
	fieldsCompareMap["LocalNames"] = compareArrays(compositeOrg.LocalNames, fsOrg.LocalNames)
	fieldsCompareMap["FormerNames"] = compareArrays(compositeOrg.FormerNames, fsOrg.FormerNames)
	fieldsCompareMap["Aliases"] = compareArrays(compositeOrg.Aliases, fsOrg.Aliases)
	fieldsCompareMap["Identifiers"] = s.checkIdentifiers(compositeOrg, fsOrg)
	return fieldsCompareMap
}



func (s *orgServiceImpl) makeUppIdentifierMap(fsOrg combinedOrg) (uppIdMap map[string]error) {
	uppIdMap = make(map[string]error)
	v1UUIDMap, _, err := s.concorder.v2tov1(fsOrg.UUID)
	if err != nil {
		return
	}
	uppIdMap[fsOrg.UUID] = fmt.Errorf("No UPP identifier found for: %v", fsOrg.UUID)
	for v1UUID,_ := range v1UUIDMap {
		uppIdMap[v1UUID] = fmt.Errorf("No UPP identifier found for: %v", v1UUID)
	}
	return
}

func (s *orgServiceImpl) checkIdentifiers(compositeOrg combinedOrg, fsOrg combinedOrg) error {
	var uppIdMap map[string]error = s.makeUppIdentifierMap(fsOrg)
	if uppIdMap == nil || len(uppIdMap) < 1 {
		return fmt.Errorf("No concorded uuids for org: %v", fsOrg.UUID)
	}
	for _, identifierVal := range compositeOrg.Identifiers {
		if identifierVal.Authority == uppIdentifier {
			uppIdMap[identifierVal.IdentifierValue] = nil
		}
	}
	//TODO check FT-TME && FACTSET identifiers too on composite
	var errorMsg string
	for _, err := range uppIdMap {
		if err != nil {
			errorMsg = errorMsg + "\n" + err.Error()
		}
	}
	if errorMsg != "" {
		return fmt.Errorf(errorMsg)
	}
	return nil
}

