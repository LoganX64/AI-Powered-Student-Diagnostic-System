import { useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

const BASE_URL = import.meta.env.VITE_BACKEND_URL as string;

export function ResetPasswordPage() {
  const [params] = useSearchParams();
  const token = params.get("token") || "";
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (password !== confirm) {
      setError("Passwords do not match");
      return;
    }
    setLoading(true);
    try {
      const res = await fetch(`${BASE_URL}/auth/reset-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ token, new_password: password }),
      });
      const payload = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(payload.error || "Reset failed");
      setDone(true);
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
          <CardTitle>Choose a new password</CardTitle>
          <CardDescription>Enter a new password for your account.</CardDescription>
        </CardHeader>
        <CardContent>
          {!token ? (
            <p className="text-sm text-destructive">Missing reset token in the URL.</p>
          ) : done ? (
            <p className="text-sm text-muted-foreground">
              Password updated. <Link to="/admin-signin" className="underline">Sign in</Link>.
            </p>
          ) : (
            <form onSubmit={handleSubmit}>
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="pw">New password</FieldLabel>
                  <Input id="pw" type="password" required value={password} onChange={(e) => setPassword(e.target.value)} />
                </Field>
                <Field>
                  <FieldLabel htmlFor="confirm">Confirm password</FieldLabel>
                  <Input id="confirm" type="password" required value={confirm} onChange={(e) => setConfirm(e.target.value)} />
                </Field>
                <Button type="submit" disabled={loading} className="w-full">
                  {loading ? "Updating..." : "Update password"}
                </Button>
                {error && <p className="text-sm text-destructive">{error}</p>}
              </FieldGroup>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
