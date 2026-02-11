---
name: stripe-go-integration
description: Implement Stripe payment processing in Go for robust, PCI-compliant payment flows including checkout, subscriptions, and webhooks. Use when integrating Stripe payments in Go applications, building subscription systems, or implementing secure checkout flows.
---

# Stripe Go Integration

Master Stripe payment processing with Go using the official stripe-go SDK for robust, PCI-compliant payment flows including checkout, subscriptions, webhooks, and refunds.

## When to Use This Skill

- Implementing payment processing in Go web applications
- Setting up subscription billing systems in Go
- Handling one-time payments and recurring charges
- Processing refunds and disputes
- Managing customer payment methods
- Implementing SCA (Strong Customer Authentication) for European payments
- Building marketplace payment flows with Stripe Connect

## Core Concepts

### 1. SDK Setup

```go
import (
    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/checkout/session"
    "github.com/stripe/stripe-go/v81/customer"
    "github.com/stripe/stripe-go/v81/paymentintent"
    "github.com/stripe/stripe-go/v81/subscription"
    "github.com/stripe/stripe-go/v81/webhook"
)

func init() {
    stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
}
```

### 2. Payment Flows

**Checkout Session (Hosted)**
- Stripe-hosted payment page
- Minimal PCI compliance burden
- Fastest implementation
- Supports one-time and recurring payments

**Payment Intents (Custom UI)**
- Full control over payment UI
- Requires Stripe.js for PCI compliance
- More complex implementation
- Better customization options

**Setup Intents (Save Payment Methods)**
- Collect payment method without charging
- Used for subscriptions and future payments
- Requires customer confirmation

### 3. Webhooks

**Critical Events:**
- `payment_intent.succeeded`: Payment completed
- `payment_intent.payment_failed`: Payment failed
- `customer.subscription.updated`: Subscription changed
- `customer.subscription.deleted`: Subscription canceled
- `charge.refunded`: Refund processed
- `invoice.payment_succeeded`: Subscription payment successful

## Quick Start

```go
package main

import (
    "fmt"
    "os"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/checkout/session"
)

func main() {
    stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

    params := &stripe.CheckoutSessionParams{
        PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
        LineItems: []*stripe.CheckoutSessionLineItemParams{
            {
                PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
                    Currency: stripe.String("usd"),
                    ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
                        Name: stripe.String("Premium Subscription"),
                    },
                    UnitAmount: stripe.Int64(2000), // $20.00
                    Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
                        Interval: stripe.String("month"),
                    },
                },
                Quantity: stripe.Int64(1),
            },
        },
        Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
        SuccessURL: stripe.String("https://yourdomain.com/success?session_id={CHECKOUT_SESSION_ID}"),
        CancelURL:  stripe.String("https://yourdomain.com/cancel"),
    }

    s, err := session.New(params)
    if err != nil {
        panic(err)
    }

    fmt.Println(s.URL)
}
```

## Payment Implementation Patterns

### Pattern 1: One-Time Payment (Hosted Checkout)

```go
package payments

import (
    "context"
    "fmt"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/checkout/session"
)

type CheckoutService struct {
    successURL string
    cancelURL  string
}

func NewCheckoutService(successURL, cancelURL string) *CheckoutService {
    return &CheckoutService{
        successURL: successURL,
        cancelURL:  cancelURL,
    }
}

func (s *CheckoutService) CreateCheckoutSession(ctx context.Context, amount int64, currency, orderID, userID string) (*stripe.CheckoutSession, error) {
    params := &stripe.CheckoutSessionParams{
        PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
        LineItems: []*stripe.CheckoutSessionLineItemParams{
            {
                PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
                    Currency: stripe.String(currency),
                    ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
                        Name:   stripe.String("Purchase"),
                        Images: stripe.StringSlice([]string{"https://example.com/product.jpg"}),
                    },
                    UnitAmount: stripe.Int64(amount),
                },
                Quantity: stripe.Int64(1),
            },
        },
        Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
        SuccessURL: stripe.String(s.successURL + "?session_id={CHECKOUT_SESSION_ID}"),
        CancelURL:  stripe.String(s.cancelURL),
        Metadata: map[string]string{
            "order_id": orderID,
            "user_id":  userID,
        },
    }
    params.Context = ctx

    sess, err := session.New(params)
    if err != nil {
        return nil, fmt.Errorf("failed to create checkout session: %w", err)
    }

    return sess, nil
}
```

### Pattern 2: Custom Payment Intent Flow

