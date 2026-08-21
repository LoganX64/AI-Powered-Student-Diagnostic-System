interface AdminSidebarText {
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

interface AdminCoachesText {
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

interface AdminStudentsText {
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

interface AdminSubjectsText {
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

interface AdminTestsText {
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

interface AdminDashboardText {
  title: string;
}

interface AdminPageText {
  sidebar: AdminSidebarText;
  dashboard: AdminDashboardText;
  coaches: AdminCoachesText;
  students: AdminStudentsText;
  subjects: AdminSubjectsText;
  tests: AdminTestsText;
}


