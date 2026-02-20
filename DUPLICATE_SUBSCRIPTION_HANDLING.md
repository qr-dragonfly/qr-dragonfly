# Duplicate Subscription Handling

## Overview

Updated the subscription flow to properly handle cases where a user already has an active subscription. Instead of returning an error, the system now:

1. Returns the existing subscription
2. Updates AWS Cognito user attributes immediately
3. Refreshes the frontend user session to reflect the updated entitlements

## Changes Made

### Backend: user-service

#### `/backend/user-service/internal/stripe/client.go`

- **`CreateSubscriptionWithPaymentMethod()`**: Changed to return existing subscription instead of error when duplicate is found
- Before: `return nil, fmt.Errorf("customer already has an active %s subscription")`
- After: `return existingSub, nil`

#### `/backend/user-service/internal/httpapi/stripe.go`

- **`handleCreateSubscription()`**: Now updates Cognito user attributes immediately after subscription creation/retrieval
- Added logic to extract entitlement from subscription items
- Calls `updateUserEntitlementByEmail()` synchronously
- Returns entitlement in API response: `{"subscriptionId": "...", "status": "...", "entitlement": "basic"}`

### Frontend

#### `/frontend/src/pages/Subscription/StripeCheckoutPage.vue`

- Added call to `reloadCurrentUser()` after successful subscription creation
- This fetches the latest user attributes from Cognito, including updated `custom:user_type`
- User session is refreshed before redirecting to subscription page

#### `/frontend/src/api/stripe/stripe.api.ts`

- Updated `CreateSubscriptionResponse` interface to include optional `entitlement` field

## User Flow

### New Subscription

1. User enters payment details and submits
2. Backend creates Stripe subscription
3. Backend updates Cognito user attributes (custom:user_type, custom:entitlements)
4. Frontend receives successful response
5. Frontend reloads user session from Cognito
6. User redirected to subscription page with updated tier displayed

### Existing Subscription (Duplicate Attempt)

1. User enters payment details and submits
2. Backend finds existing active subscription for the same price
3. Backend returns existing subscription (not an error)
4. Backend updates Cognito user attributes (ensures they match subscription)
5. Frontend receives successful response
6. Frontend reloads user session from Cognito
7. User redirected to subscription page with correct tier displayed

## Benefits

1. **No Failed Payments**: Users with existing subscriptions won't see confusing errors
2. **Immediate Entitlement Updates**: No need to wait for webhooks
3. **Consistent State**: Cognito attributes are updated synchronously with the API call
4. **Better UX**: User sees their upgraded status immediately after successful payment

## Testing

To test duplicate subscription handling:

1. Create a subscription for a user
2. Try to create the same subscription again (same price ID)
3. Verify:
   - API returns 200 (not 500)
   - Response includes existing subscription ID
   - Cognito user attributes are correct
   - Frontend shows correct user tier immediately
   - No duplicate subscription created in Stripe dashboard

## Logs Example

```
[Stripe] Creating subscription for user@example.com with priceID: price_1SyFfGLj4DF9Rxip5N5tPUNu
[Stripe] Customer ID: cus_ABC123
[Stripe] Checking for active subscriptions for customer cus_ABC123 with price price_1SyFfGLj4DF9Rxip5N5tPUNu
[Stripe] Found existing active subscription sub_XYZ789, returning it
subscription created/found successfully: sub_XYZ789, status: active
updating user user@example.com entitlement to enterprise
```

## Related Files

- [STRIPE_DUPLICATE_PREVENTION.md](./STRIPE_DUPLICATE_PREVENTION.md) - Original duplicate prevention strategy
- [backend/user-service/internal/stripe/client.go](./backend/user-service/internal/stripe/client.go)
- [backend/user-service/internal/httpapi/stripe.go](./backend/user-service/internal/httpapi/stripe.go)
- [frontend/src/pages/Subscription/StripeCheckoutPage.vue](./frontend/src/pages/Subscription/StripeCheckoutPage.vue)
