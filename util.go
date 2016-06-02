package main

import (
	"crypto/md5"
	"github.com/pborman/uuid"
	"fmt"
)

var hasher = md5.New()
var emptyUUID = uuid.UUID{}

func v1ToUUID(v1id string) string {
	return uuid.NewMD5(uuid.UUID{}, []byte(v1id)).String()
}

func v2ToUUID(fsid string) string {
	md5data := md5.Sum([]byte(fsid))
	hasher.Reset()
	return uuid.NewHash(hasher, emptyUUID, md5data[:], 3).String()
}

func canonical(uuids ...string) (c string) {
	for _, s := range uuids {
		if c == "" || s < c {
			c = s
		}
	}
	return
}

func compareStrings(compositeField string, fsField string) error {
	if compositeField == fsField {
		return nil
	}
	return fmt.Errorf("Not identical - compositeField: %v, fsField: %v", compositeField, fsField)
}

func compareArrays(compositeField []string, fsField []string) error {
	if compositeField == nil && fsField == nil {
		return nil
	}
	if compositeField == nil || fsField == nil {
		return fmt.Errorf("One of the field array is nil - compositeField: %v, fsField: %v", compositeField, fsField)
	}
	if len(compositeField) != len(fsField) {
		return fmt.Errorf("Field arrays have different sizes - compositeField: %v, fsField: %v", compositeField, fsField)
	}
	for compositeValue, _ := range compositeField {
		var found bool = false
		for fsValue, _ := range fsField {
			if fsValue == compositeValue {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Not identical - compositeField: %v, fsField: %v", compositeField, fsField)
		}
	}
	return nil
}
