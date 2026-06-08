export interface AdminSidebarText {
  brand: string;
  navItems: {
    dashboard: string;
    coaches: string;
    students: string;
    subjects: string;
    tests: string;
    settings: string;
    getHelp: string;
  };
}

export interface AdminCoachesText {
  title: string;
  createTitle: string;
  createDescription: string;
  nameLabel: string;
  namePlaceholder: string;
  emailLabel: string;
  emailPlaceholder: string;
  passwordLabel: string;
  passwordPlaceholder: string;
  creatingText: string;
  createButton: string;
  allCoachesTitle: string;
  emptyState: string;
  tableHeaders: {
    id: string;
    name: string;
    email: string;
    action: string;
  };
  deleteTitle: string;
  deleteDescription: string;
  cancelButton: string;
  deleteButton: string;
}

export interface AdminStudentsText {
  title: string;
  createTitle: string;
  createDescription: string;
  nameLabel: string;
  namePlaceholder: string;
  codeLabel: string;
  codePlaceholder: string;
  coachIdLabel: string;
  coachIdPlaceholder: string;
  creatingText: string;
  createButton: string;
  allStudentsTitle: string;
  emptyState: string;
  tableHeaders: {
    id: string;
    name: string;
    code: string;
    coachId: string;
    action: string;
  };
  deleteTitle: string;
  deleteDescription: string;
  cancelButton: string;
  deleteButton: string;
}

export interface AdminSubjectsText {
  title: string;
  createTitle: string;
  createDescription: string;
  nameLabel: string;
  namePlaceholder: string;
  creatingText: string;
  createButton: string;
  allSubjectsTitle: string;
  emptyState: string;
  tableHeaders: {
    id: string;
    name: string;
    action: string;
  };
  deleteTitle: string;
  deleteDescription: string;
  cancelButton: string;
  deleteButton: string;
}

export interface AdminTestsText {
  title: string;
  sectionTitle: string;
  sectionDescription: string;
  tabs: {
    test: string;
    questions: string;
    assign: string;
  };
  createTest: {
    title: string;
    description: string;
    titleLabel: string;
    titlePlaceholder: string;
    subjectIdLabel: string;
    subjectIdPlaceholder: string;
    coachIdLabel: string;
    coachIdPlaceholder: string;
    durationLabel: string;
    durationPlaceholder: string;
    creatingText: string;
    createButton: string;
  };
  createQuestions: {
    title: string;
    description: string;
    testIdLabel: string;
    testIdPlaceholder: string;
    questionLabel: string;
    questionTextPlaceholder: string;
    optionLabels: string[];
    optionPlaceholders: string[];
    correctAnswerLabel: string;
    marksLabel: string;
    negMarksLabel: string;
    expTimeLabel: string;
    importanceLabel: string;
    importanceOptions: string[];
    difficultyLabel: string;
    difficultyOptions: string[];
    typeLabel: string;
    typeOptions: string[];
    conceptTagLabel: string;
    conceptTagPlaceholder: string;
    addQuestionButton: string;
    submittingText: string;
    submitButton: string;
    readyCount: string;
  };
  createAssignment: {
    title: string;
    description: string;
    studentIdLabel: string;
    studentIdPlaceholder: string;
    testIdLabel: string;
    testIdPlaceholder: string;
    coachIdLabel: string;
    coachIdPlaceholder: string;
    assigningText: string;
    assignButton: string;
  };
  deleteTitle: string;
  deleteDescription: string;
  cancelButton: string;
  deleteButton: string;
  tableHeaders: {
    id: string;
    title: string;
    subjectId: string;
    duration: string;
    action: string;
    studentId: string;
    testId: string;
  };
}

export interface AdminDashboardText {
  title: string;
}

export interface AdminPageText {
  sidebar: AdminSidebarText;
  dashboard: AdminDashboardText;
  coaches: AdminCoachesText;
  students: AdminStudentsText;
  subjects: AdminSubjectsText;
  tests: AdminTestsText;
}

