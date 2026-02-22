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

// GetEntitlementForEmail looks up the customer's current active Stripe subscription
// and returns the matching plan tier ("basic", "enterprise", or "free").
func (c *Client) GetEntitlementForEmail(email string) (string, error) {
	customerID, err := c.findCustomerByEmail(email)
	if err != nil || customerID == "" {
		return "free", nil
	}

	sub, err := c.findAnyActiveSubscription(customerID)
	if err != nil {
		return "free", fmt.Errorf("listing subscriptions: %w", err)
	}
	if sub == nil || sub.Items == nil || len(sub.Items.Data) == 0 {
		return "free", nil
	}

	priceID := sub.Items.Data[0].Price.ID
	if basicID, err := c.GetPriceIDForPlan("basic"); err == nil && priceID == basicID {
		return "basic", nil
	}
	if enterpriseID, err := c.GetPriceIDForPlan("enterprise"); err == nil && priceID == enterpriseID {
		return "enterprise", nil
	}
	return "free", nil
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

// CreateSubscriptionWithPaymentMethod creates a subscription using a payment method ID.
// If the customer already has an active subscription on a different plan, it is upgraded
// (or downgraded) in-place rather than creating a second subscription alongside it.
func (c *Client) CreateSubscriptionWithPaymentMethod(customerEmail, paymentMethodID, priceID string) (*stripe.Subscription, error) {
	fmt.Printf("[Stripe] Creating subscription for %s with priceID: %s\n", customerEmail, priceID)

	// Find or create customer
	customerID, err := c.findOrCreateCustomer(customerEmail)
	if err != nil {
		fmt.Printf("[Stripe] Error finding/creating customer: %v\n", err)
		return nil, fmt.Errorf("failed to find/create customer: %w", err)
	}
	fmt.Printf("[Stripe] Customer ID: %s\n", customerID)

	// Attach payment method and set as default before anything else
	fmt.Printf("[Stripe] Attaching payment method %s to customer %s\n", paymentMethodID, customerID)
	pmParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	}
	if _, err = paymentmethod.Attach(paymentMethodID, pmParams); err != nil {
		fmt.Printf("[Stripe] Error attaching payment method: %v\n", err)
		return nil, fmt.Errorf("failed to attach payment method: %w", err)
	}
	custParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethodID),
		},
	}
	if _, err = customer.Update(customerID, custParams); err != nil {
		return nil, fmt.Errorf("failed to set default payment method: %w", err)
	}
	fmt.Printf("[Stripe] Payment method attached successfully\n")

	// Check for existing active subscription on ANY plan
	existingSub, err := c.findAnyActiveSubscription(customerID)
	if err != nil {
		fmt.Printf("[Stripe] Error checking existing subscriptions: %v\n", err)
		return nil, fmt.Errorf("failed to check existing subscriptions: %w", err)
	}

	if existingSub != nil {
		// Already on the exact same plan — nothing to do
		if len(existingSub.Items.Data) > 0 && existingSub.Items.Data[0].Price.ID == priceID {
			fmt.Printf("[Stripe] Customer already on target plan, returning existing sub %s\n", existingSub.ID)
			return existingSub, nil
		}

		// Different plan — upgrade/downgrade in-place to avoid duplicate active subscriptions
		fmt.Printf("[Stripe] Upgrading existing sub %s to new price %s\n", existingSub.ID, priceID)
		updateParams := &stripe.SubscriptionParams{
			Items: []*stripe.SubscriptionItemsParams{
				{
					ID:    stripe.String(existingSub.Items.Data[0].ID),
					Price: stripe.String(priceID),
				},
			},
			ProrationBehavior: stripe.String("create_prorations"),
			Expand:            stripe.StringSlice([]string{"latest_invoice.payment_intent"}),
		}
		updateParams.AddMetadata("customer_email", customerEmail)
		updated, err := subscription.Update(existingSub.ID, updateParams)
		if err != nil {
			fmt.Printf("[Stripe] Error updating subscription: %v\n", err)
			return nil, fmt.Errorf("failed to update subscription: %w", err)
		}
		fmt.Printf("[Stripe] Subscription updated successfully: %s (status: %s)\n", updated.ID, updated.Status)
		// Cancel any other lingering active subscriptions for this customer
		c.cancelOtherSubscriptions(customerID, updated.ID)
		return updated, nil
	}

	// No existing subscription — create a fresh one
	fmt.Printf("[Stripe] No existing subscription found, creating new one for customer %s with price %s\n", customerID, priceID)
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
	subParams.AddMetadata("customer_email", customerEmail)

	sub, err := subscription.New(subParams)
	if err != nil {
		fmt.Printf("[Stripe] Error creating subscription: %v\n", err)
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	fmt.Printf("[Stripe] Subscription created successfully: %s (status: %s)\n", sub.ID, sub.Status)
	// Cancel any other lingering active subscriptions for this customer
	c.cancelOtherSubscriptions(customerID, sub.ID)
	return sub, nil
}

// cancelOtherSubscriptions cancels all active or trialing subscriptions for a customer
// except the one with keepSubID. Used to clean up stale parallel subscriptions after
// an upgrade or plan change.
func (c *Client) cancelOtherSubscriptions(customerID, keepSubID string) {
	for _, status := range []string{"active", "trialing"} {
		params := &stripe.SubscriptionListParams{
			Customer: stripe.String(customerID),
		}
		params.Filters.AddFilter("status", "", status)
		iter := subscription.List(params)
		for iter.Next() {
			sub := iter.Subscription()
			if sub.ID == keepSubID {
				continue
			}
			fmt.Printf("[Stripe] Cancelling stale subscription %s for customer %s\n", sub.ID, customerID)
			if _, err := subscription.Cancel(sub.ID, nil); err != nil {
				fmt.Printf("[Stripe] Warning: failed to cancel stale subscription %s: %v\n", sub.ID, err)
			}
		}
	}
}

// GetSubscription retrieves a subscription by ID
func (c *Client) GetSubscription(subscriptionID string) (*stripe.Subscription, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return sub, nil
}

// findAnyActiveSubscription returns the first active (or trialing) subscription for a customer,
// regardless of which price/plan it is on.
func (c *Client) findAnyActiveSubscription(customerID string) (*stripe.Subscription, error) {
	fmt.Printf("[Stripe] Checking for any active subscriptions for customer %s\n", customerID)
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
	}
	params.Filters.AddFilter("status", "", "active")
	params.Expand = []*string{stripe.String("data.items")}

	iter := subscription.List(params)
	for iter.Next() {
		sub := iter.Subscription()
		fmt.Printf("[Stripe] Found active subscription %s\n", sub.ID)
		return sub, nil
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	// Also check trialing
	params2 := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
	}
	params2.Filters.AddFilter("status", "", "trialing")
	params2.Expand = []*string{stripe.String("data.items")}

	iter2 := subscription.List(params2)
	for iter2.Next() {
		sub := iter2.Subscription()
		fmt.Printf("[Stripe] Found trialing subscription %s\n", sub.ID)
		return sub, nil
	}
	return nil, iter2.Err()
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
