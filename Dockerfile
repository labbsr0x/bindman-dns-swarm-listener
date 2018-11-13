# BUILD
FROM golang:1.11-alpine as builder

RUN apk add --no-cache git mercurial \
    && go get github.com/gorilla/mux \
    && go get github.com/labbsr0x/sandman-dns-webhook/src/client \
    && go get github.com/labbsr0x/sandman-swarm-listener/src/cmd

WORKDIR $GOPATH/src/github.com/labbsr0x/sandman-swarm-listener/src/cmd

ADD . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /listener .

# PACK
FROM scratch

ENV REVERSE_PROXY_ADDRESS ""
ENV MANAGER_ADDRESS ""

COPY --from=builder /listener /
CMD ["./listener"]
