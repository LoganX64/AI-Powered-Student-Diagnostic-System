# SQI Engine v2 — Scoring System Report

## 1. DATABASE TABLE — `attempt_results`

Defined in `backend/migrations/000001_init.up.sql:150`:

```sql
CREATE TABLE attempt_results (
    id SERIAL PRIMARY KEY,
    attempt_id INT UNIQUE,
    sqi_score FLOAT,
    raw_score FLOAT,          -- exists in schema but NOT used in code
    analysis_json JSONB,
    version VARCHAR(10) DEFAULT 'v1',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (attempt_id) REFERENCES attempts(id) ON DELETE CASCADE
);
```

**Key finding:** The `raw_score` column exists in the DB but is **never written to or read from** in any Go code. Only `sqi_score`, `analysis_json`, and `version` are used. The `version` is always set to `"v2"` when inserting (`student_handler.go:335`).

---

## 2. API ENDPOINTS FOR SQI SCORES

| # | Method | Path | Handler | Purpose |
|---|--------|------|---------|---------|
| 1 | `POST` | `/student/submit/:id` | `SubmitAnswers` | Submits answers, runs SQI engine, stores + returns score |
| 2 | `GET` | `/admin/students/:id/sqi` | `GetStudentSQI` | Lists all SQI scores for a student (admin) |
| 3 | `GET` | `/coach/students/:id/sqi` | `GetStudentSQI` | Lists all SQI scores for a student (coach) |
| 4 | `GET` | `/admin/students/:id/assignments/:assignmentId` | `GetAssignmentResults` | Full result + SQI for one assignment (admin) |
| 5 | `GET` | `/coach/students/:id/assignments/:assignmentId` | `GetAssignmentResults` | Full result + SQI for one assignment (coach) |
| 6 | `GET` | `/admin/students/:id/subjects/:subject_id/results` | `GetStudentSubjectResults` | SQI per subject with optional `?test_id=` filter |

---

## 3. PAYLOADS & RESPONSES

### Endpoint 1: `POST /student/submit/:id` — The main scoring endpoint

**Request Payload:**
```json
{
  "answers": [
    {
      "question_id": 1,
      "selected_answer": "B",
      "time_spent": 0.3,
      "seen": true,
      "marked_for_review": false,
      "revisited": false,
      "changed_answer": false,
      "was_initially_wrong": false
    },
    {
      "question_id": 10,
      "seen": false,
      "selected_answer": "",
      "time_spent": 0,
      "marked_for_review": false,
      "revisited": false,
      "changed_answer": false,
      "was_initially_wrong": false
    }
  ]
}
```

**Response:**
```json
{
  "attempt_id": 1,
  "sqi_score": 24.83,
  "total_time_spent": 21.9,
  "test_duration": 120,
  "analysis": {
    "overall_sqi": 24.83,
    "dimensions": {
      "mastery": 30.5,
      "speed": 22.1,
      "risk": 28.7,
      "coverage": 18.0
    },
    "exam_summary": {
      "exam_type": "competitive",
      "has_negative_marking": true,
      "total_questions": 10,
      "attempted": 9,
      "correct": 5,
      "wrong": 4,
      "skipped": 0,
      "unseen": 1,
      "total_marks_earned": 9.0,
      "total_marks_lost": 1.75,
      "net_score": 7.25,
      "max_possible_score": 20.0,
      "score_percent": 36.25
    },
    "attempt_profile": {
      "guessed_wrong": 2,
      "carefully_wrong": 2,
      "guessed_right": 1,
      "carefully_right": 4,
      "seen_abandoned": 0,
      "never_reached": 1,
      "neg_marks_from_guess": 0.75,
      "neg_marks_from_careful": 1.0
    },
    "concept_profiles": [
      {
        "concept_tag": "calculus_derivatives",
        "subject": "Math",
        "status": "confused",
        "priority_rank": 1,
        "evidence": {
          "total_questions": 1,
          "attempted": 1,
          "correct": 0,
          "wrong": 1,
          "skipped": 0,
          "unseen": 0,
          "accuracy_pct": 0.0,
          "avg_time_ratio": 0.9,
          "neg_marks_cost": 1.0,
          "guess_count": 0,
          "genuine_wrong": 1,
          "changed_to_correct": 0,
          "changed_to_wrong": 0,
          "mastery_score": 0.0,
          "priority_score": 0.85
        }
      }
    ],
    "behavior_flags": {
      "panic_guesser": {
        "detected": true,
        "confidence": 0.5,
        "evidence": "2 of 4 wrong answers were guesses (fast+wrong)"
      },
      "time_mismanager": { "detected": false, "confidence": 0, "evidence": "" },
      "overconfident": { "detected": false, "confidence": 0, "evidence": "" },
      "review_wasted": { "detected": false, "confidence": 0, "evidence": "" },
      "early_exhaustion": { "detected": false, "confidence": 0, "evidence": "" },
      "risky_attempter": { "detected": false, "confidence": 0, "evidence": "" },
      "strong_starter": { "detected": false, "confidence": 0, "evidence": "" }
    },
    "first_half_accuracy": 60.0,
    "second_half_accuracy": 40.0
  }
}
```

