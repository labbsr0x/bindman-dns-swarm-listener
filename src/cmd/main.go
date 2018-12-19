package main

import (
	listener "github.com/labbsr0x/bindman-dns-swarm-listener/src/listener"
	hook "github.com/labbsr0x/bindman-dns-webhook/src/client"
)

func main() {
	l := listener.New(new(hook.BindmanHTTPHelper))
	go l.Sync()   // fire and forget
	go l.Listen() // fire and forget
	select {}     //keep alive magic
}
