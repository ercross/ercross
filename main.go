package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/ws", connectWithRemotePeer)

	fmt.Println("Signaling server listening on port " + port)
	_ = http.ListenAndServe(":"+port, nil)
}
