package main

import (
	"log"
)

func main() {
	store := NewPostgressStore()
	if err := store.Init(); err != nil {
		log.Fatal(err)
	}

	server := NewAPIServer("localhost:8080", store)
	server.Run()
}
