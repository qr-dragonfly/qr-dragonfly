<script setup lang="ts"></script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">Terms of Service</h1>
      <p class="subtitle">Last updated: January 27, 2026</p>
    </header>

    <section class="card legal">
      <h2>1. Acceptance of Terms</h2>
      <p>
        By accessing or using QR-Dragonfly ("Service"), you agree to be bound by these Terms of Service ("Terms"). 
        If you do not agree to these Terms, you may not use the Service.
      </p>

      <h2>2. Description of Service</h2>
      <p>
        QR-Dragonfly is a QR code management platform that allows users to create, manage, and track QR codes that 
        redirect to URLs. The Service includes analytics and tracking capabilities for QR code usage.
      </p>

      <h2>3. Account Registration</h2>
      <p>
        To use the Service, you must create an account by providing a valid email address and password. You are 
        responsible for maintaining the confidentiality of your account credentials and for all activities that 
        occur under your account.
      </p>

      <h2>4. Subscription Plans</h2>
      <p>The Service offers the following subscription tiers:</p>
      <ul>
        <li><strong>Free:</strong> Up to 5 active QR codes, 20 total QR codes</li>
        <li><strong>Basic:</strong> Up to 50 active QR codes, 200 total QR codes (paid subscription)</li>
        <li><strong>Enterprise:</strong> Up to 2,000 active QR codes, 10,000 total QR codes (paid subscription)</li>
      </ul>
      <p>
        Active QR codes are those currently enabled and redirecting. Inactive QR codes count toward your total but not your active limit.
      </p>

      <h2>5. Payment and Billing</h2>
      <p>
        Paid subscriptions are billed on a recurring basis through Stripe. By subscribing to a paid plan, you authorize 
        us to charge your payment method on a recurring basis until you cancel. You may cancel your subscription at any 
        time through the Customer Portal. Cancellations take effect at the end of the current billing period.
      </p>
      <p>
        We reserve the right to change our pricing at any time. Price changes will be communicated in advance and will 
        take effect at the start of your next billing cycle.
      </p>

      <h2>6. Acceptable Use</h2>
      <p>You agree not to use the Service to:</p>
      <ul>
        <li>Violate any local, state, national, or international law</li>
        <li>Distribute malware, viruses, or other harmful code</li>
        <li>Engage in phishing, spam, or fraudulent activities</li>
        <li>Redirect to content that is illegal, harmful, threatening, abusive, harassing, defamatory, or obscene</li>
        <li>Infringe upon intellectual property rights of others</li>
        <li>Impersonate any person or entity</li>
        <li>Attempt to gain unauthorized access to the Service or other users' accounts</li>
      </ul>

      <h2>7. Data and Privacy</h2>
      <p>
        We collect and process data in accordance with our Privacy Policy. This includes your account information, 
        QR code data, and analytics about QR code usage (including IP addresses, user agents, and geographic location).
      </p>
      <p>
        You retain ownership of the content you create using the Service. By using the Service, you grant us a 
        license to store, process, and display your content as necessary to provide the Service.
      </p>

      <h2>8. Service Availability</h2>
      <p>
        We strive to maintain high availability but do not guarantee uninterrupted access to the Service. We may 
        modify, suspend, or discontinue the Service at any time without notice. We are not liable for any 
        modification, suspension, or discontinuation of the Service.
      </p>

      <h2>9. Termination</h2>
      <p>
        We may terminate or suspend your account immediately, without prior notice, for conduct that we believe 
        violates these Terms or is harmful to the Service, other users, or third parties. Upon termination, your 
        right to use the Service will immediately cease.
      </p>

      <h2>10. Limitation of Liability</h2>
      <p>
        TO THE MAXIMUM EXTENT PERMITTED BY LAW, QR-DRAGONFLY SHALL NOT BE LIABLE FOR ANY INDIRECT, INCIDENTAL, 
        SPECIAL, CONSEQUENTIAL, OR PUNITIVE DAMAGES, OR ANY LOSS OF PROFITS OR REVENUES, WHETHER INCURRED DIRECTLY 
        OR INDIRECTLY, OR ANY LOSS OF DATA, USE, GOODWILL, OR OTHER INTANGIBLE LOSSES.
      </p>

      <h2>11. Disclaimer of Warranties</h2>
      <p>
        THE SERVICE IS PROVIDED "AS IS" AND "AS AVAILABLE" WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, 
        INCLUDING BUT NOT LIMITED TO IMPLIED WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND 
        NON-INFRINGEMENT.
      </p>

      <h2>12. Indemnification</h2>
      <p>
        You agree to indemnify and hold harmless QR-Dragonfly and its officers, directors, employees, and agents from 
        any claims, damages, losses, liabilities, and expenses arising out of your use of the Service or violation of 
        these Terms.
      </p>

      <h2>13. Changes to Terms</h2>
      <p>
        We reserve the right to modify these Terms at any time. We will notify users of material changes via email or 
        through the Service. Your continued use of the Service after such modifications constitutes acceptance of the 
        updated Terms.
      </p>

      <h2>14. Governing Law</h2>
      <p>
        These Terms shall be governed by and construed in accordance with the laws of the State of Minnesota, United States, 
        without regard to its conflict of law provisions.
      </p>

      <h2>15. Contact</h2>
      <p>
        If you have any questions about these Terms, please contact us through the Service or at the email address 
        provided in your account dashboard.
      </p>
    </section>
  </main>
</template>

<style scoped src="../Auth/AuthPage.scss" lang="scss"></style>
<style scoped>
.legal {
  max-width: 800px;
  line-height: 1.6;
}

.legal h2 {
  margin-top: 2rem;
  margin-bottom: 1rem;
  font-size: 1.25rem;
  font-weight: 600;
}

.legal p {
  margin: 0.75rem 0;
}

.legal ul {
  margin: 0.75rem 0;
  padding-left: 2rem;
}

.legal li {
  margin: 0.5rem 0;
}
</style>
