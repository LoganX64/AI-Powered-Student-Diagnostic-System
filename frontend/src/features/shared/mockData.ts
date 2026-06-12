// ─── Settings Page Mock Data ──────────────────────────────────────────────────

export const mockUserProfile = {
  name: "Admin User",
  email: "admin@example.com",
  phone: "+1 (555) 123-4567",
  avatar: "/avatars/shadcn.jpg",
  role: "admin",
  joinedAt: "2024-01-15",
  lastActive: "2024-06-10",
};

export const mockNotificationPreferences = {
  emailNotifications: true,
  pushNotifications: true,
  weeklyDigest: false,
  testAlerts: true,
  studentActivity: true,
};

// ─── Get Help Page Mock Data ──────────────────────────────────────────────────

export const mockFaqs = [
  {
    id: 1,
    question: "How do I create a new test?",
    answer: "Navigate to 'Create Test' in the sidebar. Fill in the test details including title, subject, duration, and exam date. Then add questions and assign students.",
  },
  {
    id: 2,
    question: "How do I add new students?",
    answer: "Go to the Students page and click 'Create Student'. Enter the student's name and unique code. Assign them to a coach if needed.",
  },
  {
    id: 3,
    question: "What is SQI (Student Quality Index)?",
    answer: "SQI is a composite score measuring student performance across all assessments. It considers test scores, completion rates, and improvement trends.",
  },
  {
    id: 4,
    question: "How do I view student performance?",
    answer: "Click on any student in the Students list to view their detailed profile, including test history, SQI scores, and assignment status.",
  },
  {
    id: 5,
    question: "Can I edit a test after creating it?",
    answer: "Yes, admins can edit test details and questions. Navigate to All Tests, find the test, and click the edit button.",
  },
];

// ─── Accounts Page Mock Data ──────────────────────────────────────────────────

export const mockAccountInfo = {
  id: 1,
  name: "Admin User",
  email: "admin@example.com",
  role: "admin",
  status: "active",
  joinedAt: "2024-01-15",
  lastLogin: "2024-06-10T14:30:00Z",
  emailVerified: true,
  twoFactorEnabled: false,
};

export const mockApiKeys = [
  {
    id: "key_1",
    name: "Production API Key",
    prefix: "sk_prod_***",
    createdAt: "2024-02-01",
    lastUsed: "2024-06-10",
    status: "active",
  },
  {
    id: "key_2",
    name: "Development API Key",
    prefix: "sk_dev_***",
    createdAt: "2024-03-15",
    lastUsed: "2024-06-08",
    status: "active",
  },
];

export const mockSessions = [
  {
    id: "sess_1",
    device: "Chrome on Windows",
    ip: "192.168.1.100",
    lastActive: "2024-06-10T14:30:00Z",
    current: true,
  },
  {
    id: "sess_2",
    device: "Safari on iPhone",
    ip: "10.0.0.50",
    lastActive: "2024-06-09T09:15:00Z",
    current: false,
  },
];

// ─── Billing Page Mock Data (Admin-only) ─────────────────────────────────────

export const mockBillingPlan = {
  name: "Professional",
  price: 49,
  billingCycle: "monthly",
  nextBillingDate: "2024-07-15",
  features: [
    "Unlimited students",
    "Unlimited tests",
    "50 coaches",
    "Advanced analytics",
    "Priority support",
  ],
};

export const mockUsage = {
  students: { used: 245, limit: 500, percentage: 49 },
  coaches: { used: 12, limit: 50, percentage: 24 },
  storage: { used: 2.4, limit: 10, unit: "GB", percentage: 24 },
  tests: { used: 89, limit: -1, percentage: 0 },
};

export const mockBillingHistory = [
  {
    id: "inv_1",
    date: "2024-06-01",
    amount: 49,
    status: "paid",
    description: "Monthly subscription - June 2024",
  },
  {
    id: "inv_2",
    date: "2024-05-01",
    amount: 49,
    status: "paid",
    description: "Monthly subscription - May 2024",
  },
  {
    id: "inv_3",
    date: "2024-04-01",
    amount: 49,
    status: "paid",
    description: "Monthly subscription - April 2024",
  },
  {
    id: "inv_4",
    date: "2024-03-01",
    amount: 39,
    status: "paid",
    description: "Monthly subscription - March 2024",
  },
];

// ─── Notifications Page Mock Data ─────────────────────────────────────────────

export type Notification = {
  id: string;
  title: string;
  message: string;
  type: "info" | "warning" | "success" | "alert";
  read: boolean;
  createdAt: string;
};

export const mockNotifications: Notification[] = [
  {
    id: "notif_1",
    title: "New Student Registered",
    message: "John Smith has been registered and assigned to Coach Sarah.",
    type: "info",
    read: false,
    createdAt: "2024-06-10T10:30:00Z",
  },
  {
    id: "notif_2",
    title: "Test Submission Alert",
    message: "15 students have not yet submitted the Mathematics Mid-Term.",
    type: "warning",
    read: false,
    createdAt: "2024-06-10T09:15:00Z",
  },
  {
    id: "notif_3",
    title: "Test Completed",
    message: "Physics Chapter 5 Test has been completed by all assigned students.",
    type: "success",
    read: true,
    createdAt: "2024-06-09T16:45:00Z",
  },
  {
    id: "notif_4",
    title: "At-Risk Student Detected",
    message: "3 students have scored below 40% in their last 3 assessments.",
    type: "alert",
    read: false,
    createdAt: "2024-06-09T14:20:00Z",
  },
  {
    id: "notif_5",
    title: "System Maintenance Scheduled",
    message: "Scheduled maintenance on June 15th from 2:00 AM to 4:00 AM UTC.",
    type: "info",
    read: true,
    createdAt: "2024-06-08T08:00:00Z",
  },
  {
    id: "notif_6",
    title: "New Coach Added",
    message: "Coach Michael Brown has been added to the platform.",
    type: "success",
    read: true,
    createdAt: "2024-06-07T11:30:00Z",
  },
  {
    id: "notif_7",
    title: "Low Completion Rate",
    message: "Only 60% of students completed the Chemistry Quiz.",
    type: "warning",
    read: true,
    createdAt: "2024-06-06T15:00:00Z",
  },
];
