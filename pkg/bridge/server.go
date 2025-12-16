package bridge

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	bridgepb "github.com/Goden-Gun/transport-lib/gen/go/bridge/v1"
	"github.com/Goden-Gun/transport-lib/pkg/envelope"
)

// NewServer builds a gRPC server that wires stream events to Handler.
func NewServer(opts Options) (Server, error) {
	if opts.Address == "" {
		return nil, errors.New("server address is required")
	}
	return &server{opts: opts}, nil
}

type server struct {
	opts       Options
	grpcServer *grpc.Server
	lis        net.Listener
	stopOnce   sync.Once
}

func (s *server) Serve(ctx context.Context, handler Handler) error {
	if handler == nil {
		return errors.New("handler is required")
	}
	lis, err := net.Listen("tcp", s.opts.Address)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.opts.Address, err)
	}
	s.lis = lis
	var serverOpts []grpc.ServerOption
	if !s.opts.Insecure {
		tlsConf := &tls.Config{}
		if s.opts.TLSCertFile != "" && s.opts.TLSKeyFile != "" {
			cert, loadErr := tls.LoadX509KeyPair(s.opts.TLSCertFile, s.opts.TLSKeyFile)
			if loadErr != nil {
				return loadErr
			}
			tlsConf.Certificates = []tls.Certificate{cert}
		}
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsConf)))
	}
	srv := grpc.NewServer(serverOpts...)
	bridgepb.RegisterSidecarBridgeServer(srv, &bridgeService{handler: handler})
	s.grpcServer = srv
	go func() {
		<-ctx.Done()
		s.Close()
	}()
	return srv.Serve(lis)
}

func (s *server) Close() error {
	s.stopOnce.Do(func() {
		if s.grpcServer != nil {
			s.grpcServer.GracefulStop()
		}
		if s.lis != nil {
			_ = s.lis.Close()
		}
	})
	return nil
}

type bridgeService struct {
	bridgepb.UnimplementedSidecarBridgeServer
	handler Handler
}

type session struct {
	meta   RegisterMeta
	stream bridgepb.SidecarBridge_StreamServer
	sendMu sync.Mutex
}

func (s *session) SendDeliver(ctx context.Context, env envelope.TransportEnvelope) error {
	envelope.NormalizeEnvelope(&env)
	resp := &bridgepb.StreamResponse{Payload: &bridgepb.StreamResponse_Deliver{Deliver: &bridgepb.DeliverFrame{Envelope: &env}}}
	return s.send(ctx, resp)
}

func (s *session) SendBroadcast(ctx context.Context, env envelope.TransportEnvelope) error {
	envelope.NormalizeEnvelope(&env)
	resp := &bridgepb.StreamResponse{Payload: &bridgepb.StreamResponse_Broadcast{Broadcast: &bridgepb.BroadcastFrame{Envelope: &env}}}
	return s.send(ctx, resp)
}

func (s *session) SendHeartbeat(ctx context.Context, nonce string) error {
	resp := &bridgepb.StreamResponse{Payload: &bridgepb.StreamResponse_Heartbeat{Heartbeat: &bridgepb.HeartbeatFrame{Nonce: nonce}}}
	return s.send(ctx, resp)
}

func (s *session) send(ctx context.Context, resp *bridgepb.StreamResponse) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.stream.Send(resp)
}

func (s *session) Metadata() RegisterMeta {
	return s.meta
}

func (s *session) Close() error {
	return s.stream.Context().Err()
}

func (svc *bridgeService) Stream(stream bridgepb.SidecarBridge_StreamServer) error {
	ctx := stream.Context()
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	reg := first.GetRegister()
	if reg == nil {
		return errors.New("register frame required")
	}
	meta := RegisterMeta{
		NodeID:    reg.NodeId,
		Namespace: reg.Namespace,
		Version:   reg.BridgeVersion,
	}
	sess := &session{meta: meta, stream: stream}
	if err := svc.handler.OnRegister(ctx, sess, meta); err != nil {
		return err
	}
	defer svc.handler.OnClose(ctx, sess)
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		switch payload := req.GetPayload().(type) {
		case *bridgepb.StreamRequest_Ingress:
			if payload.Ingress != nil && payload.Ingress.Envelope != nil {
				if err := svc.handler.OnIngress(ctx, sess, *payload.Ingress.Envelope); err != nil {
					return err
				}
			}
		case *bridgepb.StreamRequest_Ack:
			if payload.Ack != nil {
				ack := Ack{MessageID: payload.Ack.MessageId, BroadcastID: payload.Ack.BroadcastId}
				if err := svc.handler.OnAck(ctx, sess, ack); err != nil {
					return err
				}
			}
		case *bridgepb.StreamRequest_Heartbeat:
			nonce := ""
			if payload.Heartbeat != nil {
				nonce = payload.Heartbeat.Nonce
			}
			if err := svc.handler.OnHeartbeat(ctx, sess, nonce); err != nil {
				return err
			}
		case *bridgepb.StreamRequest_Register:
			// duplicate register ignored
		}
	}
}
