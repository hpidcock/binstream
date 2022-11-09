package main

import (
	"regexp"
)

var binRegex = regexp.MustCompile(`juju-(.+)-([^-]+)-([^-]+)\.tgz`)

func matchBinName(name string) (string, string, string, bool) {
	v := binRegex.FindStringSubmatch(name)
	if v == nil {
		return "", "", "", false
	}
	return v[1], v[2], v[3], true
}
