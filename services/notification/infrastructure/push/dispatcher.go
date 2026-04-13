package push

import (
	"context"
	"fmt"

	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
)

// Dispatcher routes push messages to the correct adapter (FCM or APNs) based
// on the token type. It implements domain/ports.PushPort.
type Dispatcher struct {
	fcm  *FCMAdapter
	apns *APNsAdapter
}

var _ domainports.PushPort = (*Dispatcher)(nil)

func NewDispatcher(fcm *FCMAdapter, apns *APNsAdapter) *Dispatcher {
	return &Dispatcher{fcm: fcm, apns: apns}
}

func (d *Dispatcher) Send(ctx context.Context, tokenType string, msg domainports.PushMessage) error {
	switch tokenType {
	case "fcm":
		if d.fcm == nil {
			return fmt.Errorf("dispatcher: FCM adapter not configured")
		}
		return d.fcm.Send(ctx, msg)
	case "apns":
		if d.apns == nil {
			return fmt.Errorf("dispatcher: APNs adapter not configured")
		}
		return d.apns.Send(ctx, msg)
	default:
		return fmt.Errorf("dispatcher: unknown token type %q", tokenType)
	}
}

// NoopPushPort is a push adapter that discards all messages. Used when push
// credentials are not configured (e.g. during local development).
type NoopPushPort struct{}

var _ domainports.PushPort = (*NoopPushPort)(nil)

func (*NoopPushPort) Send(_ context.Context, _ string, _ domainports.PushMessage) error {
	return nil
}
