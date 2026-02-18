<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUser } from '../../composables/useUser'
import { createPortalSession } from '../../api/stripe/stripe.api'

const route = useRoute()
const router = useRouter()
const { user, userType, reload: reloadUser } = useUser()

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
const showConfirmModal = ref(false)
const pendingPlan = ref<PlanTier | null>(null)
const actionLoading = ref(false)
const waitingForWebhook = ref(false)
const webhookCheckInterval = ref<number | null>(null)

const planAction = computed(() => {
  if (!pendingPlan.value || !currentPlan.value) return null
  
  const currentIndex = plans.findIndex(p => p.userType === currentPlan.value?.userType)
  const targetIndex = plans.findIndex(p => p.userType === pendingPlan.value?.userType)
  
  if (targetIndex > currentIndex) return 'upgrade'
  if (targetIndex < currentIndex) return 'downgrade'
  return null
})

onMounted(() => {
  // Check for success query param
  if (route.query.success === 'true') {
    checkoutSuccess.value = true
    waitingForWebhook.value = true
    
    // Poll to check if user type has been updated
    let pollCount = 0
    const maxPolls = 20 // Poll for up to 20 seconds
    
    webhookCheckInterval.value = window.setInterval(async () => {
      pollCount++
      
      // Refresh user data from Cognito
      await reloadUser()
      
      // Stop polling after max attempts or if account is upgraded
      if (pollCount >= maxPolls || userType.value !== 'free') {
        waitingForWebhook.value = false
        if (webhookCheckInterval.value) {
          clearInterval(webhookCheckInterval.value)
          webhookCheckInterval.value = null
        }
      }
    }, 1000) // Check every second
    
    // Clear the success message after 30 seconds
    setTimeout(() => {
      checkoutSuccess.value = false
      waitingForWebhook.value = false
      if (webhookCheckInterval.value) {
        clearInterval(webhookCheckInterval.value)
        webhookCheckInterval.value = null
      }
    }, 5000)
  }
})

onUnmounted(() => {
  // Cleanup interval on component unmount
  if (webhookCheckInterval.value) {
    clearInterval(webhookCheckInterval.value)
  }
})

function handlePlanClick(plan: PlanTier) {
  // Don't do anything for current plan
  if (plan.userType === userType.value) {
    return
  }
  
  pendingPlan.value = plan
  showConfirmModal.value = true
  checkoutError.value = null
}

function closeModal() {
  showConfirmModal.value = false
  pendingPlan.value = null
  actionLoading.value = false
}

async function confirmPlanChange() {
  if (!pendingPlan.value) return
  
  actionLoading.value = true
  checkoutError.value = null
  
  try {
    if (planAction.value === 'upgrade') {
      // For upgrades, use embedded checkout page (Stripe Elements)
      router.push({ path: '/checkout', query: { plan: pendingPlan.value.userType } })
      
      // Alternative: Use Stripe Checkout (hosted page)
      // Uncomment below to redirect to Stripe's hosted checkout instead:
      // const response = await createCheckoutSession(pendingPlan.value.userType as 'basic' | 'enterprise')
      // window.location.href = response.url
    } else if (planAction.value === 'downgrade') {
      // For downgrades (including to free), use the portal
      const response = await createPortalSession()
      window.location.href = response.url
    }
  } catch (err: any) {
    console.error('Plan change error:', err)
    
    // Check for specific error messages
    const errorMessage = err?.payload?.error || err?.message || 'Unknown error'
    
    if (errorMessage.includes('already has an active subscription')) {
      checkoutError.value = 'You already have an active subscription for this plan. Please manage your billing to make changes.'
    } else if (err?.status === 401) {
      checkoutError.value = 'Please log in again to continue.'
    } else {
      checkoutError.value = `Failed to ${planAction.value}. Please try again.`
    }
    
    actionLoading.value = false
    closeModal()
  }
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

function getPlanButtonText(plan: PlanTier): string {
  if (!currentPlan.value) return 'Select'
  
  const currentIndex = plans.findIndex(p => p.userType === currentPlan.value?.userType)
  const targetIndex = plans.findIndex(p => p.userType === plan.userType)
  
  if (targetIndex > currentIndex) return 'Upgrade'
  if (targetIndex < currentIndex) return 'Downgrade'
  return 'Select'
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
      <div class="successContent">
        <span>✓ Subscription successful!</span>
        <div v-if="waitingForWebhook" class="spinnerContainer">
          <div class="spinner"></div>
          <span class="spinnerText">Updating your account...</span>
        </div>
        <span v-else>Your account has been upgraded. This may take a few moments to reflect.</span>
      </div>
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
          <button 
            v-else 
            class="button" 
            :class="{ primary: plan.highlight, secondary: plan.userType === 'free' }"
            @click="handlePlanClick(plan)"
          >
            {{ getPlanButtonText(plan) }}
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

    <!-- Confirmation Modal -->
    <div v-if="showConfirmModal" class="modalOverlay" @click="closeModal">
      <div class="modal" @click.stop>
        <div class="modalHeader">
          <h2 class="modalTitle">Confirm {{ planAction }}</h2>
          <button class="modalClose" @click="closeModal">×</button>
        </div>
        
        <div class="modalBody">
          <p v-if="planAction === 'upgrade'">
            You're about to upgrade to <strong>{{ pendingPlan?.name }}</strong> plan ({{ pendingPlan?.price }}/{{ pendingPlan?.priceDetail }}).
          </p>
          <p v-else-if="planAction === 'downgrade'">
            You're about to downgrade to <strong>{{ pendingPlan?.name }}</strong> plan ({{ pendingPlan?.price }}/{{ pendingPlan?.priceDetail }}).
          </p>
          
          <div class="planComparison">
            <div class="comparisonItem">
              <span class="label">Active QR codes:</span>
              <span class="value">
                {{ currentPlan?.maxActive }} → {{ pendingPlan?.maxActive }}
              </span>
            </div>
            <div class="comparisonItem">
              <span class="label">Total QR codes:</span>
              <span class="value">
                {{ currentPlan?.maxTotal }} → {{ pendingPlan?.maxTotal }}
              </span>
            </div>
          </div>
          
          <p v-if="planAction === 'upgrade'" class="modalNote">
            You'll be redirected to complete your payment.
          </p>
          <p v-else-if="planAction === 'downgrade'" class="modalNote">
            You'll be redirected to the billing portal to manage your subscription. Changes will take effect at the end of your current billing period.
          </p>
        </div>
        
        <div class="modalFooter">
          <button class="button secondary" @click="closeModal" :disabled="actionLoading">Cancel</button>
          <button 
            class="button primary" 
            @click="confirmPlanChange" 
            :disabled="actionLoading"
          >
            {{ actionLoading ? 'Processing...' : `Confirm ${planAction}` }}
          </button>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped src="../HomePage/HomePage.scss" lang="scss"></style>
<style scoped src="./SubscriptionPage.scss" lang="scss"></style>
