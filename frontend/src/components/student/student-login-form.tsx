// import { Link } from "react-router-dom";
import { cn } from "../../lib/utils";
import { Button } from "../ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "../ui/card";
import { FieldGroup, Field, FieldLabel } from "../ui/field";
import { Input } from "../ui/input";

type LoginFormData = {
  student_code: string;
};

type LoginFormProps = {
  onSubmit?: (data: LoginFormData) => void;
  loading?: boolean;
  className?: string;
};

export function StudentLoginForm({
  className,
  onSubmit,
  loading,
}: LoginFormProps) {
  const handleSubmit: React.FormEventHandler<HTMLFormElement> = (event) => {
    event.preventDefault();

    const formData = new FormData(event.currentTarget);
    onSubmit?.({
      student_code: formData.get("student_code")?.toString() ?? "",
    });
  };

  return (
    <div className={cn("flex w-full max-w-md flex-col gap-6", className)}>
      <Card className="w-full">
        <CardHeader>
          <CardTitle>Login to your account</CardTitle>
          <CardDescription>
            Enter your Student code below to login to your account
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="student_code">Student Code</FieldLabel>
                <Input
                  id="student_code"
                  name="student_code"
                  type="text"
                  placeholder="Enter your student code"
                  required
                />
              </Field>

              <Field>
                <Button type="submit" disabled={loading}>
                  {loading ? "Logging in..." : "Login"}
                </Button>
                {/* <Button variant="outline" type="button" disabled={loading}> 
                  Login with Google
                </Button> */}
                {/* <FieldDescription className="text-center">
                  Don&apos;t have an account?{" "}
                  <Link
                    to="/student-signup"
                    className="underline hover:no-underline"
                  >
                    Sign up
                  </Link>
                </FieldDescription> */}
              </Field>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
