package listener

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	hook "github.com/labbsr0x/sandman-dns-webhook/src/client"
	hookTypes "github.com/labbsr0x/sandman-dns-webhook/src/types"
	docker "github.com/labbsr0x/sandman-swarm-listener/src/docker"
	dockerTypes "github.com/labbsr0x/sandman-swarm-listener/src/docker/types"
	"github.com/labbsr0x/sandman-swarm-listener/src/types"
)

// SwarmListener owns a Docker Client and a Hook Client
type SwarmListener struct {
	DockerClient  *docker.Client
	WebhookClient *hook.DNSWebhookClient
}

// New instantiates a new swarm listener
func New() *SwarmListener {
	toReturn := SwarmListener{}

	dockerClient, err := docker.NewEnvClient()
	hookTypes.PanicIfError(hookTypes.Error{Message: fmt.Sprintf("Not possible to start the swarm listener; something went wrong while creating the Docker Client: %s", err), Code: types.ErrInitDockerClient, Err: err})
	toReturn.DockerClient = dockerClient

	hookClient, err := hook.New()
	hookTypes.PanicIfError(hookTypes.Error{Message: fmt.Sprintf("Not possible to start the swarm listener; something went wrong while creating the sandman dns manager hook client: %s", err), Code: types.ErrInitHookClient, Err: err})
	toReturn.WebhookClient = hookClient

	return &toReturn
}

// Listen prepares the ground to listen to docker events. it blocks the main thread, keeping it alive
func (sl *SwarmListener) Listen() {
	listeningCtx, cancel := context.WithCancel(context.Background())
	events, errs := sl.DockerClient.Events(listeningCtx, dockerTypes.EventsOptions{})
	go sl.handleEvents(listeningCtx, events)
	go sl.handleErrors(listeningCtx, errs, cancel)
	go sl.gracefulStop(cancel)
	select {} // keep alive magic
}

// handleEvents deals with the events being dispatched by the docker swarm cluster
func (sl *SwarmListener) handleEvents(ctx context.Context, events <-chan dockerTypes.Message) {
	for {
		select {
		case <-ctx.Done():
			logrus.Info("Stopping events handler")
			return
		case event := <-events:
			sl.treatEvent(event)
		}
	}
}

// treatEvent analyses the docker event and take actions accordingly
func (sl *SwarmListener) treatEvent(event dockerTypes.Message) {
	if sl.isDNSEvent(event) {
		logrus.Infof("Got event! Type: %v; Action: %v; Service Name: %v", event.Type, event.Action, event.Actor.Attributes["name"])

	}
}

// treatMessage identifies if the event is a DNS update
func (sl *SwarmListener) isDNSEvent(event dockerTypes.Message) bool {
	return event.Scope == "swarm" && event.Type == "service" && (event.Action == "create" || event.Action == "remove" || event.Action == "update")
}

// gracefullStop cancels gracefully the running goRoutines
func (sl *SwarmListener) gracefulStop(cancel context.CancelFunc) {
	stopCh := make(chan os.Signal)

	signal.Notify(stopCh, syscall.SIGTERM)
	signal.Notify(stopCh, syscall.SIGINT)

	<-stopCh // waits for a stop signal
	sl.stop(0, cancel)
}

// handleErrors deals with errors dispatched in the communication with the docker swarm cluster
func (sl *SwarmListener) handleErrors(ctx context.Context, errs <-chan error, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			logrus.Info("Stopping error handler")
			return
		case err := <-errs:
			logrus.Errorf("Error communicating with the docker swarm cluster: %s", err)
			sl.stop(types.ErrTalkToDocker, cancel)
		}
	}
}

// stops the whole listener
func (sl *SwarmListener) stop(returnCode int, cancel context.CancelFunc) {
	logrus.Infof("Stopping routines...")
	cancel()
	time.Sleep(2 * time.Second)
	os.Exit(returnCode)
}
