<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUser } from '../../composables/useUser'
import { createSubscription } from '../../api/stripe/stripe.api'

const router = useRouter()
const route = useRoute()
useUser()

// Stripe publishable key
const STRIPE_PK = 'pk_test_51Sxy89Lj4DF9Rxipkx5cl936ms3Yc8KNzf6t3h7TtUecFRwAK5f0G1AfrFpt0v4yazK3IV3SdQG43noPaDw8JyKb00kpWH4e1B'

// Get plan from route query, default to 'basic'
const selectedPlan = computed(() => {
  const plan = route.query.plan as string
  return plan === 'enterprise' ? 'enterprise' : 'basic'
})

const planInfo = computed(() => {
  return selectedPlan.value === 'enterprise'
    ? { name: 'Enterprise Plan', price: '$65', detail: '/month + tax' }
    : { name: 'Basic Plan', price: '$6', detail: '/month + tax' }
})

const loading = ref(true)
const processing = ref(false)
const error = ref<string | null>(null)
const success = ref(false)
const cardholderName = ref('')

let stripe: any = null
let cardElement: any = null

onMounted(async () => {
  try {
    // Load Stripe.js dynamically
    const script = document.createElement('script')
    script.src = 'https://js.stripe.com/v3/'
    script.async = true
    document.head.appendChild(script)

    await new Promise((resolve, reject) => {
      script.onload = resolve
      script.onerror = reject
    })

    // Wait for next tick to ensure DOM is ready
    await new Promise(resolve => setTimeout(resolve, 100))

    // Initialize Stripe
    stripe = (window as any).Stripe(STRIPE_PK)
    const elements = stripe.elements()

    // Create card element
    cardElement = elements.create('card', {
      style: {
        base: {
          color: 'rgba(255, 255, 255, 0.87)',
          fontFamily: 'system-ui, Avenir, Helvetica, Arial, sans-serif',
          fontSize: '16px',
          '::placeholder': {
            color: 'rgba(255, 255, 255, 0.4)',
          },
        },
        invalid: {
          color: '#ef4444',
          iconColor: '#ef4444',
        },
      },
    })

    // Make sure the element exists before mounting
    const cardElementContainer = document.getElementById('card-element')
    if (!cardElementContainer) {
      throw new Error('Card element container not found')
    }

    cardElement.mount('#card-element')
    
    cardElement.on('change', (event: any) => {
      if (event.error) {
        error.value = event.error.message
      } else {
        error.value = null
      }
    })

    loading.value = false
  } catch (err) {
    console.error('Failed to load Stripe:', err)
    error.value = 'Failed to load payment form. Please refresh the page.'
    loading.value = false
  }
})

