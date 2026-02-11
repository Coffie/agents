// Package webhook provides a production-ready Stripe webhook handler for Go applications.
package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

// EventHandler defines the interface for handling Stripe events.
type EventHandler interface {
	HandlePaymentIntentSucceeded(ctx context.Context, pi *stripe.PaymentIntent) error
	HandlePaymentIntentFailed(ctx context.Context, pi *stripe.PaymentIntent) error
	HandleSubscriptionCreated(ctx context.Context, sub *stripe.Subscription) error
	HandleSubscriptionUpdated(ctx context.Context, sub *stripe.Subscription) error
	HandleSubscriptionDeleted(ctx context.Context, sub *stripe.Subscription) error
	HandleInvoicePaid(ctx context.Context, inv *stripe.Invoice) error
	HandleInvoicePaymentFailed(ctx context.Context, inv *stripe.Invoice) error
	HandleCheckoutSessionCompleted(ctx context.Context, sess *stripe.CheckoutSession) error
	HandleChargeRefunded(ctx context.Context, ch *stripe.Charge) error
}

// Handler handles incoming Stripe webhooks.
type Handler struct {
	endpointSecret string
	eventHandler   EventHandler
	db             *sql.DB
	logger         *slog.Logger
}

// Config holds configuration for the webhook handler.
type Config struct {
	EndpointSecret string
	EventHandler   EventHandler
	DB             *sql.DB // For idempotency tracking
	Logger         *slog.Logger
}

// NewHandler creates a new webhook handler.
func NewHandler(cfg Config) (*Handler, error) {
	if cfg.EndpointSecret == "" {
		cfg.EndpointSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
	}
	if cfg.EndpointSecret == "" {
		return nil, errors.New("webhook endpoint secret is required")
	}

	if cfg.EventHandler == nil {
		return nil, errors.New("event handler is required")
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Handler{
		endpointSecret: cfg.EndpointSecret,
		eventHandler:   cfg.EventHandler,
		db:             cfg.DB,
		logger:         cfg.Logger,
	}, nil
}

// ServeHTTP implements http.Handler for the webhook endpoint.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify signature
	sig := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(body, sig, h.endpointSecret)
	if err != nil {
		h.logger.Error("webhook signature verification failed", "error", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	// Check idempotency
	if h.db != nil {
		processed, err := h.isEventProcessed(r.Context(), event.ID)
		if err != nil {
			h.logger.Error("failed to check event idempotency", "error", err, "event_id", event.ID)
		}
		if processed {
			h.logger.Info("event already processed", "event_id", event.ID)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	// Process event
	if err := h.handleEvent(r.Context(), &event); err != nil {
		h.logger.Error("failed to handle event",
			"error", err,
			"event_type", event.Type,
			"event_id", event.ID,
		)
		// Return 200 to prevent retries for application errors
		// Return 500 only for transient errors that should be retried
		w.WriteHeader(http.StatusOK)
		return
	}

	// Mark event as processed
	if h.db != nil {
		if err := h.markEventProcessed(r.Context(), event.ID, string(event.Type)); err != nil {
			h.logger.Error("failed to mark event as processed", "error", err, "event_id", event.ID)
		}
	}

	h.logger.Info("webhook processed",
		"event_type", event.Type,
		"event_id", event.ID,
	)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"received": true})
}

// handleEvent routes the event to the appropriate handler.
func (h *Handler) handleEvent(ctx context.Context, event *stripe.Event) error {
	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return fmt.Errorf("unmarshal payment_intent: %w", err)
		}
		return h.eventHandler.HandlePaymentIntentSucceeded(ctx, &pi)

	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return fmt.Errorf("unmarshal payment_intent: %w", err)
		}
		return h.eventHandler.HandlePaymentIntentFailed(ctx, &pi)

	case "customer.subscription.created":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return fmt.Errorf("unmarshal subscription: %w", err)
		}
		return h.eventHandler.HandleSubscriptionCreated(ctx, &sub)

	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return fmt.Errorf("unmarshal subscription: %w", err)
		}
		return h.eventHandler.HandleSubscriptionUpdated(ctx, &sub)

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return fmt.Errorf("unmarshal subscription: %w", err)
		}
		return h.eventHandler.HandleSubscriptionDeleted(ctx, &sub)

	case "invoice.paid", "invoice.payment_succeeded":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			return fmt.Errorf("unmarshal invoice: %w", err)
		}
		return h.eventHandler.HandleInvoicePaid(ctx, &inv)

	case "invoice.payment_failed":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			return fmt.Errorf("unmarshal invoice: %w", err)
		}
		return h.eventHandler.HandleInvoicePaymentFailed(ctx, &inv)

	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return fmt.Errorf("unmarshal checkout_session: %w", err)
		}
		return h.eventHandler.HandleCheckoutSessionCompleted(ctx, &sess)

	case "charge.refunded":
		var ch stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &ch); err != nil {
			return fmt.Errorf("unmarshal charge: %w", err)
		}
		return h.eventHandler.HandleChargeRefunded(ctx, &ch)

	default:
		h.logger.Debug("unhandled event type", "type", event.Type)
		return nil
	}
}

