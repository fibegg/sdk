package fibe

import (
	"net/url"
	"regexp"
	"strconv"
)

var nameIdentifierPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
var numericIdentifierPattern = regexp.MustCompile(`^\d+$`)

func identifierPath(prefix, identifier string) string {
	return prefix + "/" + url.PathEscape(identifier)
}

func int64Identifier(id int64) string {
	return strconv.FormatInt(id, 10)
}

func nameIdentifier(value string) bool {
	return value != "new" && nameIdentifierPattern.MatchString(value) && !numericIdentifierPattern.MatchString(value)
}

func identifierNameBase(name string, fallback string) string {
	if nameIdentifier(name) {
		return name
	}
	if nameIdentifier(fallback) {
		return fallback
	}
	return "resource"
}
