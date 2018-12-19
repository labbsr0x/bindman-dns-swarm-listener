package listener

import (
	"math"
	"strings"
	"time"
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
	HostName    string
	Tags        []string
}

// backoffWait sleeps thread exponentially longer depending on the trial index
func backoffWait(max uint, triesLeft uint, baseDuration time.Duration) {
	waitSeconds := time.Duration(math.Exp2(float64(max-triesLeft))+1) * baseDuration
	time.Sleep(waitSeconds)
}

// check checks if the service is ok; aggregate error strings on a slice
func (s *SandmanService) check(contextTags []string) (ok bool, errs []string) {
	ok = false

	// dumb implementation but linear O(n + m)
	rm := make(map[string]bool)
	for ri := 0; ri < len(contextTags); ri++ {
		rm[contextTags[ri]] = true
	}

	errs = append(errs, "No matching tags found")
	for ri := 0; ri < len(s.Tags); ri++ {
		if rm[s.Tags[ri]] {
			ok = true
			errs = nil
		}
	}

	if strings.Trim(s.HostName, " ") == "" {
		ok = false
		errs = append(errs, "Hostname cannot be empty")
	}

	return ok, errs
}
