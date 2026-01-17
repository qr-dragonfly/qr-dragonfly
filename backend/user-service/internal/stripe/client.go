package stripe

import (
	"fmt"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/webhook"
)

type Config struct {
	SecretKey         string
	WebhookSecret     string
	BasicPriceID      string
	PortalReturnURL   string
	EnterprisePriceID string
	SuccessURL        string
	CancelURL         string
}

type Client struct {
	cfg Config
}

func NewClient(cfg Config) *Client {
	stripe.Key = cfg.SecretKey
	return &Client{cfg: cfg}
}

// CreateCheckoutSession creates a Stripe Checkout session for a subscription
func (c *Client) CreateCheckoutSession(customerEmail string, priceID string) (*stripe.CheckoutSession, error) {
	params := &stripe.CheckoutSessionParams{
		CustomerEmail: stripe.String(customerEmail),
		Mode:          stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(c.cfg.SuccessURL),
		CancelURL:  stripe.String(c.cfg.CancelURL),
	}
	sess, err := checkoutsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe session creation failed: %w", err)
	}

	return sess, nil
}

// CreateCustomerPortalSession creates a Stripe Customer Portal session for subscription management
func (c *Client) CreateCustomerPortalSession(customerEmail string) (*stripe.BillingPortalSession, error) {
	// First, find or create a customer by email
	customerID, err := c.findOrCreateCustomer(customerEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to find/create customer: %w", err)
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(c.cfg.PortalReturnURL),
	}

	portalSession, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe portal session creation failed: %w", err)
	}

	return portalSession, nil
}

// findOrCreateCustomer finds a customer by email or creates one if it doesn't exist
func (c *Client) findOrCreateCustomer(email string) (string, error) {
	// Search for existing customer
	params := &stripe.CustomerSearchParams{
		SearchParams: stripe.SearchParams{
			Query: fmt.Sprintf("email:'%s'", email),
		},
	}
	params.Limit = stripe.Int64(1)

	iter := customer.Search(params)
	if iter.Next() {
		return iter.Customer().ID, nil
	}

	// Customer doesn't exist, create one
	createParams := &stripe.CustomerParams{
		Email: stripe.String(email),
	}
	cust, err := customer.New(createParams)
	if err != nil {
		return "", fmt.Errorf("failed to create customer: %w", err)
	}

	return cust.ID, nil
}

// ConstructEvent validates and constructs a Stripe webhook event
func (c *Client) ConstructEvent(payload []byte, signature string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, signature, c.cfg.WebhookSecret)
}

// GetPriceIDForPlan returns the Stripe price ID for a given plan tier
func (c *Client) GetPriceIDForPlan(plan string) (string, error) {
	switch plan {
	case "basic":
		return c.cfg.BasicPriceID, nil
	case "enterprise":
		return c.cfg.EnterprisePriceID, nil
	default:
		return "", fmt.Errorf("invalid plan: %s", plan)
	}
}