```go
package payments

import (
    "context"
    "fmt"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/paymentintent"
)

type PaymentIntentService struct{}

func (s *PaymentIntentService) CreatePaymentIntent(ctx context.Context, amount int64, currency string, customerID *string) (string, error) {
    params := &stripe.PaymentIntentParams{
        Amount:   stripe.Int64(amount),
        Currency: stripe.String(currency),
        AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
            Enabled: stripe.Bool(true),
        },
        Metadata: map[string]string{
            "integration_check": "accept_a_payment",
        },
    }

    if customerID != nil {
        params.Customer = customerID
    }

    params.Context = ctx

    intent, err := paymentintent.New(params)
    if err != nil {
        return "", fmt.Errorf("failed to create payment intent: %w", err)
    }

    return intent.ClientSecret, nil
}

func (s *PaymentIntentService) ConfirmPaymentIntent(ctx context.Context, paymentIntentID, paymentMethodID string) (*stripe.PaymentIntent, error) {
    params := &stripe.PaymentIntentConfirmParams{
        PaymentMethod: stripe.String(paymentMethodID),
    }
    params.Context = ctx

    intent, err := paymentintent.Confirm(paymentIntentID, params)
    if err != nil {
        return nil, fmt.Errorf("failed to confirm payment intent: %w", err)
    }

    return intent, nil
}
```

### Pattern 3: Subscription Creation

```go
package payments

import (
    "context"
    "fmt"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/subscription"
)

type SubscriptionService struct{}

type SubscriptionResult struct {
    SubscriptionID string
    ClientSecret   string
    Status         stripe.SubscriptionStatus
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, customerID, priceID string) (*SubscriptionResult, error) {
    params := &stripe.SubscriptionParams{
        Customer: stripe.String(customerID),
        Items: []*stripe.SubscriptionItemsParams{
            {
                Price: stripe.String(priceID),
            },
        },
        PaymentBehavior: stripe.String("default_incomplete"),
        PaymentSettings: &stripe.SubscriptionPaymentSettingsParams{
            SaveDefaultPaymentMethod: stripe.String("on_subscription"),
        },
    }
    params.AddExpand("latest_invoice.payment_intent")
    params.Context = ctx

    sub, err := subscription.New(params)
    if err != nil {
        return nil, fmt.Errorf("failed to create subscription: %w", err)
    }

    result := &SubscriptionResult{
        SubscriptionID: sub.ID,
        Status:         sub.Status,
    }

    if sub.LatestInvoice != nil && sub.LatestInvoice.PaymentIntent != nil {
        result.ClientSecret = sub.LatestInvoice.PaymentIntent.ClientSecret
    }

    return result, nil
}

func (s *SubscriptionService) CancelSubscription(ctx context.Context, subscriptionID string) error {
    params := &stripe.SubscriptionCancelParams{}
    params.Context = ctx

    _, err := subscription.Cancel(subscriptionID, params)
    if err != nil {
        return fmt.Errorf("failed to cancel subscription: %w", err)
    }

    return nil
}
```

### Pattern 4: Customer Portal

```go
package payments

import (
    "context"
    "fmt"

    "github.com/stripe/stripe-go/v81"
    portalsession "github.com/stripe/stripe-go/v81/billingportal/session"
)

type PortalService struct {
    returnURL string
}

func NewPortalService(returnURL string) *PortalService {
    return &PortalService{returnURL: returnURL}
}

func (s *PortalService) CreatePortalSession(ctx context.Context, customerID string) (string, error) {
    params := &stripe.BillingPortalSessionParams{
        Customer:  stripe.String(customerID),
        ReturnURL: stripe.String(s.returnURL),
    }
    params.Context = ctx

    sess, err := portalsession.New(params)
    if err != nil {
        return "", fmt.Errorf("failed to create portal session: %w", err)
    }

    return sess.URL, nil
}
```

## Webhook Handling

### Gin Framework