export const adminPageText: AdminPageText = {
  sidebar: {
    brand: "Admin Panel",
    navItems: {
      dashboard: "Dashboard",
      coaches: "Coaches",
      students: "Students",
      subjects: "Subjects",
      tests: "Tests",
      settings: "Settings",
      getHelp: "Get Help",
    },
  },
  dashboard: {
    title: "Dashboard",
  },
  coaches: {
    title: "Coaches",
    createTitle: "Create Coach",
    createDescription: "Add a new coach to your organization.",
    nameLabel: "Full Name",
    namePlaceholder: "John Smith",
    emailLabel: "Email",
    emailPlaceholder: "coach@academy.com",
    passwordLabel: "Password",
    passwordPlaceholder: "Min. 8 characters",
    creatingText: "Creating…",
    createButton: "Create Coach",
    allCoachesTitle: "All Coaches",
    emptyState: "No coaches yet. Create one above.",
    tableHeaders: {
      id: "ID",
      name: "Name",
      email: "Email",
      action: "Action",
    },
    deleteTitle: "Delete Coach",
    deleteDescription:
      "Are you sure you want to delete this coach? This action cannot be undone.",
    cancelButton: "Cancel",
    deleteButton: "Delete",
  },
  students: {
    title: "Students",
    createTitle: "Create Student",
    createDescription: "Add a new student and assign them to a coach.",
    nameLabel: "Full Name",
    namePlaceholder: "Alice",
    codeLabel: "Student Code",
    codePlaceholder: "STU001",
    coachIdLabel: "Coach ID",
    coachIdPlaceholder: "1",
    creatingText: "Creating…",
    createButton: "Create Student",
    allStudentsTitle: "All Students",
    emptyState: "No students yet. Create one above.",
    tableHeaders: {
      id: "ID",
      name: "Name",
      code: "Code",
      coachId: "Coach ID",
      action: "Action",
    },
    deleteTitle: "Delete Student",
    deleteDescription:
      "Are you sure you want to delete this student? This action cannot be undone.",
    cancelButton: "Cancel",
    deleteButton: "Delete",
  },
  subjects: {
    title: "Subjects",
    createTitle: "Create Subject",
    createDescription: "Add a new subject to your organization.",
    nameLabel: "Subject Name",
    namePlaceholder: "Mathematics",
    creatingText: "Creating…",
    createButton: "Create Subject",
    allSubjectsTitle: "All Subjects",
    emptyState: "No subjects yet. Create one above.",
    tableHeaders: {
      id: "ID",
      name: "Name",
      action: "Action",
    },
    deleteTitle: "Delete Subject",
    deleteDescription:
      "Are you sure you want to delete this subject? This action cannot be undone.",
    cancelButton: "Cancel",
    deleteButton: "Delete",
  },
  tests: {
    title: "Tests",
    sectionTitle: "Tests & Assignments",
    sectionDescription:
      "Create a test, add questions to it, then assign it to a student.",
    tabs: {
      test: "Test",
      questions: "Questions",
      assign: "Assign",
    },
    createTest: {
      title: "Create Test",
      description:
        "Create a new test under a subject and assign it to a coach.",
      titleLabel: "Title",
      titlePlaceholder: "Mathematics Midterm Exam 2026",
      subjectIdLabel: "Subject ID",
      subjectIdPlaceholder: "1",
      coachIdLabel: "Coach ID",
      coachIdPlaceholder: "1",
      durationLabel: "Duration (minutes)",
      durationPlaceholder: "120",
      creatingText: "Creating…",
      createButton: "Create Test",
    },
    createQuestions: {
      title: "Add Questions to Test",
      description: "Add one or more questions to an existing test.",
      testIdLabel: "Test ID",
      testIdPlaceholder: "1",
      questionLabel: "Question",
      questionTextPlaceholder: "What is 2 + 2?",
      optionLabels: ["A", "B", "C", "D"],
      optionPlaceholders: [
        "Option A",
        "Option B",
        "Option C",
        "Option D",
      ],
      correctAnswerLabel: "Correct Answer",
      marksLabel: "Marks",
      negMarksLabel: "Neg. Marks",
      expTimeLabel: "Exp. Time (min)",
      importanceLabel: "Importance",
      importanceOptions: ["A (High)", "B (Medium)", "C (Low)"],
      difficultyLabel: "Difficulty",
      difficultyOptions: ["Easy", "Medium", "Hard"],
      typeLabel: "Type",
      typeOptions: ["Theory", "Numerical", "Applied"],
      conceptTagLabel: "Concept Tag",
      conceptTagPlaceholder: "basic_arithmetic",
      addQuestionButton: "Add Another Question",
      submittingText: "Submitting…",
      submitButton: "Submit",
      readyCount: "question(s) ready",
    },
    createAssignment: {
      title: "Assign Test to Student",
      description: "Link a test to a student under a specific coach.",
      studentIdLabel: "Student ID",
      studentIdPlaceholder: "1",
      testIdLabel: "Test ID",
      testIdPlaceholder: "1",
      coachIdLabel: "Coach ID",
      coachIdPlaceholder: "1",
      assigningText: "Assigning…",
      assignButton: "Assign Test",
    },
    deleteTitle: "Delete",
    deleteDescription:
      "Are you sure you want to delete this? This action cannot be undone.",
    cancelButton: "Cancel",
    deleteButton: "Delete",
    tableHeaders: {
      id: "ID",
      title: "Title",
      subjectId: "Subject ID",
      duration: "Duration",
      action: "Action",
      studentId: "Student ID",
      testId: "Test ID",
    },
  },
};
