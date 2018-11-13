package main

import (
	listener "github.com/labbsr0x/sandman-swarm-listener/src/listener"
)

func main() {
	go listener.New().Listen() // fire and forget
	select {}                  //keep alive magic
}
