package main

import (
	listener "github.com/labbsr0x/bindman-dns-swarm-listener/src/listener"
	hook "github.com/labbsr0x/bindman-dns-webhook/src/client"
)

func main() {
	go listener.New(new(hook.BindmanHTTPHelper)).Listen() // fire and forget
	select {}                                             //keep alive magic
}
