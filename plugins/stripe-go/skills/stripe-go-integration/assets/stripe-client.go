// Package stripe provides a production-ready Stripe client wrapper for Go applications.
package stripe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/paymentintent"
	"github.com/stripe/stripe-go/v81/paymentmethod"
	"github.com/stripe/stripe-go/v81/refund"
	"github.com/stripe/stripe-go/v81/subscription"
)

// Client wraps Stripe operations with proper error handling and context support.
type Client struct {
	apiKey    string
	timeout   time.Duration
	mu        sync.RWMutex
	customers map[string]*stripe.Customer // Simple in-memory cache
}

// Config holds configuration for the Stripe client.
type Config struct {
	APIKey  string
	Timeout time.Duration
}

// NewClient creates a new Stripe client with the given configuration.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("STRIPE_SECRET_KEY")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("stripe API key is required")
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	stripe.Key = cfg.APIKey

	return &Client{
		apiKey:    cfg.APIKey,
		timeout:   cfg.Timeout,
		customers: make(map[string]*stripe.Customer),
	}, nil
}

// withContext creates a context with timeout.
func (c *Client) withContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.timeout)
}

// CreateCustomer creates a new Stripe customer.
func (c *Client) CreateCustomer(ctx context.Context, email, name string) (*stripe.Customer, error) {
	ctx, cancel := c.withContext(ctx)
	defer cancel()

	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	params.Context = ctx

	cust, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("create customer: %w", handleStripeError(err))
	}

	c.mu.Lock()
	c.customers[cust.ID] = cust
	c.mu.Unlock()

	return cust, nil
}

// GetCustomer retrieves a customer by ID.
func (c *Client) GetCustomer(ctx context.Context, customerID string) (*stripe.Customer, error) {
	// Check cache first
	c.mu.RLock()
	if cust, ok := c.customers[customerID]; ok {
		c.mu.RUnlock()
		return cust, nil
	}
	c.mu.RUnlock()

	ctx, cancel := c.withContext(ctx)
	defer cancel()

	params := &stripe.CustomerParams{}
	params.Context = ctx

	cust, err := customer.Get(customerID, params)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", handleStripeError(err))
	}

	c.mu.Lock()
	c.customers[cust.ID] = cust
	c.mu.Unlock()

	return cust, nil
}

// CheckoutSessionParams holds parameters for creating a checkout session.
type CheckoutSessionParams struct {
	CustomerID  string
	PriceID     string
	Mode        string // "payment", "subscription", "setup"
	SuccessURL  string
	CancelURL   string
	Metadata    map[string]string
	Quantity    int64
}

// CreateCheckoutSession creates a new checkout session.
func (c *Client) CreateCheckoutSession(ctx context.Context, p CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	ctx, cancel := c.withContext(ctx)
	defer cancel()

	if p.Quantity == 0 {
		p.Quantity = 1
	}

	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(p.SuccessURL),
		CancelURL:  stripe.String(p.CancelURL),
		Mode:       stripe.String(p.Mode),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(p.PriceID),
				Quantity: stripe.Int64(p.Quantity),
			},
		},
	}

	if p.CustomerID != "" {
		params.Customer = stripe.String(p.CustomerID)
	}

	if len(p.Metadata) > 0 {
		params.Metadata = p.Metadata
	}

	params.Context = ctx

	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", handleStripeError(err))
	}

	return sess, nil
}

// PaymentIntentParams holds parameters for creating a payment intent.
type PaymentIntentParams struct {
	Amount         int64
	Currency       string
	CustomerID     string
	Description    string
	Metadata       map[string]string
	IdempotencyKey string
}

// CreatePaymentIntent creates a new payment intent.
func (c *Client) CreatePaymentIntent(ctx context.Context, p PaymentIntentParams) (*stripe.PaymentIntent, error) {
	ctx, cancel := c.withContext(ctx)
	defer cancel()

	if p.Currency == "" {
		p.Currency = "usd"
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(p.Amount),
		Currency: stripe.String(p.Currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	if p.CustomerID != "" {
		params.Customer = stripe.String(p.CustomerID)
	}

	if p.Description != "" {
		params.Description = stripe.String(p.Description)
	}

	if len(p.Metadata) > 0 {
		params.Metadata = p.Metadata
	}

	if p.IdempotencyKey != "" {
		params.IdempotencyKey = stripe.String(p.IdempotencyKey)
	}

	params.Context = ctx

	intent, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("create payment intent: %w", handleStripeError(err))
	}

	return intent, nil
}

// ConfirmPaymentIntent confirms a payment intent with a payment method.
func (c *Client) ConfirmPaymentIntent(ctx context.Context, intentID, paymentMethodID string) (*stripe.PaymentIntent, error) {
	ctx, cancel := c.withContext(ctx)
	defer cancel()

	params := &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String(paymentMethodID),
	}
	params.Context = ctx

	intent, err := paymentintent.Confirm(intentID, params)
	if err != nil {
		return nil, fmt.Errorf("confirm payment intent: %w", handleStripeError(err))
	}

	return intent, nil
}

