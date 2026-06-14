# 🎓 AI-Powered Student Diagnostic System

A comprehensive system that provides deep performance analysis, concept-level insights, and AI-assisted improvement guidance for coaching institutes and students.

## 📋 Table of Contents

- [Project Overview](#project-overview)
- [System Architecture](#system-architecture)
- [Frontend Structure](#frontend-structure)
- [Backend Structure](#backend-structure)
- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)

---

## 🎯 Project Overview

The AI-Powered Student Diagnostic System moves beyond raw marks and percentages by providing:

- **Deep Performance Analysis**: Detailed breakdown of student performance across concepts and topics
- **Concept-Level Insights**: Understanding of specific learning gaps and strengths
- **Prioritized Learning Recommendations**: AI-assisted suggestions for improvement
- **Multi-Role Support**: Separate interfaces for students and administrators
- **Custom Metrics**: Student Quality Index (SQI) for comprehensive performance evaluation

---

## 🏗️ System Architecture

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

## 💻 Frontend Structure

The frontend is built with **React 19**, **TypeScript**, **Vite**, **Tailwind CSS**, and **Radix UI**.

### Project Root: `/frontend`

```
frontend/
├── index.html                  # Entry HTML file
├── package.json               # Dependencies and scripts
├── vite.config.ts             # Vite configuration
├── tsconfig.json              # TypeScript configuration
├── eslint.config.js           # ESLint rules
├── components.json            # Component metadata
├── src/
│   ├── main.tsx               # Application entry point
│   ├── App.tsx                # Root component
│   ├── App.css                # Global styles
│   ├── index.css              # Base styles
│   ├── assets/                # Static assets (images, fonts, etc.)
│   ├── components/            # Reusable UI components
│   │   ├── login-form.tsx     # Login form component
│   │   ├── signup-form.tsx    # User registration form
│   │   └── ui/                # UI component library
│   │       ├── button.tsx     # Button component
│   │       ├── card.tsx       # Card layout component
│   │       ├── field.tsx      # Form field wrapper
│   │       ├── input.tsx      # Input field component
│   │       ├── label.tsx      # Label component
│   │       └── separator.tsx  # Divider component
│   ├── features/              # Feature-specific modules
│   │   ├── admin/             # Admin dashboard and features
│   │   ├── auth/              # Authentication pages and logic
│   │   │   └── AuthPage.tsx   # Auth entry page
│   │   ├── student/           # Student portal features
│   │   ├── sqi/               # SQI (Student Quality Index) features
│   │   └── test/              # Test-related features
│   ├── services/              # API communication
│   │   └── auth.service.ts    # Authentication API calls
│   ├── lib/                   # Utility libraries
│   │   └── utils.ts           # Helper functions
│   ├── types/                 # TypeScript type definitions
│   └── utils/                 # Utility functions
├── public/                    # Public static files
└── node_modules/              # Dependencies (generated)
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

## 🔧 Backend Structure

The backend is built with **Go 1.25.1**, **Gin framework**, and **PostgreSQL**.

### Project Root: `/backend`

```
backend/
├── go.mod                     # Go module definition
├── cmd/
│   ├── api/
│   │   └── main.go            # Application entry point
│   ├── createsuperadmin/       # CLI to create super admin
│   ├── createadmin/            # CLI to create admin
│   ├── check_migrations/       # Migration checker
│   └── resetdb/                # DB reset utility
├── internal/                  # Private application code
│   ├── auth/                  # Authentication logic
│   ├── config/                # Configuration management
│   │   └── config.go          # Config parsing and setup
│   ├── handler/               # HTTP request handlers
│   │   ├── admin_handler.go   # Admin endpoints
│   │   ├── auth_handler.go    # Authentication endpoints
│   │   ├── coach_handler.go   # Coach endpoints
│   │   └── student_handler.go # Student endpoints
│   ├── helpers/               # Helper functions
│   │   ├── weights.go         # SQI weight functions v1
│   │   └── weights_v2.go      # SQI weight functions v2
│   ├── middleware/            # HTTP middleware
│   │   ├── auth.go            # Authentication middleware
│   │   └── roleMiddleware.go  # Role-based access control
│   ├── repository/            # Data access layer
│   │   ├── db.go              # Database operations
│   │   └── validators.go      # Input validation
│   ├── routes/                # Route definitions
│   │   └── routes.go          # API route setup
│   └── services/              # Business logic
│       ├── sqi_engine.go      # Student Quality Index calculations v1
│       └── sqi_engine_v2.go   # Student Quality Index calculations v2
├── utils/                     # Shared utilities
│   ├── jwt.go                 # JWT token handling
│   └── password.go            # Password hashing and verification
├── migrations/                # Database migrations
│   ├── 000001_init.up.sql     # Initial schema (10 tables)
│   └── 000001_init.down.sql   # Rollback initial schema
├── Ai-student-diagnosis.postman_collection.json  # API documentation
└── README.md                  # Backend-specific documentation
```

### Backend Technologies

| Technology     | Purpose               |
| -------------- | --------------------- |
| **Go 1.25.1**  | Backend language      |
| **Gin**        | Web framework         |
| **PostgreSQL** | Database              |
| **JWT**        | Authentication tokens |

### Backend Architecture Layers

#### 1. **Handler Layer** (`internal/handler/`)

Handles HTTP requests and responses:

- `auth_handler.go`: Login, signup, Google OAuth, password management
- `student_handler.go`: Student login and test submission
- `admin_handler.go`: Admin CRUD for tests, questions, students, coaches, subjects, assignments
- `coach_handler.go`: Coach-specific CRUD endpoints

#### 2. **Service Layer** (`internal/services/`)

Contains business logic:

- `sqi_engine.go`: Calculates Student Quality Index metrics (v1)
- `sqi_engine_v2.go`: Enhanced SQI calculations (v2)

#### 3. **Repository Layer** (`internal/repository/`)

Data access and database operations:

- `db.go`: Database queries and operations
- `validators.go`: Input validation rules

#### 4. **Middleware** (`internal/middleware/`)

Request interceptors:

- `auth.go`: Validates JWT tokens
- `roleMiddleware.go`: Checks user roles and permissions

#### 5. **Utilities** (`utils/`)

Shared functionality:

- `jwt.go`: Token generation and validation
- `password.go`: Secure password handling

---

## 🗄️ Database Schema

### Migrations

The system uses SQL migrations for schema management:

- **000001_init.up.sql**: Creates base tables for tenants, users, coaches, students, subjects, tests, questions, assignments, attempts, answer_logs, and attempt_results
- **000001_init.down.sql**: Drops all tables

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

## 🔐 Authentication & Authorization

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

## 🚀 Getting Started

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

## 📚 API Documentation

The backend API is documented in the Postman collection:

- **File**: `backend/Ai-student-diagnosis.postman_collection.json`
- Import into Postman to view all available endpoints

### Main API Routes

- **Authentication**: `/auth/*` (login, register-admin, Google OAuth)
- **Student**: `/student/*` (login, submit test answers)
- **Admin**: `/admin/*` (CRUD for tests, questions, students, coaches, subjects, assignments, SQI)
- **Coach**: `/coach/*` (CRUD for tests, questions, students, subjects, assignments, SQI)

---

## 🔄 Development Workflow

1. **Frontend**: Modify React components in `frontend/src/`
2. **Backend**: Update Go files in `backend/internal/`
3. **Database**: Create migrations for schema changes
4. **Testing**: Use Postman collection for API testing

---

## 📦 Project Features

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

## 📖 Additional Resources

- **Planning Document**: See `planning.md` for detailed vision and roadmap
- **Backend README**: See `backend/README.md` for backend-specific details
- **Frontend README**: See `frontend/README.md` for frontend-specific details

---

## 📝 Notes

- All TypeScript code must pass ESLint checks
- Go code follows Go conventions and best practices
- Database changes require corresponding migrations
- Frontend components use Tailwind CSS for styling
- Backend API responses follow RESTful conventions

---

## 🤝 Contributing

When contributing to this project:

1. Follow the existing code structure and conventions
2. Update relevant documentation
3. Test changes thoroughly
4. Create migrations for database changes
5. Ensure all linting passes

---

**Last Updated**: June 2026
