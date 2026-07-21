package main

import (
	"fmt"
	"time"
)

func main() {
	frames := make(chan string)
	heartbeat := make(chan string)
	disconnect := make(chan bool)

	// Simulate a session sending frames
	go func ()  {
		for i := 1; i <= 3; i++ {
			time.Sleep(200 * time.Millisecond)
			frames <- fmt.Sprintf("frame %d", i)
		}		
	} ()

	// Simulate a heartbeat ticker
	go func() {
		for i := 1; i <= 6; i ++ {
			time.Sleep(100 * time.Millisecond)
			heartbeat <- "ping"
		}
	}()

	// Simulate disconnection after 500ms
	go func ()  {
		time.Sleep(500 * time.Millisecond)		
		disconnect <- true
	}()

    // select lets the server react to whichever event arrives first.
    // In the proctoring server this is the core of each session handler —
    // react to frames, heartbeats, disconnections, and context cancellation.

	for {
		select {
		case frame := <-frames:
			fmt.Printf("received: %s\n", frame)
		case <- heartbeat:
			fmt.Println("heartbeat received")
		case <- disconnect:
			fmt.Println("session disconnected — cleaning up")
			return
		}
	}


	
}