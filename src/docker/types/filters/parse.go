package filters

import (
	"encoding/json"

	"github.com/labbsr0x/sandman-swarm-listener/src/docker/types/versions"
)

// Args stores a mapping of keys to a set of multiple values.
type Args struct {
	fields map[string]map[string]bool
}

// Len returns the number of keys in the mapping
func (args Args) Len() int {
	return len(args.fields)
}

// ToParamWithVersion encodes Args as a JSON string. If version is less than 1.22
// then the encoded format will use an older legacy format where the values are a
// list of strings, instead of a set.
//
// Deprecated: Use ToJSON
func ToParamWithVersion(version string, a Args) (string, error) {
	if a.Len() == 0 {
		return "", nil
	}

	if version != "" && versions.LessThan(version, "1.22") {
		buf, err := json.Marshal(convertArgsToSlice(a.fields))
		return string(buf), err
	}

	return ToJSON(a)
}

// ToJSON returns the Args as a JSON encoded string
func ToJSON(a Args) (string, error) {
	if a.Len() == 0 {
		return "", nil
	}
	buf, err := json.Marshal(a)
	return string(buf), err
}

func convertArgsToSlice(f map[string]map[string]bool) map[string][]string {
	m := map[string][]string{}
	for k, v := range f {
		values := []string{}
		for kk := range v {
			if v[kk] {
				values = append(values, kk)
			}
		}
		m[k] = values
	}
	return m
}
