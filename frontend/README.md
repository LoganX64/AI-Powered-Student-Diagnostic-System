# AI-Powered Student Diagnostic System - Frontend

This is the frontend of the AI-Powered Student Diagnostic System, built with **React 19**, **TypeScript**, **Vite**, and **Tailwind CSS**. It provides role-based dashboards for Admin, Coach, and Student users.

## 📂 Project Structure

```text
frontend/
├── index.html                    # SPA entry point
├── package.json                  # Dependencies and scripts
├── vite.config.ts                # Vite configuration
├── tsconfig.json                 # TypeScript configuration
├── components.json               # Shadcn/ui configuration
├── eslint.config.js              # ESLint rules (flat config)
├── .env                          # Environment variables (VITE_BACKEND_URL, VITE_PORT)
├── public/
│   ├── favicon.svg               # Favicon icon
│   └── icons.svg                 # Icon sprite
├── routes/
│   └── routes.ts                 # All route definitions (createBrowserRouter)
└── src/
    ├── main.tsx                  # Application entry point
    ├── index.css                 # Global styles (Tailwind + Shadcn theme)
    │
    ├── components/
    │   ├── ProtectedRoute.tsx    # Auth guard (admin/coach/student role detection)
    │   ├── ui/                   # Shadcn/ui core components (28 files)
    │   │   ├── button.tsx
    │   │   ├── card.tsx
    │   │   ├── dialog.tsx
    │   │   ├── table.tsx
    │   │   └── ...               # alert-dialog, avatar, badge, breadcrumb, chart,
    │   │                         # checkbox, drawer, dropdown-menu, field, input,
    │   │                         # label, progress, select, separator, sheet, sidebar,
    │   │                         # skeleton, sonner, switch, tabs, textarea, toggle,
    │   │                         # toggle-group, toggle-variants, tooltip
    │   ├── admin/                # Admin-specific form components
    │   │   ├── signup-form.tsx
    │   │   ├── QuestionCard.tsx
    │   │   └── forms/
    │   │       ├── CreateAssignmentForm.tsx
    │   │       ├── CreateCoachForm.tsx
    │   │       ├── CreateQuestionsForm.tsx
    │   │       ├── CreateStudentForm.tsx
    │   │       ├── CreateSubjectForm.tsx
    │   │       ├── CreateTestForm.tsx
    │   │       ├── EditTestDialog.tsx
    │   │       └── QuestionFormFields.tsx
    │   ├── shared/               # Shared dashboard/layout components
    │   │   ├── DashboardChart.tsx
    │   │   ├── DashboardHeader.tsx
    │   │   ├── DashboardLayout.tsx
    │   │   ├── DashboardSectionCards.tsx
    │   │   ├── DashboardSidebar.tsx
    │   │   ├── DashboardTable.tsx
    │   │   ├── login-page.tsx
    │   │   ├── nav-main.tsx
    │   │   ├── nav-secondary.tsx
    │   │   └── nav-user.tsx
    │   └── student/              # Student-specific components
    │       ├── exam-header.tsx
    │       └── student-login-form.tsx
    │
    ├── contexts/
    │   └── RoleContext.tsx        # React context for role ("admin" | "coach")
    │
    ├── features/
    │   ├── landing/
    │   │   ├── LandingPage.tsx   # Marketing landing page
    │   │   └── AboutPage.tsx     # About page
    │   ├── admin/
    │   │   ├── AdminSigninPage.tsx
    │   │   └── AdminSignupPage.tsx
    │   ├── coach/
    │   │   └── CoachSigninPage.tsx
    │   ├── student/
    │   │   ├── StudentLoginPage.tsx
    │   │   ├── StudentInstructionsPage.tsx
    │   │   ├── StudentQuizPage.tsx
    │   │   └── StudentSubmittedPage.tsx
    │   └── shared/               # Dashboard pages (used by both admin & coach)
    │       ├── AccountsPage.tsx
    │       ├── AllTestsPage.tsx
    │       ├── BillingPage.tsx
    │       ├── CoachDetailPage.tsx
    │       ├── CoachesPage.tsx
    │       ├── DashboardPage.tsx
    │       ├── GetHelpPage.tsx
    │       ├── NotificationsPage.tsx
    │       ├── QuestionsPage.tsx
    │       ├── SettingsPage.tsx
    │       ├── StudentDetailPage.tsx
    │       ├── StudentsPage.tsx
    │       ├── StudentSQIPage.tsx
    │       ├── SubjectsPage.tsx
    │       ├── TestDetailPage.tsx
    │       ├── TestsPage.tsx
    │       └── mockData.ts
    │
    ├── hooks/
    │   ├── use-mobile.ts         # useIsMobile() - responsive breakpoint hook
    │   ├── useExamTimer.ts       # useExamTimer() - countdown timer with sessionStorage
    │   └── useRole.ts            # useRole() - derives role from URL path
    │
    ├── lib/
    │   ├── api.ts                # apiFetch<T>() - shared fetch wrapper with JWT auth
    │   └── utils.ts              # cn() (tailwind-merge), formatDateDDMMYYYY()
    │
    ├── services/
    │   ├── auth.service.ts       # login(), register() - admin auth endpoints
    │   ├── admin.service.ts      # Full admin CRUD
    │   ├── coach.service.ts      # Coach-scoped CRUD (mirrors admin under /coach)
    │   ├── student.service.ts    # loginStudent() - student login
    │   └── dashboard.service.ts  # Role-aware service (auto-selects /admin or /coach)
    │
    └── types/
        ├── student.ts            # Placeholder
        └── static/               # Typed UI text for all pages (i18n-ready)
            ├── index.ts
            ├── about.ts
            ├── admin.ts
            ├── auth.ts
            ├── coach.ts
            ├── landing.ts
            └── student.ts
```

