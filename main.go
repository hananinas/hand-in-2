package main

import (
	"hospital/internal/client"
	"hospital/internal/server"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the server in a separate goroutine
	go func() {
		defer wg.Done()
		server.StartServer()
	}()

	// Start the client
	client.StartClient()

	// Wait for the server to finish
	wg.Wait()
}
