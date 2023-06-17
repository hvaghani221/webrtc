package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/pion/webrtc/v3"

	"github.com/hvaghani221/webrtc/utils"
)

var (
	localCandidateFile    = "answer.candidates"
	localOfferFile        = "answer.sdp"
	localDescriptionFile  = "answer.desc"
	remoteCandidateFile   = "offer.candidates"
	remoteOfferFile       = "offer.sdp"
	remoteDescriptionFile = "offer.desc"
)

var rootPath string

func init() {
	flag.StringVar(&rootPath, "rootPath", "./", "Path where the files will be stored")
	flag.Parse()

	localCandidateFile = rootPath + localCandidateFile
	localOfferFile = rootPath + localOfferFile
	remoteCandidateFile = rootPath + remoteCandidateFile
	remoteOfferFile = rootPath + remoteOfferFile
	localDescriptionFile = rootPath + localDescriptionFile
	remoteDescriptionFile = rootPath + remoteDescriptionFile

	os.Remove(localCandidateFile)
	os.Remove(localOfferFile)
	os.Remove(remoteCandidateFile)
	os.Remove(remoteOfferFile)
	os.Remove(localDescriptionFile)
	os.Remove(remoteDescriptionFile)
}

func main() {
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
	sdp := utils.ReturnOrPanic(utils.WaitForOffer(remoteOfferFile))
	utils.PanicIf(peerConnection.SetRemoteDescription(sdp))
	fmt.Println("Received sdp request")

	answer := utils.ReturnOrPanic(peerConnection.CreateAnswer(nil))

	utils.PanicIf(utils.WriteOfferTo(localOfferFile, answer))
	fmt.Println("Sent SDP request")

	fmt.Println("SetLocalDescription")
	utils.PanicIf(peerConnection.SetLocalDescription(answer))

	fmt.Println("Waiting for remote candidates")
	remoteCandidates := utils.ReturnOrPanic(utils.WaitForCandidates(remoteCandidateFile))
	fmt.Println("Adding remote ICECandidates")
	for _, rc := range remoteCandidates {
		utils.PanicIf(peerConnection.AddICECandidate(rc))
	}

	fmt.Println("Signaling all the pending candidates")
	utils.PanicIf(utils.WriteCandidatesTo(localCandidateFile, pendingCandidates))

	fmt.Println("Sharing final candidate")
	utils.PanicIf(utils.WriteCandidatesTo(localDescriptionFile, finalLocalCandidate))

	remoteCandidates = utils.ReturnOrPanic(utils.WaitForCandidates(remoteDescriptionFile))
	for _, rc := range remoteCandidates {
		utils.PanicIf(peerConnection.AddICECandidate(rc))
	}
	fmt.Println("Received remote final candidate")
	select {}
}
