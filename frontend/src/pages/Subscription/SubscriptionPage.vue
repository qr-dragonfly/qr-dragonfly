<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUser } from '../../composables/useUser'
import { createPortalSession } from '../../api/stripe/stripe.api'

const route = useRoute()
const router = useRouter()
const { user, userType } = useUser()

type PlanTier = {
  name: string
  userType: 'free' | 'basic' | 'enterprise'
  price: string
  priceDetail: string
  maxActive: number
  maxTotal: number
  features: string[]
  highlight?: boolean
}

const plans: PlanTier[] = [
  {
    name: 'Free',
    userType: 'free',
    price: '$0',
    priceDetail: 'forever',
    maxActive: 5,
    maxTotal: 20,
    features: [
      'Up to 5 active QR codes',
      'Up to 20 total QR codes',
      'Basic click tracking',
      'HTTPS URLs only',
      'Create & download codes',
    ],
  },
  {
    name: 'Basic',
    userType: 'basic',
    price: '$6',
    priceDetail: 'per month + tax',
    maxActive: 50,
    maxTotal: 200,
    features: [
      'Up to 50 active QR codes',
      'Up to 200 total QR codes',
      'Advanced click analytics',
      'HTTPS URLs only',
      'Priority support',
    ],
    highlight: true,
  },
  {
    name: 'Enterprise',
    userType: 'enterprise',
    price: '$65',
    priceDetail: 'per month + tax',
    maxActive: 2000,
    maxTotal: 10000,
    features: [
      'Up to 2,000 active QR codes',
      'Up to 10,000 total QR codes',
      'Advanced analytics & exports',
      'Multi-account management (coming soon)',
      'HTTPS URLs only',
      'Dedicated support',
    ],
  },
]

const currentPlan = computed(() => {
  return plans.find((p) => p.userType === userType.value) || plans[0]
})

const checkoutError = ref<string | null>(null)
const checkoutSuccess = ref(false)
const portalLoading = ref(false)

onMounted(() => {
  // Check for success query param
  if (route.query.success === 'true') {
    checkoutSuccess.value = true
    // Clear the query param after 5 seconds
    setTimeout(() => {
      checkoutSuccess.value = false
    }, 5000)
  }
})

async function handleSubscribe(plan: 'basic' | 'enterprise') {
  // Navigate to Stripe Elements checkout page
  router.push({ path: '/checkout', query: { plan } })
}

async function handleManageBilling() {
  portalLoading.value = true
  checkoutError.value = null

  try {
    const response = await createPortalSession()
    // Redirect to Stripe Customer Portal
    window.location.href = response.url
  } catch (err) {
    console.error('Portal error:', err)
    checkoutError.value = 'Failed to open billing portal. Please try again.'
    portalLoading.value = false
  }
}

</script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">Subscription Plans</h1>
      <p class="subtitle">Choose the plan that fits your needs.</p>
    </header>

    <section v-if="user && currentPlan" class="card currentPlan">
      <h2 class="sectionTitle">Your current plan</h2>
      <div class="planBadge">
        <span class="planName">{{ currentPlan.name }}</span>
        <span class="planPrice">{{ currentPlan.price }}<span class="planDetail">/{{ currentPlan.priceDetail }}</span></span>
      </div>
      <p class="muted">
        You have {{ currentPlan.maxActive }} active and {{ currentPlan.maxTotal }} total QR code slots.
      </p>
      <div v-if="userType !== 'free'" class="billingActions">
        <button class="button secondary" @click="handleManageBilling" :disabled="portalLoading">
          {{ portalLoading ? 'Loading...' : '⚙️ Manage Billing & Payments' }}
        </button>
        <p class="billingHint">Update payment method, view invoices, or cancel subscription</p>
      </div>
    </section>

    <div v-if="checkoutSuccess" class="card success">
      ✓ Subscription successful! Your account has been upgraded. This may take a few moments to reflect.
    </div>

    <div v-if="checkoutError" class="card error">{{ checkoutError }}</div>

    <div class="plansGrid">
      <div v-for="plan in plans" :key="plan.userType" class="planCard" :class="{ highlight: plan.highlight, current: plan.userType === userType }">
        <div class="planHeader">
          <h3 class="planTitle">{{ plan.name }}</h3>
          <div class="planPricing">
            <span class="planPrice">{{ plan.price }}</span>
            <span class="planDetail">{{ plan.priceDetail }}</span>
          </div>
        </div>

        <ul class="featureList">
          <li v-for="(feature, idx) in plan.features" :key="idx" class="feature">{{ feature }}</li>
        </ul>

        <div class="planActions">
          <button v-if="plan.userType === userType" class="button current" disabled>Current Plan</button>
          <button v-else-if="plan.userType === 'free'" class="button secondary" disabled>Default Plan</button>
          <button 
            v-else 
            class="button" 
            @click="handleSubscribe(plan.userType)"
          >
            Subscribe
          </button>
        </div>
      </div>
    </div>

    <section class="card faq">
      <h2 class="sectionTitle">Frequently Asked Questions</h2>

      <div class="faqItem">
        <h3 class="faqQuestion">What's the difference between active and total QR codes?</h3>
        <p class="faqAnswer">
          Active QR codes are currently enabled and users can scan them. Total QR codes includes both active and inactive codes in your account.
        </p>
      </div>

      <div class="faqItem">
        <h3 class="faqQuestion">Can I upgrade or downgrade anytime?</h3>
        <p class="faqAnswer">
          Yes! Contact our sales team to change your plan. Changes will be reflected in your next billing cycle.
        </p>
      </div>

      <div class="faqItem">
        <h3 class="faqQuestion">Do you offer custom enterprise plans?</h3>
        <p class="faqAnswer">
          Absolutely. For organizations needing more than 10,000 QR codes or specialized features, please contact our sales team.
        </p>
      </div>
    </section>
  </main>
</template>

<style scoped src="../HomePage/HomePage.scss" lang="scss"></style>
<style scoped src="./SubscriptionPage.scss" lang="scss"></style>
