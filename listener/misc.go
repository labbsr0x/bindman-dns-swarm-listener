package listener

import (
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	// ErrInitDockerClient error code for problems while creating the Docker Client
	ErrInitDockerClient = iota

	// ErrInitHookClient error code for problems while creating the Hook Client
	ErrInitHookClient = iota

	// ErrTalkToDocker error code for problems while communicating with docker
	ErrTalkToDocker = iota

	// ErrReadingTags error code for problems while reading the BINDMAN_DNS_TAGS environment variable
	ErrReadingTags = iota

	// ErrReadingReverseProxyAddress error code for problems while reading the BINDMAN_REVERSE_PROXY_ADDRESS environment variable
	ErrReadingReverseProxyAddress = iota

	// ErrListingServicesForSync error code for problems when listing the swarm services for syncing
	ErrListingServicesForSync = iota
)

// SandmanService groups together the service name, the host name and the tags
type SandmanService struct {
	ServiceName string
	HostName    []string
	Tags        []string
}

// backoffWait sleeps thread exponentially longer depending on the trial index
func backoffWait(max uint, triesLeft uint, baseDuration time.Duration) {
	waitTime := time.Duration(math.Exp2(float64(max-triesLeft))+1) * baseDuration
	time.Sleep(waitTime)
}

// check checks if the service is ok; aggregate error strings on a slice
func (s *SandmanService) check(contextTags []string) (errs []string) {
	// dumb implementation but linear O(n + m)
	rm := make(map[string]bool)
	for ri := 0; ri < len(contextTags); ri++ {
		rm[contextTags[ri]] = true
	}

	errs = append(errs, "No matching tags found")
	for ri := 0; ri < len(s.Tags); ri++ {
		if rm[s.Tags[ri]] {
			errs = nil
		}
	}

	if len(s.HostName) < 1 {
		errs = append(errs, "Hostname cannot be empty")
	}

	for _, hostName := range s.HostName {
		if strings.TrimSpace(hostName) == "" {
			errs = append(errs, "Hostname cannot be empty")
		}
	}

	return
}

func getHostNamesFromLabel(text string) []string {
	// AND operator ";"
	for _, t := range strings.Split(text, ";") {
		t = strings.TrimSpace(t)
		if strings.HasPrefix(t, "Host:") {
			t = strings.TrimPrefix(t, "Host:")
			t = strings.Replace(t, ",", " ", -1)
			return strings.FieldsFunc(t, unicode.IsSpace)
		}
	}
	return nil
}

func getHostNamesFromLabelRegex(text string) []string {
	// r, _ := regexp.Compile(`(?m)\x60([^\x60]*)\x60|Host\:([^;|^\s]*)`)
	// r, _ := regexp.Compile(`Host\((((\x60([^\x60])*\x60)\,?\s?)*)\)|Host\:([^;|^\s]*)`)
	r, _ := regexp.Compile(`(?m)Host\(((\x60[^\x60]*\x60\,?\s?)*)\)|Host\:([^;|^\s]*)`)
	result := make([]string, 0)
	for _, match := range r.FindAllStringSubmatch(text, -1) {
		if len(match[1]) == 0 {
			result = append(result, match[3])
		} else {
			itensTmp := strings.Split(strings.ReplaceAll(match[1], "`", ""), ",")
			for _, iten := range itensTmp {
				result = append(result, strings.TrimSpace(iten))
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func findTraefikLabelForHostNames(labels map[string]string) []string {
	result := findLabel(traefikV1xHostNameLabelPattern(), labels)
	result = append(result, findLabel(traefikV2xHostNameLabelPattern(), labels)...)
	if len(result) == 0 {
		return nil
	}
	return result
}

func findTraefikLabelForEntryPoints(labels map[string]string) []string {
	result := findLabel(traefikV1xEntryPointsLabelPattern(), labels)
	result = append(result, findLabel(traefikV2xEntryPointsLabelPattern(), labels)...)
	if len(result) == 0 {
		return nil
	}
	return result
}

func findLabel(pattern string, labels map[string]string) []string {
	result := make([]string, 0)
	r, _ := regexp.Compile(pattern)
	for k := range labels {
		if r.MatchString(k) {
			result = append(result, k)
		}
	}
	return result
}

func traefikV1xHostNameLabelPattern() string {
	return `(?m)^traefik\.frontend\.rule$`
}

func traefikV1xEntryPointsLabelPattern() string {
	return `(?m)^traefik\.frontend\.entryPoints$`
}

// - traefik.http.routers.*.rule
func traefikV2xHostNameLabelPattern() string {
	return `(?m)^traefik\.http\.routers\.[\S]*\.rule$`
}

// - traefik.http.routers.*.entryPoints
func traefikV2xEntryPointsLabelPattern() string {
	return `(?m)^traefik\.http\.routers\.[\S]*\.entryPoints$`
}

// ToFqdn converts the name into a fqdn appending a trailing dot.
func ToFqdn(name string) string {
	n := len(name)
	if n == 0 || name[n-1] == '.' {
		return name
	}
	return name + "."
}

// UnFqdn converts the fqdn into a name removing the trailing dot.
func UnFqdn(name string) string {
	n := len(name)
	if n != 0 && name[n-1] == '.' {
		return name[:n-1]
	}
	return name
}
