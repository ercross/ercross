# Ercross
Let's call a running instance of [Ercross Webcam Provider](https://github.com/ercross/ercross-webcam-client) app cp1,
where cp means connection-peer.  

Ercross serves as the:
1. signaling server (ss1) to cp1 
2. connection-peer (cp2) for cp1


## How it works 
1. An instance of Ercross webcam provider app (i.e., cp1) sends a WebRTC offer (i.e., RTCSessionDescription) to ss1
2. ss1 returns a WebRTC answer (i.e., another instance of a RTCSessionDescription)
3. cp1 initiates a WebRTC PeerConnection with cp2
4. cp1 continuously captures its camera video feeds
5. cp1 sends captured video streams in RTP packets to cp2
6. cp2 decodes the received feed 
7. and routes the decoded stream as input into any configured virtual webcam driver (e.g., OBS) available on the PC
8. User can then choose OBS as webcam during any virtual meeting

## Discoverability
Both the PC running cp2 and the phone running cp1 must be on the same local WIFI network.   

To get the IP address assigned to the machine Ercross is running on, run
`ifconfig` on MacOS or `ipconfig` on Linux.  
Look for the IP address under the `en0` or `en1` interface, which is usually used for WiFi connections.
It will be in the format `inet 192.168.x.x.`

## Dependencies Concerns
- Ercross has only been tested on MacOS.
- Ercross is currently configured to work with [OBS](https://obsproject.com/) as virtual webcam driver provider

## Installation
1. Build from source using `GOOS=[your-OS] GOARCH=[your-cpu-architecture] go build -o Ercross *.go`
2. Make the binary executable if needed `chmod +x ercross` on Linux/MacOS
3. Add binary to your system PATH `sudo mv myprogram /usr/local/bin/` or simply double-click on the binary if on Windows
4. Run the binary `ercross start`

## Limitations
- Single Connection: Ercross is currently designed handle only one active connection (websocket or peerConnection)