// isEventProcessed checks if an event has already been processed.
func (h *Handler) isEventProcessed(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	err := h.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM webhook_events WHERE stripe_event_id = $1)",
		eventID,
	).Scan(&exists)
	return exists, err
}

// markEventProcessed records that an event has been processed.
func (h *Handler) markEventProcessed(ctx context.Context, eventID, eventType string) error {
	_, err := h.db.ExecContext(ctx,
		"INSERT INTO webhook_events (stripe_event_id, event_type, processed_at) VALUES ($1, $2, $3) ON CONFLICT (stripe_event_id) DO NOTHING",
		eventID, eventType, time.Now(),
	)
	return err
}

// DefaultEventHandler provides a base implementation of EventHandler.
// Embed this in your own handler and override methods as needed.
type DefaultEventHandler struct {
	Logger *slog.Logger
}

func (h *DefaultEventHandler) HandlePaymentIntentSucceeded(ctx context.Context, pi *stripe.PaymentIntent) error {
	h.Logger.Info("payment succeeded", "payment_intent_id", pi.ID, "amount", pi.Amount)
	return nil
}

func (h *DefaultEventHandler) HandlePaymentIntentFailed(ctx context.Context, pi *stripe.PaymentIntent) error {
	msg := ""
	if pi.LastPaymentError != nil {
		msg = pi.LastPaymentError.Message
	}
	h.Logger.Warn("payment failed", "payment_intent_id", pi.ID, "error", msg)
	return nil
}

func (h *DefaultEventHandler) HandleSubscriptionCreated(ctx context.Context, sub *stripe.Subscription) error {
	h.Logger.Info("subscription created", "subscription_id", sub.ID, "customer", sub.Customer.ID)
	return nil
}

func (h *DefaultEventHandler) HandleSubscriptionUpdated(ctx context.Context, sub *stripe.Subscription) error {
	h.Logger.Info("subscription updated", "subscription_id", sub.ID, "status", sub.Status)
	return nil
}

func (h *DefaultEventHandler) HandleSubscriptionDeleted(ctx context.Context, sub *stripe.Subscription) error {
	h.Logger.Info("subscription canceled", "subscription_id", sub.ID)
	return nil
}

func (h *DefaultEventHandler) HandleInvoicePaid(ctx context.Context, inv *stripe.Invoice) error {
	h.Logger.Info("invoice paid", "invoice_id", inv.ID, "amount_paid", inv.AmountPaid)
	return nil
}

func (h *DefaultEventHandler) HandleInvoicePaymentFailed(ctx context.Context, inv *stripe.Invoice) error {
	h.Logger.Warn("invoice payment failed", "invoice_id", inv.ID)
	return nil
}

func (h *DefaultEventHandler) HandleCheckoutSessionCompleted(ctx context.Context, sess *stripe.CheckoutSession) error {
	h.Logger.Info("checkout completed", "session_id", sess.ID, "customer", sess.Customer)
	return nil
}

func (h *DefaultEventHandler) HandleChargeRefunded(ctx context.Context, ch *stripe.Charge) error {
	h.Logger.Info("charge refunded", "charge_id", ch.ID, "amount_refunded", ch.AmountRefunded)
	return nil
}

// GinHandler returns a gin.HandlerFunc for use with Gin framework.
// Usage: router.POST("/webhook", webhook.GinHandler(handler))
func GinHandler(h *Handler) func(c interface {
	Request() *http.Request
	Writer() http.ResponseWriter
}) {
	return func(c interface {
		Request() *http.Request
		Writer() http.ResponseWriter
	}) {
		h.ServeHTTP(c.Writer(), c.Request())
	}
}
