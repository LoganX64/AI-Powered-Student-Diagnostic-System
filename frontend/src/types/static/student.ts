export interface StudentLoginFormText {
  cardTitle: string;
  cardDescription: string;
  codeLabel: string;
  codePlaceholder: string;
  submitButton: string;
  loadingText: string;
}

export interface StudentInstructionsText {
  title: string;
  subtitle: string;
  instructions: string[];
  examDurationLabel: string;
  examDurationValue: string;
  examDurationUnit: string;
  beginButton: string;
}

export interface StudentQuizText {
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

export interface StudentSubmittedText {
  title: string;
  description: string;
  redirectText: string;
  redirectNowButton: string;
}

export interface StudentPageText {
  login: StudentLoginFormText;
  instructions: StudentInstructionsText;
  quiz: StudentQuizText;
  submitted: StudentSubmittedText;
}

export const studentPageText: StudentPageText = {
  login: {
    cardTitle: "Login to your account",
    cardDescription:
      "Enter your Student code below to login to your account",
    codeLabel: "Student Code",
    codePlaceholder: "Enter your student code",
    submitButton: "Login",
    loadingText: "Logging in...",
  },
  instructions: {
    title: "Exam Instructions",
    subtitle: "Please read all instructions carefully before beginning the test.",
    instructions: [
      "Read each question carefully before selecting your answer.",
      "Each question carries fixed marks. There may be negative marking for wrong answers.",
      "Do not refresh the page or navigate away during the test — your progress may be lost.",
      "Use the question navigator on the right side of the quiz to jump between questions.",
      "You can mark a question for review and return to it later.",
      "Once you click Accept and Begin, the timer will start and cannot be paused.",
      "Submit your answers before the timer reaches zero. The test will auto-submit on time expiry.",
      "Ensure a stable internet connection throughout the test.",
    ],
    examDurationLabel: "Exam duration:",
    examDurationValue: "1",
    examDurationUnit: "hours",
    beginButton: "Accept & Begin",
  },
  quiz: {
    questionLabel: "Question",
    ofLabel: "of",
    diagramAltText: "Diagram for question",
    questionsSidebarTitle: "Questions",
    statusCurrent: "Current",
    statusAnswered: "Answered",
    statusNotAnswered: "Not answered",
    answeredCount: "answered",
    submitButton: "Submit",
    nextButton: "Next",
  },
  submitted: {
    title: "Your test has been submitted",
    description:
      "Thank you for completing the assessment. Your answers have been recorded successfully.",
    redirectText: "You will be redirected to the login page in",
    redirectNowButton: "Redirect Now",
  },
};
