package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// SSEClient represents an SSE client connection
type SSEClient struct {
	ID     int
	Stream chan string
}

// SSEServer manages clients and broadcasting messages
type SSEServer struct {
	Clients      map[int]*SSEClient
	AddClient    chan *SSEClient
	RemoveClient chan int
	Broadcast    chan string
}

func NewSSEServer() *SSEServer {
	return &SSEServer{
		Clients:      make(map[int]*SSEClient),
		AddClient:    make(chan *SSEClient),
		RemoveClient: make(chan int),
		Broadcast:    make(chan string),
	}
}

// Run starts the SSE server and handles adding/removing clients and broadcasting messages
func (server *SSEServer) Run() {
	for {
		select {
		case client := <-server.AddClient:
			server.Clients[client.ID] = client
			log.Printf("Client %d connected", client.ID)

		case id := <-server.RemoveClient:
			delete(server.Clients, id)
			log.Printf("Client %d disconnected", id)

		case message := <-server.Broadcast:
			for _, client := range server.Clients {
				client.Stream <- message
			}
		}
	}
}

// SSEHandler handles incoming SSE connections
func (server *SSEServer) SSEHandler(w http.ResponseWriter, r *http.Request) {
	clientID := len(server.Clients) + 1
	client := &SSEClient{
		ID:     clientID,
		Stream: make(chan string),
	}

	// Register new client
	server.AddClient <- client

	// Ensure headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	go func() {
		<-ctx.Done() // Wait until the client disconnects
		server.RemoveClient <- client.ID
	}()

	// Stream messages to client
	for message := range client.Stream {
		fmt.Fprintf(w, "data: %s\n\n", message)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

// BroadcastMessages periodically sends messages to all clients
func (server *SSEServer) BroadcastMessages() {
	for {
		time.Sleep(2 * time.Second)
		message := fmt.Sprintf("Current Time: %s", time.Now().Format("2006-01-02 15:04:05"))
		server.Broadcast <- message
	}
}

func main() {
	server := NewSSEServer()
	go server.Run()
	go server.BroadcastMessages()

	http.HandleFunc("/", server.SSEHandler)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
