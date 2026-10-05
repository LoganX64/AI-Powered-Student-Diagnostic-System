interface LegalSection {
  title: string;
  body: string;
}

interface LegalPageText {
  title: string;
  lastUpdated: string;
  sections: LegalSection[];
}

export const termsPageText: LegalPageText = {
  title: "Terms of Service",
  lastUpdated: "October 5, 2026",
  sections: [
    {
      title: "1. Acceptance of Terms",
      body: "By accessing or using EduQuant, you agree to be bound by these Terms of Service. If you do not agree, do not use the platform.",
    },
    {
      title: "2. Use of Service",
      body: "EduQuant provides online assessments, proctoring, and analytics for educational institutions. You may use the service only for its intended educational purposes and in compliance with applicable laws.",
    },
    {
      title: "3. Exam Integrity & Proctoring",
      body: "Proctoring features, including webcam capture and live monitoring, are used solely for the purpose of exam integrity. Recordings are processed and stored to detect and prevent academic misconduct.",
    },
    {
      title: "4. Accounts",
      body: "You are responsible for safeguarding your account credentials and for all activity that occurs under your account. Notify your administrator immediately of any unauthorized use.",
    },
    {
      title: "5. Limitation of Liability",
      body: "The service is provided \"as is\" without warranties of any kind. EduQuant is not liable for indirect, incidental, or consequential damages arising from your use of the platform.",
    },
    {
      title: "6. Changes to These Terms",
      body: "We may update these terms from time to time. Continued use of the platform after changes take effect constitutes acceptance of the revised terms.",
    },
    {
      title: "7. Contact",
      body: "Questions about these terms can be directed to your organization's administrator or to the support team listed on the Get Help page.",
    },
  ],
};

export const privacyPageText: LegalPageText = {
  title: "Privacy Policy",
  lastUpdated: "October 5, 2026",
  sections: [
    {
      title: "1. Data We Collect",
      body: "We collect account details (name, email, role), exam activity (answers, timing, proctoring events), and limited technical telemetry needed to operate the service.",
    },
    {
      title: "2. How We Use It",
      body: "Your data is used to deliver assessments, compute diagnostic analytics, enforce exam integrity, and improve the platform. We do not sell personal data.",
    },
    {
      title: "3. Video Proctoring & Exam Data",
      body: "Exam recordings and live video are processed only for proctoring and review by authorized staff of your organization. Access is logged and scoped to the students and assignments you manage.",
    },
    {
      title: "4. Sharing",
      body: "Data is shared only within your organization (administrators, coaches, proctors) and with infrastructure providers required to run the service under contractual safeguards.",
    },
    {
      title: "5. Retention",
      body: "Exam records are retained as configured by your organization. You may request deletion of your data through your administrator, subject to legal retention requirements.",
    },
    {
      title: "6. Your Rights",
      body: "You may request access to, correction of, or deletion of your personal data, and you may withdraw from a proctored exam at any time by notifying your administrator.",
    },
    {
      title: "7. Contact",
      body: "Privacy questions can be directed to your organization's administrator or to the support team listed on the Get Help page.",
    },
  ],
};
