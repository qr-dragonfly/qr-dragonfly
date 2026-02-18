# Stripe Quick Setup Guide

## You're seeing this error because Stripe isn't fully configured yet!

Error: `No such price: 'price_replace_with_enterprise_price_id'`

This means the placeholder values in `.env` need to be replaced with real Stripe Price IDs.

## Quick Setup (5 minutes)

### Step 1: Get Price IDs from Stripe

You already created your products! Now you need to find the **Price IDs** (not Product IDs).

**Important:** You have Product IDs (`prod_xxx`) but you need Price IDs (`price_xxx`)

#### How to Find Price IDs:

1. Go to https://dashboard.stripe.com/test/products
2. Click on your **$6 Basic Plan** product (`prod_Tw7ufXoGEDHIvL`)
3. In the "Pricing" section, you'll see the price details
4. Click on the price amount (`$6.00 / month`)
5. Look for the **Price ID** in the details (starts with `price_`)
6. **Copy this Price ID** - this is what you need!

7. Go back and click on your **$65 Enterprise Plan** product (`prod_Tw7v0c9ZrQ1PTQ`)
8. Click on the price amount (`$65.00 / month`)
9. **Copy the Price ID**

**Alternative method:** Use the Stripe API ID field directly visible on each product's pricing section.

If you don't see prices yet:

1. Click "Add another price" on each product
2. Set: Recurring, $6/month (or $65/month), Monthly billing
3. Save and copy the Price ID

### Step 2: Setup Webhook

1. Go to https://dashboard.stripe.com/test/webhooks/create
2. **Endpoint URL**: `http://localhost:8081/api/stripe/webhook` (or your deployed URL)
3. **Events to send**: Select these 3 events:
   - `checkout.session.completed`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
4. Click **Add endpoint**
5. Click **Reveal** under "Signing secret"
6. **Copy the webhook secret** (looks like `whsec_abc123...`)

### Step 3: Update .env File

Edit `backend/user-service/.env`:

```bash
# Replace these placeholder values with your real IDs from steps above:
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret_here
STRIPE_BASIC_PRICE_ID=price_your_basic_price_id_here
STRIPE_ENTERPRISE_PRICE_ID=price_your_enterprise_price_id_here
```

### Step 4: Restart User Service

```bash
cd backend/user-service
go run cmd/server/main.go
```

You should see in the logs:

```
stripe configured with basic price: price_xxxxx, enterprise price: price_xxxxx
```

## Test Your Setup

1. Go to http://localhost:5173/subscription
2. Click "Upgrade" on any plan
3. You should be redirected to Stripe Checkout
4. Use test card: `4242 4242 4242 4242`
5. Any expiry date in the future, any CVC

## Troubleshooting

### "No such price" error persists

- Double-check you copied the Price IDs correctly (not Product IDs)
- Make sure you saved the `.env` file
- Restart the user-service

### Webhook not working

- Make sure webhook endpoint is accessible
- For local development, use Stripe CLI: `stripe listen --forward-to localhost:8081/api/stripe/webhook`

### User type not updating after payment

- Check user-service logs for webhook errors
- Verify the email used in Stripe matches your Cognito user email

## Production Deployment

When deploying to production:

1. Switch to **live mode** in Stripe Dashboard (toggle in top right)
2. Create the same products/prices in live mode
3. Get live API keys and webhook secret
4. Update your production environment variables
5. Update webhook endpoint URL to your production domain

## Need Help?

- Stripe Dashboard: https://dashboard.stripe.com
- Stripe Testing Docs: https://stripe.com/docs/testing
- Webhook Testing: https://stripe.com/docs/webhooks/test
