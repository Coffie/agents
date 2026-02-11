# Getting Started with Stripe Go Integration

This guide covers setting up Stripe payments in a Go application from scratch.

## Prerequisites

- Go 1.21+
- Stripe account (test mode)
- Basic understanding of HTTP handlers in Go

## Installation

```bash
go get github.com/stripe/stripe-go/v81
```

## Project Structure

```
your-app/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── payments/
│   │   ├── client.go      # Stripe client wrapper
│   │   ├── handlers.go    # HTTP handlers
│   │   └── webhooks.go    # Webhook processing
│   └── models/
│       └── payment.go     # Database models
├── migrations/
│   └── 001_create_payments.sql
├── go.mod
└── .env
```

## Environment Setup

Create a `.env` file:

```bash
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PUBLISHABLE_KEY=pk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
DATABASE_URL=postgres://user:pass@localhost/myapp?sslmode=disable
```

## Step 1: Initialize Stripe

```go
// internal/payments/client.go
package payments

import (
    "os"

    "github.com/stripe/stripe-go/v81"
)

func init() {
    stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
}
```

## Step 2: Create Payment Handlers

```go
// internal/payments/handlers.go
package payments

import (
    "encoding/json"
    "net/http"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/checkout/session"
)

type CreateCheckoutRequest struct {
    PriceID    string `json:"price_id"`
    SuccessURL string `json:"success_url"`
    CancelURL  string `json:"cancel_url"`
}

func CreateCheckoutHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateCheckoutRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    params := &stripe.CheckoutSessionParams{
        Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
        LineItems: []*stripe.CheckoutSessionLineItemParams{
            {
                Price:    stripe.String(req.PriceID),
                Quantity: stripe.Int64(1),
            },
        },
        SuccessURL: stripe.String(req.SuccessURL),
        CancelURL:  stripe.String(req.CancelURL),
    }

    sess, err := session.New(params)
    if err != nil {
        http.Error(w, "failed to create session", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{
        "url": sess.URL,
    })
}
```

## Step 3: Set Up Webhooks

```go
// internal/payments/webhooks.go
package payments

import (
    "encoding/json"
    "io"
    "log"
    "net/http"
    "os"

    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/webhook"
)

func WebhookHandler(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "read error", http.StatusBadRequest)
        return
    }

    sig := r.Header.Get("Stripe-Signature")
    event, err := webhook.ConstructEvent(body, sig, os.Getenv("STRIPE_WEBHOOK_SECRET"))
    if err != nil {
        http.Error(w, "invalid signature", http.StatusBadRequest)
        return
    }

    switch event.Type {
    case "checkout.session.completed":
        var session stripe.CheckoutSession
        json.Unmarshal(event.Data.Raw, &session)
        log.Printf("Checkout completed: %s", session.ID)
        // Fulfill order here

    case "payment_intent.succeeded":
        var pi stripe.PaymentIntent
        json.Unmarshal(event.Data.Raw, &pi)
        log.Printf("Payment succeeded: %s", pi.ID)
    }

    w.WriteHeader(http.StatusOK)
}
```

## Step 4: Wire Up Routes

```go
// cmd/server/main.go
package main

import (
    "log"
    "net/http"

    "your-app/internal/payments"
)

func main() {
    http.HandleFunc("POST /api/checkout", payments.CreateCheckoutHandler)
    http.HandleFunc("POST /webhook", payments.WebhookHandler)

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Step 5: Test with Stripe CLI

```bash
# Install Stripe CLI
brew install stripe/stripe-cli/stripe

# Login
stripe login

# Forward webhooks to local server
stripe listen --forward-to localhost:8080/webhook

# In another terminal, trigger test events
stripe trigger payment_intent.succeeded
```

## Test Cards

| Card Number         | Scenario            |
|---------------------|---------------------|
| 4242424242424242    | Success             |
| 4000000000000002    | Declined            |
| 4000002500003155    | Requires 3D Secure  |
| 4000000000009995    | Insufficient funds  |

## Next Steps

1. **Add database storage** - Store payment records for auditing
2. **Implement subscriptions** - Add recurring billing
3. **Add customer management** - Create and manage Stripe customers
4. **Set up monitoring** - Track payment success rates
5. **Go to production** - Switch to live keys and enable webhooks

## Common Issues

### Webhook Signature Verification Fails

- Ensure you're using the raw request body
- Check that `STRIPE_WEBHOOK_SECRET` matches your endpoint
- Don't modify the body before verification

### Context Timeout

- Pass context with appropriate timeouts
- Stripe operations typically complete within 5-10 seconds

### Idempotency

- Use idempotency keys for mutations
- Store event IDs to prevent duplicate processing
