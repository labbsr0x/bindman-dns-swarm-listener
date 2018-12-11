# BUILD
FROM golang:1.11-alpine as builder

RUN apk add --no-cache git mercurial 

RUN mkdir -p $GOPATH/src/github.com/labbsr0x/sandman-swarm-listener/src
WORKDIR $GOPATH/src/github.com/labbsr0x/sandman-swarm-listener/src

ADD ./src ./
RUN go get -v ./...

WORKDIR $GOPATH/src/github.com/labbsr0x/sandman-swarm-listener/src/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /listener .

# PACK
FROM scratch

ENV BINDMAN_REVERSE_PROXY_ADDRESS ""
ENV BINDMAN_DNS_MANAGER_ADDRESS ""
ENV BINDMAN_DNS_TAGS ""
ENV BINDMAN_DNS_TTL ""
ENV BINDMAN_MODE ""

ENV DOCKER_HOST = ""
ENV DOCKER_API_VERSION = ""
ENV DOCKER_TLS_VERIFY ""
ENV DOCKER_CERT_PATH ""

COPY --from=builder /listener /
CMD ["./listener"]
