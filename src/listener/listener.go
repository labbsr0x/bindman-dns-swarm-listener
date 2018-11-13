package listener

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	hook "github.com/labbsr0x/sandman-dns-webhook/src/client"
	hookTypes "github.com/labbsr0x/sandman-dns-webhook/src/types"

	dockerTypes "github.com/docker/docker/api/types"
	dockerEvents "github.com/docker/docker/api/types/events"
	docker "github.com/docker/docker/client"
)

// SwarmListener owns a Docker Client and a Hook Client
type SwarmListener struct {
	DockerClient  *docker.Client
	WebhookClient *hook.DNSWebhookClient
	TTL           int
	managedNames  map[string]string
}

// New instantiates a new swarm listener
func New() *SwarmListener {
	toReturn := SwarmListener{}

	dockerClient, err := docker.NewEnvClient()
	hookTypes.PanicIfError(hookTypes.Error{Message: fmt.Sprintf("Not possible to start the swarm listener; something went wrong while creating the Docker Client: %s", err), Code: ErrInitDockerClient, Err: err})
	toReturn.DockerClient = dockerClient

	hookClient, err := hook.New()
	hookTypes.PanicIfError(hookTypes.Error{Message: fmt.Sprintf("Not possible to start the swarm listener; something went wrong while creating the sandman dns manager hook client: %s", err), Code: ErrInitHookClient, Err: err})
	toReturn.WebhookClient = hookClient

	ttl := os.Getenv("SANDMAN_DNS_TTL")
	ttl = strings.Trim(ttl, " ")
	toReturn.TTL, err = strconv.Atoi(ttl)
	if err != nil {
		logrus.Errorf("Invalid TTL. Going default.")
		toReturn.TTL = 3600
	}

	toReturn.managedNames = make(map[string]string)
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
func (sl *SwarmListener) handleEvents(ctx context.Context, events <-chan dockerEvents.Message) {
	for {
		select {
		case <-ctx.Done():
			logrus.Info("Stopping events handler")
			return
		case event := <-events:
			go sl.treatEvent(ctx, event)
		}
	}
}

// treatEvent analyses the docker event and take actions accordingly. will retry tree times before it gives up
func (sl *SwarmListener) treatEvent(ctx context.Context, event dockerEvents.Message) {
	if sl.isDNSEvent(event) {
		logrus.Infof("Got DNS Event! Action: %v; Service Name: %v", event.Action, event.Actor.Attributes["name"])
		var retries uint = 3
		for retries > 0 {
			service, _, err := sl.DockerClient.ServiceInspectWithRaw(ctx, event.Actor.Attributes["name"], dockerTypes.ServiceInspectOptions{})
			if err == nil {
				name := service.Spec.Annotations.Labels["traefik.frontend.rule"]
				tags := strings.Split(service.Spec.Annotations.Labels["traefik.frontend.entryPoints"], ",")
				sl.delegate(event.Action, name, tags)
				break
			} else {
				backoffWait(3, retries) // exponential backoff for retrial
				retries--
			}
		}
		if retries == 0 {
			logrus.Errorf("Exhausted retries to inspect the service '%v' and %v its DNS Bindings", event.Actor.Attributes["name"], event.Action)
		}
	}
}

// delegate appropriately calls the dns manager to handle the addition or removal of a DNS rule
func (sl *SwarmListener) delegate(action string, hostName string, tags []string) {
	if strings.Trim(hostName, " ") != "" {
		if action == "remove" || action == "update" {
			sl.WebhookClient.RemoveRecord(hostName)
		}

		if action == "create" || action == "update" {
			sl.WebhookClient.AddRecord(hostName, tags, sl.TTL)
		}
	}
}

// treatMessage identifies if the event is a DNS update
func (sl *SwarmListener) isDNSEvent(event dockerEvents.Message) bool {
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
			sl.stop(ErrTalkToDocker, cancel)
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

// backoffWait sleeps thread exponentially longer depending on the trial index
func backoffWait(max uint, triesLeft uint) {
	waitSeconds := time.Duration(math.Exp2(float64(max-triesLeft))+1) * time.Second
	time.Sleep(waitSeconds)
}

const (
	// ErrInitDockerClient error code for problems while creating the Docker Client
	ErrInitDockerClient = iota

	// ErrInitHookClient error code for problems while creating the Hook Client
	ErrInitHookClient = iota

	// ErrTalkToDocker error code for problems while communicating with docker
	ErrTalkToDocker = iota
)
