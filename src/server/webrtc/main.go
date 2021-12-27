// +build !js

package main

import (
    "context"
    "fmt"
    "os"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"strings"

    "github.com/pion/webrtc/v3"
    "github.com/pion/mediadevices/pkg/codec/opus"
    "github.com/pion/mediadevices/pkg/codec/mmal"
    "github.com/pion/mediadevices"
    _ "github.com/pion/mediadevices/pkg/driver/camera"
    _ "github.com/pion/mediadevices/pkg/driver/microphone"
    "github.com/pion/mediadevices/pkg/prop"
)

// Allows compressing offer/answer to bypass terminal input limits.
const compress = false

// MustReadStdin blocks until input is received from stdin
func MustReadStdin() string {
	r := bufio.NewReader(os.Stdin)

	var in string
	for {
		var err error
		in, err = r.ReadString('\n')
		if err != io.EOF {
			if err != nil {
				panic(err)
			}
		}
		in = strings.TrimSpace(in)
		if len(in) > 0 {
			break
		}
	}

	fmt.Println("")

	return in
}

// Encode encodes the input in base64
// It can optionally zip the input before encoding
func Encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	if compress {
		b = zip(b)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes the input from base64
// It can optionally unzip the input after decoding
func Decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	if compress {
		b = unzip(b)
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}

func zip(in []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(in)
	if err != nil {
		panic(err)
	}
	err = gz.Flush()
	if err != nil {
		panic(err)
	}
	err = gz.Close()
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

func unzip(in []byte) []byte {
	var b bytes.Buffer
	_, err := b.Write(in)
	if err != nil {
		panic(err)
	}
	r, err := gzip.NewReader(&b)
	if err != nil {
		panic(err)
	}
	res, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return res
}

func main() {

    // configure audio codec
    opusParams, err := opus.NewParams()
    if err != nil {
        panic("could not configure audio codec")
    }

    // configure video codec
    x264Params, _ := mmal.NewParams()

    codecSelector := mediadevices.NewCodecSelector(
        mediadevices.WithVideoEncoders(&x264Params),
        mediadevices.WithAudioEncoders(&opusParams),
    )

    stream, _ := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
        Audio: func(constraint *mediadevices.MediaTrackConstraints) {
            constraint.ChannelCount = prop.Int(2)
        },
        Video: func(constraint *mediadevices.MediaTrackConstraints) {
            constraint.Width = prop.Int(640)
            constraint.Height = prop.Int(480)
        },
        Codec: codecSelector,
    })

    if stream == nil {
        panic("unable to find media devices")
    }

    peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
        ICEServers: []webrtc.ICEServer{
            {
                URLs: []string{"stun:stun.l.google.com:19302"},
            },
        },
    })

    if err != nil {
	fmt.Println("unable to create peer connection")
        panic(err)
    }

    defer func() {
        if cErr := peerConnection.Close(); cErr != nil {
            fmt.Printf("cannot close peerConnection: %v\n", cErr)
        }
    }()

    _, iceConnectedCtxCancel := context.WithCancel(context.Background())

    if len(stream.GetVideoTracks()) > 0 {
        _, videoTrackErr := peerConnection.AddTrack(stream.GetVideoTracks()[0].(*mediadevices.VideoTrack))
        if videoTrackErr != nil {
	    fmt.Println("unable to add video track")
            panic(videoTrackErr)
        }
    }

    if len(stream.GetAudioTracks()) > 0 {
        _, audioTrackErr := peerConnection.AddTrack(stream.GetAudioTracks()[0].(*mediadevices.AudioTrack))
        if audioTrackErr != nil {
	    fmt.Println("unable to add audio track")
            panic(audioTrackErr)
        }
    }

    // Set the handler for ICE connection state
    // This will notify you when the peer has connected/disconnected
    peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
        fmt.Printf("Connection State has changed %s \n", connectionState.String())
        if connectionState == webrtc.ICEConnectionStateConnected {
            iceConnectedCtxCancel()
        }
    })

    // Set the handler for Peer connection state
    // This will notify you when the peer has connected/disconnected
    peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
        fmt.Printf("Peer Connection State has changed: %s\n", s.String())

        if s == webrtc.PeerConnectionStateFailed {
            // Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
            // Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
            // Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
            fmt.Println("Peer Connection has gone to failed exiting")
            os.Exit(0)
        }
    })

    // Wait for the offer to be pasted
    offer := webrtc.SessionDescription{}
    Decode(MustReadStdin(), &offer)

    // Set the remote SessionDescription
    if err = peerConnection.SetRemoteDescription(offer); err != nil {
	fmt.Println("unable to set remote description")
        panic(err)
    }

    // Create answer
    answer, err := peerConnection.CreateAnswer(nil)
    if err != nil {
	fmt.Println("unable to create answer")
        panic(err)
    }

    // Create channel that is blocked until ICE Gathering is complete
    gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

    // Sets the LocalDescription, and starts our UDP listeners
    if err = peerConnection.SetLocalDescription(answer); err != nil {
	fmt.Println("unable to set local description")
        panic(err)
    }

    // Block until ICE Gathering is complete, disabling trickle ICE
    // we do this because we only can exchange one signaling message
    // in a production application you should exchange ICE Candidates via OnICECandidate
    <-gatherComplete

    // Output the answer in base64 so we can paste it in browser
    fmt.Println(Encode(*peerConnection.LocalDescription()))

    // Block forever
    select {}
}
