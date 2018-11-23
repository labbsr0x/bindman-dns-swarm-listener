package listener

import (
	"math"
	"time"
)

const (
	// ErrInitDockerClient error code for problems while creating the Docker Client
	ErrInitDockerClient = iota

	// ErrInitHookClient error code for problems while creating the Hook Client
	ErrInitHookClient = iota

	// ErrTalkToDocker error code for problems while communicating with docker
	ErrTalkToDocker = iota
)

// SandmanService groups together the service name, the host name and the tags
type SandmanService struct {
	ServiceName string
	HostName    string
	Tags        []string
}

// backoffWait sleeps thread exponentially longer depending on the trial index
func backoffWait(max uint, triesLeft uint) {
	waitSeconds := time.Duration(math.Exp2(float64(max-triesLeft))+1) * time.Second
	time.Sleep(waitSeconds)
}