```go
package handlers

import (
    "io"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/webhook"
)

type WebhookHandler struct {
    endpointSecret string
    paymentService PaymentService
}

func NewWebhookHandler(paymentService PaymentService) *WebhookHandler {
    return &WebhookHandler{
        endpointSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
        paymentService: paymentService,
    }
}

func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
        return
    }

    sig := c.GetHeader("Stripe-Signature")

    event, err := webhook.ConstructEvent(body, sig, h.endpointSecret)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
        return
    }

    switch event.Type {
    case "payment_intent.succeeded":
        var paymentIntent stripe.PaymentIntent
        if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse payment intent"})
            return
        }
        h.handlePaymentSuccess(c.Request.Context(), &paymentIntent)

    case "payment_intent.payment_failed":
        var paymentIntent stripe.PaymentIntent
        if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse payment intent"})
            return
        }
        h.handlePaymentFailure(c.Request.Context(), &paymentIntent)

    case "customer.subscription.deleted":
        var sub stripe.Subscription
        if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse subscription"})
            return
        }
        h.handleSubscriptionCanceled(c.Request.Context(), &sub)

    case "invoice.payment_succeeded":
        var invoice stripe.Invoice
        if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse invoice"})
            return
        }
        h.handleInvoicePaid(c.Request.Context(), &invoice)
    }

    c.JSON(http.StatusOK, gin.H{"received": true})
}

func (h *WebhookHandler) handlePaymentSuccess(ctx context.Context, pi *stripe.PaymentIntent) {
    // Update order status
    // Send confirmation email
    // Fulfill order
    log.Printf("Payment succeeded: %s", pi.ID)
}

func (h *WebhookHandler) handlePaymentFailure(ctx context.Context, pi *stripe.PaymentIntent) {
    // Notify customer
    // Update order status
    if pi.LastPaymentError != nil {
        log.Printf("Payment failed: %s - %s", pi.ID, pi.LastPaymentError.Message)
    }
}

func (h *WebhookHandler) handleSubscriptionCanceled(ctx context.Context, sub *stripe.Subscription) {
    // Update user access
    // Send cancellation email
    log.Printf("Subscription canceled: %s", sub.ID)
}

func (h *WebhookHandler) handleInvoicePaid(ctx context.Context, inv *stripe.Invoice) {
    // Record payment
    // Extend subscription access
    log.Printf("Invoice paid: %s", inv.ID)
}
```

### Chi Framework

```go
package handlers

import (
    "encoding/json"
    "io"
    "net/http"
    "os"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/webhook"
)

func StripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "failed to read body", http.StatusBadRequest)
        return
    }

    sig := r.Header.Get("Stripe-Signature")
    endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

    event, err := webhook.ConstructEvent(body, sig, endpointSecret)
    if err != nil {
        http.Error(w, "invalid signature", http.StatusBadRequest)
        return
    }

    switch event.Type {
    case "payment_intent.succeeded":
        var pi stripe.PaymentIntent
        json.Unmarshal(event.Data.Raw, &pi)
        // Handle success
    case "payment_intent.payment_failed":
        var pi stripe.PaymentIntent
        json.Unmarshal(event.Data.Raw, &pi)
        // Handle failure
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]bool{"received": true})
}
```

### net/http (Standard Library)

```go
package main

import (
    "encoding/json"
    "io"
    "log"
    "net/http"
    "os"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/webhook"
)

func main() {
    http.HandleFunc("/webhook", handleWebhook)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "failed to read body", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    sig := r.Header.Get("Stripe-Signature")
    endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

    event, err := webhook.ConstructEvent(body, sig, endpointSecret)
    if err != nil {
        log.Printf("Webhook signature verification failed: %v", err)
        http.Error(w, "invalid signature", http.StatusBadRequest)
        return
    }

    // Handle event types
    switch event.Type {
    case "payment_intent.succeeded":
        var pi stripe.PaymentIntent
        if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
            log.Printf("Failed to unmarshal payment intent: %v", err)
            http.Error(w, "parse error", http.StatusBadRequest)
            return
        }
        log.Printf("Payment succeeded: %s, amount: %d", pi.ID, pi.Amount)

    case "checkout.session.completed":
        var session stripe.CheckoutSession
        if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
            log.Printf("Failed to unmarshal checkout session: %v", err)
            http.Error(w, "parse error", http.StatusBadRequest)
            return
        }
        log.Printf("Checkout completed: %s", session.ID)
    }

    w.WriteHeader(http.StatusOK)
}
```

## Customer Management

