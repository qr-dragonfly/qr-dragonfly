# Stripe Subscription Cleanup Guide

## Problem

You may have multiple active subscriptions for the same customer/plan in Stripe, which can cause:

- Double billing
- Confusion about which subscription is "active"
- Webhook conflicts

## How to Check for Duplicates

### Via Stripe Dashboard

1. Go to https://dashboard.stripe.com/test/customers
2. Search for customer by email
3. Click on the customer
4. Check the "Subscriptions" tab
5. Look for multiple "Active" subscriptions with the same price

### Via Stripe CLI

```bash
# List all active subscriptions for a customer
stripe subscriptions list --customer cus_XXXXX --status active

# Search for customer by email first
stripe customers search --query "email:'customer@example.com'"
```

## How to Cancel Duplicate Subscriptions

### Via Stripe Dashboard

1. Go to the customer's page
2. Click on the duplicate subscription
3. Click "Cancel subscription"
4. Choose "Cancel immediately" (not at period end)
5. Confirm cancellation

### Via Stripe CLI

```bash
# Cancel a specific subscription
stripe subscriptions cancel sub_XXXXX

# Cancel immediately without refund
stripe subscriptions cancel sub_XXXXX --invoice-now=false
```

## Prevention (Now Implemented)

The backend now includes duplicate prevention:

1. **Before creating checkout session**: Checks if customer already has active subscription for that price
2. **Before creating direct subscription**: Returns existing subscription if found
3. **Error message**: Returns clear error if duplicate detected

## What Changed

### Backend ([stripe/client.go](backend/user-service/internal/stripe/client.go))

- ✅ Added `findActiveSubscription()` - checks for existing subscriptions
- ✅ Added `findCustomerByEmail()` - finds customer without creating one
- ✅ Updated `CreateCheckoutSession()` - prevents duplicate checkout sessions
- ✅ Updated `CreateSubscriptionWithPaymentMethod()` - returns existing subscription

### Frontend ([SubscriptionPage.vue](frontend/src/pages/Subscription/SubscriptionPage.vue))

- ✅ Better error handling for duplicate subscription errors
- ✅ Clear user message: "You already have an active subscription for this plan"

## Testing the Fix

1. Restart the backend:

   ```bash
   cd /Users/bender/Desktop/image-code
   docker-compose restart user-service
   ```

2. Try to subscribe to a plan you already have
3. Should see error: "You already have an active subscription for this plan"

## Cleaning Up Existing Duplicates

### Step 1: Identify the duplicates

```bash
# Get customer ID
stripe customers search --query "email:'icfenderbender@gmail.com'"

# List their subscriptions
stripe subscriptions list --customer cus_XXXXX --status active
```

### Step 2: Keep the most recent one

Look at the `created` timestamp. Keep the newest subscription, cancel the older ones.

### Step 3: Cancel old subscriptions

```bash
stripe subscriptions cancel sub_OLD_ONE
stripe subscriptions cancel sub_ANOTHER_OLD_ONE
# Keep: sub_NEWEST_ONE
```

### Step 4: Verify in dashboard

Check that only ONE active subscription remains for each plan.

## Future Considerations

### For Production

Consider adding these Stripe features:

1. **Subscription Update Instead of Create**
   - Instead of creating new subscriptions, update existing ones
   - Changes plan without creating duplicates

2. **Idempotency Keys**
   - Use idempotency keys for all Stripe API calls
   - Prevents duplicate charges on retry

3. **Subscription Schedules**
   - Use Stripe's subscription schedules for plan changes
   - Better handles upgrades/downgrades

## Example Cleanup Script

```bash
#!/bin/bash
# cleanup-duplicates.sh

EMAIL="icfenderbender@gmail.com"

echo "Finding customer..."
CUSTOMER_ID=$(stripe customers search --query "email:'$EMAIL'" | jq -r '.data[0].id')

if [ -z "$CUSTOMER_ID" ] || [ "$CUSTOMER_ID" = "null" ]; then
  echo "Customer not found"
  exit 1
fi

echo "Customer ID: $CUSTOMER_ID"
echo ""
echo "Active subscriptions:"
stripe subscriptions list --customer "$CUSTOMER_ID" --status active

echo ""
echo "To cancel a subscription, run:"
echo "stripe subscriptions cancel sub_XXXXX"
```

Save this script, make it executable (`chmod +x cleanup-duplicates.sh`), and run it to help identify duplicates.

## Support

If you continue to see duplicates after implementing these fixes, check:

1. Frontend is sending only ONE request (check Network tab)
2. Backend logs show only ONE subscription creation
3. Stripe webhook logs show the creation event

The most common cause is users clicking "Subscribe" multiple times while the first request is processing.
