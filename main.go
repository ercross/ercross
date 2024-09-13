package main

import (
	"encoding/json"
	"fmt"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"net/http"
	"os/exec"
	"time"
)

const signalServerPort = "15002"

var peerConnection *webrtc.PeerConnection

func main() {
	mustInitPeerConnection()
	peerConnection.OnTrack(handleWebRTCStream)
	mustStartSignalServer()
}

func mustStartSignalServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/offer", handleOffer)
	fmt.Println("Signal server listening on port: ", signalServerPort)
	err := http.ListenAndServe(":"+signalServerPort, mux)
	if err != nil {
		panic(err)
	}
}

func mustInitPeerConnection() {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	pcn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(fmt.Errorf("Failed to create peer connection: %v\n", err))
	}

	peerConnection = pcn
}

// todo the ice candidate of client is not being sent to this server
func handleOffer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("***** Received an offer *****")
	var offer webrtc.SessionDescription
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		sendJsonErrorResponse(w, "Failed to decode offer", http.StatusBadRequest)
		fmt.Printf("Failed to decode body: %v\n", err)
		return
	}

	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		sendJsonErrorResponse(w, "Failed to set remote description", http.StatusInternalServerError)
		fmt.Printf("Failed to set remote description: %v\n", err)
		return
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		sendJsonErrorResponse(w, "Failed to create answer", http.StatusInternalServerError)
		fmt.Printf("Failed to create answer: %v\n", err)
		return
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		sendJsonErrorResponse(w, "failed to set local description", http.StatusInternalServerError)
		fmt.Printf("Failed to set local description: %v\n", err)
		return
	}

	_ = json.NewEncoder(w).Encode(peerConnection.LocalDescription())
}

func sendJsonErrorResponse(w http.ResponseWriter, err string, status int) {
	w.WriteHeader(status)
	sErr := json.NewEncoder(w).Encode(err)
	if sErr != nil {
		fmt.Printf("Failed to encode error: %v\n", sErr)
	}
}

func handleWebRTCStream(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	fmt.Println("****** Preparing to handle track ******")
	cmd := exec.Command(
		"ffmpeg",
		"-f", "rawvideo",
		"-pix_fmt", "yuv420p",
		"-s", "640x480",
		"-i", "-",
		"-f", "avfoundation",
		"OBS Virtual Camera",
	)

	// pipe stdin to ffmpeg
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(fmt.Errorf("Failed to create stdin pipe: %v\n", err))
	}

	if err = cmd.Start(); err != nil {
		panic(fmt.Errorf("Failed to start command: %v\n", err))
	}

	// send periodic RTP picture loss indication (PLI) to request keyframes
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for range ticker.C {
			rtcpErr := peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())},
			})
			if rtcpErr != nil {
				fmt.Printf("Failed to send rtcp: %v\n", rtcpErr)
			}
		}
	}()

	// read RTP packets and write them to ffmpeg stdin
	for {
		rtp, _, err := track.ReadRTP()
		if err != nil {
			panic(fmt.Errorf("Failed to read RTP: %v\n", err))
		}

		_, err = stdin.Write(rtp.Payload)
		if err != nil {
			fmt.Printf("Failed to write payload: %v\n", err)
		}
	}
}
