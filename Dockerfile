# BUILD
FROM golang:1.12.5-stretch as builder

# RUN apk add --no-cache gcc build-base git mercurial 

# ENV DIR $GOPATH/src/github.com/labbsr0x/bindman-dns-swarm-listener/src
# RUN mkdir -p ${DIR}
WORKDIR /build-app

ADD ./ ./
RUN go mod download
RUN go test ./...

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /listener .

# PACK
FROM alpine

ENV BINDMAN_REVERSE_PROXY_ADDRESS ""
ENV BINDMAN_DNS_MANAGER_ADDRESS ""
ENV BINDMAN_DNS_TAGS ""
ENV BINDMAN_DNS_TTL ""
ENV BINDMAN_MODE ""

ENV DOCKER_HOST ""
ENV DOCKER_API_VERSION ""
ENV DOCKER_TLS_VERIFY ""
ENV DOCKER_CERT_PATH ""

COPY --from=builder /listener /
CMD ["./listener"]
