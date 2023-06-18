package firebase

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/pion/webrtc/v3"
	"google.golang.org/api/option"

	"github.com/hvaghani221/webrtc/utils"
)

type client struct {
	cloudstore *firestore.Client
	doc        *firestore.DocumentRef
	requests   utils.Requests
	waitQueue  map[string]*safeChannel
	ctx        context.Context
	cancelFunc context.CancelFunc
	mutex      sync.RWMutex
}

type safeChannel struct {
	channel chan struct{}
	once    sync.Once
}

func newSafeChannel() *safeChannel {
	channel := make(chan struct{})
	return &safeChannel{
		channel: channel,
		once:    sync.Once{},
	}
}

func (channel *safeChannel) close() {
	channel.once.Do(func() {
		close(channel.channel)
	})
}

func Init() (*client, error) {
	// Use a service account
	ctx, cancelFunc := context.WithCancel(context.Background())
	sa := option.WithCredentialsFile("firebase.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		cancelFunc()
		return nil, fmt.Errorf("firebase init: %e", err)
	}

	cli, err := app.Firestore(ctx)
	if err != nil {
		cancelFunc()
		return nil, fmt.Errorf("firebase init: %e", err)
	}

	collection := cli.Collection("connections")
	doc := collection.Doc("test")
	c := &client{
		cloudstore: cli,
		doc:        doc,
		waitQueue: map[string]*safeChannel{
			"OfferCandidate":    newSafeChannel(),
			"OfferOffer":        newSafeChannel(),
			"AnswerCandidate":   newSafeChannel(),
			"AnswerOffer":       newSafeChannel(),
			"OfferDescription":  newSafeChannel(),
			"AnswerDescription": newSafeChannel(),
		},
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
	go c.watch()
	return c, nil
}

func (c *client) watch() {
	snapshot := c.doc.Snapshots(c.ctx)
	go func() {
		<-c.ctx.Done()
		snapshot.Stop()
	}()
	for {
		value, err := snapshot.Next()
		if err != nil {
			fmt.Println("Stopped watching because of the error: ", err)
			break
		}
		var req utils.Requests
		if err := value.DataTo(&req); err != nil {
			panic(err)
		}
		c.requests = req
		c.signalClose(req)
	}
}

func (c *client) signalClose(req utils.Requests) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if req.OfferCandidate != nil {
		c.waitQueue[utils.OfferCandidate].close()
	}
	if req.AnswerCandidate != nil {
		c.waitQueue[utils.AnswerCandidate].close()
	}
	if req.OfferOffer != nil {
		c.waitQueue[utils.OfferOffer].close()
	}
	if req.AnswerOffer != nil {
		c.waitQueue[utils.AnswerOffer].close()
	}
	if req.OfferDescription != nil {
		c.waitQueue[utils.OfferDescription].close()
	}
	if req.AnswerDescription != nil {
		c.waitQueue[utils.AnswerDescription].close()
	}
}

func (c *client) Close() {
	c.cancelFunc()
	c.cloudstore.Close()
}

func (c *client) ShareOffer(key string, offer webrtc.SessionDescription) error {
	_, err := c.doc.Update(c.ctx, []firestore.Update{
		{
			Path:  key,
			Value: offer,
		},
	})
	return err
}

func (c *client) WaitForOffer(key string) webrtc.SessionDescription {
	c.mutex.RLock()
	waitChan := c.waitQueue[key]
	c.mutex.RUnlock()

	for range waitChan.channel {
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return *utils.GetFromRequest[*webrtc.SessionDescription](c.requests, key)
}

func (c *client) ShareCandidate(key string, candidates []*webrtc.ICECandidate) error {
	updates := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		updates = append(updates, candidate.ToJSON().Candidate)
	}
	_, err := c.doc.Update(c.ctx, []firestore.Update{
		{
			Path:  key,
			Value: updates,
		},
	})
	return err
}

func (c *client) WaitForCandidates(key string) []webrtc.ICECandidateInit {
	c.mutex.RLock()
	waitChan := c.waitQueue[key]
	c.mutex.RUnlock()

	for range waitChan.channel {
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	res := utils.GetFromRequest[[]string](c.requests, key)
	candidates := make([]webrtc.ICECandidateInit, 0, len(res))
	for _, c := range res {
		candidates = append(candidates, webrtc.ICECandidateInit{Candidate: c})
	}
	return candidates
}