```go
package payments

import (
    "context"
    "fmt"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/customer"
    "github.com/stripe/stripe-go/v81/paymentmethod"
)

type CustomerService struct{}

func (s *CustomerService) CreateCustomer(ctx context.Context, email, name string, paymentMethodID *string) (*stripe.Customer, error) {
    params := &stripe.CustomerParams{
        Email: stripe.String(email),
        Name:  stripe.String(name),
        Metadata: map[string]string{
            "source": "go_app",
        },
    }

    if paymentMethodID != nil {
        params.PaymentMethod = paymentMethodID
        params.InvoiceSettings = &stripe.CustomerInvoiceSettingsParams{
            DefaultPaymentMethod: paymentMethodID,
        }
    }

    params.Context = ctx

    cust, err := customer.New(params)
    if err != nil {
        return nil, fmt.Errorf("failed to create customer: %w", err)
    }

    return cust, nil
}

func (s *CustomerService) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
    // Attach payment method to customer
    attachParams := &stripe.PaymentMethodAttachParams{
        Customer: stripe.String(customerID),
    }
    attachParams.Context = ctx

    _, err := paymentmethod.Attach(paymentMethodID, attachParams)
    if err != nil {
        return fmt.Errorf("failed to attach payment method: %w", err)
    }

    // Set as default
    updateParams := &stripe.CustomerParams{
        InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
            DefaultPaymentMethod: stripe.String(paymentMethodID),
        },
    }
    updateParams.Context = ctx

    _, err = customer.Update(customerID, updateParams)
    if err != nil {
        return fmt.Errorf("failed to set default payment method: %w", err)
    }

    return nil
}

func (s *CustomerService) ListPaymentMethods(ctx context.Context, customerID string) ([]*stripe.PaymentMethod, error) {
    params := &stripe.PaymentMethodListParams{
        Customer: stripe.String(customerID),
        Type:     stripe.String("card"),
    }
    params.Context = ctx

    iter := paymentmethod.List(params)

    var methods []*stripe.PaymentMethod
    for iter.Next() {
        methods = append(methods, iter.PaymentMethod())
    }

    if err := iter.Err(); err != nil {
        return nil, fmt.Errorf("failed to list payment methods: %w", err)
    }

    return methods, nil
}
```

## Refund Handling

```go
package payments

import (
    "context"
    "fmt"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/refund"
)

type RefundService struct{}

func (s *RefundService) CreateRefund(ctx context.Context, paymentIntentID string, amount *int64, reason *string) (*stripe.Refund, error) {
    params := &stripe.RefundParams{
        PaymentIntent: stripe.String(paymentIntentID),
    }

    if amount != nil {
        params.Amount = amount // Partial refund
    }

    if reason != nil {
        params.Reason = reason // "duplicate", "fraudulent", "requested_by_customer"
    }

    params.Context = ctx

    ref, err := refund.New(params)
    if err != nil {
        return nil, fmt.Errorf("failed to create refund: %w", err)
    }

    return ref, nil
}
```

## Error Handling

```go
package payments

import (
    "errors"
    "fmt"

    "github.com/stripe/stripe-go/v81"
)

var (
    ErrCardDeclined      = errors.New("card was declined")
    ErrExpiredCard       = errors.New("card has expired")
    ErrInsufficientFunds = errors.New("insufficient funds")
    ErrProcessingError   = errors.New("processing error")
)

func HandleStripeError(err error) error {
    if err == nil {
        return nil
    }

    var stripeErr *stripe.Error
    if errors.As(err, &stripeErr) {
        switch stripeErr.Code {
        case stripe.ErrorCodeCardDeclined:
            return fmt.Errorf("%w: %s", ErrCardDeclined, stripeErr.Message)
        case stripe.ErrorCodeExpiredCard:
            return fmt.Errorf("%w: %s", ErrExpiredCard, stripeErr.Message)
        case stripe.ErrorCodeInsufficientFunds:
            return fmt.Errorf("%w: %s", ErrInsufficientFunds, stripeErr.Message)
        case stripe.ErrorCodeProcessingError:
            return fmt.Errorf("%w: %s", ErrProcessingError, stripeErr.Message)
        default:
            return fmt.Errorf("stripe error (%s): %s", stripeErr.Code, stripeErr.Message)
        }
    }

    return err
}

// Usage example
func ProcessPayment(ctx context.Context, amount int64) error {
    _, err := CreatePaymentIntent(ctx, amount, "usd", nil)
    if err != nil {
        return HandleStripeError(err)
    }
    return nil
}
```

## Testing