async function handleSubmit() {
  if (!stripe || !cardElement) return
  
  processing.value = true
  error.value = null

  try {
    // Create payment method
    const { error: stripeError, paymentMethod } = await stripe.createPaymentMethod({
      type: 'card',
      card: cardElement,
      billing_details: {
        name: cardholderName.value,
      },
    })

    if (stripeError) {
      error.value = stripeError.message
      processing.value = false
      return
    }

    // Create subscription with payment method
    await createSubscription(selectedPlan.value as 'basic' | 'enterprise', paymentMethod.id)
    
    // Success!
    success.value = true
    setTimeout(() => {
      router.push('/subscription?success=true')
    }, 2000)
  } catch (err: any) {
    error.value = err.message || 'Failed to create subscription. Please try again.'
    processing.value = false
  }
}
</script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">Subscribe to {{ planInfo.name }}</h1>
      <p class="subtitle">Complete your payment securely with Stripe</p>
    </header>

    <section v-if="success" class="card successCard">
      <div class="successIcon">✓</div>
      <h2>Subscription Created!</h2>
      <p>Your subscription has been successfully activated. Redirecting you back...</p>
    </section>

    <section v-else class="card paymentForm">
      <form @submit.prevent="handleSubmit">
        <div class="planInfo">
          <div class="planName">{{ planInfo.name }}</div>
          <div class="planPrice">{{ planInfo.price }}<span class="priceDetail">{{ planInfo.detail }}</span></div>
        </div>

        <div class="formGroup">
          <label class="label" for="cardholder-name">Cardholder Name</label>
          <input
            id="cardholder-name"
            v-model="cardholderName"
            type="text"
            class="input"
            placeholder="John Doe"
            required
          />
        </div>

        <div class="formGroup">
          <label class="label">Card Information</label>
          <div id="card-element" class="cardElement">
            <div v-if="loading" class="loadingText">Loading payment form...</div>
          </div>
        </div>

        <p v-if="error" class="errorMessage">{{ error }}</p>

        <div class="actions">
          <button
            type="submit"
            class="button primary"
            :disabled="loading || processing || !cardholderName.trim()"
          >
            {{ processing ? 'Processing...' : 'Subscribe' }}
          </button>
          <button
            type="button"
            class="button secondary"
            @click="router.push('/subscription')"
          >
            Cancel
          </button>
        </div>

        <div class="securityNotice">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
          <span>Secured by Stripe - Your payment information is encrypted</span>
        </div>
      </form>
    </section>
  </main>
</template>

<style scoped lang="scss">
@use '../../styles/variables' as *;

.paymentForm {
  max-width: 600px;
  margin: 24px auto;
}

.successCard {
  max-width: 600px;
  margin: 24px auto;
  text-align: center;
  padding: 48px 24px;
}

.successIcon {
  width: 64px;
  height: 64px;
  background: rgba(34, 197, 94, 0.2);
  color: rgb(34, 197, 94);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  margin: 0 auto 24px;
}

.successCard h2 {
  margin: 0 0 12px 0;
  font-size: 24px;
}

.successCard p {
  margin: 0;
  opacity: 0.8;
}

.planInfo {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 8px;
  margin-bottom: 24px;
}

.planName {
  font-size: 18px;
  font-weight: 600;
}

.planPrice {
  font-size: 24px;
  font-weight: 700;
  color: $color-link;
}

.priceDetail {
  font-size: 14px;
  opacity: 0.7;
  font-weight: 500;
}

.formGroup {
  margin-bottom: 20px;
}

.label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 600;
  opacity: 0.9;
}

.input {
  width: 100%;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: inherit;
  font-size: 16px;
  transition: all 0.2s;

  &:focus {
    outline: none;
    border-color: $color-link;
    background: rgba(255, 255, 255, 0.08);
  }

  &::placeholder {
    color: rgba(255, 255, 255, 0.4);
  }
}

.cardElement {
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  transition: all 0.2s;

  &:focus-within {
    border-color: $color-link;
    background: rgba(255, 255, 255, 0.08);
  }
}

.loadingText {
  color: rgba(255, 255, 255, 0.4);
  font-size: 14px;
}

.errorMessage {
  color: $color-error;
  font-size: 14px;
  margin: 16px 0;
  padding: 12px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 6px;
}

.actions {
  display: flex;
  gap: 12px;
  margin-top: 24px;
}

.button {
  flex: 1;
  padding: 14px 24px;
  border: 1px solid transparent;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;

  &.primary {
    background: $color-link;
    color: white;

    &:hover:not(:disabled) {
      opacity: 0.9;
      transform: translateY(-1px);
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  &.secondary {
    background: transparent;
    border-color: var(--border-color);
    color: inherit;

    &:hover {
      border-color: var(--border-color-hover);
      background: rgba(255, 255, 255, 0.05);
    }
  }
}

.securityNotice {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
  font-size: 13px;
  opacity: 0.6;
  justify-content: center;

  svg {
    opacity: 0.8;
  }
}

.card {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  padding: 24px;
  background: rgba(255, 255, 255, 0.02);
  backdrop-filter: blur(10px);
}
</style>
