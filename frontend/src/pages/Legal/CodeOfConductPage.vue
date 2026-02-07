<script setup lang="ts"></script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">Code of Conduct</h1>
      <p class="subtitle">Community Guidelines for QR-Dragonfly Users</p>
    </header>

    <section class="card legal">
      <h2>Our Commitment</h2>
      <p>
        QR-Dragonfly is committed to providing a safe, respectful, and professional environment for all users. 
        This Code of Conduct outlines our expectations for user behavior and the consequences for unacceptable conduct.
      </p>

      <h2>Expected Behavior</h2>
      <p>All users are expected to:</p>
      <ul>
        <li>Use the Service in a lawful and ethical manner</li>
        <li>Create QR codes that direct to legitimate, legal content</li>
        <li>Respect the privacy and data of others</li>
        <li>Provide accurate information when creating an account</li>
        <li>Comply with all applicable laws and regulations</li>
        <li>Report suspicious or abusive behavior to our team</li>
      </ul>

      <h2>Prohibited Conduct</h2>
      <p>The following behaviors are strictly prohibited:</p>
      
      <h3>Illegal Activities</h3>
      <ul>
        <li>Using QR codes to facilitate illegal activities</li>
        <li>Distributing malware, viruses, or other harmful software</li>
        <li>Engaging in fraud, scams, or deceptive practices</li>
        <li>Violating intellectual property rights</li>
      </ul>

      <h3>Harmful Content</h3>
      <ul>
        <li>Directing to content that promotes violence or hatred</li>
        <li>Sharing sexually explicit or inappropriate material</li>
        <li>Distributing content that harasses, threatens, or bullies others</li>
        <li>Publishing false or misleading information</li>
      </ul>

      <h3>Platform Abuse</h3>
      <ul>
        <li>Attempting to circumvent subscription limits or quotas</li>
        <li>Creating multiple accounts to evade restrictions</li>
        <li>Interfering with or disrupting the Service</li>
        <li>Attempting unauthorized access to accounts or systems</li>
        <li>Using the Service for spam or unsolicited marketing</li>
      </ul>

      <h3>Phishing and Deception</h3>
      <ul>
        <li>Creating QR codes that impersonate legitimate entities</li>
        <li>Using QR codes to harvest personal information without consent</li>
        <li>Redirecting users to misleading or deceptive websites</li>
      </ul>

      <h2>Consequences</h2>
      <p>Violations of this Code of Conduct may result in:</p>
      <ul>
        <li><strong>Warning:</strong> A formal notice of the violation</li>
        <li><strong>Content Removal:</strong> Deletion of offending QR codes</li>
        <li><strong>Account Suspension:</strong> Temporary loss of access to the Service</li>
        <li><strong>Account Termination:</strong> Permanent ban from the Service</li>
        <li><strong>Legal Action:</strong> Reporting to law enforcement if applicable</li>
      </ul>

      <h2>Reporting Violations</h2>
      <p>
        If you encounter content or behavior that violates this Code of Conduct, please report it immediately. 
        You can contact our support team through your account dashboard. All reports will be reviewed promptly 
        and handled with confidentiality.
      </p>

      <h2>Content Moderation</h2>
      <p>
        We reserve the right to review, monitor, and remove any QR codes or content that violates this Code of Conduct 
        or our Terms of Service. While we strive to maintain a safe platform, we do not pre-screen all content and 
        are not responsible for user-generated content.
      </p>

      <h2>Privacy and Data Protection</h2>
      <p>
        When using QR codes that collect user data (through analytics or redirects), you must:
      </p>
      <ul>
        <li>Comply with applicable data protection laws (GDPR, CCPA, etc.)</li>
        <li>Provide appropriate privacy notices to end users</li>
        <li>Obtain necessary consent for data collection</li>
        <li>Use collected data responsibly and ethically</li>
      </ul>

      <h2>Commercial Use</h2>
      <p>
        If you use QR-Dragonfly for commercial purposes, you must:
      </p>
      <ul>
        <li>Comply with advertising and marketing regulations</li>
        <li>Clearly identify sponsored or promotional content</li>
        <li>Honor opt-out requests and unsubscribe mechanisms</li>
        <li>Maintain appropriate subscription tier for your usage volume</li>
      </ul>

      <h2>Updates to Code of Conduct</h2>
      <p>
        We may update this Code of Conduct from time to time to reflect changes in our community standards or legal 
        requirements. Significant changes will be communicated to users via email or through the Service.
      </p>

      <h2>Questions</h2>
      <p>
        If you have questions about this Code of Conduct or need clarification on what constitutes acceptable use, 
        please contact our support team.
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

.legal h3 {
  margin-top: 1.5rem;
  margin-bottom: 0.75rem;
  font-size: 1.1rem;
  font-weight: 600;
  opacity: 0.9;
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
