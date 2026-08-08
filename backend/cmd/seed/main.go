package main

import (
	"ai-student-diagnostic/backend/internal/config"
	"ai-student-diagnostic/backend/utils"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config.LoadConfig()

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tenants").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to check tenants: %v", err)
	}
	if count > 0 {
		fmt.Println("Database already seeded. Skipping.")
		os.Exit(0)
	}

	fmt.Println("Seeding demo data...")

	// ── 1. Tenant ──────────────────────────────────────────────
	var tenantID int
	err = db.QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`, "Demo Academy").Scan(&tenantID)
	if err != nil {
		log.Fatalf("Failed to create tenant: %v", err)
	}
	fmt.Printf("  Tenant: Demo Academy (id=%d)\n", tenantID)

	// ── 2. Users ──────────────────────────────────────────────
	hash1, _ := utils.HashPassword("admin123")
	hash2, _ := utils.HashPassword("coach123")

	var superAdminID, adminID, coachUserID int

	err = db.QueryRow(`INSERT INTO users (tenant_id, email, password, role) VALUES (NULL, $1, $2, 'super_admin') RETURNING id`,
		"admin@demo.com", hash1).Scan(&superAdminID)
	if err != nil {
		log.Fatalf("Failed to create super admin: %v", err)
	}

	err = db.QueryRow(`INSERT INTO users (tenant_id, email, password, role) VALUES ($1, $2, $3, 'admin') RETURNING id`,
		tenantID, "admin2@demo.com", hash1).Scan(&adminID)
	if err != nil {
		log.Fatalf("Failed to create admin: %v", err)
	}

	err = db.QueryRow(`INSERT INTO users (tenant_id, email, password, role) VALUES ($1, $2, $3, 'coach') RETURNING id`,
		tenantID, "coach@demo.com", hash2).Scan(&coachUserID)
	if err != nil {
		log.Fatalf("Failed to create coach user: %v", err)
	}
	fmt.Printf("  Users: super_admin(id=%d), admin(id=%d), coach(id=%d)\n", superAdminID, adminID, coachUserID)

	// ── 3. Coach ──────────────────────────────────────────────
	var coachID int
	err = db.QueryRow(`INSERT INTO coaches (tenant_id, user_id, name) VALUES ($1, $2, $3) RETURNING id`,
		tenantID, coachUserID, "John Smith").Scan(&coachID)
	if err != nil {
		log.Fatalf("Failed to create coach: %v", err)
	}
	fmt.Printf("  Coach: John Smith (id=%d)\n", coachID)

	// ── 4. Students ───────────────────────────────────────────
	type studentInfo struct {
		id    int
		code  string
		name  string
	}
	students := []studentInfo{}
	for _, s := range []struct {
		code, name string
	}{
		{"STU001", "Alice"},
		{"STU002", "Bob"},
		{"STU003", "Charlie"},
	} {
		var sid int
		err = db.QueryRow(`INSERT INTO students (tenant_id, coach_id, student_code, name) VALUES ($1, $2, $3, $4) RETURNING id`,
			tenantID, coachID, s.code, s.name).Scan(&sid)
		if err != nil {
			log.Fatalf("Failed to create student %s: %v", s.name, err)
		}
		students = append(students, studentInfo{id: sid, code: s.code, name: s.name})
	}
	fmt.Printf("  Students: %d created\n", len(students))

	// ── 5. Subjects ───────────────────────────────────────────
	var mathSubjectID, physicsSubjectID int
	err = db.QueryRow(`INSERT INTO subjects (tenant_id, name) VALUES ($1, $2) RETURNING id`,
		tenantID, "Mathematics").Scan(&mathSubjectID)
	if err != nil {
		log.Fatalf("Failed to create subject: %v", err)
	}
	err = db.QueryRow(`INSERT INTO subjects (tenant_id, name) VALUES ($1, $2) RETURNING id`,
		tenantID, "Physics").Scan(&physicsSubjectID)
	if err != nil {
		log.Fatalf("Failed to create subject: %v", err)
	}
	fmt.Printf("  Subjects: Mathematics(id=%d), Physics(id=%d)\n", mathSubjectID, physicsSubjectID)

	// ── 6. Tests ──────────────────────────────────────────────
	var mathTestID, physicsTestID int
	err = db.QueryRow(`INSERT INTO tests (tenant_id, title, subject_id, subject_name, coach_id, duration, exam_date) VALUES ($1,$2,$3,$4,$5,60,'2026-06-15') RETURNING id`,
		tenantID, "Mathematics Midterm Exam 2026", mathSubjectID, "Mathematics", coachID).Scan(&mathTestID)
	if err != nil {
		log.Fatalf("Failed to create math test: %v", err)
	}

	err = db.QueryRow(`INSERT INTO tests (tenant_id, title, subject_id, subject_name, coach_id, duration, exam_date) VALUES ($1,$2,$3,$4,$5,45,'2026-06-20') RETURNING id`,
		tenantID, "Physics Unit Test", physicsSubjectID, "Physics", coachID).Scan(&physicsTestID)
	if err != nil {
		log.Fatalf("Failed to create physics test: %v", err)
	}
	fmt.Printf("  Tests: Math(id=%d), Physics(id=%d)\n", mathTestID, physicsTestID)

	// ── 7. Questions ──────────────────────────────────────────
	type q struct {
		text, a, b, c, d, correct, importance, difficulty, qtype, concept string
		marks, negMarks, expectedTime                                      float64
	}

	mathQuestions := []q{
		{"What is 2 + 3?", "4", "5", "6", "7", "B", "high", "E", "mcq", "addition", 4, 1, 30},
		{"Solve: 2x = 10", "x=3", "x=4", "x=5", "x=6", "C", "high", "M", "mcq", "algebra", 4, 1, 60},
		{"Find sqrt(144)", "10", "11", "12", "13", "C", "high", "H", "integer", "roots", 4, 1, 90},
		{"What is 15% of 200?", "25", "30", "35", "40", "B", "high", "E", "mcq", "percentages", 4, 1, 30},
		{"Which are prime numbers?", "2,3", "4,5", "6,7", "8,9", "A", "medium", "M", "multi", "number_theory", 4, 1, 60},
		{"What is the area of a circle r=7? (use pi=22/7)", "144", "148", "154", "158", "C", "medium", "E", "mcq", "geometry", 4, 1, 45},
		{"Evaluate: integral of 2x dx", "x^2", "x^2 + c", "2x^2", "2", "B", "medium", "H", "integer", "calculus", 4, 1, 120},
		{"What is 7 factorial?", "5040", "40320", "362880", "6", "A", "low", "M", "mcq", "combinatorics", 4, 1, 60},
		{"Which are even prime numbers?", "2", "2,3", "2,4", "None", "A", "low", "E", "multi", "number_theory", 4, 1, 30},
		{"Solve: x^2 - 5x + 6 = 0", "x=1,2", "x=2,3", "x=3,4", "x=4,5", "B", "low", "H", "mcq", "quadratic", 4, 1, 90},
	}

	for _, question := range mathQuestions {
		_, err = db.Exec(`INSERT INTO questions
			(test_id, question_text, option_a, option_b, option_c, option_d,
			 correct_answer, marks, neg_marks, importance, difficulty, type, expected_time, concept_tag)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			mathTestID, question.text, question.a, question.b, question.c, question.d,
			question.correct, question.marks, question.negMarks, question.importance, question.difficulty,
			question.qtype, question.expectedTime, question.concept)
		if err != nil {
			log.Fatalf("Failed to create math question: %v", err)
		}
	}

	physicsQuestions := []q{
		{"What is the SI unit of force?", "Joule", "Newton", "Watt", "Pascal", "B", "high", "E", "mcq", "mechanics", 4, 1, 30},
		{"F = ma. If m=5, a=10, F=?", "50", "55", "60", "15", "A", "high", "M", "integer", "newton_law", 4, 1, 60},
		{"What is the speed of light?", "3x10^6", "3x10^8", "3x10^10", "3x10^5", "B", "high", "H", "mcq", "electromagnetism", 4, 1, 45},
		{"Unit of electric current?", "Volt", "Ampere", "Ohm", "Watt", "B", "high", "E", "mcq", "electricity", 4, 1, 30},
		{"Which are vector quantities?", "Force,Velocity", "Speed,Mass", "Time,Energy", "None", "A", "medium", "M", "multi", "vectors", 4, 1, 60},
		{"What is KE of 2kg at 3m/s?", "6J", "9J", "12J", "18J", "B", "medium", "E", "mcq", "energy", 4, 1, 45},
		{"Find g if h=4.9m, t=1s", "9.8", "4.9", "10", "19.6", "A", "medium", "H", "integer", "gravity", 4, 1, 90},
		{"What is Ohm's law?", "V=IR", "V=I/R", "V=I+R", "V=I-R", "A", "low", "E", "mcq", "ohms_law", 4, 1, 30},
		{"Which are renewable sources?", "Solar,Wind", "Coal,Oil", "Gas,Nuclear", "Petroleum", "A", "low", "M", "multi", "energy_sources", 4, 1, 45},
		{"Frequency of AC in India?", "50Hz", "60Hz", "100Hz", "120Hz", "A", "low", "H", "mcq", "electricity", 4, 1, 60},
	}

	for _, question := range physicsQuestions {
		_, err = db.Exec(`INSERT INTO questions
			(test_id, question_text, option_a, option_b, option_c, option_d,
			 correct_answer, marks, neg_marks, importance, difficulty, type, expected_time, concept_tag)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			physicsTestID, question.text, question.a, question.b, question.c, question.d,
			question.correct, question.marks, question.negMarks, question.importance, question.difficulty,
			question.qtype, question.expectedTime, question.concept)
		if err != nil {
			log.Fatalf("Failed to create physics question: %v", err)
		}
	}
	fmt.Printf("  Questions: 20 created (10 per test)\n")

	// ── 8. Assignments ────────────────────────────────────────
	var assignAlice, assignBob, assignCharlie int
	err = db.QueryRow(`INSERT INTO assignments (student_id, test_id, coach_id) VALUES ($1,$2,$3) RETURNING id`,
		students[0].id, mathTestID, coachID).Scan(&assignAlice)
	if err != nil {
		log.Fatalf("Failed to create assignment: %v", err)
	}

	err = db.QueryRow(`INSERT INTO assignments (student_id, test_id, coach_id) VALUES ($1,$2,$3) RETURNING id`,
		students[1].id, mathTestID, coachID).Scan(&assignBob)
	if err != nil {
		log.Fatalf("Failed to create assignment: %v", err)
	}

	err = db.QueryRow(`INSERT INTO assignments (student_id, test_id, coach_id) VALUES ($1,$2,$3) RETURNING id`,
		students[2].id, physicsTestID, coachID).Scan(&assignCharlie)
	if err != nil {
		log.Fatalf("Failed to create assignment: %v", err)
	}
	fmt.Printf("  Assignments: Alice->Math(%d), Bob->Math(%d), Charlie->Physics(%d)\n", assignAlice, assignBob, assignCharlie)

	// ── 9. Attempt (Alice submitted Math) ─────────────────────
	var attemptID int
	err = db.QueryRow(`INSERT INTO attempts (assignment_id, started_at, submitted_at) VALUES ($1, NOW() - INTERVAL '7 days', NOW() - INTERVAL '6 days') RETURNING id`,
		assignAlice).Scan(&attemptID)
	if err != nil {
		log.Fatalf("Failed to create attempt: %v", err)
	}

	_, err = db.Exec(`UPDATE assignments SET status = 'submitted' WHERE id = $1`, assignAlice)
	if err != nil {
		log.Fatalf("Failed to update assignment status: %v", err)
	}
	fmt.Printf("  Attempt: Alice Math attempt (id=%d)\n", attemptID)

	// ── 10. Answer Logs (Alice's answers) ─────────────────────
	aliceAnswers := []struct {
		qid                   int
		selected              string
		isCorrect             bool
		timeSpent             float64
		markedForReview       bool
		revisited             bool
		changedAnswer         bool
		wasInitiallyWrong     bool
	}{
		{1, "B", true, 25, false, false, false, false},
		{2, "C", true, 40, false, false, false, false},
		{3, "14", false, 90, false, false, true, true},
		{4, "B", true, 30, false, false, false, false},
		{5, "A", true, 60, true, true, false, false},
		{6, "A", false, 35, false, false, true, true},
		{7, "100", false, 120, false, false, false, false},
		{8, "A", true, 20, false, false, false, false},
		{9, "", false, 0, false, false, false, false},
		{10, "D", true, 45, false, false, false, false},
	}

	for _, a := range aliceAnswers {
		seen := a.selected != "" || a.timeSpent > 0
		_, err = db.Exec(`INSERT INTO answer_logs
			(attempt_id, question_id, selected_answer, is_correct, time_spent,
			 marked_for_review, revisited, changed_answer, was_initially_wrong, seen)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			attemptID, a.qid, a.selected, a.isCorrect, a.timeSpent,
			a.markedForReview, a.revisited, a.changedAnswer, a.wasInitiallyWrong, seen)
		if err != nil {
			log.Fatalf("Failed to create answer log: %v", err)
		}
	}
	fmt.Printf("  Answer Logs: 10 answers for Alice\n")

	// ── 11. Attempt Result ────────────────────────────────────
	sqiScore := 62.5
	rawScore := 6.0
	analysisJSON := `{
		"exam_summary": {
			"total_questions": 10,
			"attempted": 9,
			"correct": 6,
			"wrong": 3,
			"skipped": 1,
			"accuracy": 66.67,
			"raw_score": 6.0,
			"max_score": 40.0,
			"negative_marks": 3.0,
			"net_score": 3.0,
			"sqi_score": 62.5
		},
		"topic_breakdown": [
			{"topic": "addition", "correct": 1, "total": 1, "accuracy": 100},
			{"topic": "algebra", "correct": 1, "total": 1, "accuracy": 100},
			{"topic": "roots", "correct": 0, "total": 1, "accuracy": 0},
			{"topic": "percentages", "correct": 1, "total": 1, "accuracy": 100},
			{"topic": "number_theory", "correct": 1, "total": 2, "accuracy": 50},
			{"topic": "geometry", "correct": 0, "total": 1, "accuracy": 0},
			{"topic": "calculus", "correct": 0, "total": 1, "accuracy": 0},
			{"topic": "combinatorics", "correct": 1, "total": 1, "accuracy": 100},
			{"topic": "quadratic", "correct": 1, "total": 1, "accuracy": 100}
		],
		"behavior_flags": {
			"time_mismanager": {"detected": false},
			"overconfident": {"detected": false},
			"panic_guesser": {"detected": false}
		}
	}`

	_, err = db.Exec(`INSERT INTO attempt_results (attempt_id, sqi_score, raw_score, analysis_json, version)
		VALUES ($1, $2, $3, $4, 'v2')`, attemptID, sqiScore, rawScore, analysisJSON)
	if err != nil {
		log.Fatalf("Failed to create attempt result: %v", err)
	}
	fmt.Printf("  Attempt Result: SQI=%.1f, Raw=%.1f\n", sqiScore, rawScore)

	fmt.Println("\nSeeding complete!")
	fmt.Println("──────────────────────────────────────────────")
	fmt.Println("Login credentials:")
	fmt.Println("  Super Admin: admin@demo.com / admin123")
	fmt.Println("  Admin:       admin2@demo.com / admin123")
	fmt.Println("  Coach:       coach@demo.com / coach123")
	fmt.Println("  Students:    STU001 (Alice), STU002 (Bob), STU003 (Charlie)")
	fmt.Println("──────────────────────────────────────────────")
	fmt.Println("Alice (STU001) has a completed Math attempt with SQI results.")
}
