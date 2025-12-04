package bridge

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	bridgepb "github.com/Goden-Gun/transport-lib/gen/go/bridge/v1"
	"github.com/Goden-Gun/transport-lib/pkg/envelope"
)

var (
	// ErrNotStarted indicates client used before Start.
	ErrNotStarted = errors.New("bridge client not started")
)

type client struct {
	opts Options

	deliverCh   chan envelope.TransportEnvelope
	broadcastCh chan envelope.TransportEnvelope

	conn      *grpc.ClientConn
	stream    bridgepb.SidecarBridge_StreamClient
	cancel    context.CancelFunc
	runCancel context.CancelFunc
	started   atomic.Bool
	closed    atomic.Bool

	startOnce sync.Once
	stopOnce  sync.Once

	sendMu sync.Mutex

	inflight chan struct{}
	wg       sync.WaitGroup
	recvErr  chan error
}

// NewClient creates a gRPC bridge client.
func NewClient(opts Options) (Client, error) {
	if opts.Address == "" {
		return nil, errors.New("bridge address is required")
	}
	if opts.NodeID == "" {
		return nil, errors.New("node id is required")
	}
	if opts.Namespace == "" {
		return nil, errors.New("namespace is required")
	}
	if opts.DeliverBuffer <= 0 {
		opts.DeliverBuffer = 128
	}
	if opts.BroadcastBuffer <= 0 {
		opts.BroadcastBuffer = 128
	}
	if len(opts.SupportedVersions) == 0 {
		opts.SupportedVersions = []string{envelope.Version}
	}
	if opts.BridgeVersion == "" {
		opts.BridgeVersion = envelope.Version
	}
	c := &client{
		opts:        opts,
		deliverCh:   make(chan envelope.TransportEnvelope, opts.DeliverBuffer),
		broadcastCh: make(chan envelope.TransportEnvelope, opts.BroadcastBuffer),
	}
	if opts.EnableBackpressure && opts.MaxInFlightDeliver > 0 {
		c.inflight = make(chan struct{}, opts.MaxInFlightDeliver)
	}
	return c, nil
}

// Start dials the remote gRPC bridge and begins stream consumption.
func (c *client) Start(ctx context.Context) error {
	var err error
	c.startOnce.Do(func() {
		runCtx, cancel := context.WithCancel(ctx)
		c.runCancel = cancel
		c.wg.Add(1)
		go c.run(runCtx)
	})
	return err
}

func (c *client) run(ctx context.Context) {
	defer c.wg.Done()
	retry := c.opts.ReconnectBackoff
	if retry <= 0 {
		retry = time.Second
	}
	maxRetry := c.opts.MaxReconnectBackoff
	if maxRetry <= 0 {
		maxRetry = 15 * time.Second
	}
	for {
		if err := c.connect(ctx); err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(retry):
				if retry < maxRetry {
					retry *= 2
					if retry > maxRetry {
						retry = maxRetry
					}
				}
				continue
			}
		}
		retry = c.opts.ReconnectBackoff
		if retry <= 0 {
			retry = time.Second
		}
		select {
		case <-ctx.Done():
			c.cleanup()
			return
		case <-c.recvErr:
			c.cleanup()
			if ctx.Err() != nil {
				return
			}
		}
	}
}

func (c *client) connect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	var dialOpts []grpc.DialOption
	if c.opts.Insecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		tlsConf := &tls.Config{}
		if c.opts.TLSCertFile != "" || c.opts.TLSKeyFile != "" {
			cert, tlsErr := tls.LoadX509KeyPair(c.opts.TLSCertFile, c.opts.TLSKeyFile)
			if tlsErr != nil {
				return fmt.Errorf("load tls cert: %w", tlsErr)
			}
			tlsConf.Certificates = []tls.Certificate{cert}
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConf)))
	}
	dialTimeout := c.opts.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}
	dctx, cancelDial := context.WithTimeout(ctx, dialTimeout)
	conn, dialErr := grpc.DialContext(dctx, c.opts.Address, dialOpts...)
	cancelDial()
	if dialErr != nil {
		return fmt.Errorf("dial bridge: %w", dialErr)
	}
	c.conn = conn
	client := bridgepb.NewSidecarBridgeClient(conn)
	streamCtx := dctx
	if len(c.opts.Metadata) > 0 {
		md := metadata.New(c.opts.Metadata)
		streamCtx = metadata.NewOutgoingContext(dctx, md)
	}
	stream, streamErr := client.Stream(streamCtx)
	if streamErr != nil {
		return fmt.Errorf("create stream: %w", streamErr)
	}
	c.stream = stream
	reg := &bridgepb.RegisterFrame{
		NodeId:            c.opts.NodeID,
		Namespace:         c.opts.Namespace,
		SupportedVersions: c.opts.SupportedVersions,
		BridgeVersion:     c.opts.BridgeVersion,
	}
	req := &bridgepb.StreamRequest{Payload: &bridgepb.StreamRequest_Register{Register: reg}}
	if sendErr := stream.Send(req); sendErr != nil {
		return fmt.Errorf("send register: %w", sendErr)
	}
	c.started.Store(true)
	c.recvErr = make(chan error, 1)
	c.wg.Add(1)
	go c.heartbeatLoop(ctx)
	go c.consume(ctx)
	return nil
}