## 🛠️ Tech Stack

| Technology               | Purpose                                    |
| ------------------------ | ------------------------------------------ |
| **React 19**             | UI framework                               |
| **TypeScript**           | Type-safe JavaScript                       |
| **Vite**                 | Fast build tool and dev server             |
| **Tailwind CSS 4**       | Utility-first styling                      |
| **Shadcn/ui**            | Accessible component primitives (Radix UI) |
| **React Router 7**       | Client-side routing                        |
| **TanStack React Table** | Table state and rendering                  |
| **Recharts**             | Chart components                           |
| **Zod**                  | Schema validation                          |
| **Lucide React**         | Icon library                               |
| **Sonner**               | Toast notifications                        |
| **Vaul**                 | Drawer component                           |

## 🚀 Getting Started

### Prerequisites

- Node.js (v18 or higher)
- npm, yarn, or pnpm

### Installation

1. Navigate to the frontend directory:

   ```bash
   cd frontend
   ```

2. Install dependencies:

   Recommended (pnpm):

   ```bash
   pnpm install
   ```

   Or using npm:

   ```bash
   npm install
   ```

3. Configure environment variables (`.env`):

   ```env
   VITE_BACKEND_URL=http://localhost:8080
   VITE_PORT=5173
   ```

4. Start the development server:

   ```bash
   npm run dev
   ```

5. Build for production:

   ```bash
   npm run build
   ```

### Quick Start (development)

From repository root — copy `.env`, install, and run:

```bash
# frontend
cd frontend
cp .env.example .env
pnpm install    # or `npm install`
pnpm dev        # or `npm run dev`

# backend (see ../backend)
cd ../backend
cp .env.example .env
go mod download
go run ./cmd/api/main.go
```

Notes:

- `pnpm` is recommended for faster installs and deterministic lockfiles (repo includes `pnpm-lock.yaml`).
- Ensure `VITE_BACKEND_URL` in `frontend/.env` points to your running backend (default: `http://localhost:8080`).

## 🔐 Authentication & Roles

The application supports three user roles with separate login flows:

