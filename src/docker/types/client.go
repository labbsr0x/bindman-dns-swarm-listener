package types

import "github.com/labbsr0x/sandman-swarm-listener/src/docker/types/filters"

// EventsOptions holds parameters to filter events with.
type EventsOptions struct {
	Since   string
	Until   string
	Filters filters.Args
}
