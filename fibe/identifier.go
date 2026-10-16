package fibe

import (
	"net/url"
	"strconv"
)

func identifierPath(prefix, identifier string) string {
	return prefix + "/" + url.PathEscape(identifier)
}

func int64Identifier(id int64) string {
	return strconv.FormatInt(id, 10)
}
