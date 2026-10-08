# 🎓 AI-Powered Student Diagnostic System (SDS)

A full-stack, multi-tenant SaaS platform delivering high-precision diagnostic assessments, multidimensional **Student Quality Index (SQI v2)** scoring, real-time exam proctoring, and comprehensive analytics for coaching institutes, educators, and students.

---

## 📋 Table of Contents

- [Project Overview](#project-overview)
- [Core Features](#core-features)
- [System Architecture](#system-architecture)
- [Technology Stack](#technology-stack)
- [Frontend Architecture](#frontend-architecture)
- [Backend Architecture](#backend-architecture)
- [Database Schema & Migrations](#database-schema--migrations)
- [Authentication, Security & Quota Enforcement](#authentication-security--quota-enforcement)
- [API Route Reference](#api-route-reference)
- [Environment Configuration](#environment-configuration)
- [Getting Started](#getting-started)
- [Testing & Quality Assurance](#testing--quality-assurance)
- [Deployment](#deployment)

---

## Project Overview

The **AI-Powered Student Diagnostic System** goes far beyond traditional percentage-based test scores. It models student performance across multidimensional learning axes (speed, accuracy, conceptual mastery, behavioral revision patterns, and consistency) while providing coaching institutes with an enterprise-grade SaaS infrastructure.

### Key Highlights

- **Multi-Tenant SaaS Architecture**: Complete tenant isolation for coaching institutes and academies with per-tenant branding and settings.
- **Multidimensional SQI Engine (v2)**: Calculates granular Student Quality Index scores, topic-level learning gap identification, and prioritized study recommendations.
- **Resilient Exam Engine**: Autosave buffering (in-memory or Redis Streams), exam state recovery on disconnect, and automated background sweepers for expired attempts.
- **Live Proctoring & Video Audit**: Real-time WebSocket live streaming for coaches/proctors, chunked video uploads with Cloudinary or local disk storage, and secure tokenized video playback.
- **4-Tier Role-Based Access Control**: Tailored workflows for **Super Admin**, **Admin**, **Coach**, and **Student**.
- **Subscription & Quota Management**: Flexible tier plans (Free, Pro, Enterprise) with middleware-enforced quotas on students, coaches, tests, video storage, and SQI features.
- **Notification & Communication System**: In-app notifications with unread badges, preference toggles, and SMTP email services for password resets and system alerts.

---

## Core Features

| Feature | Description | Target Role |
| :--- | :--- | :--- |
| **SQI v2 Engine** | Multi-factor performance evaluation (Accuracy, Speed, Consistency, Revision Behavior, Topic Mastery) with asynchronous batch computation. | Admin, Coach, Student |
| **Live Proctoring** | Real-time candidate stream monitoring via WebSockets (`/view/students/:id/live`) and background video chunk recording. | Admin, Coach |
| **Exam Engine & Autosave** | Resilient client-side sync, server-side autosave buffer, offline state recovery, and auto-submission sweeper after exam expiry grace period. | Student |
| **Batch Management** | Organize students into cohorts/batches, assign batch tests with a single click, and seamlessly transfer students between batches. | Admin, Coach |
| **Tenant & Plan Billing** | Plan limits enforcement (`QuotaMiddleware`), automated usage tracking (active students, coaches, tests, video storage overage). | Super Admin, Admin |
| **Platform Administration** | Global platform metrics, tenant lifecycle management (create, suspend, reactivate), and platform subscription plans. | Super Admin |
| **Notifications & Mailer** | Real-time notification center, unread counters, notification preferences, and transactional emails via SMTP. | All Roles |

---

## System Architecture

```
                                  ┌──────────────────────────────────────────────────────────┐
                                  │                  React 19 Frontend (SPA)                 │
                                  │  • Super Admin Portal     • Admin Management Dashboard   │
                                  │  • Coach Diagnostic View  • Student Exam & Result Portal │
                                  └─────────────────────────────┬────────────────────────────┘
                                                                │ HTTPS / WebSockets (WSS)
                                                                ▼
                                  ┌──────────────────────────────────────────────────────────┐
                                  │                    Go (Gin) API Layer                    │
                                  │  • HttpOnly JWT Auth      • Rate Limiter (Redis/Memory)  │
                                  │  • Quota & Role Guard     • CSRF & Tenant Isolation MW   │
                                  └──────┬──────────────────────┬──────────────────────┬─────┘
                                         │                      │                      │
                 ┌───────────────────────┴──────────┐           │           ┌──────────┴───────────────────────┐
                 ▼                                  ▼           ▼           ▼                                  ▼
   ┌───────────────────────────┐      ┌───────────────────────────┐   ┌───────────────────────────┐      ┌───────────────────────────┐
   │     Services Layer        │      │       LiveView Hub        │   │        Job Queue          │      │     Storage Backend       │
   │  • Attempt & Exam Engine  │      │  • Student WS Streamer    │   │  • In-Process / Redis     │      │  • Cloudinary Storage     │
   │  • SQI Engine v2          │      │  • Coach Live Proctor     │   │  • Async Batch SQI Jobs   │      │  • Local Filesystem       │
   │  • Autosave Buffer        │      │  • Real-Time Broadcast    │   │  • Finalize Worker        │      │  • Tokenized Streaming    │
   │  • Sweeper / Mailer       │      └───────────────────────────┘   └───────────────────────────┘      └───────────────────────────┘
   └─────────────┬─────────────┘                                                    │
                 │                                                                  │
                 ▼                                                                  ▼
   ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
   │                                              PostgreSQL & Redis Storage                                                 │
   │  • Multi-tenant Schemas (tenants, users, coaches, students, tests, assignments, attempts, answer_logs, attempt_results)   │
   │  • Subscriptions, Plans, Batches, Notifications, Login Attempts, Password Resets                                        │
   │  • Redis Caching, Shared Rate Limiting & Autosave Buffer Streams                                                         │
   └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Technology Stack

### Frontend

- **Core**: React 19, TypeScript, Vite 8
- **Routing**: React Router 7 (`react-router-dom`)
- **Styling & UI**: Tailwind CSS v4, Radix UI Primitives, Lucide Icons, Geist Font Variable
- **State & Data Display**: TanStack React Table, Recharts, Motion (Framer Motion), Vaul, Sonner Toasts
- **Drag & Drop**: `@dnd-kit/core`, `@dnd-kit/sortable`
- **Validation**: Zod

### Backend

- **Language & Runtime**: Go 1.25.1
- **Web Framework**: Gin Gonic (`github.com/gin-gonic/gin`)
- **Database**: PostgreSQL 12+ via `github.com/lib/pq`
- **Cache & Message Broker**: Redis (`github.com/redis/go-redis/v9`)
- **Migrations**: `github.com/golang-migrate/migrate/v4`
- **Real-Time Communication**: WebSockets (`github.com/gorilla/websocket`)
- **Authentication**: JWT (`github.com/golang-jwt/jwt/v5`), bcrypt (`golang.org/x/crypto`)
- **Rate Limiting**: `golang.org/x/time/rate` & Redis sliding limiters

---

## Frontend Architecture

The frontend application resides in [`/frontend`](file:///d:/project%20files/AI-Powered-Student-Diagnostic-System/frontend) and is organized into modular feature domains:

```
frontend/
├── index.html                    # Single-page entry with dynamic CSP meta tags
├── package.json                  # Dependencies & scripts
├── vite.config.ts                # Vite build & Tailwind CSS plugin configuration
├── vercel.json                   # SPA routing rewrites for production deployment
├── routes/
│   └── routes.tsx                # Central application routing definitions & guards
├── src/
│   ├── main.tsx                  # App bootstrapping & provider registration
│   ├── index.css                 # Theme tokens, utilities & animations
│   ├── components/               # UI and shared components
│   │   ├── ProtectedRoute.tsx    # Role & authentication route guard
│   │   ├── admin/                # Admin views (batches, tests, students, coaches)
│   │   ├── coach/                # Coach portal components
│   │   ├── student/              # Student assessment components
│   │   ├── shared/               # Reusable widgets (SQI charts, dialogs, filters)
│   │   └── ui/                   # Design system primitives (buttons, modals, tables, toasts)
│   ├── config/                   # API client endpoints & environment resolution
│   ├── contexts/                 # React Contexts (DashboardContext, AuthContext)
│   ├── features/                 # Modular application pages
│   │   ├── admin/                # Admin dashboards, exam creation, batch managers
│   │   ├── coach/                # Coach dashboard & student tracking
│   │   ├── landing/              # Public landing page & marketing hero
│   │   ├── shared/               # Shared analytical views & test previews
│   │   ├── student/              # Student test player, instructions, feedback
│   │   └── super-admin/          # Super Admin global stats, tenants & plans
│   ├── hooks/                    # Custom hooks (useExamTimer, useAnswerTracker, useRole)
│   ├── lib/                      # Helper utilities (token management, formatting)
│   ├── services/                 # Typed API service wrappers
│   └── types/                    # TypeScript interfaces & static models
```

### Frontend NPM Scripts

```bash
cd frontend
pnpm dev        # Start Vite development server
pnpm build      # Typecheck with tsc and create production bundle
pnpm lint       # Execute ESLint rules
pnpm preview    # Locally preview production bundle
```

---

## Backend Architecture

The backend application resides in [`/backend`](file:///d:/project%20files/AI-Powered-Student-Diagnostic-System/backend) following clean architectural patterns:

```
backend/
├── cmd/
│   ├── api/                      # Main HTTP server entrypoint (main.go)
│   ├── createsuperadmin/         # CLI command to seed root Super Admin accounts
│   ├── check_migrations/         # Migration sanity check tool
│   ├── resetdb/                  # Database reset and schema re-migration utility
│   └── seed/                     # Development sample data seeder
├── internal/
│   ├── auth/                     # Authentication handler & token signing
│   ├── cache/                    # Redis client initialization & connection pooling
│   ├── config/                   # Environment configuration loader with validation
│   ├── handler/                  # HTTP controllers (36 handlers and comprehensive tests)
│   │   ├── admin_handler.go      # Tenant admin management endpoints
│   │   ├── assignment_handler.go # Test assignment & batch assignment handlers
│   │   ├── batch_handler.go      # Student cohort/batch management
│   │   ├── billing_handler.go    # Subscription plans, checkout & webhooks
│   │   ├── coach_handler.go      # Coach operations & student oversight
│   │   ├── exam_handler.go       # Exam execution, autosave & submission
│   │   ├── notification_handler.go# In-app notification management
│   │   ├── profile_handler.go    # User profile & password management
│   │   ├── sqi_handler.go        # Diagnostic calculation triggers & stats
│   │   ├── student_handler.go    # Student login, test state, and submission
│   │   ├── super_admin_handler.go# Multi-tenant oversight & global stats
│   │   ├── tenant_settings_handler.go # Tenant customization & branding
│   │   ├── test_paper_handler.go # Test papers, questions & subjects
│   │   └── video_handler.go      # Proctoring video chunks & tokenized streaming
│   ├── helper/                   # Math & SQI weight matrix algorithms
│   ├── liveview/                 # WebSocket Hub & real-time exam proctoring
│   ├── middleware/               # Auth, Role, Tenant, Quota, CSRF & Rate Limiters
│   ├── queue/                    # Queue abstraction (in-process channel or Redis Streams)
│   ├── repository/               # SQL data access layer (29 repository files & tests)
│   ├── routes/                   # Route setup, CORS, HTTP pipelines & shutdown hooks
│   ├── services/                 # Domain business logic:
│   │   ├── assignment_service.go # Assignment validation & allocation
│   │   ├── attempt_service.go    # Attempt lifecycle & answer finalization
│   │   ├── auth_service.go       # User credentials & role checks
│   │   ├── autosave_buffer.go    # Batched exam answer flusher
│   │   ├── job_service.go        # Asynchronous SQI chunk computation worker
│   │   ├── mailer.go             # Transactional email dispatcher (SMTP)
│   │   ├── notification_service.go # System alerts & unread count dispatch
│   │   ├── sqi_engine_v2.go      # Full SQI v2 multi-parameter calculation
│   │   └── sweeper.go            # Background attempt expiration sweeper
│   ├── storage/                  # Object storage drivers (Cloudinary & Local FS)
│   ├── testutil/                 # Mock databases, fixtures & assertions for unit tests
│   └── types/                    # Domain model definitions
├── migrations/                   # 20 sequential database migration files (.up.sql & .down.sql)
├── utils/                        # Shared utilities (JWT, Safe responses, Passwords)
├── docker-compose.yml            # Local development orchestration (Postgres + Redis)
└── go.mod                        # Go module specifications & dependencies
```

---

## Database Schema & Migrations

The system relies on 20 versioned migrations managed via `golang-migrate` and automatically executed upon server startup:

| Migration | Name | Purpose |
| :--- | :--- | :--- |
| `000001` | `init` | Base schema: tenants, users, coaches, students, subjects, tests, questions, assignments, attempts, answer_logs, attempt_results |
| `000002` | `add_attempt_constraint` | Enforces one active attempt per assignment |
| `000003` | `add_coach_soft_delete` | Adds `deleted_at` soft-delete column to coaches |
| `000004` | `fix_student_code_unique` | Adjusts student code uniqueness constraint |
| `000005` | `noop` | Migration placeholder alignment |
| `000006` | `add_subject_name_to_tests` | Denormalizes subject name on tests for optimized querying |
| `000007` | `fix_sqi_values` | Normalizes SQI decimal precision and default value constraints |
| `000008` | `soft_delete_tests_and_subjects` | Implements soft-delete lifecycle for tests and subjects |
| `000009` | `add_pg_trgm_search_indexes` | Trigram GIN indexes for fast student & coach search |
| `000010` | `add_login_attempts` | Brute-force tracking and security audit log table |
| `000011` | `exam_foundation` | Autosave states, time tracking, and exam session persistence |
| `000012` | `restore_global_student_code_unique`| Enforces global student identification code uniqueness |
| `000013` | `add_coach_subjects` | Multi-subject association mapping for coaches |
| `000014` | `super_admin_settings` | System-wide configuration table and super-admin metadata |
| `000015` | `subscriptions` | SaaS subscription plans, tenant tier allocation, feature flags |
| `000016` | `notifications` | In-app user notifications, unread tracking, and preference options |
| `000017` | `storage_overage` | Tracks video proctoring disk usage and storage limits |
| `000018` | `allow_reassignment` | Enables reassigning tests for remediations or retakes |
| `000019` | `questions_test_id_index` | B-tree index on question foreign keys for low-latency retrieval |
| `000020` | `password_resets` | Secure token hashes and expiry timestamps for password reset flows |

---

## Authentication, Security & Quota Enforcement

### 1. Authentication & Cookie Strategy
- **JWT Storage**: Issued inside secure, `HttpOnly` cookies to protect against Cross-Site Scripting (XSS).
- **Environment Aware**:
  - `APP_ENV=development`: Sets `SameSite=Lax` with `Secure=false` for frictionless local development.
  - `APP_ENV=production`: Sets `SameSite=None` with `Secure=true` to enable secure cross-origin communication between isolated domains (e.g. Vercel frontend and Render API).
- **Session Identification**: Endpoints accept the `X-Role` header to resolve role context during multi-session switching.

### 2. Quota Enforcement (`QuotaMiddleware`)
Protects against resource abuse by verifying tenant subscription limits before executing actions:
- `CheckStudentLimit()`: Restricts creating students when plan limits are reached.
- `CheckCoachLimit()`: Controls coach account capacity.
- `CheckTestLimit()`: Validates maximum test paper creations.
- `CheckSQIAccess()`: Restricts deep SQI analytics to entitled tiers.
- `CheckVideoProctoringAccess()`: Restricts video proctoring and chunk uploads to premium tiers.

### 3. API Resilience & Error Masking
- **Safe Errors (`utils.SafeErrorResponse`)**: In development mode, returns exact raw error messages for debugging; in production mode, displays sanitized generic messages while preserving full stack traces in server logs.
- **Rate Limiting**: Sliding window and token bucket rate limiters protect login attempts, exam autosave endpoints, and video uploads.
- **CSRF Protection**: Mutating requests in cross-site production mode enforce valid content types (`application/json` or `multipart/form-data`).

---

## API Route Reference

### Authentication & Profile (`/auth`)
- `POST /auth/login` - Authenticate admin, coach, or super admin
- `POST /auth/register-admin` - Register initial tenant admin account
- `POST /auth/logout` - Clear authentication cookies
- `POST /auth/forgot-password` - Request password reset email
- `POST /auth/reset-password` - Reset password with token
- `GET  /auth/profile` - Retrieve current user profile
- `PUT  /auth/profile` - Update profile information
- `PUT  /auth/password` - Change password

### Student Portal (`/student`)
- `POST /student/login` - Student login via student code & credentials
- `POST /student/logout` - Student session logout
- `GET  /student/assignments` - List assigned tests & status
- `GET  /student/assignments/:id/questions` - Fetch exam questions
- `POST /student/assignments/:id/start` - Initialize test session
- `POST /student/assignments/:id/autosave` - Buffer interim question answers
- `GET  /student/assignments/:id/state` - Fetch persisted exam state on reconnect
- `POST /student/assignments/:id/submit` - Finalize and submit assessment
- `POST /student/assignments/:id/video-chunk` - Upload proctoring video chunk
- `GET  /student/assignments/:id/live` - WebSocket stream for candidate camera

### Admin Endpoints (`/admin`)
- `POST /admin/register-coach` - Provision coach account (quota checked)
- `POST /admin/students` | `GET /admin/students` - Student CRUD and soft-delete
- `POST /admin/subjects` | `GET /admin/subjects` - Subject CRUD
- `POST /admin/tests` | `GET /admin/tests` - Test paper & question management
- `POST /admin/assignments` | `POST /admin/assignments/batch` - Test assignment
- `POST /admin/batches` | `GET /admin/batches` - Cohort/batch management
- `PATCH /admin/students/:id/batch` - Transfer student between batches
- `GET  /admin/students/:id/sqi` | `POST /admin/students/sqi-batch` - SQI diagnostics
- `POST /admin/sqi/compute` | `POST /admin/sqi/compute-batch` - Trigger asynchronous SQI computation
- `GET  /admin/jobs/:id` - Check status of background SQI processing job
- `GET  /admin/assignments/:id/video-chunks` - List recorded proctoring chunks
- `POST /admin/assignments/:id/video-token` - Generate short-lived video viewing token
- `GET  /admin/assignments/:id/video-merged` - Stream full merged proctoring video
- `GET  /admin/tenant/settings` | `PUT /admin/tenant/settings` - Tenant settings & branding
- `GET  /admin/subscription` | `POST /admin/subscription/checkout` - Plan subscription & checkout
- `GET  /admin/notifications` | `PUT /admin/notifications/:id/read` - Notifications & preferences

### Coach Endpoints (`/coach`)
- Scoped equivalents of student management, batch assignment, test creation, SQI insights, proctoring videos, and notifications restricted to the coach's assigned subjects and students.

### Super Admin Endpoints (`/super-admin`)
- `GET  /super-admin/stats` - Platform-wide aggregated metrics
- `GET  /super-admin/tenants` | `POST /super-admin/tenants` - Tenant management
- `PUT  /super-admin/tenants/:id/suspend` | `PUT /super-admin/tenants/:id/reactivate` - Lifecycle controls
- `GET  /super-admin/tenants/:id/admins` | `POST /super-admin/tenants/:id/admins` - Tenant admin provision
- `GET  /super-admin/plans` | `POST /super-admin/plans` | `PUT /super-admin/plans/:id` - Subscription plan management
- `PUT  /super-admin/tenants/:id/subscription` - Assign plan to tenant

### Live Proctoring View (`/view`)
- `GET /view/students/:id/live` - WebSocket connection for real-time video observation
- `GET /view/students/:id/live/status` - Query current live session connectivity

---

## Environment Configuration

### Backend Configuration (`backend/.env`)

| Variable | Required | Default / Example | Purpose |
| :--- | :---: | :--- | :--- |
| `PORT` | Yes | `8080` | Port for the HTTP server |
| `APP_ENV` | Yes | `development` / `production` | Controls error verbosity and cookie security |
| `DB_URL` | Yes | `postgres://user:pass@localhost:5432/sds_db?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | Yes | *Min 32 random characters* | Key used to sign user authentication tokens |
| `JWT_EXPIRY` | Yes | `4h` | Token expiration duration |
| `JWT_ISSUER` | Yes | `ai-student-diagnostic` | Token issuer identifier |
| `VIDEO_TOKEN_SECRET` | No | *Min 32 random characters* | Secret for video-stream playback tokens |
| `ALLOWED_ORIGINS` | Yes | `http://localhost:5173` | Comma-separated list of allowed CORS origins |
| `FRONTEND_URL` | Yes | `http://localhost:5173` | Base frontend URL for email password-reset links |
| `REDIS_URL` | No | `redis://localhost:6379` | Connection string for Redis cache, queue & rate limiters |
| `QUEUE_MODE` | No | `standard` (or `scale`) | `standard` uses in-process queues; `scale` engages Redis Streams |
| `CLOUDINARY_URL` | No | `cloudinary://key:secret@cloud_name` | Cloud storage for proctoring videos (falls back to local) |
| `UPLOAD_DIR` | No | `./uploads` | Local directory for storing video chunks |
| `SMTP_HOST` | No | `smtp.gmail.com` | SMTP email server hostname |
| `SMTP_PORT` | No | `587` | SMTP server port |
| `SMTP_USER` | No | `user@example.com` | SMTP authentication username |
| `SMTP_APP_PASSWORD` | No | `app-password` | SMTP authentication password |
| `SMTP_FROM` | No | `no-reply@example.com` | Sender address for system emails |
| `SUBMIT_GRACE_SECONDS` | No | `30` | Grace window before sweeper auto-submits expired tests |
| `COMPUTE_CHUNK_SIZE` | No | `25` | Batch size for asynchronous SQI processing jobs |

### Frontend Configuration (`frontend/.env`)

| Variable | Required | Default / Example | Purpose |
| :--- | :---: | :--- | :--- |
| `VITE_BACKEND_URL` | Yes | `http://localhost:8080` | Backend API origin URL |
| `VITE_WS_URL` | No | `ws://localhost:8080` | WebSocket origin (falls back to HTTP URL conversion) |

---

## Getting Started

### Prerequisites

- **Go**: v1.25.1+
- **Node.js**: v18+ (with `pnpm` or `npm`)
- **PostgreSQL**: v12+
- **Redis** *(Optional, recommended for scale features)*: v6+

### 1. Clone the Repository

```bash
git clone https://github.com/LoganX64/AI-Powered-Student-Diagnostic-System.git
cd AI-Powered-Student-Diagnostic-System
```

### 2. Setup the Backend

```bash
cd backend

# Create environment configuration
cp .env.example .env

# Download Go dependencies
go mod download

# Run database migrations and start server
go run cmd/api/main.go
```
*Note: Database migrations run automatically upon startup.*

To create an initial Super Admin user:
```bash
go run cmd/createsuperadmin/main.go -email admin@example.com -password YourSecurePassword
```

### 3. Setup the Frontend

```bash
cd ../frontend

# Create environment configuration
cp .env.example .env

# Install dependencies (pnpm recommended)
pnpm install

# Start the Vite development server
pnpm dev
```

Open [http://localhost:5173](http://localhost:5173) in your browser.

---

## Testing & Quality Assurance

### 1. Automated Backend Test Suite
The Go backend contains extensive unit and integration tests across all handlers, services, repositories, middleware, cache, storage, and utility packages (55+ test files, 350+ test functions):

```bash
cd backend

# Run all backend tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests with code coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

- **Test Payloads & Fixtures**: See [`backend/TEST_PAYLOADS.md`](file:///d:/project%20files/AI-Powered-Student-Diagnostic-System/backend/TEST_PAYLOADS.md) for sample API payloads and request fixtures.
- **Postman Collection**: Import [`backend/Ai-student-diagnosis.postman_collection.json`](file:///d:/project%20files/AI-Powered-Student-Diagnostic-System/backend/Ai-student-diagnosis.postman_collection.json) to execute manual and automated endpoint testing.

### 2. Frontend Validation & Linting

```bash
cd frontend

# Run ESLint validation
pnpm lint

# Run TypeScript typecheck and build validation
pnpm build
```

---

## Deployment

The codebase is preconfigured for continuous deployment on **Render (Backend)** and **Vercel (Frontend)**:

- **Backend (Render)**: Utilizes `render.yaml` with managed PostgreSQL. Configure production environment variables in the Render dashboard.
- **Frontend (Vercel)**: Includes `frontend/vercel.json` with SPA rewrites to `index.html`. Set `VITE_BACKEND_URL` to your production API domain.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
