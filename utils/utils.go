package utils

import (
	"fmt"
	"os"

	"github.com/pion/webrtc/v3"
)

const (
	OfferCandidate    = "OfferCandidate"
	OfferOffer        = "OfferOffer"
	AnswerCandidate   = "AnswerCandidate"
	AnswerOffer       = "AnswerOffer"
	OfferDescription  = "OfferDescription"
	AnswerDescription = "AnswerDescription"
)

type Requests struct {
	OfferCandidate    []string
	OfferOffer        *webrtc.SessionDescription
	AnswerCandidate   []string
	AnswerOffer       *webrtc.SessionDescription
	OfferDescription  []string
	AnswerDescription []string
}

func GetFromRequest[T any](r Requests, key string) T {
	var temp any
	switch key {

	case OfferCandidate:
		temp = r.OfferCandidate
	case OfferOffer:
		temp = r.OfferOffer
	case AnswerCandidate:
		temp = r.AnswerCandidate
	case AnswerOffer:
		temp = r.AnswerOffer
	case OfferDescription:
		temp = r.OfferDescription
	case AnswerDescription:
		temp = r.AnswerDescription
	}
	t, _ := temp.(T)
	return t
}

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
