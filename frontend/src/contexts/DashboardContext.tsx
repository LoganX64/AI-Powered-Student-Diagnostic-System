import { createContext, useContext, useState, useEffect, type ReactNode } from "react";
import { useRole } from "@/hooks/useRole";
import {
  getDashboardCounts,
  getStudents,
  getCoaches,
  getStudentSQI,
  type DashboardCounts,
} from "@/services/dashboard.service";
import type { Student, Coach } from "@/services/types";

export type StudentWithSQI = Student & {
  average_sqi: number;
  total_tests: number;
};

export type CoachRow = {
  id: number;
  name: string;
  email: string;
  studentsCount: number;
  avgStudentSqi: number;
  status: "Active" | "Inactive";
  joinedDate: string;
};

type DashboardContextValue = {
  counts: DashboardCounts;
  studentsWithSQI: StudentWithSQI[];
  coachRows: CoachRow[];
  loading: boolean;
  role: string;
};

const DashboardContext = createContext<DashboardContextValue | null>(null);

export function DashboardProvider({ children }: { children: ReactNode }) {
  const role = useRole();
  const [counts, setCounts] = useState<DashboardCounts>({
    totalCoaches: 0,
    totalStudents: 0,
    testsCreated: 0,
  });
  const [studentsWithSQI, setStudentsWithSQI] = useState<StudentWithSQI[]>([]);
  const [coachRows, setCoachRows] = useState<CoachRow[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const c = await getDashboardCounts();
        if (!cancelled) setCounts(c);

        const studentsRes = await getStudents({ limit: 100 });
        const students = studentsRes.data ?? [];

        const sqiResults = await Promise.all(
          students.map(async (s) => {
            try {
              const sqi = await getStudentSQI(s.student_id, { compute: true });
              return { ...s, average_sqi: sqi.average_sqi, total_tests: sqi.total_tests };
            } catch {
              return { ...s, average_sqi: 0, total_tests: 0 };
            }
          })
        );

        if (!cancelled) setStudentsWithSQI(sqiResults);

        if (role === "admin") {
          const coachesRes = await getCoaches({ limit: 100 });
          const coaches = coachesRes.data ?? [];
          const rows: CoachRow[] = coaches.map((c: Coach) => ({
            id: c.coach_id,
            name: c.name,
            email: c.email,
            studentsCount: 0,
            avgStudentSqi: 0,
            status: c.deleted_at ? ("Inactive" as const) : ("Active" as const),
            joinedDate: "—",
          }));
          if (!cancelled) setCoachRows(rows);
        }
      } catch {
        // keep defaults
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [role]);

  return (
    <DashboardContext.Provider value={{ counts, studentsWithSQI, coachRows, loading, role }}>
      {children}
    </DashboardContext.Provider>
  );
}

export function useDashboard() {
  const ctx = useContext(DashboardContext);
  if (!ctx) throw new Error("useDashboard must be used within DashboardProvider");
  return ctx;
}
