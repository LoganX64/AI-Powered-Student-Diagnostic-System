# 🎓 AI-Powered Student Diagnostic System

A comprehensive system that provides deep performance analysis, concept-level insights, and AI-assisted improvement guidance for coaching institutes and students.

## 📋 Table of Contents

- [Project Overview](#project-overview)
- [System Architecture](#system-architecture)
- [Frontend Structure](#frontend-structure)
- [Backend Structure](#backend-structure)
- [Getting Started](#getting-started)

---

## Project Overview

The AI-Powered Student Diagnostic System moves beyond raw marks and percentages by providing:

- **Deep Performance Analysis**: Detailed breakdown of student performance across concepts and topics
- **Concept-Level Insights**: Understanding of specific learning gaps and strengths
- **Prioritized Learning Recommendations**: AI-assisted suggestions for improvement
- **Multi-Role Support**: Separate interfaces for students and administrators
- **Custom Metrics**: Student Quality Index (SQI) for comprehensive performance evaluation

---

## System Architecture

```
┌────────────────────┐
│   React Frontend   │
│ (Student + Admin)  │
└─────────┬──────────┘
          ↓
┌────────────────────┐
│    Go Backend      │
│────────────────────│
│ API Layer          │
│ SQI Engine         │
│ Insight Engine     │
│ AI Integration     │
└─────────┬──────────┘
          ↓
┌────────────────────┐
│    PostgreSQL      │
│────────────────────│
│ Students           │
│ Attempts           │
│ Tests              │
│ Results            │
└────────────────────┘
```

---

## Frontend Structure

The frontend is built with **React 19**, **TypeScript**, **Vite**, **Tailwind CSS**, and **Radix UI**.

### Project Root: `/frontend`

```
frontend/
├── index.html                    # Entry HTML file
├── package.json                  # Dependencies and scripts
├── vite.config.ts                # Vite configuration
├── tsconfig.json                 # TypeScript base configuration
├── tsconfig.app.json             # TypeScript app-specific config
├── tsconfig.node.json            # TypeScript node-specific config
├── eslint.config.js              # ESLint rules
├── components.json               # Component metadata
├── .env                          # Environment variables
├── .env.example                  # Environment template
├── routes/
│   └── routes.ts                 # Route definitions
├── src/
│   ├── main.tsx                  # Application entry point
│   ├── index.css                 # Base styles
│   ├── app/
│   │   └── dashboard/            # Dashboard application shell
│   ├── assets/                   # Static assets (images, fonts, etc.)
│   ├── components/               # Reusable UI components
│   │   ├── ProtectedRoute.tsx    # Route guard component
│   │   ├── admin/                # Admin-specific components
│   │   ├── coach/                # Coach-specific components
│   │   ├── student/              # Student-specific components
│   │   ├── shared/               # Shared components across roles
│   │   └── ui/                   # UI component library
│   │       ├── button.tsx
│   │       ├── card.tsx
│   │       ├── chart.tsx
│   │       ├── dialog.tsx
│   │       ├── dropdown-menu.tsx
│   │       ├── field.tsx
│   │       ├── input.tsx
│   │       ├── label.tsx
│   │       ├── progress.tsx
│   │       ├── select.tsx
│   │       ├── separator.tsx
│   │       ├── sidebar.tsx
│   │       ├── table.tsx
│   │       ├── tabs.tsx
│   │       ├── textarea.tsx
│   │       ├── tooltip.tsx
│   │       └── ... (29 components total)
│   ├── contexts/                 # React context providers
│   │   └── DashboardContext.tsx  # Dashboard state context
│   ├── features/                 # Feature-specific modules
│   │   ├── admin/                # Admin auth pages
│   │   │   ├── AdminSigninPage.tsx
│   │   │   └── AdminSignupPage.tsx
│   │   ├── coach/                # Coach auth pages
│   │   │   └── CoachSigninPage.tsx
│   │   ├── landing/              # Landing page features
│   │   ├── shared/               # Shared feature components
│   │   ├── sqi/                  # SQI (Student Quality Index) features
│   │   ├── student/              # Student portal features
│   │   └── test/                 # Test-related features
│   ├── hooks/                    # Custom React hooks
│   │   ├── use-mobile.ts        # Mobile detection hook
│   │   ├── useAnswerTracker.ts  # Answer tracking hook
│   │   ├── useExamTimer.ts      # Exam timer hook
│   │   └── useRole.ts           # Role detection hook
│   ├── lib/                      # Utility libraries
│   │   ├── api.ts               # API client configuration
│   │   ├── token.ts             # Token management
│   │   └── utils.ts             # Helper functions
│   ├── services/                 # API communication
│   │   ├── auth.service.ts      # Authentication API calls
│   │   ├── dashboard.service.ts # Dashboard API calls
│   │   ├── student.service.ts   # Student API calls
│   │   └── types.ts             # Service type definitions
│   ├── types/                    # TypeScript type definitions
│   │   └── static/              # Static type definitions
│   └── utils/                    # Utility functions (currently empty)
├── public/                       # Public static files
└── dist/                         # Build output (generated)
```

### Frontend Technologies

| Technology       | Purpose                         |
| ---------------- | ------------------------------- |
| **React 19**     | UI framework                    |
| **TypeScript**   | Type-safe JavaScript            |
| **Vite**         | Fast build tool and dev server  |
| **Tailwind CSS** | Utility-first styling           |
| **Radix UI**     | Accessible component primitives |
| **React Router** | Client-side routing             |
| **Lucide React** | Icon library                    |

### Frontend Build Scripts

```bash
npm run dev        # Start development server
npm run build      # Build for production (TypeScript + Vite)
npm run lint       # Run ESLint checks
npm run preview    # Preview production build locally
```

---

## Backend Structure

The backend is built with **Go 1.25.1**, **Gin framework**, and **PostgreSQL**.

### Project Root: `/backend`

```
backend/
├── go.mod                                    # Go module definition
├── go.sum                                    # Go module checksums
├── .env                                      # Environment variables
├── .env.example                              # Environment template
├── cmd/
│   ├── api/
│   │   └── main.go                           # Application entry point
│   ├── createsuperadmin/                      # CLI to create super admin
│   ├── createadmin/                           # CLI to create admin
│   ├── check_migrations/                      # Migration checker
│   └── resetdb/                               # DB reset utility
├── internal/                                 # Private application code
│   ├── auth/
│   │   └── auth.go                           # Authentication logic
│   ├── config/
│   │   └── config.go                         # Config parsing and setup
│   ├── handler/                              # HTTP request handlers
│   │   ├── admin_handler.go                  # Admin endpoints
│   │   ├── assignment_handler.go             # Assignment endpoints
│   │   ├── coach_handler.go                  # Coach endpoints
│   │   ├── coach_ops.go                      # Coach operations
│   │   ├── helpers.go                        # Handler helper functions
│   │   ├── student_handler.go                # Student endpoints
│   │   ├── student_ops.go                    # Student operations
│   │   ├── subject_handler.go                # Subject endpoints
│   │   └── test_paper_handler.go             # Test paper endpoints
│   ├── helper/
│   │   └── weights_v2.go                     # SQI weight functions v2
│   ├── middleware/                            # HTTP middleware
│   │   ├── auth.go                           # Authentication middleware
│   │   └── roleMiddleware.go                 # Role-based access control
│   ├── repository/                           # Data access layer
│   │   ├── db.go                             # Database operations
│   │   └── validators.go                     # Input validation
│   ├── routes/
│   │   └── routes.go                         # API route setup
│   ├── services/                             # Business logic
│   │   ├── assignment_service.go             # Assignment business logic
│   │   ├── attempt_service.go                # Attempt business logic
│   │   ├── auth_service.go                   # Authentication business logic
│   │   └── sqi_engine_v2.go                  # SQI calculations v2
│   └── types/
│       └── diagnostic.go                     # Diagnostic type definitions
├── utils/                                    # Shared utilities
│   ├── jwt.go                                # JWT token handling
│   ├── pagination.go                         # Pagination utilities
│   ├── password.go                           # Password hashing and verification
│   └── response.go                           # Safe error response handler (env-aware)
├── migrations/                               # Database migrations
│   ├── 000001_init.up.sql                    # Initial schema (10 tables)
│   ├── 000001_init.down.sql                  # Rollback initial schema
│   ├── 000002_add_attempt_constraint.*       # Attempt constraint migration
│   ├── 000003_add_coach_soft_delete.*        # Coach soft-delete migration
│   ├── 000004_fix_student_code_unique.*      # Student code unique constraint
│   └── 000006_add_subject_name_to_tests.*    # Subject name in tests
├── Ai-student-diagnosis.postman_collection.json  # API documentation
├── HANDLER_HELPERS.md                        # Handler helpers documentation
├── TEST_PAYLOADS.md                          # Test payloads documentation
└── README.md                                 # Backend-specific documentation
```

### Backend Technologies

| Technology     | Purpose               |
| -------------- | --------------------- |
| **Go 1.25.1**  | Backend language      |
| **Gin**        | Web framework         |
| **PostgreSQL** | Database              |
| **JWT**        | Authentication tokens |

### Environment Configuration

The backend uses environment variables for configuration. Copy `.env.example` to `.env` and update values:

```bash
cd backend
cp .env.example .env
```

| Variable | Values | Description |
|----------|--------|-------------|
| `APP_ENV` | `development` (default) | Raw errors returned to client for frontend debugging |
| `APP_ENV` | `production` | Generic errors returned; real errors logged server-side only |
| `PORT` | `8080` | Server port |
| `DB_URL` | `postgres://...` | PostgreSQL connection string |
| `JWT_SECRET` | string | Secret key for JWT token signing |
| `JWT_EXPIRY` | `4h` | Token expiration duration |

#### Error Handling by Environment

The backend uses `utils.SafeErrorResponse` to handle errors differently based on environment:

| Environment | Client sees | Server logs |
|-------------|-------------|-------------|
| **Development** | Raw `err.Error()` (e.g., `pq: duplicate key value violates constraint`) | Yes |
| **Production** | Generic message (e.g., `failed to create student`) | Yes |

This allows frontend developers to see exact error details during development while protecting sensitive information in production.

**Production deployment:**
```bash
APP_ENV=production go run cmd/api/main.go
```

Or set in `.env.production` and load with a tool like `godotenv`.

### Backend Architecture Layers

#### 1. **Handler Layer** (`internal/handler/`)

Handles HTTP requests and responses:

- `admin_handler.go`: Admin CRUD for tests, questions, students, coaches, subjects, assignments
- `assignment_handler.go`: Assignment endpoint handlers
- `coach_handler.go`: Coach-specific CRUD endpoints
- `coach_ops.go`: Coach operation helpers
- `helpers.go`: Handler helper functions
- `student_handler.go`: Student login and test submission
- `student_ops.go`: Student operation helpers
- `subject_handler.go`: Subject endpoint handlers
- `test_paper_handler.go`: Test paper endpoint handlers

#### 2. **Service Layer** (`internal/services/`)

Contains business logic:

- `assignment_service.go`: Assignment business logic
- `attempt_service.go`: Attempt business logic
- `auth_service.go`: Authentication business logic
- `sqi_engine_v2.go`: Enhanced SQI calculations (v2)

#### 3. **Repository Layer** (`internal/repository/`)

Data access and database operations:

- `db.go`: Database queries and operations
- `validators.go`: Input validation rules

#### 4. **Middleware** (`internal/middleware/`)

Request interceptors:

- `auth.go`: Validates JWT tokens
- `roleMiddleware.go`: Checks user roles and permissions

#### 5. **Types** (`internal/types/`)

Type definitions:

- `diagnostic.go`: Diagnostic type definitions

#### 6. **Helper** (`internal/helper/`)

Helper functions:

- `weights_v2.go`: SQI weight functions v2

#### 7. **Utilities** (`utils/`)

Shared functionality:

- `jwt.go`: Token generation and validation
- `pagination.go`: Pagination utilities
- `password.go`: Secure password handling
- `response.go`: Safe error response handler (env-aware)

---

## Database Schema

### Migrations

The system uses SQL migrations for schema management:

- **000001_init**: Creates base tables for tenants, users, coaches, students, subjects, tests, questions, assignments, attempts, answer_logs, and attempt_results
- **000002_add_attempt_constraint**: Adds attempt constraints
- **000003_add_coach_soft_delete**: Adds soft-delete support for coaches
- **000004_fix_student_code_unique**: Fixes student code unique constraint
- **000006_add_subject_name_to_tests**: Adds subject name to tests table

### Key Tables

- **tenants**: Organizations/institutes (multi-tenant isolation)
- **users**: Authentication accounts (super_admin, admin, coach roles)
- **coaches**: Coach profiles linked to user accounts
- **students**: Student records (with soft-delete support)
- **subjects**: Subjects per tenant
- **tests**: Test definitions (title, subject, coach, duration, exam_date)
- **questions**: MCQ questions with metadata (marks, difficulty, importance, concept_tag)
- **assignments**: Links students to tests via coaches
- **attempts**: Student test attempts
- **answer_logs**: Per-question answer records with behavioral signals
- **attempt_results**: SQI scores and full analysis JSON per attempt

---

## Authentication & Authorization

### Authentication Flow

1. User logs in via signup/login forms
2. Backend validates credentials and generates JWT token
3. Frontend stores token and includes in API requests
4. Authentication middleware validates token for protected routes

### Authorization

- Role-based access control (RBAC) via `roleMiddleware`
- Supports multiple roles: Super Admin, Admin, Coach, Student
- Endpoints protected by role requirements

---

## Getting Started

### Prerequisites

- **Node.js** 18+ (Frontend)
- **Go** 1.25.1+ (Backend)
- **PostgreSQL** 12+ (Database)
- **npm** or **yarn** (Package manager)

### Frontend Setup

```bash
cd frontend
npm install
npm run dev       # Development server (http://localhost:5173)
npm run build     # Production build
```

### Backend Setup

```bash
cd backend
go mod download   # Download dependencies
go run cmd/api/main.go  # Run server (http://localhost:8080)
```

### Database Setup

```bash
# Run migrations
migrate -path migrations -database "postgresql://..." up

# Or manually run SQL files
psql -U <user> -d <database> -f migrations/000001_init.up.sql
```

---

## API Documentation

The backend API is documented in the Postman collection:

- **File**: `backend/Ai-student-diagnosis.postman_collection.json`
- Import into Postman to view all available endpoints

### Main API Routes

- **Authentication**: `/auth/*` (login, register-admin, Google OAuth)
- **Student**: `/student/*` (login, submit test answers)
- **Admin**: `/admin/*` (CRUD for tests, questions, students, coaches, subjects, assignments, SQI)
- **Coach**: `/coach/*` (CRUD for tests, questions, students, subjects, assignments, SQI)

---

## Development Workflow

1. **Frontend**: Modify React components in `frontend/src/`
2. **Backend**: Update Go files in `backend/internal/`
3. **Database**: Create migrations for schema changes
4. **Testing**: Use Postman collection for API testing

---

## Project Features

### Implemented

- ✅ User authentication (signup/login, Google OAuth)
- ✅ Role-based access control (Super Admin, Admin, Coach, Student)
- ✅ JWT token management
- ✅ Password security
- ✅ Multi-tenant architecture with data isolation
- ✅ Database migrations with auto-migration on startup
- ✅ Full CRUD for tests, questions, students, coaches, subjects
- ✅ Student test submission with SQI scoring
- ✅ Pagination and search on list endpoints
- ✅ Student soft-delete with audit trail
- ✅ Coach-specific endpoints and data isolation

### In Development

- 🔄 SQI calculation engine (v1 and v2)
- 🔄 AI-powered insights and recommendations
- 🔄 Advanced analytics dashboards
- 🔄 Super admin system monitoring

---

## Additional Resources

- **Planning Document**: See `planning.md` for detailed vision and roadmap
- **Architecture**: See `architecture.md` for system architecture details
- **Backend README**: See `backend/README.md` for backend-specific details
- **Frontend README**: See `frontend/README.md` for frontend-specific details
- **Code Review**: See `code_review.md` for code review guidelines
- **Error Handling**: See `error_handling.md` for error handling patterns
- **Observability**: See `observability.md` for monitoring and logging
- **Backend Features**: See `bkfeatures.md` for backend feature list

---

## Notes

- All TypeScript code must pass ESLint checks
- Go code follows Go conventions and best practices
- Database changes require corresponding migrations
- Frontend components use Tailwind CSS for styling
- Backend API responses follow RESTful conventions

---

## Contributing

When contributing to this project:

1. Follow the existing code structure and conventions
2. Update relevant documentation
3. Test changes thoroughly
4. Create migrations for database changes
5. Ensure all linting passes

---

**Last Updated**: June 2026
