import { useState } from "react";
import { Link } from "react-router-dom";
import { BarChart3Icon, AlertCircleIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { studentLoginSchema, zodErrors } from "@/lib/validations";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/components/ui/card";
import { FieldGroup, Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

type LoginFormData = {
  student_code: string;
};

type LoginFormProps = {
  onSubmit?: (data: LoginFormData) => void;
  loading?: boolean;
  error?: string;
  className?: string;
};

export function StudentLoginForm({
  className,
  onSubmit,
  loading,
  error,
}: LoginFormProps) {
  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleSubmit: React.FormEventHandler<HTMLFormElement> = (event) => {
    event.preventDefault();
    setErrors({});

    const formData = new FormData(event.currentTarget);
    const result = studentLoginSchema.safeParse({
      student_code: formData.get("student_code")?.toString() ?? "",
    });
    if (!result.success) {
      setErrors(zodErrors(result.error));
      return;
    }

    onSubmit?.(result.data);
  };

  return (
    <div className={cn("flex w-full max-w-md flex-col gap-6", className)}>
      <Link to="/" className="flex items-center justify-center gap-2">
        <BarChart3Icon className="size-6 text-primary" />
        <span className="text-lg font-bold">EduQuant</span>
      </Link>
      <Card className="w-full">
        <CardHeader className="text-center">
          <CardTitle className="text-xl">Login to your account</CardTitle>
          <CardDescription>
            Enter your Student code below to login to your account
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit}>
            <FieldGroup>
              {error && (
                <div className="flex items-center gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  <AlertCircleIcon className="size-4 shrink-0" />
                  <span>{error}</span>
                </div>
              )}
              <Field>
                <FieldLabel htmlFor="student_code">Student Code</FieldLabel>
                <Input
                  id="student_code"
                  name="student_code"
                  type="text"
                  placeholder="Enter your student code"
                  required
                />
                {errors.student_code && (
                  <p className="text-sm text-destructive">{errors.student_code}</p>
                )}
              </Field>

              <Field>
                <Button type="submit" className="w-full" disabled={loading}>
                  {loading ? "Logging in..." : "Login"}
                </Button>
              </Field>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
