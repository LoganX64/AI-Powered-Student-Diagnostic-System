interface StudentLoginFormText {
  cardTitle: string;
  cardDescription: string;
  codeLabel: string;
  codePlaceholder: string;
  submitButton: string;
  loadingText: string;
}

interface StudentInstructionsText {
  title: string;
  subtitle: string;
  instructions: string[];
  examDurationLabel: string;
  examDurationValue: string;
  examDurationUnit: string;
  beginButton: string;
}

interface StudentQuizText {
  questionLabel: string;
  ofLabel: string;
  diagramAltText: string;
  questionsSidebarTitle: string;
  statusCurrent: string;
  statusAnswered: string;
  statusNotAnswered: string;
  answeredCount: string;
  submitButton: string;
  nextButton: string;
}

interface StudentSubmittedText {
  title: string;
  description: string;
  redirectText: string;
  redirectNowButton: string;
}

interface StudentPageText {
  login: StudentLoginFormText;
  instructions: StudentInstructionsText;
  quiz: StudentQuizText;
  submitted: StudentSubmittedText;
}


