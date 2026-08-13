import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Eye, EyeOff, BarChart3Icon, AlertCircleIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { loginSchema, zodErrors } from "@/lib/validations";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/components/ui/card";
import {
  FieldGroup,
  Field,
  FieldLabel,
  FieldDescription,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { login } from "@/services/auth.service";

export interface FooterLink {
  label: string;
  linkText: string;
  href: string;
}

export interface LoginPageProps extends React.ComponentProps<"div"> {
  title: string;
  description: string;
  emailPlaceholder?: string;
  dashboardPath: string;
  footerLinks?: FooterLink[];
}

export function LoginPage({
  title,
  description,
  emailPlaceholder = "m@example.com",
  dashboardPath,
  footerLinks = [],
  className,
  ...props
}: LoginPageProps) {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrors({});

    const result = loginSchema.safeParse({ email, password });
    if (!result.success) {
      setErrors(zodErrors(result.error));
      return;
    }

    setLoading(true);

    try {
      const res = await login({ email, password });
      if (res.role === "coach") {
        localStorage.setItem("coach_token", res.token);
        localStorage.setItem("coach_role", res.role);
      } else {
        localStorage.setItem("admin_token", res.token);
        localStorage.setItem("admin_role", res.role);
      }
      navigate(dashboardPath);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Login failed";

      if (message.toLowerCase().includes("credential")) {
        setErrors({ form: "Invalid email or password" });
      } else {
        setErrors({ form: message });
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      className={cn("flex w-full max-w-md flex-col gap-6", className)}
      {...props}
    >
      <Link to="/" className="flex items-center justify-center gap-2">
        <BarChart3Icon className="size-6 text-primary" />
        <span className="text-lg font-bold">EduQuant</span>
      </Link>
      <Card className="w-full">
        <CardHeader className="text-center">
          <CardTitle className="text-xl">{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit}>
            <FieldGroup>
              {errors.form && (
                <div className="flex items-center gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  <AlertCircleIcon className="size-4 shrink-0" />
                  <span>{errors.form}</span>
                </div>
              )}
              <Field>
                <FieldLabel htmlFor="email">Email</FieldLabel>
                <Input
                  id="email"
                  type="email"
                  placeholder={emailPlaceholder}
                  value={email}
                  onChange={(e) => {
                    setEmail(e.target.value);
                    if (errors.email) setErrors((prev) => ({ ...prev, email: "" }));
                  }}
                  required
                />
                {errors.email && (
                  <p className="text-sm text-destructive">{errors.email}</p>
                )}
              </Field>
              <Field>
                <div className="flex items-center">
                  <FieldLabel htmlFor="password">Password</FieldLabel>
                  <a
                    href="#"
                    className="ml-auto text-sm underline-offset-4 hover:underline"
                  >
                    Forgot your password?
                  </a>
                </div>
                <div className="relative">
                  <Input
                    id="password"
                    type={showPassword ? "text" : "password"}
                    value={password}
                    onChange={(e) => {
                      setPassword(e.target.value);
                      if (errors.password) setErrors((prev) => ({ ...prev, password: "" }));
                    }}
                    required
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword((prev) => !prev)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                    tabIndex={-1}
                  >
                    {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
                {errors.password && (
                  <p className="text-sm text-destructive">{errors.password}</p>
                )}
              </Field>
              <Field>
                <Button type="submit" className="w-full" disabled={loading}>
                  {loading ? "Logging in..." : "Login"}
                </Button>
                {footerLinks.map((link, index) => (
                  <FieldDescription key={index} className="text-center">
                    {link.label}{" "}
                    <Link
                      to={link.href}
                      className="underline hover:no-underline"
                    >
                      {link.linkText}
                    </Link>
                  </FieldDescription>
                ))}
              </Field>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
      <FieldDescription className="px-6 text-center">
        By clicking continue, you agree to our{" "}
        <a href="#" className="underline hover:no-underline">Terms of Service</a>{" "}
        and{" "}
        <a href="#" className="underline hover:no-underline">Privacy Policy</a>.
      </FieldDescription>
    </div>
  );
}