### Endpoint 2/3: `GET /admin/students/:id/sqi` or `GET /coach/students/:id/sqi`

**Query params:** `?include_analysis=true` (admin only)

**Response (without analysis):**
```json
{
  "student_id": 1,
  "name": "Alice",
  "attempts": [
    { "attempt_id": 1, "test_id": 1, "sqi_score": 24.83 },
    { "attempt_id": 2, "test_id": 2, "sqi_score": 72.10 }
  ],
  "average_sqi": 48.47,
  "total_tests": 2
}
```

**Response (with `?include_analysis=true`):**
```json
{
  "student_id": 1,
  "name": "Alice",
  "attempts": [
    {
      "attempt_id": 1,
      "test_id": 1,
      "sqi_score": 24.83,
      "analysis": { "overall_sqi": 24.83, "dimensions": {...}, ... }
    }
  ],
  "average_sqi": 24.83,
  "total_tests": 1
}
```

### Endpoint 4/5: `GET /admin/students/:id/assignments/:assignmentId` or coach equivalent

**Response:**
```json
{
  "student": { "id": 1, "name": "Alice", "student_code": "STU001" },
  "test": { "id": 1, "title": "Mathematics Midterm Exam 2026" },
  "assignment": { "id": 1, "status": "submitted", "assigned_at": "..." },
  "attempt": { "id": 1, "submitted_at": "2026-06-20T10:30:00Z" },
  "sqi_score": 24.83,
  "analysis": { "overall_sqi": 24.83, "dimensions": {...}, ... },
  "answers": [
    {
      "question_id": 1,
      "question_text": "What is 2 + 2?",
      "option_a": "3", "option_b": "4", "option_c": "5", "option_d": "6",
      "correct_answer": "B",
      "selected_answer": "B",
      "is_correct": true,
      "marks": 1,
      "time_spent": 0.3,
      "marked_for_review": false,
      "changed_answer": false,
      "seen": true,
      "concept_tag": "basic_arithmetic",
      "difficulty": "E"
    }
  ]
}
```

### Endpoint 6: `GET /admin/students/:id/subjects/:subject_id/results`

**Response:**
```json
{
  "student_id": 1,
  "student_name": "Alice",
  "subject_id": 1,
  "subject_name": "Math",
  "results": [
    {
      "attempt_id": 1,
      "test_id": 1,
      "test_title": "Mathematics Midterm",
      "sqi_score": 24.83,
      "analysis": { "overall_sqi": 24.83, "dimensions": {...}, ... }
    }
  ],
  "average_sqi": 24.83,
  "total_attempts": 1,
  "calculation": "sqi_engine"
}
```

---

## 4. SQI ENGINE V2 — HOW SCORES ARE COMPUTED

**Formula** (`backend/internal/services/sqi_engine_v2.go:270`):
```
overall_sqi = 0.35*mastery + 0.25*speed + 0.25*risk + 0.15*coverage
```

Each dimension is 0-100:
- **Mastery** — weighted accuracy (correct on hard/important = more, wrong on easy = penalized more)
- **Speed** — quality of time usage (fast+correct=100, slow+wrong=15)
- **Risk** — negative marking management (starts at 100, deductions for guessing/bad patterns)
- **Coverage** — how much of the paper was engaged with

**Weight system** (`backend/internal/helper/weights_v2.go`):

| Factor | Value | Weight |
|--------|-------|--------|
| Importance A (high) | 1.5 | |
| Importance B (medium) | 1.0 | |
| Importance C (low) | 0.6 | |
| Difficulty E (easy) | 0.8 | |
| Difficulty M (medium) | 1.0 | |
| Difficulty H (hard) | 1.3 | |
| Type integer | 1.2 | |
| Type multi | 1.1 | |
| Type mcq | 1.0 | |

**Storage:** `overall_sqi` stored as `attempt_results.sqi_score`. Full payload stored as `attempt_results.analysis_json` (JSONB). Version `"v2"` stored in `attempt_results.version`.

---

## 5. KEY SOURCE FILES

| File | Purpose |
|------|---------|
| `backend/internal/services/sqi_engine_v2.go` | Core SQI engine — Analyze() entry point |
| `backend/internal/helper/weights_v2.go` | Weight multipliers and utility functions |
| `backend/internal/handler/student_handler.go` | SubmitAnswers — triggers SQI + stores result |
| `backend/internal/handler/admin_handler.go` | GetStudentSQI, GetAssignmentResults, GetStudentSubjectResults |
| `backend/internal/handler/coach_handler.go` | GetStudentSQI, GetAssignmentResults (coach scope) |
| `backend/internal/routes/routes.go` | Route registration |
| `backend/migrations/000001_init.up.sql` | DB schema including attempt_results table |

---

## 6. NOTES

- The `raw_score` column in `attempt_results` is unused dead schema — consider removing or repurposing.
- The `version` field is hardcoded to `"v2"` in `student_handler.go:335` — no v1 data exists in practice.
- The `GetStudentSQI` admin handler supports `?include_analysis=true` query param; the coach handler does not.
- All SQI scores are rounded to 2 decimal places via `helper.Round2V2()`.
- The `DiagnosticPayloadV2` struct is the full analysis JSONB stored in the database and returned in API responses.
