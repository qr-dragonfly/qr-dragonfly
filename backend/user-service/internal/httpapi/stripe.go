package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitoTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/stripe/stripe-go/v81"
)

type createCheckoutSessionRequest struct {
	Plan string `json:"plan"` // "basic" or "enterprise"
}

type createSubscriptionRequest struct {
	Plan            string `json:"plan"`            // "basic" or "enterprise"
	PaymentMethodID string `json:"paymentMethodId"` // Stripe payment method ID
}

type checkoutSessionResponse struct {
	SessionID  string `json:"sessionId"`
	SessionURL string `json:"url"`
}

// handleCreateCheckoutSession creates a Stripe Checkout session for subscription
func (srv *Server) handleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	access, _ := readCookie(r, "access_token")
	if access == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not_authenticated"})
		return
	}

	if srv.StripeClient == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "stripe_not_configured"})
		return
	}

	// Get user from access token
	user, err := getUserFromAccessToken(ctx, srv.Cognito, access)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not_authenticated"})
		return
	}

	var req createCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	req.Plan = strings.TrimSpace(strings.ToLower(req.Plan))
	if req.Plan != "basic" && req.Plan != "enterprise" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_plan"})
		return
	}

	priceID, err := srv.StripeClient.GetPriceIDForPlan(req.Plan)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_plan"})
		return
	}

	checkoutSession, err := srv.StripeClient.CreateCheckoutSession(user.Email, priceID)
	if err != nil {
		log.Printf("stripe checkout error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "checkout_failed"})
		return
	}

	writeJSON(w, http.StatusOK, checkoutSessionResponse{
		SessionID:  checkoutSession.ID,
		SessionURL: checkoutSession.URL,
	})
}

// handleCreateSubscription creates a subscription directly with a payment method
func (srv *Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	access, _ := readCookie(r, "access_token")
	if access == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not_authenticated"})
		return
	}

	if srv.StripeClient == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "stripe_not_configured"})
		return
	}

	// Get user from access token
	user, err := getUserFromAccessToken(ctx, srv.Cognito, access)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not_authenticated"})
		return
	}

	var req createSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	req.Plan = strings.TrimSpace(strings.ToLower(req.Plan))
	req.PaymentMethodID = strings.TrimSpace(req.PaymentMethodID)

	if req.Plan != "basic" && req.Plan != "enterprise" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_plan"})
		return
	}

	if req.PaymentMethodID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payment_method_required"})
		return
	}

	priceID, err := srv.StripeClient.GetPriceIDForPlan(req.Plan)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_plan"})
		return
	}

	log.Printf("creating subscription for %s, plan: %s, priceID: %s", user.Email, req.Plan, priceID)

	sub, err := srv.StripeClient.CreateSubscriptionWithPaymentMethod(user.Email, req.PaymentMethodID, priceID)
	if err != nil {
		log.Printf("stripe subscription error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "subscription_failed"})
		return
	}

	log.Printf("subscription created successfully: %s, status: %s", sub.ID, sub.Status)

	// Subscription created successfully
	// The webhook will handle updating user entitlements
	writeJSON(w, http.StatusOK, map[string]any{
		"subscriptionId": sub.ID,
		"status":         string(sub.Status),
	})
}

// handleCreatePortalSession creates a Stripe Customer Portal session for subscription management
func (srv *Server) handleCreatePortalSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	access, _ := readCookie(r, "access_token")
	if access == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not_authenticated"})
		return
	}

	if srv.StripeClient == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "stripe_not_configured"})
		return
	}

	// Get user from access token
	user, err := getUserFromAccessToken(ctx, srv.Cognito, access)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not_authenticated"})
		return
	}

	portalSession, err := srv.StripeClient.CreateCustomerPortalSession(user.Email)
	if err != nil {
		log.Printf("stripe portal session error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "portal_session_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"url": portalSession.URL,
	})
}

