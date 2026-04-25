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
│   └── api/
│       └── main.go            # Application entry point
├── internal/                  # Private application code
│   ├── auth/                  # Authentication logic
│   ├── config/                # Configuration management
│   │   └── config.go          # Config parsing and setup
│   ├── handler/               # HTTP request handlers
│   │   ├── admin_handler.go   # Admin endpoints
│   │   ├── auth_handler.go    # Authentication endpoints
│   │   └── student_handler.go # Student endpoints
│   ├── helpers/               # Helper functions
│   ├── middleware/            # HTTP middleware
│   │   ├── auth.go            # Authentication middleware
│   │   └── roleMiddleware.go  # Role-based access control
│   ├── repository/            # Data access layer
│   │   ├── db.go              # Database operations
│   │   └── validators.go      # Input validation
│   ├── routes/                # Route definitions
│   │   └── routes.go          # API route setup
│   └── services/              # Business logic
│       └── sqi_engine.go      # Student Quality Index calculations
├── utils/                     # Shared utilities
│   ├── jwt.go                 # JWT token handling
│   └── password.go            # Password hashing and verification
├── migrations/                # Database migrations
│   ├── 000001_init.up.sql     # Initial schema
│   ├── 000001_init.down.sql   # Rollback initial schema
│   ├── 000002_seed_data.up.sql     # Seed data
│   └── 000002_seed_data.down.sql   # Remove seed data
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

- `auth_handler.go`: Login, signup, token refresh
- `student_handler.go`: Student-specific endpoints
- `admin_handler.go`: Admin-specific endpoints

#### 2. **Service Layer** (`internal/services/`)

Contains business logic:

- `sqi_engine.go`: Calculates Student Quality Index metrics

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

- **000001_init.up.sql**: Creates base tables for users, students, tests, attempts, and results
- **000001_init.down.sql**: Drops all tables
- **000002_seed_data.up.sql**: Populates initial data
- **000002_seed_data.down.sql**: Cleans up seed data

### Key Tables (inferred from structure)

- **users**: Authentication and user management
- **students**: Student profile information
- **tests**: Test definitions and metadata
- **attempts**: Student test attempts/submissions
- **results**: Test results and scoring data

---

## 🔐 Authentication & Authorization

### Authentication Flow

1. User logs in via signup/login forms
2. Backend validates credentials and generates JWT token
3. Frontend stores token and includes in API requests
4. Authentication middleware validates token for protected routes

### Authorization

- Role-based access control (RBAC) via `roleMiddleware`
- Supports multiple roles: Student, Admin
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

# Or manually run SQL files in order
psql -U <user> -d <database> -f migrations/000001_init.up.sql
psql -U <user> -d <database> -f migrations/000002_seed_data.up.sql
```

---

## 📚 API Documentation

The backend API is documented in the Postman collection:

- **File**: `backend/Ai-student-diagnosis.postman_collection.json`
- Import into Postman to view all available endpoints

### Main API Routes

- **Authentication**: `/api/auth/*` (login, signup, refresh token)
- **Student**: `/api/student/*` (student dashboard, results, recommendations)
- **Admin**: `/api/admin/*` (user management, analytics, system settings)

---

## 🔄 Development Workflow

1. **Frontend**: Modify React components in `frontend/src/`
2. **Backend**: Update Go files in `backend/internal/`
3. **Database**: Create migrations for schema changes
4. **Testing**: Use Postman collection for API testing

---

## 📦 Project Features

### Implemented

- ✅ User authentication (signup/login)
- ✅ Role-based access control
- ✅ JWT token management
- ✅ Password security
- ✅ SQI calculation engine
- ✅ Database migrations

### In Development

- 🔄 Student dashboard
- 🔄 Admin analytics
- 🔄 Test submission and evaluation
- 🔄 AI-powered insights

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

**Last Updated**: April 2026
