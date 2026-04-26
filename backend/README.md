# AI-Powered Student Diagnostic System - Backend

The backend for the AI-Powered Student Diagnostic System is built with Go, providing a high-performance and scalable API to handle student assessments, diagnostic calculations (SQI), and user management.

## 🏗️ Folder Structure

```text
backend/
├── cmd/
│   └── api/                # Application entry point (main.go)
├── internal/
│   ├── auth/               # JWT & Password security logic
│   ├── config/             # Environment & Configuration management
│   ├── handler/            # HTTP Handlers (Controllers)
│   ├── helpers/            # Common helper functions
│   ├── middleware/         # Gin Middlewares (Auth, Logging, etc.)
│   ├── repository/         # Database access layer (Postgres/MongoDB)
│   ├── routes/             # API Route definitions
│   └── services/           # Core Business Logic (SQI Engine, etc.)
├── migrations/             # SQL Migration files for database schema
├── utils/                  # Shared utility functions
├── .env                    # Environment variables (Local development)
├── go.mod                  # Go module definition and dependencies
└── go.sum                  # Dependency checksums
```

## 📦 Key Packages & Dependencies

- **Web Framework**: [Gin-Gonic v1.12.0](https://github.com/gin-gonic/gin) - A high-performance HTTP web framework.
- **Database Driver**: [Lib/pq](https://github.com/lib/pq) - Pure Go Postgres driver.
- **Migrations**: [Golang-Migrate v4](https://github.com/golang-migrate/migrate) - Tool for handling database schema versioning.
- **Environment Config**: [Godotenv](https://github.com/joho/godotenv) - Loads environment variables from `.env`.
- **Authentication**: [Golang-JWT v5](https://github.com/golang-jwt/jwt) - JSON Web Token implementation.
- **Validation**: [Go-Playground Validator](https://github.com/go-playground/validator) - Request payload validation.

## 🚀 Getting Started

### Prerequisites

- Go (v1.25 or later recommended)
- PostgreSQL (Ensure a database is created)

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
   The server will start on `http://localhost:8080`.

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
*Note: This is a destructive action and will wipe all existing data in the database.*

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
