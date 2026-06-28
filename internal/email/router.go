package email

import (
	"context"
	"fmt"
)

type Handler interface {
	Send(ctx context.Context, event Event) error
}

type Router struct {
	handlers map[string]Handler
}

func NewRouter(email *Service) *Router {
	return &Router{
		handlers: map[string]Handler{
			"signup_thankyou":  NewTemplateHandler[SignupThankYouData](email, "signup_thankyou", "Thanks for signing up"),
			"order_success":    NewTemplateHandler[OrderSuccessData](email, "order_success", "Your order was placed"),
			"order_failed":     NewTemplateHandler[OrderFailedData](email, "order_failed", "Your order failed"),
			"order_cancelled":  NewTemplateHandler[OrderCancelledData](email, "order_cancelled", "Your order was cancelled"),
			"payment_success":  NewTemplateHandler[PaymentSuccessData](email, "payment_success", "Payment received"),
			"payment_failed":   NewTemplateHandler[PaymentFailedData](email, "payment_failed", "Payment failed"),
			"payment_refunded": NewTemplateHandler[PaymentRefundedData](email, "payment_refunded", "Payment refunded"),
		},
	}
}

func (r *Router) Handle(ctx context.Context, data Event) error {
	handler, ok := r.handlers[data.EventType]
	if !ok {
		return fmt.Errorf("no email handler registered for event type %q", data.EventType)
	}

	return handler.Send(ctx, data)
}
