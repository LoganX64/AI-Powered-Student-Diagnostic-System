import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import {
  BarChart3Icon,
  BrainCircuitIcon,
  TargetIcon,
  UsersIcon,
  LineChartIcon,
  SparklesIcon,
} from "lucide-react";

const features = [
  {
    icon: <BarChart3Icon className="size-6" />,
    title: "Deep Performance Analysis",
    description:
      "Detailed breakdown of student performance across concepts and topics, going beyond raw marks and percentages.",
  },
  {
    icon: <BrainCircuitIcon className="size-6" />,
    title: "Concept-Level Insights",
    description:
      "Understand specific learning gaps and strengths at the concept level for targeted interventions.",
  },
  {
    icon: <TargetIcon className="size-6" />,
    title: "Prioritized Recommendations",
    description:
      "AI-assisted suggestions that prioritize the most impactful areas for student improvement.",
  },
  {
    icon: <UsersIcon className="size-6" />,
    title: "Multi-Role Support",
    description:
      "Dedicated interfaces for administrators, coaches, and students with role-appropriate tools.",
  },
  {
    icon: <LineChartIcon className="size-6" />,
    title: "Student Quality Index",
    description:
      "Custom SQI metric that provides a comprehensive view of student performance across all dimensions.",
  },
  {
    icon: <SparklesIcon className="size-6" />,
    title: "AI-Powered Diagnostics",
    description:
      "Intelligent analysis engine that processes test results to generate actionable diagnostic reports.",
  },
];

function HeroIllustration() {
  return (
    <svg
      viewBox="0 0 400 320"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className="w-full max-w-md"
    >
      {/* Background shapes */}
      <rect x="40" y="40" width="320" height="240" rx="16" fill="hsl(var(--muted))" opacity="0.5" />
      <rect x="56" y="56" width="288" height="208" rx="12" fill="hsl(var(--card))" stroke="hsl(var(--border))" strokeWidth="1" />

      {/* Chart bars */}
      <rect x="88" y="180" width="28" height="60" rx="4" fill="hsl(var(--primary))" opacity="0.8" />
      <rect x="128" y="150" width="28" height="90" rx="4" fill="hsl(var(--primary))" opacity="0.6" />
      <rect x="168" y="120" width="28" height="120" rx="4" fill="hsl(var(--primary))" opacity="0.9" />
      <rect x="208" y="140" width="28" height="100" rx="4" fill="hsl(var(--primary))" opacity="0.7" />
      <rect x="248" y="100" width="28" height="140" rx="4" fill="hsl(var(--primary))" opacity="0.85" />
      <rect x="288" y="130" width="28" height="110" rx="4" fill="hsl(var(--primary))" opacity="0.65" />

      {/* Line chart overlay */}
      <polyline
        points="102,170 142,140 182,110 222,130 262,90 302,120"
        fill="none"
        stroke="hsl(var(--chart-2, #22c55e))"
        strokeWidth="3"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="102" cy="170" r="4" fill="hsl(var(--chart-2, #22c55e))" />
      <circle cx="142" cy="140" r="4" fill="hsl(var(--chart-2, #22c55e))" />
      <circle cx="182" cy="110" r="4" fill="hsl(var(--chart-2, #22c55e))" />
      <circle cx="222" cy="130" r="4" fill="hsl(var(--chart-2, #22c55e))" />
      <circle cx="262" cy="90" r="4" fill="hsl(var(--chart-2, #22c55e))" />
      <circle cx="302" cy="120" r="4" fill="hsl(var(--chart-2, #22c55e))" />

      {/* Axis lines */}
      <line x1="80" y1="250" x2="320" y2="250" stroke="hsl(var(--muted-foreground))" strokeWidth="1" opacity="0.3" />
      <line x1="80" y1="80" x2="80" y2="250" stroke="hsl(var(--muted-foreground))" strokeWidth="1" opacity="0.3" />

      {/* Floating card */}
      <rect x="240" y="60" width="120" height="50" rx="8" fill="hsl(var(--background))" stroke="hsl(var(--border))" strokeWidth="1" />
      <text x="256" y="80" fontSize="10" fill="hsl(var(--muted-foreground))" fontFamily="sans-serif">SQI Score</text>
      <text x="256" y="98" fontSize="16" fontWeight="bold" fill="hsl(var(--primary))" fontFamily="sans-serif">87.5</text>
    </svg>
  );
}

