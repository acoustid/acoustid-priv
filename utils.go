package priv

import "strings"

func IsValidCatalogName(name string) bool {
	if strings.HasPrefix(name, "_") {
		return false
	}
	return true
}

func IsValidTrackID(name string) bool {
	if strings.HasPrefix(name, "_") {
		return false
	}
	return true
}
