# Stripe Integration Setup

This document explains how to configure Stripe for subscription payments in the QR-Dragonfly application.

## Overview

The application uses Stripe Checkout for subscription payments with two tiers:

- **Basic**: $6/month - 50 active QR codes, 200 total
- **Enterprise**: $65/month - 2,000 active QR codes, 10,000 total

When a user subscribes, their account is automatically upgraded via Stripe webhooks.

## Backend Configuration

### Required Environment Variables

Add these to your user-service environment (`.env` file or deployment config):

```bash
# Stripe API Keys
STRIPE_SECRET_KEY=sk_test_xxxxx              # Your Stripe secret key (from Stripe Dashboard)
STRIPE_WEBHOOK_SECRET=whsec_xxxxx            # Webhook signing secret (from Stripe Dashboard)

# Stripe Product Price IDs
STRIPE_BASIC_PRICE_ID=price_xxxxx            # Price ID for Basic plan
STRIPE_ENTERPRISE_PRICE_ID=price_xxxxx       # Price ID for Enterprise plan

# Redirect URLs (optional, defaults provided)
STRIPE_SUCCESS_URL=http://localhost:5173/subscription?success=true
STRIPE_CANCEL_URL=http://localhost:5173/subscription
```

### Setup Steps

1. **Create Stripe Account**
   - Go to https://stripe.com and create an account
   - Get your API keys from Dashboard → Developers → API keys

2. **Create Products and Prices**
   - Go to Dashboard → Products → Add product (https://dashboard.stripe.com/test/products)
   - Create "Basic" product:
     - Name: "Basic Plan"
     - Pricing: $6.00 USD / month (recurring)
     - Copy the Price ID (starts with `price_`)
   - Create "Enterprise" product:
     - Name: "Enterprise Plan"
     - Pricing: $65.00 USD / month (recurring)
     - Copy the Price ID (starts with `price_`)

3. **Configure Webhooks**
   - Go to Dashboard → Developers → Webhooks
   - Add endpoint: `https://your-domain.com/api/stripe/webhook`
   - Select events to listen to:
     - `checkout.session.completed`
     - `customer.subscription.updated`
     - `customer.subscription.deleted`
   - Copy the webhook signing secret (starts with `whsec_`)

4. **Set Environment Variables**
   - Add all the variables listed above to your deployment
   - For local development, use test keys (start with `sk_test_` and `pk_test_`)

## Frontend Configuration

The frontend automatically connects to the backend Stripe endpoints. No additional configuration needed.

### User Flow

1. User clicks "Subscribe" button on subscription page
2. Frontend calls `/api/stripe/checkout-session` with plan type
3. User is redirected to Stripe Checkout
4. After payment, Stripe redirects back to `/subscription?success=true`
5. Stripe webhook updates user type in Cognito
6. User's account is upgraded (may take a few moments to reflect)

## Testing

### Test Mode

1. Use Stripe test API keys during development
2. Use test card numbers from https://stripe.com/docs/testing
   - Success: `4242 4242 4242 4242`
   - 3D Secure: `4000 0025 0000 3155`
   - Declined: `4000 0000 0000 0002`

3. Test the webhook locally using Stripe CLI:
   ```bash
   stripe listen --forward-to localhost:8081/api/stripe/webhook
   stripe trigger checkout.session.completed
   ```

### Verify Integration

1. Start both services:

   ```bash
   # Terminal 1: Frontend
   cd frontend && npm run dev

   # Terminal 2: User Service
   cd backend/user-service && go run cmd/server/main.go
   ```

2. Log in to the app
3. Go to Subscription page
4. Click "Subscribe" on Basic or Enterprise
5. Complete checkout with test card
6. Verify user type is updated in Cognito

## Webhook Events

The application handles these Stripe webhook events:

- **checkout.session.completed**: Upgrades user after successful payment
- **customer.subscription.updated**: Updates user tier if plan changes
- **customer.subscription.deleted**: Downgrades user to free tier on cancellation

## Production Deployment

1. Switch to live API keys (start with `sk_live_` and `pk_live_`)
2. Update webhook endpoint to production URL
3. Update success/cancel URLs to production domain
4. Test with real payment method
5. Monitor webhook delivery in Stripe Dashboard

## Troubleshooting

### Webhook Not Received

- Check webhook endpoint is publicly accessible
- Verify webhook secret is correct
- Check Stripe Dashboard → Webhooks for delivery attempts and errors

### User Type Not Updating

- Check user-service logs for webhook processing errors
- Verify email in Stripe matches email in Cognito
- Ensure custom:user_type attribute exists in Cognito User Pool

### Checkout Session Fails

- Verify price IDs are correct
- Check Stripe API key has correct permissions
- Review user-service logs for detailed error messages

## Security Notes

- Never commit API keys to version control
- Use environment variables for all secrets
- Webhook signature verification is mandatory (handled automatically)
- Always use HTTPS in production for webhook endpoint
- Stripe API keys should have minimal required permissions
