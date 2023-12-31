package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	webrtc "github.com/pion/webrtc/v3"

	"github.com/hvaghani221/webrtc/firebase"
	"github.com/hvaghani221/webrtc/utils"
)

const (
	localCandidateFile    = utils.OfferCandidate
	localOfferFile        = utils.OfferOffer
	localDescriptionFile  = utils.OfferDescription
	remoteCandidateFile   = utils.AnswerCandidate
	remoteOfferFile       = utils.AnswerOffer
	remoteDescriptionFile = utils.AnswerDescription
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
			fmt.Println("Added ICE candidate to the pending list")
			pendingCandidates = append(pendingCandidates, c)
		} else {
			fmt.Println("OnICECandidate desc non nil")
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

	dataChannel := utils.ReturnOrPanic(peerConnection.CreateDataChannel("data", nil))
	dataChannel.OnOpen(func() {
		fmt.Println("Data channel connected")
		utils.PanicIf(dataChannel.SendText("Hello from offer"))
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message received from dataChannel %s: %s\n", dataChannel.Label(), string(msg.Data))
	})

	fmt.Println("Created an offer")
	offer := utils.ReturnOrPanic(peerConnection.CreateOffer(nil))

	fmt.Println("SetLocalDescription")
	utils.PanicIf(peerConnection.SetLocalDescription(offer))

	fmt.Println("Sent SDP request")
	utils.PanicIf(client.ShareOffer(localOfferFile, offer))

	fmt.Println("Received sdp request")
	sdp := client.WaitForOffer(remoteOfferFile)
	// fmt.Println("Setting RemoteDescription sdp")
	utils.PanicIf(peerConnection.SetRemoteDescription(sdp))

	time.Sleep(time.Millisecond * 100)
	fmt.Println("Signaling all the pending candidates")
	utils.PanicIf(client.ShareCandidate(localCandidateFile, pendingCandidates))

	remoteCandidates := client.WaitForCandidates(remoteCandidateFile)
	fmt.Println("Received candidates")

	fmt.Println("Adding remote ICE candidates to the peer connection")
	for _, rc := range remoteCandidates {
		utils.PanicIf(peerConnection.AddICECandidate(rc))
	}

	fmt.Println("Sharing final candidate")
	utils.PanicIf(client.ShareCandidate(localDescriptionFile, finalLocalCandidate))

	remoteCandidates = client.WaitForCandidates(remoteDescriptionFile)
	for _, rc := range remoteCandidates {
		utils.PanicIf(peerConnection.AddICECandidate(rc))
	}
	fmt.Println("Received remote final candidate")

	fmt.Println("Infinite Block...")
	select {}
}
