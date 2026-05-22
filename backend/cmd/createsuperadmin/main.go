package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"ai-student-diagnostic/backend/internal/config"
	"ai-student-diagnostic/backend/utils"

	_ "github.com/lib/pq"
)

func main() {
	email := flag.String("email", os.Getenv("CREATE_SUPER_ADMIN_EMAIL"), "super admin email")
	password := flag.String("password", os.Getenv("CREATE_SUPER_ADMIN_PASSWORD"), "super admin password")
	flag.Parse()

	*email = strings.TrimSpace(*email)

	if *email == "" || *password == "" {
		log.Fatal("email and password are required")
	}

	cfg := config.LoadConfig()

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	var superAdminCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'super_admin'").Scan(&superAdminCount); err != nil {
		log.Fatalf("failed to check existing super admins: %v", err)
	}
	if superAdminCount > 0 {
		log.Fatal("super admin already exists; refusing to create another initial super admin")
	}

	var emailExists bool
	if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", *email).Scan(&emailExists); err != nil {
		log.Fatalf("failed to check email: %v", err)
	}
	if emailExists {
		log.Fatalf("user with email %s already exists", *email)
	}

	hashedPassword, err := utils.HashPassword(*password)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	var userID int
	if err := db.QueryRow(`
		INSERT INTO users (tenant_id, email, password, role)
		VALUES (NULL, $1, $2, 'super_admin')
		RETURNING id
	`, *email, hashedPassword).Scan(&userID); err != nil {
		log.Fatalf("failed to create super admin: %v", err)
	}

	fmt.Printf("Super admin created successfully\n")
	fmt.Printf("user_id: %d\n", userID)
	fmt.Printf("email: %s\n", *email)
}
