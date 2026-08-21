interface CoachSidebarText {
  brand: string;
  user: {
    name: string;
    email: string;
  };
  navItems: {
    dashboard: string;
    students: string;
    subjects: string;
    tests: string;
    settings: string;
    getHelp: string;
  };
}

interface CoachStudentsText {
  title: string;
  allStudentsTitle: string;
  emptyState: string;
  tableHeaders: {
    id: string;
    name: string;
    code: string;
    action: string;
  };
  deleteTitle: string;
  deleteDescription: string;
  cancelButton: string;
  deleteButton: string;
}

interface CoachSubjectsText {
  title: string;
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

interface CoachTestsText {
  title: string;
  sectionTitle: string;
  sectionDescription: string;
  tabs: {
    test: string;
    questions: string;
    assign: string;
  };
  createdTestsTitle: string;
  assignmentsTitle: string;
  tableHeaders: {
    id: string;
    title: string;
    subjectId: string;
    duration: string;
    action: string;
    studentId: string;
    testId: string;
  };
  durationUnit: string;
  deleteTitle: string;
  deleteDescription: string;
  deleteAssignmentTitle: string;
  deleteAssignmentDescription: string;
  cancelButton: string;
  deleteButton: string;
}

interface CoachPageText {
  sidebar: CoachSidebarText;
  students: CoachStudentsText;
  subjects: CoachSubjectsText;
  tests: CoachTestsText;
}


