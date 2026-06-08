export interface CoachSidebarText {
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

export interface CoachStudentsText {
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

export interface CoachSubjectsText {
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

export interface CoachTestsText {
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

export interface CoachPageText {
  sidebar: CoachSidebarText;
  students: CoachStudentsText;
  subjects: CoachSubjectsText;
  tests: CoachTestsText;
}

export const coachPageText: CoachPageText = {
  sidebar: {
    brand: "Coach Portal",
    user: {
      name: "Coach Alex",
      email: "coach@example.com",
    },
    navItems: {
      dashboard: "Dashboard",
      students: "Students",
      subjects: "Subjects",
      tests: "Tests",
      settings: "Settings",
      getHelp: "Get Help",
    },
  },
  students: {
    title: "Students",
    allStudentsTitle: "All Students",
    emptyState: "No students yet. Create one above.",
    tableHeaders: {
      id: "ID",
      name: "Name",
      code: "Code",
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
    createdTestsTitle: "Created Tests",
    assignmentsTitle: "Assignments",
    tableHeaders: {
      id: "ID",
      title: "Title",
      subjectId: "Subject ID",
      duration: "Duration",
      action: "Action",
      studentId: "Student ID",
      testId: "Test ID",
    },
    durationUnit: "min",
    deleteTitle: "Delete",
    deleteDescription:
      "Are you sure you want to delete this? This action cannot be undone.",
    deleteAssignmentTitle: "Delete Assignment",
    deleteAssignmentDescription:
      "Are you sure you want to delete this assignment? This action cannot be undone.",
    cancelButton: "Cancel",
    deleteButton: "Delete",
  },
};