// handleStripeWebhook processes Stripe webhook events
func (srv *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if srv.StripeClient == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "stripe_not_configured"})
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_payload"})
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	event, err := srv.StripeClient.ConstructEvent(payload, signature)
	if err != nil {
		log.Printf("webhook signature verification failed: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_signature"})
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		srv.handleCheckoutCompleted(event)
	case "customer.subscription.created":
		srv.handleSubscriptionCreated(event)
	case "customer.subscription.updated":
		srv.handleSubscriptionUpdated(event)
	case "customer.subscription.deleted":
		srv.handleSubscriptionDeleted(event)
	default:
		log.Printf("unhandled webhook event type: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func (srv *Server) handleCheckoutCompleted(event stripe.Event) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		log.Printf("error parsing checkout.session.completed: %v", err)
		return
	}

	if session.Mode != stripe.CheckoutSessionModeSubscription {
		return
	}

	customerEmail := session.CustomerEmail
	if customerEmail == "" && session.CustomerDetails != nil {
		customerEmail = session.CustomerDetails.Email
	}

	if customerEmail == "" {
		log.Printf("no email in checkout session: %s", session.ID)
		return
	}

	// Get subscription details to determine tier
	entitlement := "free" // default

	// Try to get from metadata first (if we set it during session creation)
	if meta, ok := session.Metadata["plan"]; ok {
		entitlement = meta
		log.Printf("got entitlement from metadata: %s", entitlement)
	} else if session.Subscription != nil {
		// Get subscription details
		subscriptionID := session.Subscription.ID

		if subscriptionID != "" {
			entitlement = srv.getEntitlementFromSubscriptionID(subscriptionID)
		}
	}

	log.Printf("checkout completed for %s, updating entitlement to %s", customerEmail, entitlement)
	srv.updateUserEntitlementByEmail(context.Background(), customerEmail, entitlement)
}

func (srv *Server) handleSubscriptionCreated(event stripe.Event) {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("error parsing customer.subscription.created: %v", err)
		return
	}

	if subscription.Status != stripe.SubscriptionStatusActive && subscription.Status != stripe.SubscriptionStatusTrialing {
		log.Printf("subscription %s not active/trialing, status: %s", subscription.ID, subscription.Status)
		return
	}

	// Get customer email
	customerEmail := srv.getCustomerEmail(&subscription)
	if customerEmail == "" {
		log.Printf("could not determine customer email for subscription %s", subscription.ID)
		return
	}

	// Determine entitlement from subscription items
	entitlement := "free"
	if subscription.Items != nil && len(subscription.Items.Data) > 0 {
		priceID := subscription.Items.Data[0].Price.ID
		entitlement = srv.getEntitlementFromPriceID(priceID)
	}

	log.Printf("subscription %s created for %s, setting entitlement to %s", subscription.ID, customerEmail, entitlement)
	srv.updateUserEntitlementByEmail(context.Background(), customerEmail, entitlement)
}

func (srv *Server) handleSubscriptionUpdated(event stripe.Event) {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("error parsing customer.subscription.updated: %v", err)
		return
	}

	if subscription.Status != stripe.SubscriptionStatusActive {
		log.Printf("subscription %s not active, status: %s", subscription.ID, subscription.Status)
		// If canceled or past_due, downgrade to free
		if subscription.Status == stripe.SubscriptionStatusCanceled ||
			subscription.Status == stripe.SubscriptionStatusIncomplete ||
			subscription.Status == stripe.SubscriptionStatusIncompleteExpired {
			if customerEmail := srv.getCustomerEmail(&subscription); customerEmail != "" {
				log.Printf("subscription %s inactive, downgrading customer to free", subscription.ID)
				srv.updateUserEntitlementByEmail(context.Background(), customerEmail, "free")
			}
		}
		return
	}

	// Get customer email
	customerEmail := srv.getCustomerEmail(&subscription)
	if customerEmail == "" {
		log.Printf("could not determine customer email for subscription %s", subscription.ID)
		return
	}

	// Determine entitlement from subscription items
	entitlement := "free"
	if subscription.Items != nil && len(subscription.Items.Data) > 0 {
		priceID := subscription.Items.Data[0].Price.ID
		entitlement = srv.getEntitlementFromPriceID(priceID)
	}

	log.Printf("subscription %s updated for %s, setting entitlement to %s", subscription.ID, customerEmail, entitlement)
	srv.updateUserEntitlementByEmail(context.Background(), customerEmail, entitlement)
}

