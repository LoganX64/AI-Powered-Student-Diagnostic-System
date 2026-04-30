# AI-Powered Student Diagnostic System - Backend

The backend for the AI-Powered Student Diagnostic System is built with Go, providing a high-performance and scalable API to handle student assessments, diagnostic calculations (SQI), and user management.

## 🏗️ Project Structure

```text
backend/
├── cmd/
│   ├── api/                # Application entry point (main.go)
│   └── resetdb/            # Utility to wipe and re-migrate the database
├── internal/
│   ├── auth/               # Authentication logic (JWT, Password, Google Login)
│   ├── handler/            # HTTP Controllers (Admin, Coach, Student specialized)
│   ├── middleware/         # Gin Middleware (Auth, Role-Based Access, Tenant Checks)
│   ├── routes/             # Route definitions and group permissions
│   └── service/            # Core Business logic (SQI Calculation Engine)
├── migrations/             # SQL Schema versions (Tenants, Users, Tests, etc.)
├── utils/                  # Shared utilities (JWT generator, Password hasher)
├── .env                    # Local environment variables
└── go.mod                  # Go module definition
```

## 🏢 Multi-Tenant Architecture

This system is built with a **Shared Database, Isolated Schema** approach using a `tenant_id` on every core table. This ensures that multiple organizations can use the same application without seeing each other's data.

### Hierarchy & Roles

1.  **Super-Admin**: System-level administrator. Can manage global settings and monitor system health. Explicitly blocked from viewing sensitive student diagnostic data in individual organizations.
2.  **Admin**: The owner of a specific **Tenant (Organization)**. Can create Coaches, Students, and view all diagnostic data within their own organization.
3.  **Coach**: Staff members within an organization. Can create tests and students. They can only see data for students assigned to them.
4.  **Student**: Users who attempt tests. They can only view their own login and submission status.

### Data Isolation Features

- **Tenant Validation**: Every request made by an Admin or Coach is validated against their `tenant_id`.
- **Resource Ownership**: The system verifies that students, tests, and subjects belong to the caller's organization before performing any operation.
- **SQI Privacy**: Student Quotient Index (SQI) scores are strictly isolated. Even a Super-Admin cannot view organization-level scores without specific permission.

## 📦 Key Packages & Dependencies

- **Web Framework**: [Gin-Gonic v1.12.0](https://github.com/gin-gonic/gin) - A high-performance HTTP web framework.
- **Database Driver**: [Lib/pq v1.12.3](https://github.com/lib/pq) - Pure Go Postgres driver.
- **Migrations**: [Golang-Migrate v4.19.1](https://github.com/golang-migrate/migrate) - Tool for handling database schema versioning.
- **Environment Config**: [Godotenv v1.5.1](https://github.com/joho/godotenv) - Loads environment variables from `.env`.
- **Authentication**: [Golang-JWT v5.3.1](https://github.com/golang-jwt/jwt) - JSON Web Token implementation.
- **Cryptography**: [Golang.org/x/crypto v0.49.0](https://golang.org/x/crypto) - Secure password hashing and encryption.
- **Google OAuth**: [Google API v0.276.0](https://github.com/googleapis/google-api-go-client) - Google authentication support.

## 🚀 Getting Started

### Prerequisites

- Go (v1.25.1 or later)
- PostgreSQL (v12+)

### Setup

1. **Configure Environment Variables**:
   Create a `.env` file in the `backend/` directory (or modify the existing one):

   ```env
   DB_URL=postgres://username:password@localhost:5432/db_name?sslmode=disable
   JWT_SECRET=your_secure_secret
   JWT_EXPIRY=24h
   ```

2. **Install Dependencies**:

   ```bash
   go mod download
   ```

3. **Run the Application**:
   ```bash
   go run cmd/api/main.go
   ```
   The server will start on `http://localhost:8080`. Migrations run automatically on startup.

## 📡 API Endpoints

### Authentication Routes (`/auth`)

- `POST /auth/login` - User login (Admin/Coach)
- `POST /auth/register-admin` - Register admin account
- `POST /auth/google` - Google OAuth login

### Student Routes (`/student`)

- `POST /student/login` - Student login (no auth required)
- `POST /student/submit/:id` - Submit test answers (requires JWT)

### Admin Routes (`/admin`) - _Requires Admin role_

- `POST /admin/register-coach` - Register a new coach
- `POST /admin/subjects` - Create subject
- `POST /admin/students` - Create student
- `POST /admin/tests` - Create test
- `POST /admin/tests/:id/questions` - Create questions in batch
- `POST /admin/assignments` - Assign test to student
- `GET /admin/students/:id/sqi` - Get student SQI scores
- `GET /admin/students/:id/subjects/:subject_id/results` - Get student results for a subject

### Coach Routes (`/coach`) - _Requires Coach role_

- `POST /coach/students` - Create student
- `POST /coach/tests` - Create test
- `POST /coach/tests/:id/questions` - Create questions in batch
- `POST /coach/assignments` - Assign test to student
- `POST /coach/subjects` - Create subject
- `GET /coach/students/:id/sqi` - Get student SQI scores
- `PUT /coach/password` - Update own password (requires JWT)

## 🧠 SQI Engine (Student Quotient Index)

The SQI engine (`internal/services/sqi_engine.go`) is the core diagnostic calculation system. It analyzes student performance across multiple dimensions:

### Metrics Calculated

1. **Accuracy Metrics** - Correct answer ratio, weighted accuracy by question importance
2. **Time Metrics** - Efficiency relative to expected time, time speed ratio
3. **Difficulty Metrics** - Performance across difficulty levels (E=Easy, M=Medium, H=Hard)
4. **Behavior Metrics** - Marks for review, revisited questions, answer changes
5. **Skipping Metrics** - Percentage of questions not attempted
6. **Efficiency Metrics** - Combined accuracy and time efficiency score
7. **Concept Analysis** - Per-concept performance breakdown with priority ranking

### SQI Score Range: 0-100

- **90-100**: Excellent - High accuracy, efficient time management
- **75-89**: Very Good - Strong performance with minor inefficiencies
- **50-74**: Good - Moderate performance, room for improvement
- **25-49**: Fair - Needs improvement in accuracy or efficiency
- **0-24**: Poor - Significant challenges identified

## 📝 Testing the API

Refer to [TEST_PAYLOADS.md](./TEST_PAYLOADS.md) for comprehensive API testing examples including:

- Creating subjects, students, and tests
- Batch question creation
- Student answer submissions with different payload formats
- Score calculation examples (high, medium, low SQI)

## 🗄️ Database Migrations

This project uses `golang-migrate` to manage the Postgres schema.

### Automatic Migrations

When you run the application (`go run cmd/api/main.go`), migrations are **automatically** applied. The application will check the `migrations/` folder and update the database to the latest version.

### Reset Database Schema

If you ever need to completely wipe the database (useful in development to reset test data or apply massive schema changes), a custom Go script has been provided to drop the schema and re-run all migrations from scratch.
To run the database reset script:

```bash
go run cmd/resetdb/main.go
```

_Note: This is a destructive action and will wipe all existing data in the database._

### Manual Migrations (Optional)

If you have the `migrate` CLI installed, you can manage migrations manually:

- **Apply all migrations**:

  ```bash
  migrate -path migrations -database "postgres://username:password@localhost:5432/db_name?sslmode=disable" up
  ```

- **Rollback last migration**:

  ```bash
  migrate -path migrations -database "postgres://username:password@localhost:5432/db_name?sslmode=disable" down 1
  ```

- **Create a new migration**:
  ```bash
  migrate create -ext sql -dir migrations -seq <migration_name>
  ```