export function LandingPage() {
  return (
    <div className="flex flex-col min-h-screen">
      {/* Navbar */}
      <header className="sticky top-0 z-50 border-b bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <BarChart3Icon className="size-6 text-primary" />
            <span className="text-lg font-bold">EduDiagnostics</span>
          </div>
          <nav className="hidden md:flex items-center gap-6 text-sm text-muted-foreground">
            <a href="#features" className="hover:text-foreground transition-colors">Features</a>
            <a href="/about" className="hover:text-foreground transition-colors">About</a>
          </nav>
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="sm" asChild>
              <Link to="/student-login">Student Login</Link>
            </Button>
            <Button size="sm" asChild>
              <Link to="/admin-signup">Register Now</Link>
            </Button>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="mx-auto flex w-full max-w-6xl flex-col-reverse items-center gap-12 px-4 py-20 md:flex-row md:py-28">
        <div className="flex-1 space-y-6">
          <div className="inline-block rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
            AI-Powered Student Diagnostics
          </div>
          <h1 className="text-4xl font-bold leading-tight tracking-tight md:text-5xl">
            Understand every student.
            <br />
            <span className="text-primary">Improve every outcome.</span>
          </h1>
          <p className="max-w-lg text-muted-foreground text-base leading-relaxed">
            A comprehensive system that provides deep performance analysis,
            concept-level insights, and AI-assisted improvement guidance for
            coaching institutes and students. Go beyond marks — understand the
            learning gaps.
          </p>
          <div className="flex items-center gap-4">
            <Button size="lg" asChild>
              <Link to="/admin-signup">Register Now</Link>
            </Button>
            <Button variant="outline" size="lg" asChild>
              <Link to="/student-login">Student Login</Link>
            </Button>
          </div>
        </div>
        <div className="flex-1 flex justify-center">
          <HeroIllustration />
        </div>
      </section>

      {/* Features */}
      <section id="features" className="border-t bg-muted/40">
        <div className="mx-auto max-w-6xl px-4 py-20">
          <div className="mb-12 text-center">
            <h2 className="text-3xl font-bold tracking-tight">Everything you need</h2>
            <p className="mt-2 text-muted-foreground">
              Powerful tools to diagnose, track, and improve student performance.
            </p>
          </div>
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {features.map((feature) => (
              <div
                key={feature.title}
                className="rounded-xl border bg-card p-6 shadow-sm transition-shadow hover:shadow-md"
              >
                <div className="mb-3 flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  {feature.icon}
                </div>
                <h3 className="mb-1 font-semibold">{feature.title}</h3>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  {feature.description}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* About / CTA */}
      <section className="mx-auto w-full max-w-6xl px-4 py-20">
        <div className="rounded-2xl border bg-card p-8 text-center shadow-sm md:p-12">
          <h2 className="text-2xl font-bold tracking-tight md:text-3xl">
            Ready to transform how you track student progress?
          </h2>
          <p className="mx-auto mt-3 max-w-xl text-muted-foreground">
            Create your organization account and start diagnosing student
            performance with AI-powered insights today.
          </p>
          <div className="mt-6 flex items-center justify-center gap-4">
            <Button size="lg" asChild>
              <Link to="/admin-signup">Get Started — It&apos;s Free</Link>
            </Button>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t mt-auto">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 text-xs text-muted-foreground">
          <span>&copy; 2026 EduDiagnostics. All rights reserved.</span>
          <div className="flex gap-4">
            <Link to="/about" className="hover:text-foreground transition-colors">About</Link>
            <a href="#" className="hover:text-foreground transition-colors">Privacy</a>
            <a href="#" className="hover:text-foreground transition-colors">Terms</a>
          </div>
        </div>
      </footer>
    </div>
  );
}
