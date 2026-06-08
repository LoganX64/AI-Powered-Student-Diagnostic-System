export interface LandingHero {
  badge: string;
  headingLine1: string;
  headingLine2: string;
  description: string;
  ctaPrimary: string;
  ctaSecondary: string;
}

export interface LandingFeature {
  title: string;
  description: string;
}

export interface LandingFeatures {
  sectionTitle: string;
  sectionDescription: string;
  items: LandingFeature[];
}

export interface LandingCta {
  heading: string;
  description: string;
  buttonText: string;
}

export interface LandingPageText {
  brand: string;
  nav: {
    features: string;
    about: string;
    studentLogin: string;
    registerNow: string;
  };
  hero: LandingHero;
  features: LandingFeatures;
  cta: LandingCta;
  footer: {
    copyright: string;
    about: string;
    privacy: string;
    terms: string;
  };
}

export const landingPageText: LandingPageText = {
  brand: "EduQuant",
  nav: {
    features: "Features",
    about: "About",
    studentLogin: "Student Login",
    registerNow: "Register Now",
  },
  hero: {
    badge: "AI-Powered Student Diagnostics",
    headingLine1: "Understand every student.",
    headingLine2: "Improve every outcome.",
    description:
      "A comprehensive system that provides deep performance analysis, concept-level insights, and AI-assisted improvement guidance for coaching institutes and students. Go beyond marks — understand the learning gaps.",
    ctaPrimary: "Register Now",
    ctaSecondary: "Student Login",
  },
  features: {
    sectionTitle: "Everything you need",
    sectionDescription:
      "Powerful tools to diagnose, track, and improve student performance.",
    items: [
      {
        title: "Deep Performance Analysis",
        description:
          "Detailed breakdown of student performance across concepts and topics, going beyond raw marks and percentages.",
      },
      {
        title: "Concept-Level Insights",
        description:
          "Understand specific learning gaps and strengths at the concept level for targeted interventions.",
      },
      {
        title: "Prioritized Recommendations",
        description:
          "AI-assisted suggestions that prioritize the most impactful areas for student improvement.",
      },
      {
        title: "Multi-Role Support",
        description:
          "Dedicated interfaces for administrators, coaches, and students with role-appropriate tools.",
      },
      {
        title: "Student Quality Index",
        description:
          "Custom SQI metric that provides a comprehensive view of student performance across all dimensions.",
      },
      {
        title: "AI-Powered Diagnostics",
        description:
          "Intelligent analysis engine that processes test results to generate actionable diagnostic reports.",
      },
    ],
  },
  cta: {
    heading: "Ready to transform how you track student progress?",
    description:
      "Create your organization account and start diagnosing student performance with AI-powered insights today.",
    buttonText: "Get Started — It's Free",
  },
  footer: {
    copyright: "© 2026 EduQuant. All rights reserved.",
    about: "About",
    privacy: "Privacy",
    terms: "Terms",
  },
};
