package bridge

import (
	"context"
	"time"

	"github.com/Goden-Gun/transport-lib/pkg/envelope"
)

// Client represents a bidirectional transport session.
type Client interface {
	Start(ctx context.Context) error
	PublishIngress(ctx context.Context, env envelope.TransportEnvelope) error
	SubscribeDeliver(ctx context.Context) (<-chan envelope.TransportEnvelope, error)
	SubscribeBroadcast(ctx context.Context) (<-chan envelope.TransportEnvelope, error)
	Drain(ctx context.Context) error
	Close() error
}

// Server exposes callbacks for chat workers implementing the bridge.
type Server interface {
	Serve(ctx context.Context, handler Handler) error
	Close() error
}

// Handler handles inbound frames from sidecar nodes.
type Handler interface {
	OnRegister(ctx context.Context, session Session, meta RegisterMeta) error
	OnIngress(ctx context.Context, session Session, env envelope.TransportEnvelope) error
	OnAck(ctx context.Context, session Session, ack Ack) error
	OnHeartbeat(ctx context.Context, session Session, nonce string) error
	OnClose(ctx context.Context, session Session) error
}

// Session represents an active stream between sidecar and worker.
type Session interface {
	SendDeliver(ctx context.Context, env envelope.TransportEnvelope) error
	SendBroadcast(ctx context.Context, env envelope.TransportEnvelope) error
	SendHeartbeat(ctx context.Context, nonce string) error
	Metadata() RegisterMeta
	Close() error
}

// RegisterMeta carries node metadata for stream bootstrap.
type RegisterMeta struct {
	NodeID    string
	Namespace string
	Version   string
}

// Ack models acknowledgement semantics.
type Ack struct {
	MessageID string
	Status    string
	Reason    string
}

// Options define bridge runtime parameters.
type Options struct {
	Address                 string
	Namespace               string
	NodeID                  string
	DeliverBuffer           int
	BroadcastBuffer         int
	DialTimeout             time.Duration
	HeartbeatInterval       time.Duration
	ReconnectBackoff        time.Duration
	MaxReconnectBackoff     time.Duration
	TLSCertFile             string
	TLSKeyFile              string
	Insecure                bool
	Metadata                map[string]string
	SupportedVersions       []string
	BridgeVersion           string
	EnableBackpressure      bool
	MaxInFlightDeliver      int
	GracefulShutdownTimeout time.Duration
}
