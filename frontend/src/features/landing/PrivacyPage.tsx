import { Link } from "react-router-dom";
import { motion } from "motion/react";
import { Button } from "@/components/ui/button";
import { BarChart3Icon } from "lucide-react";
import { privacyPageText } from "@/types/static/legal";

export function PrivacyPage() {
  return (
    <motion.div
      className="flex flex-col min-h-screen"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.4 }}
    >
      <header className="sticky top-0 z-50 border-b bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6 lg:px-8">
          <Link to="/" className="flex items-center gap-2">
            <BarChart3Icon className="size-6 text-primary" />
            <span className="text-lg font-bold">EduQuant</span>
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

      <main className="mx-auto w-full max-w-3xl px-4 sm:px-6 lg:px-8 py-16 space-y-8">
        <div className="space-y-2">
          <h1 className="text-4xl font-bold tracking-tight">{privacyPageText.title}</h1>
          <p className="text-sm text-muted-foreground">Last updated: {privacyPageText.lastUpdated}</p>
        </div>
        {privacyPageText.sections.map((s) => (
          <section key={s.title} className="space-y-2">
            <h2 className="text-xl font-semibold">{s.title}</h2>
            <p className="text-muted-foreground leading-relaxed">{s.body}</p>
          </section>
        ))}
      </main>

      <footer className="border-t mt-auto">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 sm:px-6 lg:px-8 text-xs text-muted-foreground">
          <span>© 2026 EduQuant. All rights reserved.</span>
          <div className="flex gap-4">
            <Link to="/about" className="hover:text-foreground transition-colors">About</Link>
            <Link to="/privacy" className="hover:text-foreground transition-colors">Privacy</Link>
            <Link to="/terms" className="hover:text-foreground transition-colors">Terms</Link>
          </div>
        </div>
      </footer>
    </motion.div>
  );
}
