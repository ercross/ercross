package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"net/http"
)

const port = "15002"

var (
	peerConnection *webrtc.PeerConnection
	conn           *websocket.Conn

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type message struct {
	Type      string                     `json:"type"`
	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
}

type errorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func newErrorMessage(msg string) errorMessage {
	return errorMessage{
		Type:    "error",
		Message: msg,
	}
}

func connectWithRemotePeer(w http.ResponseWriter, r *http.Request) {

	// handle only connection
	if conn != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Websocket connection already established with " + conn.RemoteAddr().String()))
		return
	}

	// upgrade the connection to a websocket
	var err error
	conn, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("error upgrading connection: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer conn.Close()

	if err = initPeerConnection(); err != nil {
		sendErrorMessageToRemotePeer(err, "faild to initialize peer connection. Terminating websocket connection...")
		fmt.Printf("error initializing peer connection: %v\nWebsocket terminated", err)
		return
	}
	handleIncomingMessages()
}

func handleIncomingMessages() {
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("error reading message from websocket connection: %v\n", err)
			sendErrorMessageToRemotePeer(err, "error reading last message")
			conn.Close()
			peerConnection.Close()
			break
		}

		var msg message
		if err = json.Unmarshal(msgBytes, &msg); err != nil {
			sendErrorMessageToRemotePeer(err, "can not marshal last message")
			continue
		}

		switch msg.Type {
		case "offer":
			handleOfferMessage(&msg)
		case "candidate":
			handleIceCandidateMessage(&msg)
		}
	}
}

func handleOfferMessage(msg *message) {
	err := peerConnection.SetRemoteDescription(*msg.SDP)
	if err != nil {
		sendErrorMessageToRemotePeer(err, "failed to set remote description")
		return
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		sendErrorMessageToRemotePeer(err, "error creating answer")
		return
	}
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		sendErrorMessageToRemotePeer(err, "failed to set answer")
		return
	}

	answerMsg := message{
		Type: "answer",
		SDP:  &answer,
	}

	if err = conn.WriteJSON(answerMsg); err != nil {
		sendErrorMessageToRemotePeer(err, "error sending answer")
		return
	}
}

func handleIceCandidateMessage(msg *message) {
	err := peerConnection.AddICECandidate(*msg.Candidate)
	if err != nil {
		sendErrorMessageToRemotePeer(err, "error adding ice candidate")
		return
	}
}

func initPeerConnection() error {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	var err error
	peerConnection, err = webrtc.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create new peer connection: %v", err)
	}

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		sendIceCandidateToRemotePeer(candidate)
	})

	peerConnection.OnTrack(handleRemoteTrack)

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Connection state has changed: %s\n", state.String())
		if state == webrtc.PeerConnectionStateFailed {
			fmt.Printf("Peer Connection Failed. Closing websocket... \n")
			_ = conn.Close()
		}
	})

	return nil
}

func sendIceCandidateToRemotePeer(candidate *webrtc.ICECandidate) {
	if candidate == nil {
		return
	}

	candidateJSON := candidate.ToJSON()
	iceCandidateMessage := message{
		Type:      "candidate",
		Candidate: &candidateJSON,
	}
	err := conn.WriteJSON(iceCandidateMessage)
	if err != nil {
		fmt.Printf("Failed to send iceCandidateMessage: %v\n", err)
		return
	}
}

func sendErrorMessageToRemotePeer(err error, msg string) {
	_ = conn.WriteJSON(newErrorMessage(fmt.Sprintf("%s: %v", msg, err)))
}
