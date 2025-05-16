package pubsub

import (
	"context"
	"fmt"

	ps "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

type PubSub struct {
	ps   *ps.PubSub
	host host.Host
}

func NewPubSub(ctx context.Context, h host.Host) (*PubSub, error) {
	pubsub, err := ps.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pubsub: %w", err)
	}
	return &PubSub{
		ps:   pubsub,
		host: h,
	}, nil
}

func (p *PubSub) JoinTopic(topicName string) (*ps.Topic, *ps.Subscription, error) {
	topic, err := p.ps.Join(topicName)
	if err != nil {
		return nil, nil, err
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, nil, err
	}
	return topic, sub, nil
}
