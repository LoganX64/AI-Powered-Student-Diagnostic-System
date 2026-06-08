export interface AuthFormText {
  cardTitle: string;
  cardDescription: string;
  googleButton: string;
  separatorText: string;
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

export interface AdminSignupFormText {
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

export interface CoachLoginFormText extends AuthFormText {
  adminAccessText: string;
}

export interface ValidationErrors {
  emailRequired: string;
  passwordRequired: string;
  orgNameRequired: string;
  passwordMinLength: string;
  passwordMismatch: string;
  invalidCredentials: string;
}

export interface AuthPageText {
  adminLogin: AuthFormText;
  adminSignup: AdminSignupFormText;
  coachLogin: CoachLoginFormText;
  validation: ValidationErrors;
}

export const authPageText: AuthPageText = {
  adminLogin: {
    cardTitle: "Welcome back",
    cardDescription: "Login with your Google account",
    googleButton: "Login with Google",
    separatorText: "Or continue with",
    emailLabel: "Email",
    emailPlaceholder: "m@example.com",
    passwordLabel: "Password",
    forgotPassword: "Forgot your password?",
    submitButton: "Login",
    loadingText: "Logging in...",
    footerText: "Don't have an account?",
    footerLinkText: "Sign up",
    footerLinkHref: "/admin-signup",
    legalText: "By clicking continue, you agree to our",
    termsLink: "Terms of Service",
    privacyLink: "Privacy Policy",
  },
  adminSignup: {
    cardTitle: "Create your account",
    cardDescription: "Enter your details below to create your account",
    orgNameLabel: "Organization Name",
    orgNamePlaceholder: "Innovative Academy",
    emailLabel: "Email",
    emailPlaceholder: "m@example.com",
    passwordLabel: "Password",
    confirmPasswordLabel: "Confirm Password",
    passwordHint: "Must be at least 8 characters long.",
    submitButton: "Create Account",
    loadingText: "Creating account...",
    hasAccountText: "Already have an account?",
    signInLinkText: "Sign in",
    legalText: "By clicking continue, you agree to our",
    termsLink: "Terms of Service",
    privacyLink: "Privacy Policy",
  },
  coachLogin: {
    cardTitle: "Welcome back, Coach!",
    cardDescription: "Sign in to your coach account",
    googleButton: "Login with Google",
    separatorText: "Or continue with",
    emailLabel: "Email",
    emailPlaceholder: "coach@example.com",
    passwordLabel: "Password",
    forgotPassword: "Forgot your password?",
    submitButton: "Login",
    loadingText: "Logging in...",
    adminAccessText: "Contact your admin to get access",
    footerText: "",
    footerLinkText: "",
    footerLinkHref: "",
    legalText: "By clicking continue, you agree to our",
    termsLink: "Terms of Service",
    privacyLink: "Privacy Policy",
  },
  validation: {
    emailRequired: "Email is required",
    passwordRequired: "Password is required",
    orgNameRequired: "Organization name is required",
    passwordMinLength: "Password must be at least 8 characters long",
    passwordMismatch: "Passwords do not match",
    invalidCredentials: "Invalid email or password",
  },
};
