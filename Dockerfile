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

ENV SANDMAN_REVERSE_PROXY_ADDRESS ""
ENV SANDMAN_DNS_MANAGER_ADDRESS ""
ENV SANDMAN_DNS_TAGS ""
ENV SANDMAN_DNS_TTL ""

COPY --from=builder /listener /
CMD ["./listener"]
