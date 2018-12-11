# Bindman Swarm Listener
[![Go Report Card](https://goreportcard.com/badge/github.com/labbsr0x/bindman-dns-swarm-listener)](https://goreportcard.com/report/github.com/labbsr0x/bindman-dns-swarm-listener)

This project defines a Bindman-DNS Swarm Listener that listens to a Docker Swarm event stream and delegates its services' DNS binding updates to a Bindman DNS Manager.

# Configuration

The BindmanDNS Swarm Listener is configurable through environment variables. The following are mandatory:

1. **BINDMAN_DNS_MANAGER_ADDRESS**: the address of the DNS Manager which will manage the identified DNS updates. The DNS Manager should implement the Sandman DNS Webhook;

2. **BINDMAN_REVERSE_PROXY_ADDRESS**: the address of the Reverse Proxy which will load balance requests to the Sandman managed hostnames.

3. **BINDMAN_DNS_TAGS**: a comma-separated list of dns tags enabling this listener to choose which service updates its dns manager should deal with

By the default, the Bindman DNS Swarm Listener will query the docker host through its `docker.sock` (**and thus should be mapped as a volume**). That can be changed through the following **optional** environment varibles:

1. **DOCKER_HOST**: docker server url, if necessary (e.g. remotely listening);

2. **DOCKER_API_VERSION**: the version of the docker API to reach; defaults to latest when empty; 

3. **DOCKER_TLS_VERIFY**: enable or disable TLS verification, off by default;

4. **DOCKER_CERT_PATH**: the path to load the TLS certificates from.

A TTL needs to be defined for the DNS rules being managed by the Sandman project. By default, the TTL is **3600 seconds**. To change it, write to the `BINDMAN_DNS_TTL` environment variable.

The runtime mode can also be configured with the use of the `BINDMAN_MODE` environment variable: `empty|PROD` for production mode or `DEBUG` for debug mode.

# DNS Rules

The DNS binding rules should be written as service rules according to the format expected by [Traefik](https://github.com/containous/traefik), the default Reverse Proxy and Load Balancer adopted by this project. That is, each swarm service should run with the following deployment labels defined:

```
...
deploy:
    ...
    labels:
        - traefik.frontend.rule=Host:<the service host name>
        - traefik.port=<the service exposed port>
        - traefik.frontend.entryPoints=<comma-separated dns tags>
    ...
...
```

This project delegates hostname management by adopting a Tag system, where each service launched by a Docke Swarm cluster is annotated with a set of Tags and each DNS listener is also run with its own set of Tags.

**The intersection of these two Tag sets indicates which DNS Managers will manage which service hostname attribution.**

It is through the label `traefik.frontend.entryPoints` that we are expecting these Tags to be defined, as a comma-separated list of strings.
