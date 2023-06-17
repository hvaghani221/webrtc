package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pion/webrtc/v3"
)

func PanicIf(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func ReturnOrPanic[T any](t T, err error) T {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		panic(err)
	}
	return t
}

func PrintList[T any](msg string, list []T) {
	fmt.Println(msg)
	for _, l := range list {
		fmt.Println(l)
	}
}

func WriteCandidatesTo(filename string, iceCandidates []*webrtc.ICECandidate) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	for _, c := range iceCandidates {
		fmt.Fprintln(file, c.ToJSON().Candidate)
	}
	return nil
}

func WaitForCandidates(filename string) ([]webrtc.ICECandidateInit, error) {
	var file *os.File
	var err error

	fmt.Printf("Waiting for %s ...\n", filename)
	for {
		if file, err = os.Open(filename); err == nil {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	fmt.Printf("%s detected\n", filename)

	candidates := make([]webrtc.ICECandidateInit, 0)

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}

		c := strings.TrimSpace(line)
		if c == "" {
			continue
		}
		candidates = append(candidates, webrtc.ICECandidateInit{Candidate: c})
	}

	return candidates, nil
}

func WriteOfferTo(filename string, offer webrtc.SessionDescription) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	return json.NewEncoder(file).Encode(offer)
}

func WaitForOffer(filename string) (webrtc.SessionDescription, error) {
	var file *os.File
	var err error

	fmt.Printf("Waiting for %s ...\n", filename)
	for {
		if file, err = os.Open(filename); err == nil {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	fmt.Printf("%s detecte\n", filename)
	var offer webrtc.SessionDescription
	if err = json.NewDecoder(file).Decode(&offer); err != nil {
		return offer, err
	}
	return offer, nil
}
