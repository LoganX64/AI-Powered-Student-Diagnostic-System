import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { BarChart3Icon, MailIcon, MapPinIcon, BuildingIcon } from "lucide-react";

export function AboutPage() {
  return (
    <div className="flex flex-col min-h-screen">
      {/* Navbar */}
      <header className="sticky top-0 z-50 border-b bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
          <Link to="/" className="flex items-center gap-2">
            <BarChart3Icon className="size-6 text-primary" />
            <span className="text-lg font-bold">EduDiagnostics</span>
          </Link>
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

      {/* Content */}
      <main className="mx-auto w-full max-w-3xl px-4 py-16 space-y-12">
        {/* Hero */}
        <div className="text-center space-y-4">
          <h1 className="text-4xl font-bold tracking-tight">About EduDiagnostics</h1>
          <p className="text-muted-foreground text-lg leading-relaxed max-w-2xl mx-auto">
            A comprehensive AI-powered system that provides deep performance
            analysis, concept-level insights, and prioritized improvement
            recommendations for coaching institutes and students.
          </p>
        </div>

        {/* Mission */}
        <section className="rounded-xl border bg-card p-8 shadow-sm space-y-4">
          <h2 className="text-2xl font-semibold">Our Mission</h2>
          <p className="text-muted-foreground leading-relaxed">
            We believe that understanding a student&apos;s learning journey goes far
            beyond raw marks and percentages. EduDiagnostics is built to empower
            coaching institutes and educators with actionable, concept-level
            insights — helping them identify learning gaps early and guide every
            student toward their full potential.
          </p>
        </section>

        {/* What We Do */}
        <section className="space-y-4">
          <h2 className="text-2xl font-semibold">What We Do</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="rounded-xl border bg-card p-6 shadow-sm">
              <h3 className="font-semibold mb-1">For Institutes</h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Manage coaches, students, subjects, and tests from a single
                dashboard. Track performance trends and make data-driven
                decisions.
              </p>
            </div>
            <div className="rounded-xl border bg-card p-6 shadow-sm">
              <h3 className="font-semibold mb-1">For Coaches</h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Monitor student progress, create assignments, and receive
                AI-generated insights to personalize your teaching approach.
              </p>
            </div>
            <div className="rounded-xl border bg-card p-6 shadow-sm">
              <h3 className="font-semibold mb-1">For Students</h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Take diagnostic tests, receive detailed performance reports, and
                get AI-assisted recommendations for focused improvement.
              </p>
            </div>
            <div className="rounded-xl border bg-card p-6 shadow-sm">
              <h3 className="font-semibold mb-1">For Parents</h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Gain clear visibility into your child&apos;s strengths and
                weaknesses with easy-to-understand diagnostic summaries.
              </p>
            </div>
          </div>
        </section>

        {/* Contact */}
        <section className="rounded-xl border bg-card p-8 shadow-sm space-y-6">
          <h2 className="text-2xl font-semibold">Contact Us</h2>
          <div className="grid gap-5 sm:grid-cols-3">
            <div className="flex items-start gap-3">
              <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <BuildingIcon className="size-4" />
              </div>
              <div>
                <p className="text-sm font-medium">Company</p>
                <p className="text-sm text-muted-foreground">Loganx64</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <MailIcon className="size-4" />
              </div>
              <div>
                <p className="text-sm font-medium">Helpline</p>
                <a
                  href="mailto:loganxtream@gmail.com"
                  className="text-sm text-muted-foreground hover:text-foreground transition-colors"
                >
                  loganxtream@gmail.com
                </a>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <MapPinIcon className="size-4" />
              </div>
              <div>
                <p className="text-sm font-medium">Address</p>
                <p className="text-sm text-muted-foreground">
                  Ulhasnagar, Maharashtra 421004
                </p>
              </div>
            </div>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t mt-auto">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 text-xs text-muted-foreground">
          <span>&copy; 2026 Loganx64. All rights reserved.</span>
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
