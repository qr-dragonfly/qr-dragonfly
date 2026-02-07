import { createRouter, createWebHistory } from 'vue-router'

import HomePage from '../pages/HomePage/HomePage.vue'
import RegisterPage from '../pages/Auth/RegisterPage.vue'
import ConfirmPage from '../pages/Auth/ConfirmPage.vue'
import LoginPage from '../pages/Auth/LoginPage.vue'
import AccountPage from '../pages/Auth/AccountPage.vue'
import ChangePasswordPage from '../pages/Auth/ChangePasswordPage.vue'
import ForgotPasswordPage from '../pages/Auth/ForgotPasswordPage.vue'
import ResetPasswordPage from '../pages/Auth/ResetPasswordPage.vue'

import TermsOfServicePage from '../pages/Legal/TermsOfServicePage.vue'
import PrivacyPolicyPage from '../pages/Legal/PrivacyPolicyPage.vue'
import CookiePolicyPage from '../pages/Legal/CookiePolicyPage.vue'
import CodeOfConductPage from '../pages/Legal/CodeOfConductPage.vue'

import QrCodeStatsPage from '../pages/QrCodeStats/QrCodeStatsPage.vue'
import SubscriptionPage from '../pages/Subscription/SubscriptionPage.vue'
import StripeCheckoutPage from '../pages/Subscription/StripeCheckoutPage.vue'
import AdminPage from '../pages/Admin/AdminPage.vue'
import ClientPage from '../pages/Client/ClientPage.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomePage },

    { path: '/register', name: 'register', component: RegisterPage },
    { path: '/confirm', name: 'confirm', component: ConfirmPage },
    { path: '/login', name: 'login', component: LoginPage },

    { path: '/account', name: 'account', component: AccountPage },
    { path: '/change-password', name: 'change-password', component: ChangePasswordPage },

    { path: '/forgot-password', name: 'forgot-password', component: ForgotPasswordPage },
    { path: '/reset-password', name: 'reset-password', component: ResetPasswordPage },

    { path: '/terms', name: 'terms', component: TermsOfServicePage },
    { path: '/privacy', name: 'privacy', component: PrivacyPolicyPage },
    { path: '/cookies', name: 'cookies', component: CookiePolicyPage },
    { path: '/code-of-conduct', name: 'code-of-conduct', component: CodeOfConductPage },

    { path: '/subscription', name: 'subscription', component: SubscriptionPage },
    { path: '/checkout', name: 'checkout', component: StripeCheckoutPage },
    { path: '/qr-codes/:id/stats', name: 'qr-code-stats', component: QrCodeStatsPage },
    
    { path: '/admin', name: 'admin', component: AdminPage },
    { path: '/client', name: 'client', component: ClientPage },

    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})
