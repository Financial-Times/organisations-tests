package main

import (
	"fmt"
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
	initialised      bool
	cacheFileName    string
	c                int
}

type compareResult struct {
	uuid      string
	fieldName string
	err       error
}

func (s *orgServiceImpl) compare() []compareResult {
	var results []compareResult
	for v2Uuid := range s.concorder.v2Uuids() {
		canonicalUUID, err := s.getCanonicalUUIDForV2UUID(v2Uuid)
		if err != nil {
			results = append(results, compareResult{uuid: v2Uuid, err: err})
			continue
		}
		compositeOrg, err := s.orgsRepo.orgFromURL(s.compositeOrgsURL + canonicalUUID)
		if err != nil {
			results = append(results, compareResult{uuid: v2Uuid, err: err})
			continue
		}

		fsOrg, err := s.orgsRepo.orgFromURL(s.fsURL + v2Uuid)
		if err != nil {
			results = append(results, compareResult{uuid: v2Uuid, err: err})
			continue
		}

		fieldsCompareMap := s.checkFields(&compositeOrg, &fsOrg)

		for fieldName, err := range fieldsCompareMap {
			if err != nil {
				results = append(results, compareResult{uuid: v2Uuid, fieldName: fieldName, err: err})
			}
		}
	}
	return results
}

func (s *orgServiceImpl) getCanonicalUUIDForV2UUID(v2Uuid string) (string, error) {
	v1UUIDMap, _, err := s.concorder.v2tov1(v2Uuid)
	if err != nil {
		return "", err
	}
	var uuids []string
	for v1UUID, _ := range v1UUIDMap {
		uuids = append(uuids, v1UUID)
	}
	uuids = append(uuids, v2Uuid)
	return canonical(uuids...), nil
}

func (s *orgServiceImpl) isInitialised() bool {
	return s.initialised
}

func (s *orgServiceImpl) checkFields(compositeOrg *combinedOrg, fsOrg *combinedOrg) map[string]error {
	fieldsCompareMap := make(map[string]error, 13)
	fieldsCompareMap["Type"] = compareStrings(compositeOrg.Type, fsOrg.Type)
	fieldsCompareMap["ProperName"] = compareStrings(compositeOrg.ProperName, fsOrg.ProperName)
	fieldsCompareMap["PrefLabel"] = compareStrings(compositeOrg.PrefLabel, fsOrg.PrefLabel)
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

func (s *orgServiceImpl) makeUppIdErrorMap(fsOrg *combinedOrg) (uppIdMap map[string]error) {
	uppIdMap = make(map[string]error)
	v1UUIDMap, _, err := s.concorder.v2tov1(fsOrg.UUID)
	if err != nil {
		return
	}
	uppIdMap[fsOrg.UUID] = fmt.Errorf("No UPP identifier found for fsOrg: %v", fsOrg.UUID)
	for v1UUID, _ := range v1UUIDMap {
		uppIdMap[v1UUID] = fmt.Errorf("No UPP identifier found for v1Org: %v", v1UUID)
	}
	return
}

func (s *orgServiceImpl) checkIdentifiers(compositeOrg *combinedOrg, fsOrg *combinedOrg) error {
	var uppIdErrMap map[string]error = s.makeUppIdErrorMap(fsOrg)
	if uppIdErrMap == nil || len(uppIdErrMap) < 1 {
		return fmt.Errorf("No concorded uuids for org: %v", fsOrg.UUID)
	}
	for _, uppID := range compositeOrg.AlternativeIdentifiers.Uuids {
		delete(uppIdErrMap, uppID)
	}

	var errorMsg string
	for _, err := range uppIdErrMap {
		if err != nil {
			errorMsg = errorMsg + " \n " + err.Error()
		}
	}
	if errorMsg != "" {
		return fmt.Errorf(errorMsg)
	}
	return nil
}
