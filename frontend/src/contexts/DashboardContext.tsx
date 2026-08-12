import { createContext, useContext, useState, useEffect, type ReactNode } from "react";
import { useRole, type Role } from "@/hooks/useRole";
import {
  getDashboardCounts,
  getStudents,
  getCoaches,
  getCoachStatsBatch,
  getStudentSQIBatch,
  type DashboardCounts,
} from "@/services/dashboard.service";
import type { Student, Coach, CoachStatMetric } from "@/services/types";

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
  role: Role | null;
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
        if (!role) return;
        const c = await getDashboardCounts();
        if (!cancelled) setCounts(c);

        const studentsRes = await getStudents({ limit: 100 });
        const students = studentsRes.data ?? [];

        if (students.length) {
          const sqiRes = await getStudentSQIBatch(students.map((s) => s.student_id));
          const byId = new Map(sqiRes.data.map((m) => [m.student_id, m]));
          const sqiResults = students.map((s) => {
            const m = byId.get(s.student_id);
            return { ...s, average_sqi: m?.average_sqi ?? 0, total_tests: m?.total_tests ?? 0 };
          });
          if (!cancelled) setStudentsWithSQI(sqiResults);
        }

        if (role === "admin") {
          const coachesRes = await getCoaches({ limit: 100 });
          const coaches = coachesRes.data ?? [];
          let statsById: Map<number, CoachStatMetric> | null = null;
          if (coaches.length) {
            try {
              const statsRes = await getCoachStatsBatch(coaches.map((c) => c.coach_id));
              statsById = new Map(statsRes.data.map((m) => [m.coach_id, m]));
            } catch {
              // stats are optional; keep defaults
            }
          }
          const rows: CoachRow[] = coaches.map((c: Coach) => {
            const m = statsById?.get(c.coach_id);
            return {
              id: c.coach_id,
              name: c.name,
              email: c.email,
              studentsCount: m?.student_count ?? 0,
              avgStudentSqi: m?.avg_sqi ?? 0,
              status: c.deleted_at ? ("Inactive" as const) : ("Active" as const),
              joinedDate: c.created_at ?? "—",
            };
          });
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
