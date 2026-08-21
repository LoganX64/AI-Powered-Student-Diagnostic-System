interface AboutMission {
  title: string;
  description: string;
}

interface AboutWhatWeDoItem {
  title: string;
  description: string;
}

interface AboutContact {
  company: string;
  helpline: string;
  helplineEmail: string;
  address: string;
  addressFull: string;
}

interface AboutPageText {
  hero: {
    title: string;
    description: string;
  };
  mission: AboutMission;
  whatWeDo: {
    title: string;
    items: AboutWhatWeDoItem[];
  };
  contact: {
    title: string;
    company: AboutContact;
  };
  footer: {
    copyright: string;
    about: string;
    privacy: string;
    terms: string;
  };
}

export const aboutPageText: AboutPageText = {
  hero: {
    title: "About EduQuant",
    description:
      "A comprehensive AI-powered system that provides deep performance analysis, concept-level insights, and prioritized improvement recommendations for coaching institutes and students.",
  },
  mission: {
    title: "Our Mission",
    description:
      "We believe that understanding a student's learning journey goes far beyond raw marks and percentages. EduQuant is built to empower coaching institutes and educators with actionable, concept-level insights — helping them identify learning gaps early and guide every student toward their full potential.",
  },
  whatWeDo: {
    title: "What We Do",
    items: [
      {
        title: "For Institutes",
        description:
          "Manage coaches, students, subjects, and tests from a single dashboard. Track performance trends and make data-driven decisions.",
      },
      {
        title: "For Coaches",
        description:
          "Monitor student progress, create assignments, and receive AI-generated insights to personalize your teaching approach.",
      },
      {
        title: "For Students",
        description:
          "Take diagnostic tests, receive detailed performance reports, and get AI-assisted recommendations for focused improvement.",
      },
      {
        title: "For Parents",
        description:
          "Gain clear visibility into your child's strengths and weaknesses with easy-to-understand diagnostic summaries.",
      },
    ],
  },
  contact: {
    title: "Contact Us",
    company: {
      company: "Company",
      helpline: "Helpline",
      helplineEmail: "loganxtream@gmail.com",
      address: "Address",
      addressFull: "Ulhasnagar, Maharashtra 421004",
    },
  },
  footer: {
    copyright: "© 2026 Loganx64. All rights reserved.",
    about: "About",
    privacy: "Privacy",
    terms: "Terms",
  },
};
