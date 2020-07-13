package listener

import (
	"math"
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