func (srv *Server) handleSubscriptionDeleted(event stripe.Event) {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		log.Printf("error parsing customer.subscription.deleted: %v", err)
		return
	}

	// Downgrade user to free tier
	customerEmail := srv.getCustomerEmail(&subscription)
	if customerEmail == "" {
		log.Printf("could not determine customer email for deleted subscription %s", subscription.ID)
		return
	}

	log.Printf("subscription %s deleted, downgrading %s to free", subscription.ID, customerEmail)
	srv.updateUserEntitlementByEmail(context.Background(), customerEmail, "free")
}

func (srv *Server) updateUserEntitlementByEmail(ctx context.Context, email, entitlement string) {
	// List users to find by email
	listOut, err := srv.Cognito.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
		UserPoolId: aws.String(srv.UserPoolID),
		Filter:     aws.String(fmt.Sprintf(`email = "%s"`, email)),
		Limit:      aws.Int32(1),
	})
	if err != nil || len(listOut.Users) == 0 {
		log.Printf("user not found for email %s: %v", email, err)
		return
	}

	username := aws.ToString(listOut.Users[0].Username)

	// Update both user_type (for backward compatibility) and entitlements
	_, err = srv.Cognito.AdminUpdateUserAttributes(ctx, &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(srv.UserPoolID),
		Username:   aws.String(username),
		UserAttributes: []cognitoTypes.AttributeType{
			{Name: aws.String(cognitoUserTypeAttr), Value: aws.String(entitlement)},
			{Name: aws.String(cognitoEntitlementsAttr), Value: aws.String(entitlement)},
		},
	})
	if err != nil {
		log.Printf("failed to update entitlement for %s: %v", email, err)
		return
	}

	log.Printf("updated user %s to entitlement %s", email, entitlement)
}

// getEntitlementFromPriceID maps Stripe price ID to entitlement tier
func (srv *Server) getEntitlementFromPriceID(priceID string) string {
	if srv.StripeClient == nil {
		return "free"
	}

	plan, err := srv.StripeClient.GetPriceIDForPlan("basic")
	if err == nil && plan == priceID {
		return "basic"
	}

	plan, err = srv.StripeClient.GetPriceIDForPlan("enterprise")
	if err == nil && plan == priceID {
		return "enterprise"
	}

	return "free"
}

// getCustomerEmail retrieves customer email from a subscription
func (srv *Server) getCustomerEmail(subscription *stripe.Subscription) string {
	if subscription.Customer == nil {
		return ""
	}

	// If customer is expanded, get email directly
	if subscription.Customer.Email != "" {
		return subscription.Customer.Email
	}

	// Customer metadata might have email
	if email, ok := subscription.Metadata["customer_email"]; ok {
		return email
	}

	// If customer is just an ID string, fetch the full customer object
	customerID := subscription.Customer.ID
	if customerID != "" && srv.StripeClient != nil {
		customer, err := srv.StripeClient.GetCustomer(customerID)
		if err != nil {
			log.Printf("failed to fetch customer %s: %v", customerID, err)
			return ""
		}
		return customer.Email
	}

	return ""
}

// getEntitlementFromSubscriptionID retrieves entitlement from a subscription ID
func (srv *Server) getEntitlementFromSubscriptionID(subscriptionID string) string {
	if srv.StripeClient == nil {
		return "free"
	}

	sub, err := srv.StripeClient.GetSubscription(subscriptionID)
	if err != nil {
		log.Printf("failed to get subscription %s: %v", subscriptionID, err)
		return "free"
	}

	if sub.Items != nil && len(sub.Items.Data) > 0 {
		priceID := sub.Items.Data[0].Price.ID
		return srv.getEntitlementFromPriceID(priceID)
	}

	log.Printf("no items found in subscription %s, defaulting to free", subscriptionID)
	return "free"
}
