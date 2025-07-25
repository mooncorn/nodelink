package main

import (
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Data to be broadcasted to a client.
type Data struct {
	Message string `json:"message"`
	From    int    `json:"sender"`
	To      int    `json:"receiver"`
}

// Uniquely defines an incoming client.
type Client struct {
	// Unique Client ID
	ID int
	// Client channel
	Channel chan Data
}

var ID = 0

type Event struct {
	// Data are pushed to this channel
	Message chan Data

	// New client connections
	NewClients chan Client

	// Closed client connections
	ClosedClients chan Client

	// Total client connections
	TotalClients map[int]chan Data
}

func main() {
	router := gin.Default()

	stream := NewEvent()

	router.GET("/stream", HeadersMiddleware(), stream.SSEConnMiddleware(), func(gctx *gin.Context) {
		v, ok := gctx.Get("client")
		if !ok {
			gctx.Status(http.StatusInternalServerError)
			return
		}
		client, ok := v.(Client)
		if !ok {
			gctx.Status(http.StatusInternalServerError)
			return
		}
		// Data to be sent to a specific client
		// Currently this data would be sent to the first client on every new connection
		data := Data{
			Message: "New Client in town",
			From:    client.ID,
			To:      1, // To send this data to a specified client, you can change this to the specific client ID
		}
		// This goroutine will send the above data to Message channel
		// Which will pass through listen(), where it will get sent to the specified client (To)
		go func() {
			if stream.TotalClients[data.To] == nil {
				// Client doesn't exist or disconnected
				log.Printf("Receiver - %d doesn't exist or disconnected.", data.To)
			} else {
				stream.Message <- data
			}
		}()

		gctx.Stream(func(w io.Writer) bool {
			// Stream data to client
			for {
				select {
				// Send msg to the client
				case msg, ok := <-client.Channel:
					if !ok {
						return false
					}
					gctx.SSEvent("message", msg)
					return true
				// Client exit
				case <-gctx.Request.Context().Done():
					return false
				}
			}
		})
	})
}

func HeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Next()
	}
}

// This is a middleware which creates a Client struct variable with unique UUID & Channel,
// And sets it in the connection's context.
func (stream *Event) SSEConnMiddleware() gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Increment global variable ID
		ID += 1
		// Initialize client
		client := Client{
			ID:      ID,
			Channel: make(chan Data),
		}

		// Send new connection to event to store
		stream.NewClients <- client

		defer func() {
			// Send closed connection to event server
			log.Printf("Closing connection : %d", client.ID)
			stream.ClosedClients <- client
		}()

		gctx.Set("client", client)
		gctx.Next()
	}
}

// Initializes Event and starts the event listener
func NewEvent() (event *Event) {
	event = &Event{
		Message:       make(chan Data),
		NewClients:    make(chan Client),
		ClosedClients: make(chan Client),
		TotalClients:  make(map[int]chan Data),
	}

	go event.listen()
	return
}

// It Listens all incoming requests from clients.
// Handles addition and removal of clients and broadcast messages to clients.
func (stream *Event) listen() {
	for {
		select {
		// Add new available client
		case client := <-stream.NewClients:
			stream.TotalClients[client.ID] = client.Channel
			log.Printf("Added client. %d registered clients", len(stream.TotalClients))

		// Remove closed client
		case client := <-stream.ClosedClients:
			delete(stream.TotalClients, client.ID)
			close(client.Channel)
			ID -= 1
			log.Printf("Removed client. %d registered clients", len(stream.TotalClients))

		// Broadcast message to a specific client with client ID fetched from eventMsg.To
		case eventMsg := <-stream.Message:
			stream.TotalClients[eventMsg.To] <- eventMsg
		}
	}
}
