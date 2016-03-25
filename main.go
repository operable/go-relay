package main

import "github.com/operable/go-relay/relay"

func main() {
	_, err := relay.LoadConfig("/tmp/relay.config")
	if err != nil {
		panic(err)
	}
}
