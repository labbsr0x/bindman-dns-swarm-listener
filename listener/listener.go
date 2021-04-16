package listener

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	dockerEvents "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerSwarmTypes "github.com/docker/docker/api/types/swarm"
	docker "github.com/docker/docker/client"
	hook "github.com/labbsr0x/bindman-dns-webhook/src/client"
	hookTypes "github.com/labbsr0x/bindman-dns-webhook/src/types"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

type Builder struct {
	BindmanManagerAddress string
	ReverseProxyAddress   string
	Tags                  []string
}

// SwarmListener owns a Docker Client and a Hook Client
type SwarmListener struct {
	*Builder
	DockerClient  *docker.Client
	WebHookClient *hook.DNSWebhookClient
	managedNames  *cache.Cache
	SyncLock      *sync.RWMutex
}

// New instantiates a new swarm listener
func (b *Builder) New() *SwarmListener {
	dockerClient, err := docker.NewEnvClient()
	hookTypes.PanicIfError(err)

	hookClient, err := hook.New(b.BindmanManagerAddress, http.DefaultClient)
	hookTypes.PanicIfError(err)

	if len(b.Tags) < 1 {
		hookTypes.Panic(hookTypes.Error{
			Message: fmt.Sprintf("The BINDMAN_DNS_TAGS environment variable was not defined"),
			Code:    ErrReadingTags,
			Err:     nil})
	}

	if strings.TrimSpace(b.ReverseProxyAddress) == "" {
		hookTypes.Panic(hookTypes.Error{
			Message: fmt.Sprintf("The BINDMAN_REVERSE_PROXY_ADDRESS environment variable was not defined"),
			Code:    ErrReadingReverseProxyAddress,
			Err:     nil})
	}

	return &SwarmListener{
		DockerClient:  dockerClient,
		WebHookClient: hookClient,
		managedNames:  cache.New(cache.NoExpiration, -1),
		SyncLock:      new(sync.RWMutex),
	}
}

// Sync defines a routine for syncing the dns records present in the docker swarm and being managed by the bindman dns manager
func (sl *SwarmListener) Sync() {
	var maxTries uint = 100
	var leftTries = maxTries

	for leftTries > 0 {
		func() {
			logrus.Debug("initializing sync process")
			defer sl.SyncLock.Unlock()
			sl.SyncLock.Lock()

			sFilters := filters.NewArgs()

			sFilters.Add("label", "traefik.frontend.rule")
			sFilters.Add("label", "traefik.frontend.entryPoints")
			sFilters.Add("label", "traefik.enable")

			services, err := sl.DockerClient.ServiceList(context.Background(), dockerTypes.ServiceListOptions{Filters: sFilters})
			hookTypes.PanicIfError(err)

			logrus.Debugf("%d services found on swarm cluster", len(services))
			for _, service := range services {
				ss := sl.getSandmanServiceFromDockerService(service)
				logrus.Infof("%v", ss)

				// verify required labels before call bindman manager
				if e := ss.check(sl.Tags); e != nil {
					logrus.Debugf("service does not contains required tags. errors %q", e)
					continue
				}

				for _, hostName := range ss.HostName {
					fqdn := ToFqdn(hostName)
					bs, err := sl.WebHookClient.GetRecord(fqdn, "A")
					if err != nil {
						if e, ok := err.(*hookTypes.Error); ok && e.Code == http.StatusNotFound { // means record was not found on manager; so we create it
							sl.delegate("create", ss)
						} else {
							logrus.Error("error connecting bindman manager to get record", err)
						}
						continue
					}

					if bs.Value != sl.ReverseProxyAddress { // if true, record exists and needs to be update
						logrus.Debugf("update needed to domain %s. actual %s new %q", hostName, bs.Value, sl.ReverseProxyAddress)
						sl.delegate("update", ss)
					} else {
						logrus.Debugf("no update needed to domain %s", bs.Name)
					}
				}
			}
		}()
		backoffWait(maxTries, leftTries, time.Minute) // wait time increases exponentially
		leftTries--
	}
}

// Listen prepares the ground to listen to docker events
func (sl *SwarmListener) Listen() {
	listeningCtx, cancel := context.WithCancel(context.Background())

	eFilter := filters.NewArgs()
	eFilter.Add("scope", "swarm")

	//filters.KeyValuePair{Key: "type", Value: "container"})
	events, errs := sl.DockerClient.Events(listeningCtx, dockerTypes.EventsOptions{Filters: eFilter})
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
		defer sl.SyncLock.RUnlock()
		sl.SyncLock.RLock()

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
	if errs := service.check(sl.Tags); errs != nil {
		logrus.Errorf("Invalid service %v. Errors: %v", service.ServiceName, strings.Join(errs, "; "))
		return
	}
	var err error
	switch action {
	case "remove":
		err = sl.removeService(service)
	case "update":
		err = sl.updateService(service)
	case "create":
		err = sl.createService(service)
	default:
		err = fmt.Errorf("action %s is not implemented", action)
	}

	if err != nil {
		logrus.Errorf("Error to %v the HostName '%v' from the service '%v': %v", action, service.HostName, service.ServiceName, err)
	}
}