```go
package payments_test

import (
    "context"
    "os"
    "testing"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/customer"
    "github.com/stripe/stripe-go/v81/paymentintent"
)

func TestMain(m *testing.M) {
    // Use test mode key
    stripe.Key = os.Getenv("STRIPE_TEST_SECRET_KEY")
    os.Exit(m.Run())
}

// Test card numbers
const (
    TestCardSuccess           = "4242424242424242"
    TestCardDeclined          = "4000000000000002"
    TestCard3DSecure          = "4000002500003155"
    TestCardInsufficientFunds = "4000000000009995"
)

func TestPaymentFlow(t *testing.T) {
    ctx := context.Background()

    // Create test customer
    custParams := &stripe.CustomerParams{
        Email: stripe.String("test@example.com"),
    }
    cust, err := customer.New(custParams)
    if err != nil {
        t.Fatalf("failed to create customer: %v", err)
    }
    defer customer.Del(cust.ID, nil)

    // Create payment intent
    intentParams := &stripe.PaymentIntentParams{
        Amount:   stripe.Int64(1000),
        Currency: stripe.String("usd"),
        Customer: stripe.String(cust.ID),
        PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
    }
    intentParams.Context = ctx

    intent, err := paymentintent.New(intentParams)
    if err != nil {
        t.Fatalf("failed to create payment intent: %v", err)
    }

    // Confirm with test payment method
    confirmParams := &stripe.PaymentIntentConfirmParams{
        PaymentMethod: stripe.String("pm_card_visa"),
    }
    confirmed, err := paymentintent.Confirm(intent.ID, confirmParams)
    if err != nil {
        t.Fatalf("failed to confirm payment: %v", err)
    }

    if confirmed.Status != stripe.PaymentIntentStatusSucceeded {
        t.Errorf("expected status succeeded, got %s", confirmed.Status)
    }
}

func TestDeclinedCard(t *testing.T) {
    ctx := context.Background()

    params := &stripe.PaymentIntentParams{
        Amount:   stripe.Int64(1000),
        Currency: stripe.String("usd"),
        PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
        Confirm:       stripe.Bool(true),
        PaymentMethod: stripe.String("pm_card_chargeDeclined"),
    }
    params.Context = ctx

    _, err := paymentintent.New(params)
    if err == nil {
        t.Fatal("expected error for declined card")
    }

    var stripeErr *stripe.Error
    if !errors.As(err, &stripeErr) {
        t.Fatalf("expected stripe error, got %T", err)
    }

    if stripeErr.Code != stripe.ErrorCodeCardDeclined {
        t.Errorf("expected card_declined, got %s", stripeErr.Code)
    }
}
```

## Database Schema (PostgreSQL)

```sql
-- Customers table
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stripe_customer_id VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_customers_stripe_id ON customers(stripe_customer_id);

-- Subscriptions table
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID REFERENCES customers(id),
    stripe_subscription_id VARCHAR(255) UNIQUE NOT NULL,
    stripe_price_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_customer ON subscriptions(customer_id);
CREATE INDEX idx_subscriptions_stripe_id ON subscriptions(stripe_subscription_id);

-- Payments table
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID REFERENCES customers(id),
    stripe_payment_intent_id VARCHAR(255) UNIQUE NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(50) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_payments_customer ON payments(customer_id);
CREATE INDEX idx_payments_stripe_id ON payments(stripe_payment_intent_id);

-- Webhook events table (for idempotency)
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stripe_event_id VARCHAR(255) UNIQUE NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_webhook_events_stripe_id ON webhook_events(stripe_event_id);
```

## Resources

- **references/checkout-flows.md**: Detailed checkout implementation
- **references/webhook-handling.md**: Webhook security and processing
- **references/subscription-management.md**: Subscription lifecycle
- **assets/stripe-client.go**: Production-ready Stripe client wrapper
- **assets/webhook-handler.go**: Complete webhook processor

## Best Practices

1. **Always Use Context**: Pass context through all Stripe API calls for timeouts and cancellation
2. **Idempotency Keys**: Use idempotency keys for all mutation operations
3. **Webhook Verification**: Always verify webhook signatures before processing
4. **Error Type Assertion**: Use `errors.As` to extract Stripe-specific errors
5. **Test Mode First**: Thoroughly test with test keys and test cards
6. **Metadata**: Use metadata to link Stripe objects to your database
7. **Graceful Degradation**: Handle Stripe API failures gracefully
8. **Logging**: Log payment events without sensitive data (no card numbers)

## Common Pitfalls

- **Not Verifying Webhooks**: Always verify webhook signatures
- **Missing Context**: Always pass context for proper timeout handling
- **Ignoring Errors**: Always check and handle Stripe errors appropriately
- **Hardcoded Amounts**: Use cents/smallest currency unit
- **No Idempotency**: Always use idempotency keys for mutations
- **Raw Body Modification**: Don't modify webhook body before signature verification
