package main

import (
	"fmt"
	"github.com/pion/webrtc/v4"
	"os/exec"
)

func handleTrack(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
	if track.Kind() == webrtc.RTPCodecTypeVideo {
		if err := handleVideoRemoteTrack(track); err != nil {
			panic(fmt.Errorf("error encountered while handling video track: %v", err))
		}
	}
	fmt.Printf("Track %v has not been handled\n", track.Kind())
}

func handleVideoRemoteTrack(track *webrtc.TrackRemote) error {
	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-i", "pipe:0", // read input from stdin (piped data)
		"-vf", "format=yuv420p", // convert to yuv420p format (suitable for virtual camera)
		"-f", "avfoundation", // output format for macOS
		"OBS Virtual Camera", // virtual camera target
	)

	// create pipe for sending RTP packets to FFmpeg
	stdin, err := ffmpegCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to setup ffmpeg input pipe: %w\n", err)
	}

	if err = ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg command: %w", err)
	}

	defer stdin.Close()
	defer ffmpegCmd.Wait()

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			return fmt.Errorf("failed to read rtp packet: %w", err)
		}

		if _, err = stdin.Write(rtpPacket.Payload); err != nil {
			return fmt.Errorf("failed to write rtp packet: %w", err)
		}
	}
}
