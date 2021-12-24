// +build !js

package main

import (
    "context"
    "fmt"
    "os"

    "github.com/pion/webrtc/v3"
    "github.com/pion/webrtc/v3/examples/internal/signal"
    "github.com/pion/mediadevices/pkg/codec/opus"
    "github.com/pion/mediadevices/pkg/codec/mmal"
    "github.com/pion/mediadevices"
    _ "github.com/pion/mediadevices/pkg/driver/camera"
    _ "github.com/pion/mediadevices/pkg/driver/microphone"
    "github.com/pion/mediadevices/pkg/prop"
)

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
            // constraint.SampleRate = prop.Int(8000)
        },
        Video: func(constraint *mediadevices.MediaTrackConstraints) {
            constraint.Width = prop.Int(640)
            constraint.Height = prop.Int(480)
	    // constraint.FrameRate = prop.Float(5)
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
    signal.Decode(signal.MustReadStdin(), &offer)

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
    fmt.Println(signal.Encode(*peerConnection.LocalDescription()))

    // Block forever
    select {}
}