| Role        | Login Page       | Token Storage                                | Dashboard       |
| ----------- | ---------------- | -------------------------------------------- | --------------- |
| **Admin**   | `/admin-signin`  | `admin_token` + `admin_role` in localStorage | `/admin/*`      |
| **Coach**   | `/coach-signin`  | `admin_token` + `admin_role` in localStorage | `/coach/*`      |
| **Student** | `/student-login` | `student_token` in localStorage              | N/A (exam flow) |

- **ProtectedRoute** component guards all dashboard routes, redirecting unauthenticated users to the correct login page based on URL path.
- Admin and Coach dashboards share the same page components via `dashboard.service.ts`, which dynamically switches the API prefix based on role.

## 📡 Pages & Routes

### Public Routes

| Path     | Page         |
| -------- | ------------ |
| `/`      | Landing Page |
| `/about` | About Page   |

### Auth Routes

| Path             | Page               |
| ---------------- | ------------------ |
| `/student-login` | Student Login      |
| `/admin-signin`  | Admin Sign-in      |
| `/admin-signup`  | Admin Registration |
| `/coach-signin`  | Coach Sign-in      |

### Admin Dashboard (`/admin/*`)

| Path                         | Page                             |
| ---------------------------- | -------------------------------- |
| `/admin/dashboard`           | Dashboard (charts, stats, table) |
| `/admin/coaches`             | Coaches List                     |
| `/admin/coaches/:id`         | Coach Detail                     |
| `/admin/students`            | Students List                    |
| `/admin/students/:id`        | Student Detail                   |
| `/admin/students/:id/sqi`    | Student SQI Analysis             |
| `/admin/subjects`            | Subjects List                    |
| `/admin/tests`               | Tests List                       |
| `/admin/all-tests`           | All Tests (cross-coach)          |
| `/admin/tests/:id`           | Test Detail                      |
| `/admin/tests/:id/questions` | Questions Management             |
| `/admin/settings`            | Settings                         |
| `/admin/help`                | Help                             |
| `/admin/accounts`            | Accounts                         |
| `/admin/billing`             | Billing                          |
| `/admin/notifications`       | Notifications                    |

### Coach Dashboard (`/coach/*`)

| Path                         | Page                 |
| ---------------------------- | -------------------- |
| `/coach/dashboard`           | Dashboard            |
| `/coach/students`            | Students List        |
| `/coach/students/:id`        | Student Detail       |
| `/coach/students/:id/sqi`    | Student SQI Analysis |
| `/coach/subjects`            | Subjects List        |
| `/coach/tests`               | Tests List           |
| `/coach/all-tests`           | All Tests            |
| `/coach/tests/:id`           | Test Detail          |
| `/coach/tests/:id/questions` | Questions Management |
| `/coach/settings`            | Settings             |
| `/coach/help`                | Help                 |
| `/coach/accounts`            | Accounts             |
| `/coach/notifications`       | Notifications        |

### Student Flow

| Path            | Page                             |
| --------------- | -------------------------------- |
| `/instructions` | Pre-exam Instructions            |
| `/quiz`         | Quiz/Exam (with countdown timer) |
| `/submitted`    | Post-submission Confirmation     |

## 🧩 Architecture Highlights

- **Shared Dashboard Pages**: Admin and Coach dashboards reuse the same page components from `features/shared/`. The `dashboard.service.ts` dynamically switches API prefixes based on role.
- **Static Text / i18n System**: All UI strings are centralized in `types/static/` with typed interfaces, ready for future internationalization.
- **Exam Timer**: `useExamTimer()` hook manages countdown with `sessionStorage` persistence, supporting delayed start and expiry callbacks.
- **Role Detection**: `useRole()` hook derives role from URL path; `RoleContext` provides a React context alternative.

## 📦 Build Scripts

```bash
npm run dev        # Start development server (http://localhost:5173)
npm run build      # Build for production (TypeScript + Vite)
npm run lint       # Run ESLint checks
npm run preview    # Preview production build locally
```
