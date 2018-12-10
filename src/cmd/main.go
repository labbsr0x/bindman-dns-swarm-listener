package main

import (
	listener "github.com/labbsr0x/bindman-dns-swarm-listener/src/listener"
)

func main() {
	go listener.New().Listen() // fire and forget
	select {}                  //keep alive magic
}
