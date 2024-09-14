package main

import (
	"fmt"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"os/exec"
	"time"
)

func handleRemoteTrack(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
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
