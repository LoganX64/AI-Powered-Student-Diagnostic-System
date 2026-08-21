CREATE TABLE coach_subjects (
    coach_id   INT NOT NULL,
    subject_id INT NOT NULL,
    PRIMARY KEY (coach_id, subject_id),
    FOREIGN KEY (coach_id) REFERENCES coaches(id) ON DELETE CASCADE,
    FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE CASCADE
);
