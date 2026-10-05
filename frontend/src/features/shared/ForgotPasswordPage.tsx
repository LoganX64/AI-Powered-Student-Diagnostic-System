import { useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api";

const BASE_URL = import.meta.env.VITE_BACKEND_URL as string;

export function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await fetch(`${BASE_URL}/auth/forgot-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ email }),
      });
      const payload = await res.json().catch(() => ({}));
      if (!res.ok) throw new ApiError(payload.error || "Request failed", res.status, payload);
      setSent(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Reset your password</CardTitle>
          <CardDescription>Enter the email on your account and we'll send a reset link.</CardDescription>
        </CardHeader>
        <CardContent>
          {sent ? (
            <p className="text-sm text-muted-foreground">
              If that email is registered, a password reset link has been sent. Check your inbox.
            </p>
          ) : (
            <form onSubmit={handleSubmit}>
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="email">Email</FieldLabel>
                  <Input id="email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
                </Field>
                <Button type="submit" disabled={loading} className="w-full">
                  {loading ? "Sending..." : "Send reset link"}
                </Button>
                {error && <p className="text-sm text-destructive">{error}</p>}
              </FieldGroup>
            </form>
          )}
          <p className="mt-4 text-center text-sm text-muted-foreground">
            <Link to="/admin-signin" className="underline hover:no-underline">Back to sign in</Link>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
