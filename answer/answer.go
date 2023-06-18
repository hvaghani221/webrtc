package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/pion/webrtc/v3"

	"github.com/hvaghani221/webrtc/firebase"
	"github.com/hvaghani221/webrtc/utils"
)

const (
	localCandidateFile    = utils.AnswerCandidate
	localOfferFile        = utils.AnswerOffer
	localDescriptionFile  = utils.AnswerDescription
	remoteCandidateFile   = utils.OfferCandidate
	remoteOfferFile       = utils.OfferOffer
	remoteDescriptionFile = utils.OfferDescription
)

func main() {
	client := utils.ReturnOrPanic(firebase.Init())
	defer client.Close()

	var candidateMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)
	finalLocalCandidate := make([]*webrtc.ICECandidate, 0)

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection := utils.ReturnOrPanic(webrtc.NewPeerConnection(config))

	defer func() {
		if err := peerConnection.Close(); err != nil {
			fmt.Println("Closing the peerConnection, err: ", err)
		}
	}()

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		candidateMux.Lock()
		defer candidateMux.Unlock()

		desc := peerConnection.RemoteDescription()
		if desc == nil {
			pendingCandidates = append(pendingCandidates, c)
		} else {
			fmt.Println("OnICECandidate non nil")
			finalLocalCandidate = append(finalLocalCandidate, c)
		}
	})

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

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New data channel %s %d\n", d.Label(), d.ID())

		d.OnOpen(func() {
			fmt.Println("Data channel connected")
			utils.PanicIf(d.SendText("Hello from answer"))
		})

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message received from dataChannel %s: %s\n", d.Label(), string(msg.Data))
		})
	})

	fmt.Println("Waiting for remote offer")
	sdp := client.WaitForOffer(remoteOfferFile)
	utils.PanicIf(peerConnection.SetRemoteDescription(sdp))
	fmt.Println("Received sdp request")

	answer := utils.ReturnOrPanic(peerConnection.CreateAnswer(nil))

	utils.PanicIf(client.ShareOffer(localOfferFile, answer))
	fmt.Println("Sent SDP request")

	fmt.Println("SetLocalDescription")
	utils.PanicIf(peerConnection.SetLocalDescription(answer))

	fmt.Println("Waiting for remote candidates")
	remoteCandidates := client.WaitForCandidates(remoteCandidateFile)
	fmt.Println("Adding remote ICECandidates")
	for _, rc := range remoteCandidates {
		utils.PanicIf(peerConnection.AddICECandidate(rc))
	}

	fmt.Println("Signaling all the pending candidates")
	utils.PanicIf(client.ShareCandidate(localCandidateFile, pendingCandidates))

	fmt.Println("Sharing final candidate")
	utils.PanicIf(client.ShareCandidate(localDescriptionFile, finalLocalCandidate))

	remoteCandidates = client.WaitForCandidates(remoteDescriptionFile)
	for _, rc := range remoteCandidates {
		utils.PanicIf(peerConnection.AddICECandidate(rc))
	}
	fmt.Println("Received remote final candidate")
	select {}
}
