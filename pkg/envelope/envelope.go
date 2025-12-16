package envelope

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	bridgepb "github.com/Goden-Gun/transport-lib/gen/go/bridge/v1"
)

const Version = "2025-01"

// Message exposes protobuf message for convenience.
type Message = bridgepb.Message

type Payload = bridgepb.Payload

type TextPayload = bridgepb.TextPayload

type AudioPayload = bridgepb.AudioPayload

type ErrorPayload = bridgepb.ErrorPayload

type TransportEnvelope = bridgepb.TransportEnvelope

// NormalizeMessage fills default fields for Message.
func NormalizeMessage(msg *bridgepb.Message) {
	if msg == nil {
		return
	}
	if msg.Version == "" {
		msg.Version = Version
	}
	if msg.RequestId == "" {
		msg.RequestId = uuid.NewString()
	}
	if msg.Metadata == nil {
		msg.Metadata = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if msg.Timestamp == nil || msg.Timestamp.AsTime().IsZero() {
		msg.Timestamp = timestamppb.Now()
	}
}

// ValidateIngress validates client-originated messages.
func ValidateIngress(msg *bridgepb.Message) error {
	if msg == nil {
		return errors.New("message is nil")
	}
	if msg.Kind != "request" && msg.Kind != "" {
		return errors.New("only request kind allowed")
	}
	if strings.TrimSpace(msg.Action) == "" {
		return errors.New("action is required")
	}
	payload := msg.GetPayload()
	if payload == nil || (payload.GetText() == nil && payload.GetAudio() == nil) {
		return errors.New("payload is required")
	}
	return nil
}

// NormalizeEnvelope ensures transport envelope defaults.
func NormalizeEnvelope(env *bridgepb.TransportEnvelope) {
	if env == nil {
		return
	}
	NormalizeMessage(env.Message)
	if env.Attributes == nil {
		env.Attributes = map[string]string{}
	}
	if env.CreatedAt == nil || env.CreatedAt.AsTime().IsZero() {
		env.CreatedAt = timestamppb.Now()
	}
	if env.EnvelopeVersion == "" {
		env.EnvelopeVersion = Version
	}
}

// StampTrace fills trace metadata if missing.
func StampTrace(env *bridgepb.TransportEnvelope, traceID string) {
	if env == nil || traceID == "" {
		return
	}
	env.TraceId = traceID
	if env.Attributes == nil {
		env.Attributes = map[string]string{}
	}
	env.Attributes["trace_id"] = traceID
}

// SetSlot fills slot metadata on the envelope before sending.
func SetSlot(env *bridgepb.TransportEnvelope, slotID, generation uint32) {
	if env == nil {
		return
	}
	env.SlotId = slotID
	env.SlotGeneration = generation
}

// GetSlot returns the slot metadata stored on the envelope.
func GetSlot(env *bridgepb.TransportEnvelope) (slotID, generation uint32) {
	if env == nil {
		return 0, 0
	}
	return env.GetSlotId(), env.GetSlotGeneration()
}
