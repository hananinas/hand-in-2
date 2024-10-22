package main

import (
	"hospital/internal/client"
	"hospital/internal/server"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	// Start the server in a separate goroutine
	go func() {
		defer wg.Done()
		server.StartServer()
	}()
	// Start the client
	go func() {
		defer wg.Done()
		client.StartClient(&wg)
	}()

	// Wait for the server to finish
	wg.Wait()
}
