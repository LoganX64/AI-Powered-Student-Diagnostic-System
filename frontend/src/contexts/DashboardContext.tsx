import { createContext, useContext, useState, useEffect, useMemo, type ReactNode } from "react";
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
  subjects: string;
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

    // Clear the previous role's data before refetching, and re-arm the loading
    // state. setLoading(true) is otherwise never called after mount, so a role
    // switch painted the old role's students with no spinner — and because
    // /admin/dashboard and /coach/dashboard render the same component, React
    // reconciles rather than unmounting, so that data survives the switch. A
    // coach would briefly (or, if their own list came back empty, indefinitely)
    // see the admin's full roster.
    setLoading(true);
    setStudentsWithSQI([]);
    setCoachRows([]);

    async function load() {
      try {
        if (!role) return;

        // Three independent fetches run together; the SQI and coach-stats batches
        // only depend on the student/coach lists, so they form a second parallel step.
        const [c, studentsRes, coachesRes] = await Promise.all([
          getDashboardCounts(),
          getStudents({ limit: 100 }),
          role === "admin"
            ? getCoaches({ limit: 100 })
            : Promise.resolve({ data: [] as never[] }),
        ]);
        if (cancelled) return;
        setCounts(c);

        const students = studentsRes.data ?? [];
        let sqiResults: StudentWithSQI[] = [];
        const sqiPromise = students.length
          ? getStudentSQIBatch(students.map((s) => s.student_id)).then((res) => {
              const byId = new Map(res.data.map((m) => [m.student_id, m]));
              return students.map((s) => {
                const m = byId.get(s.student_id);
                return { ...s, average_sqi: m?.average_sqi ?? 0, total_tests: m?.total_tests ?? 0 };
              });
            })
          : Promise.resolve([] as StudentWithSQI[]);

        let coachRowsPromise: Promise<CoachRow[]> = Promise.resolve([]);
        if (role === "admin") {
          const coaches = coachesRes.data ?? [];
          coachRowsPromise = (async () => {
            let statsById: Map<number, CoachStatMetric> | null = null;
            if (coaches.length) {
              try {
                const statsRes = await getCoachStatsBatch(coaches.map((c) => c.coach_id));
                statsById = new Map(statsRes.data.map((m) => [m.coach_id, m]));
              } catch {
                // stats are optional; keep defaults
              }
            }
            return coaches.map((c: Coach) => {
              const m = statsById?.get(c.coach_id);
              return {
                id: c.coach_id,
                name: c.name,
                email: c.email,
                studentsCount: m?.student_count ?? 0,
                avgStudentSqi: m?.avg_sqi ?? 0,
                status: c.deleted_at ? ("Inactive" as const) : ("Active" as const),
                joinedDate: c.created_at ?? "—",
                subjects: (c.subjects ?? []).map((s) => s.subject_name).join(", "),
              };
            });
          })();
        }

        const [sqi, rows] = await Promise.all([sqiPromise, coachRowsPromise]);
        if (cancelled) return;
        sqiResults = sqi;
        setStudentsWithSQI(sqiResults);
        if (role === "admin") setCoachRows(rows);
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

  const value = useMemo(
    () => ({ counts, studentsWithSQI, coachRows, loading, role }),
    [counts, studentsWithSQI, coachRows, loading, role],
  );

  return (
    <DashboardContext.Provider value={value}>
      {children}
    </DashboardContext.Provider>
  );
}

export function useDashboard() {
  const ctx = useContext(DashboardContext);
  if (!ctx) throw new Error("useDashboard must be used within DashboardProvider");
  return ctx;
}
