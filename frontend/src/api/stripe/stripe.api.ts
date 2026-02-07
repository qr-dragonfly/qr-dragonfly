import { requestJson } from '../http';

export interface CreateCheckoutSessionRequest {
  plan: 'basic' | 'enterprise';
}

export interface CreateCheckoutSessionResponse {
  sessionId: string;
  url: string;
}

export interface CreateSubscriptionRequest {
  plan: 'basic' | 'enterprise';
  paymentMethodId: string;
}

export interface CreateSubscriptionResponse {
  subscriptionId: string;
  status: string;
}

export interface CreatePortalSessionResponse {
  url: string;
}

export async function createCheckoutSession(
  plan: 'basic' | 'enterprise'
): Promise<CreateCheckoutSessionResponse> {
  return requestJson<CreateCheckoutSessionResponse>({
    path: '/api/stripe/checkout-session',
    method: 'POST',
    body: { plan },
  });
}

export async function createSubscription(
  plan: 'basic' | 'enterprise',
  paymentMethodId: string
): Promise<CreateSubscriptionResponse> {
  return requestJson<CreateSubscriptionResponse>({
    path: '/api/stripe/subscription',
    method: 'POST',
    body: { plan, paymentMethodId },
  });
}

export async function createPortalSession(): Promise<CreatePortalSessionResponse> {
  return requestJson<CreatePortalSessionResponse>({
    path: '/api/stripe/portal-session',
    method: 'POST',
  });
}
