package stripe

import (
	"fmt"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/paymentmethod"
	"github.com/stripe/stripe-go/v81/subscription"
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
func (c *Client) CreateCheckoutSession(customerEmail string, priceID string, plan string) (*stripe.CheckoutSession, error) {
	// Check if customer already has an active subscription for this price
	customerID, err := c.findCustomerByEmail(customerEmail)
	if err == nil && customerID != "" {
		existingSub, err := c.findActiveSubscription(customerID, priceID)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing subscriptions: %w", err)
		}
		if existingSub != nil {
			return nil, fmt.Errorf("customer already has an active subscription for this plan")
		}
	}

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
		Metadata:   map[string]string{"plan": plan},
	}
	// Store customer email in subscription metadata as a fallback
	params.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
		Metadata: map[string]string{
			"customer_email": customerEmail,
			"plan":           plan,
		},
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
	customerID, err := c.findCustomerByEmail(email)
	if err == nil && customerID != "" {
		return customerID, nil
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

// findCustomerByEmail searches for a customer by email
func (c *Client) findCustomerByEmail(email string) (string, error) {
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

	if err := iter.Err(); err != nil {
		return "", err
	}

	return "", nil
}

// ConstructEvent validates and constructs a Stripe webhook event
func (c *Client) ConstructEvent(payload []byte, signature string) (stripe.Event, error) {
	return webhook.ConstructEventWithOptions(
		payload,
		signature,
		c.cfg.WebhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
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

// CreateSubscriptionWithPaymentMethod creates a subscription using a payment method ID
func (c *Client) CreateSubscriptionWithPaymentMethod(customerEmail, paymentMethodID, priceID string) (*stripe.Subscription, error) {
	fmt.Printf("[Stripe] Creating subscription for %s with priceID: %s\n", customerEmail, priceID)

	// Find or create customer
	customerID, err := c.findOrCreateCustomer(customerEmail)
	if err != nil {
		fmt.Printf("[Stripe] Error finding/creating customer: %v\n", err)
		return nil, fmt.Errorf("failed to find/create customer: %w", err)
	}
	fmt.Printf("[Stripe] Customer ID: %s\n", customerID)

	// Check for existing active subscriptions for this price
	existingSub, err := c.findActiveSubscription(customerID, priceID)
	if err != nil {
		fmt.Printf("[Stripe] Error checking existing subscriptions: %v\n", err)
		return nil, fmt.Errorf("failed to check existing subscriptions: %w", err)
	}
	if existingSub != nil {
		// Found an existing active subscription - return it instead of creating duplicate
		fmt.Printf("[Stripe] Found existing active subscription %s for price %s, returning it\n", existingSub.ID, priceID)
		return existingSub, nil
	}
	fmt.Printf("[Stripe] No existing subscription found, proceeding to create new one\n")

	// Attach payment method to customer
	fmt.Printf("[Stripe] Attaching payment method %s to customer %s\n", paymentMethodID, customerID)
	pmParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	}
	_, err = paymentmethod.Attach(paymentMethodID, pmParams)
	if err != nil {
		fmt.Printf("[Stripe] Error attaching payment method: %v\n", err)
		return nil, fmt.Errorf("failed to attach payment method: %w", err)
	}
	fmt.Printf("[Stripe] Payment method attached successfully\n")

	// Set as default payment method
	custParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethodID),
		},
	}
	_, err = customer.Update(customerID, custParams)
	if err != nil {
		return nil, fmt.Errorf("failed to set default payment method: %w", err)
	}

	// Create subscription
	fmt.Printf("[Stripe] Creating new subscription for customer %s with price %s\n", customerID, priceID)
	subParams := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceID),
			},
		},
		PaymentSettings: &stripe.SubscriptionPaymentSettingsParams{
			PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		},
		Expand: stripe.StringSlice([]string{"latest_invoice.payment_intent"}),
	}
	// Store customer email in metadata as a fallback
	subParams.AddMetadata("customer_email", customerEmail)

	sub, err := subscription.New(subParams)
	if err != nil {
		fmt.Printf("[Stripe] Error creating subscription: %v\n", err)
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	fmt.Printf("[Stripe] Subscription created successfully: %s (status: %s)\n", sub.ID, sub.Status)
	return sub, nil
}

// GetSubscription retrieves a subscription by ID
func (c *Client) GetSubscription(subscriptionID string) (*stripe.Subscription, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return sub, nil
}

// findActiveSubscription checks if customer has an active subscription for the given price
func (c *Client) findActiveSubscription(customerID, priceID string) (*stripe.Subscription, error) {
	fmt.Printf("[Stripe] Checking for active subscriptions for customer %s with price %s\n", customerID, priceID)
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String("active"),
	}
	params.Expand = []*string{stripe.String("data.items")}

	iter := subscription.List(params)
	subCount := 0
	for iter.Next() {
		sub := iter.Subscription()
		subCount++
		fmt.Printf("[Stripe] Found subscription %s (status: %s)\n", sub.ID, sub.Status)
		if sub.Items != nil {
			for _, item := range sub.Items.Data {
				fmt.Printf("[Stripe]   - Item price: %s\n", item.Price.ID)
				if item.Price.ID == priceID {
					// Found active subscription with this price
					fmt.Printf("[Stripe] Match found! Returning existing subscription\n")
					return sub, nil
				}
			}
		}
	}

	fmt.Printf("[Stripe] Checked %d active subscriptions, no match for price %s\n", subCount, priceID)

	if err := iter.Err(); err != nil {
		fmt.Printf("[Stripe] Error listing subscriptions: %v\n", err)
		return nil, err
	}

	return nil, nil
}

// GetCustomer retrieves a customer by ID
func (c *Client) GetCustomer(customerID string) (*stripe.Customer, error) {
	cust, err := customer.Get(customerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	return cust, nil
}
