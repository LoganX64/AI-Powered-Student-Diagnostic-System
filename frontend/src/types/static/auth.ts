interface AuthFormText {
  cardTitle: string;
  cardDescription: string;
  emailLabel: string;
  emailPlaceholder: string;
  passwordLabel: string;
  forgotPassword: string;
  submitButton: string;
  loadingText: string;
  footerText: string;
  footerLinkText: string;
  footerLinkHref: string;
  legalText: string;
  termsLink: string;
  privacyLink: string;
}

interface AdminSignupFormText {
  cardTitle: string;
  cardDescription: string;
  orgNameLabel: string;
  orgNamePlaceholder: string;
  emailLabel: string;
  emailPlaceholder: string;
  passwordLabel: string;
  confirmPasswordLabel: string;
  passwordHint: string;
  submitButton: string;
  loadingText: string;
  hasAccountText: string;
  signInLinkText: string;
  legalText: string;
  termsLink: string;
  privacyLink: string;
}

interface CoachLoginFormText extends AuthFormText {
  adminAccessText: string;
}

interface ValidationErrors {
  emailRequired: string;
  passwordRequired: string;
  orgNameRequired: string;
  passwordMinLength: string;
  passwordMismatch: string;
  invalidCredentials: string;
}

interface AuthPageText {
  adminLogin: AuthFormText;
  adminSignup: AdminSignupFormText;
  coachLogin: CoachLoginFormText;
  validation: ValidationErrors;
}