// SubscriptionParams holds parameters for creating a subscription.
type SubscriptionParams struct {
	CustomerID     string
	PriceID        string
	Metadata       map[string]string
	IdempotencyKey string
}

// CreateSubscription creates a new subscription.
func (c *Client) CreateSubscription(ctx context.Context, p SubscriptionParams) (*stripe.Subscription, error) {
	ctx, cancel := c.withContext(ctx)
	defer cancel()

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(p.CustomerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(p.PriceID),
			},
		},
		PaymentBehavior: stripe.String("default_incomplete"),
		PaymentSettings: &stripe.SubscriptionPaymentSettingsParams{
			SaveDefaultPaymentMethod: stripe.String("on_subscription"),
		},
	}
	params.AddExpand("latest_invoice.payment_intent")

	if len(p.Metadata) > 0 {
		params.Metadata = p.Metadata
	}

	if p.IdempotencyKey != "" {
		params.IdempotencyKey = stripe.String(p.IdempotencyKey)
	}

	params.Context = ctx

	sub, err := subscription.New(params)
	if err != nil {
		return nil, fmt.Errorf("create subscription: %w", handleStripeError(err))
	}

	return sub, nil
}

// CancelSubscription cancels a subscription.
func (c *Client) CancelSubscription(ctx context.Context, subscriptionID string) error {
	ctx, cancel := c.withContext(ctx)
	defer cancel()

	params := &stripe.SubscriptionCancelParams{}
	params.Context = ctx

	_, err := subscription.Cancel(subscriptionID, params)
	if err != nil {
		return fmt.Errorf("cancel subscription: %w", handleStripeError(err))
	}

	return nil
}

// AttachPaymentMethod attaches a payment method to a customer.
func (c *Client) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	ctx, cancel := c.withContext(ctx)
	defer cancel()

	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	}
	params.Context = ctx

	_, err := paymentmethod.Attach(paymentMethodID, params)
	if err != nil {
		return fmt.Errorf("attach payment method: %w", handleStripeError(err))
	}

	return nil
}

// RefundParams holds parameters for creating a refund.
type RefundParams struct {
	PaymentIntentID string
	Amount          *int64  // nil for full refund
	Reason          *string // "duplicate", "fraudulent", "requested_by_customer"
	IdempotencyKey  string
}

// CreateRefund creates a refund for a payment.
func (c *Client) CreateRefund(ctx context.Context, p RefundParams) (*stripe.Refund, error) {
	ctx, cancel := c.withContext(ctx)
	defer cancel()

	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(p.PaymentIntentID),
	}

	if p.Amount != nil {
		params.Amount = p.Amount
	}

	if p.Reason != nil {
		params.Reason = p.Reason
	}

	if p.IdempotencyKey != "" {
		params.IdempotencyKey = stripe.String(p.IdempotencyKey)
	}

	params.Context = ctx

	ref, err := refund.New(params)
	if err != nil {
		return nil, fmt.Errorf("create refund: %w", handleStripeError(err))
	}

	return ref, nil
}

// Stripe error types for application-level handling
var (
	ErrCardDeclined      = errors.New("card declined")
	ErrExpiredCard       = errors.New("card expired")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidCard       = errors.New("invalid card")
	ErrProcessingError   = errors.New("processing error")
	ErrRateLimitError    = errors.New("rate limit exceeded")
	ErrAuthenticationError = errors.New("authentication failed")
)

// handleStripeError converts Stripe errors to application errors.
func handleStripeError(err error) error {
	if err == nil {
		return nil
	}

	var stripeErr *stripe.Error
	if errors.As(err, &stripeErr) {
		switch stripeErr.Type {
		case stripe.ErrorTypeCard:
			switch stripeErr.Code {
			case stripe.ErrorCodeCardDeclined:
				return fmt.Errorf("%w: %s", ErrCardDeclined, stripeErr.Message)
			case stripe.ErrorCodeExpiredCard:
				return fmt.Errorf("%w: %s", ErrExpiredCard, stripeErr.Message)
			case stripe.ErrorCodeInsufficientFunds:
				return fmt.Errorf("%w: %s", ErrInsufficientFunds, stripeErr.Message)
			case stripe.ErrorCodeIncorrectNumber, stripe.ErrorCodeIncorrectCVC:
				return fmt.Errorf("%w: %s", ErrInvalidCard, stripeErr.Message)
			case stripe.ErrorCodeProcessingError:
				return fmt.Errorf("%w: %s", ErrProcessingError, stripeErr.Message)
			}
		case stripe.ErrorTypeRateLimit:
			return fmt.Errorf("%w: %s", ErrRateLimitError, stripeErr.Message)
		case stripe.ErrorTypeAuthentication:
			return fmt.Errorf("%w: %s", ErrAuthenticationError, stripeErr.Message)
		}
		return fmt.Errorf("stripe error (%s): %s", stripeErr.Code, stripeErr.Message)
	}

	return err
}