func (sl *SwarmListener) createService(service *SandmanService) error {
	for _, hostName := range service.HostName {
		fqdn := ToFqdn(hostName)
		err := sl.WebHookClient.AddRecord(fqdn, "A", sl.ReverseProxyAddress) // adds to the dns manager
		if err != nil {
			return err
		}
	}
	sl.managedNames.Set(service.ServiceName, service, cache.NoExpiration) // adds to the cache
	return nil
}

func (sl *SwarmListener) updateService(service *SandmanService) error {
	if value, keyExists := sl.managedNames.Get(service.ServiceName); keyExists {
		if oldService, castOk := value.(*SandmanService); castOk && reflect.DeepEqual(oldService, service) {
			return nil
		}
	}
	for _, hostName := range service.HostName {
		fqdn := ToFqdn(hostName)
		err := sl.WebHookClient.UpdateRecord(&hookTypes.DNSRecord{Name: fqdn, Type: "A", Value: sl.ReverseProxyAddress})
		if err != nil {
			return err
		}
	}
	sl.managedNames.Set(service.ServiceName, service, cache.NoExpiration) // updates cache
	return nil
}

func (sl *SwarmListener) removeService(service *SandmanService) (err error) {
	// for updates, we remove the old entry and later add the new one
	if value, keyExists := sl.managedNames.Get(service.ServiceName); keyExists {
		if oldService, castOk := value.(*SandmanService); castOk {
			for _, hostName := range oldService.HostName {
				fqdn := ToFqdn(hostName)
				err = sl.WebHookClient.RemoveRecord(ToFqdn(fqdn), "A") // removes from the dns manager
				if err == nil {
					sl.managedNames.Delete(oldService.ServiceName) // removes from the cache
				}
			}
		}
	}
	return
}

// getServiceInfo retrieves service information from the available sources (cache and docker inspect)
func (sl *SwarmListener) getServiceInfo(ctx context.Context, serviceName string, action string) (*SandmanService, error) {
	var service *SandmanService
	var err error
	if action == "remove" {
		service, err = sl.getServiceInfoFromCache(serviceName)
	} else if action == "create" || action == "update" {
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
	return nil, fmt.Errorf("unable to retrieve the service '%v' information from cache", serviceName)
}

// getServiceInfoFromInspect calls docker inpect to get data from the service
func (sl *SwarmListener) getServiceInfoFromInspect(ctx context.Context, serviceName string) (*SandmanService, error) {
	var maxRetry uint = 3
	var leftTries = maxRetry
	for leftTries > 0 {
		service, _, err := sl.DockerClient.ServiceInspectWithRaw(ctx, serviceName)
		if err == nil {

			//serviceIDFilter := filters.NewArgs()
			//serviceIDFilter.Add("service", service.ID)
			////serviceIDFilter.Add("desired-state", "running")
			//
			//taskList, err := sl.DockerClient.TaskList(ctx, dockerTypes.TaskListOptions{Filters: serviceIDFilter})
			//if err != nil {
			//	return nil, err
			//}
			//
			//for _, task := range taskList {
			//	if task.Status.State != dockerSwarmTypes.TaskStateRunning {
			//		continue
			//	}
			//	fmt.Printf("%#v /n", task)
			//}

			return sl.getSandmanServiceFromDockerService(service), nil
		}
		backoffWait(maxRetry, leftTries, time.Second) // exponential backoff for retrial
		leftTries--
	}
	return nil, fmt.Errorf("exhausted retries to inspect the service '%v'", serviceName)
}

// getSandmanServiceFromDockerService gets the relevant information from a docker swarm service
func (sl *SwarmListener) getSandmanServiceFromDockerService(service dockerSwarmTypes.Service) *SandmanService {
	logrus.Infof("Docker Service to be handled: %v", service)

	hostNames := make([]string, 0)
	// rule label accepts a sequence of literal hosts.
	hostNameLabels := findTraefikLabelForHostNames(service.Spec.Annotations.Labels)
	for _, hostNameLabel := range hostNameLabels {
		hostNames = append(hostNames, getHostNamesFromLabelRegex(service.Spec.Annotations.Labels[hostNameLabel])...)
	}

	logrus.Debugf("hostNames extracted from label %q", hostNames)

	tags := make([]string, 0)
	tagsLabels := findTraefikLabelForEntryPoints(service.Spec.Annotations.Labels)
	for _, tagLabel := range tagsLabels {
		tags = append(tags, strings.Split(service.Spec.Annotations.Labels[tagLabel], ",")...)
	}

	return &SandmanService{HostName: hostNames, ServiceName: service.Spec.Name, Tags: tags}
}

// treatMessage identifies if the event is a DNS update
func (sl *SwarmListener) isDNSEvent(event dockerEvents.Message) bool {
	//return event.Scope == "swarm" && event.Type == "service" && (event.Action == "create" || event.Action == "remove" || event.Action == "update")
	return event.Type == "service" && (event.Action == "create" || event.Action == "remove" || event.Action == "update")
}

// gracefulStop cancels gracefully the running goRoutines
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
