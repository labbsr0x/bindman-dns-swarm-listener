package listener

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	hook "github.com/labbsr0x/sandman-dns-webhook/src/client"
	hookTypes "github.com/labbsr0x/sandman-dns-webhook/src/types"

	dockerTypes "github.com/docker/docker/api/types"
	dockerEvents "github.com/docker/docker/api/types/events"
	docker "github.com/docker/docker/client"

	cache "github.com/patrickmn/go-cache"
)

// SwarmListener owns a Docker Client and a Hook Client
type SwarmListener struct {
	DockerClient  *docker.Client
	WebhookClient *hook.DNSWebhookClient
	managedNames  *cache.Cache
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

	toReturn.managedNames = cache.New(cache.NoExpiration, -1*time.Second)
	return &toReturn
}

// Listen prepares the ground to listen to docker events
func (sl *SwarmListener) Listen() {
	listeningCtx, cancel := context.WithCancel(context.Background())
	events, errs := sl.DockerClient.Events(listeningCtx, dockerTypes.EventsOptions{})
	go sl.handleEvents(listeningCtx, events)
	go sl.handleErrors(listeningCtx, errs, cancel)
	go sl.gracefulStop(cancel)
	logrus.Info("Start listening...")
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
		serviceName := event.Actor.Attributes["name"]
		logrus.Infof("Got DNS Event! Action: %v; Service Name: %v", event.Action, serviceName)

		service, err := sl.getServiceInfo(ctx, serviceName, event.Action)
		if err != nil {
			logrus.Errorf("Unable to retrieve service '%v' info to %v its DNS bindings: %v", serviceName, event.Action, err)
		} else {
			sl.delegate(event.Action, service)
		}
	}
}

// delegate appropriately calls the dns manager to handle the addition or removal of a DNS rule
func (sl *SwarmListener) delegate(action string, service *SandmanService) {
	if strings.Trim(service.HostName, " ") != "" {
		var ok bool
		var err error
		// for updates, we remove the old entry and later add the new one
		if action == "remove" || action == "update" {
			if value, keyExists := sl.managedNames.Get(service.ServiceName); keyExists {
				if oldService, keyExists := value.(*SandmanService); keyExists {
					ok, err = sl.WebhookClient.RemoveRecord(oldService.HostName) // removes from the dns manager
					if ok {
						sl.managedNames.Delete(oldService.ServiceName) // removes from the cache
					}
				}
			}
		}

		if action == "create" || action == "update" {
			ok, err = sl.WebhookClient.AddRecord(service.HostName, service.Tags) // adds to the dns manager
			if ok {
				sl.managedNames.Set(service.ServiceName, service, cache.NoExpiration) // adds to the cache
			}
		}

		if !ok {
			logrus.Errorf("Error to %v the HostName '%v' from the service '%v': %v", action, service.HostName, service.ServiceName, err)
		}
	}
}

// getServiceInfo retrieves service information from the available sources (cache and docker inspect)
func (sl *SwarmListener) getServiceInfo(ctx context.Context, serviceName string, action string) (*SandmanService, error) {
	var service *SandmanService
	var err error
	if action == "remove" {
		service, err = sl.getServiceInfoFromCache(serviceName)
	}

	if action == "create" || action == "update" {
		service, err = sl.getServiceInfoFromInspect(ctx, serviceName)
	}

	return service, err
}

// getServiceInfoFromCache tries to get the service info from the cache
func (sl *SwarmListener) getServiceInfoFromCache(serviceName string) (*SandmanService, error) {
	if value, ok := sl.managedNames.Get(serviceName); ok {
		if service, ok := value.(*SandmanService); ok {
			return service, nil
		}
	}
	return nil, fmt.Errorf("Unable to retrieve the service '%v' information from cache", serviceName)
}

// getServiceInfoFromInspect calls docker inpect to get data from the service
func (sl *SwarmListener) getServiceInfoFromInspect(ctx context.Context, serviceName string) (*SandmanService, error) {
	var retries uint = 3
	for retries > 0 {
		service, _, err := sl.DockerClient.ServiceInspectWithRaw(ctx, serviceName, dockerTypes.ServiceInspectOptions{})
		if err == nil {
			hostName := strings.TrimPrefix(service.Spec.Annotations.Labels["traefik.frontend.rule"], "Host:")
			tags := strings.Split(service.Spec.Annotations.Labels["traefik.frontend.entryPoints"], ",")
			return &SandmanService{HostName: hostName, ServiceName: serviceName, Tags: tags}, nil
		}
		backoffWait(3, retries) // exponential backoff for retrial
		retries--
	}
	return nil, fmt.Errorf("Exhausted retries to inspect the service '%v'", serviceName)
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