func (c *client) consume(ctx context.Context) {
	for {
		resp, err := c.stream.Recv()
		if err != nil {
			if c.recvErr != nil {
				c.recvErr <- err
			}
			return
		}
		payload := resp.GetPayload()
		switch payload := payload.(type) {
		case *bridgepb.StreamResponse_Deliver:
			if payload.Deliver != nil && payload.Deliver.Envelope != nil {
				c.deliverCh <- *payload.Deliver.Envelope
			}
		case *bridgepb.StreamResponse_Broadcast:
			if payload.Broadcast != nil && payload.Broadcast.Envelope != nil {
				c.broadcastCh <- *payload.Broadcast.Envelope
			}
		case *bridgepb.StreamResponse_Heartbeat:
			// no-op
		}
	}
}

func (c *client) PublishIngress(ctx context.Context, env envelope.TransportEnvelope) error {
	if !c.started.Load() {
		return ErrNotStarted
	}
	if err := c.acquireSlot(ctx); err != nil {
		return err
	}
	defer c.releaseSlot()
	envelope.NormalizeEnvelope(&env)
	req := &bridgepb.StreamRequest{
		Payload: &bridgepb.StreamRequest_Ingress{
			Ingress: &bridgepb.IngressFrame{Envelope: &env},
		},
	}
	c.sendMu.Lock()
	err := c.stream.Send(req)
	c.sendMu.Unlock()
	if err != nil {
		return err
	}
	return nil
}

func (c *client) SubscribeDeliver(context.Context) (<-chan envelope.TransportEnvelope, error) {
	return c.deliverCh, nil
}

func (c *client) SubscribeBroadcast(context.Context) (<-chan envelope.TransportEnvelope, error) {
	return c.broadcastCh, nil
}

func (c *client) Drain(ctx context.Context) error {
	c.Close()
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (c *client) Close() error {
	var err error
	c.stopOnce.Do(func() {
		if c.runCancel != nil {
			c.runCancel()
		}
		if c.cancel != nil {
			c.cancel()
		}
		if c.stream != nil {
			err = c.stream.CloseSend()
		}
		if c.conn != nil {
			_ = c.conn.Close()
		}
		close(c.deliverCh)
		close(c.broadcastCh)
		c.closed.Store(true)
	})
	return err
}

func (c *client) cleanup() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if c.stream != nil {
		_ = c.stream.CloseSend()
		c.stream = nil
	}
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *client) heartbeatLoop(ctx context.Context) {
	defer c.wg.Done()
	interval := c.opts.HeartbeatInterval
	if interval <= 0 {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.sendHeartbeat(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *client) sendHeartbeat(ctx context.Context) {
	if !c.started.Load() {
		return
	}
	req := &bridgepb.StreamRequest{Payload: &bridgepb.StreamRequest_Heartbeat{Heartbeat: &bridgepb.HeartbeatFrame{Nonce: fmt.Sprintf("%d", time.Now().UnixNano())}}}
	c.sendMu.Lock()
	_ = c.stream.Send(req)
	c.sendMu.Unlock()
}

func (c *client) acquireSlot(ctx context.Context) error {
	if c.inflight == nil {
		return nil
	}
	select {
	case c.inflight <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *client) releaseSlot() {
	if c.inflight == nil {
		return
	}
	select {
	case <-c.inflight:
	default:
	}
}
