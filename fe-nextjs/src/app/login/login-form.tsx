"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useT } from "@/i18n";
import { useAuth } from "@/lib/auth/context";
import { getDemoUsers, isDemoMode } from "@/lib/demo";

export function LoginForm() {
  const t = useT();
  const router = useRouter();
  const { login } = useAuth();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setIsLoading(true);

    try {
      await login(email, password);
      // Redirect to home on success
      router.push("/");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Login failed";
      setError(message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleDemoLogin = async (demoEmail: string, demoPassword: string) => {
    setEmail(demoEmail);
    setPassword(demoPassword);
    setError("");
    setIsLoading(true);

    try {
      await login(demoEmail, demoPassword);
      router.push("/");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Login failed";
      setError(message);
    } finally {
      setIsLoading(false);
    }
  };

  const demoUsers = isDemoMode() ? getDemoUsers() : [];
  const mailpitUrl = process.env.NEXT_PUBLIC_MAILPIT_URL;

  return (
    <div className="space-y-6">
      <div className="mb-8 text-center">
        <h1 className="text-3xl font-bold text-text-primary mb-2">{t.auth.loginTitle}</h1>
        <p className="text-sm text-text-secondary">{t.auth.loginSubtitle}</p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="email" className="block text-sm font-medium text-text-primary mb-2">
            {t.auth.email}
          </label>
          <Input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={t.auth.emailPlaceholder}
            disabled={isLoading}
            required
          />
        </div>

        <div>
          <label htmlFor="password" className="block text-sm font-medium text-text-primary mb-2">
            {t.auth.password}
          </label>
          <Input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={t.auth.passwordPlaceholder}
            disabled={isLoading}
            required
          />
        </div>

        {error && <div className="text-sm text-destructive">{error}</div>}

        <Button type="submit" className="w-full" disabled={isLoading || !email || !password}>
          {isLoading ? t.common.loading : t.auth.logIn}
        </Button>
      </form>

      {demoUsers.length > 0 && (
        <div className="border-t border-border pt-6">
          <p className="mb-4 rounded-lg bg-surface-soft px-4 py-3 text-center text-xs text-text-secondary">
            {t.auth.registrationDisabled}
          </p>
          <p className="text-center text-sm font-medium text-text-primary mb-4">
            {t.auth.demoMode}
          </p>
          <div className="space-y-2">
            {demoUsers.map((user) => (
              <button
                key={user.email}
                type="button"
                onClick={() => handleDemoLogin(user.email, user.password)}
                disabled={isLoading}
                className="w-full flex flex-col items-start gap-1 px-4 py-3 text-left border border-border rounded-lg hover:bg-surface-soft transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <span className="text-sm font-medium text-text-primary">
                  {t.auth.signInAs} {user.label}
                </span>
                <span className="font-mono text-xs text-text-secondary">
                  {user.email} · {user.password}
                </span>
              </button>
            ))}
          </div>
          <p className="text-center text-xs text-text-secondary mt-3">
            {t.auth.credentialsVisible}
          </p>
          {mailpitUrl && (
            <p className="mt-2 text-center text-xs text-text-secondary">
              {t.auth.mailpitHint}{" "}
              <a
                href={mailpitUrl}
                target="_blank"
                rel="noreferrer"
                className="font-medium text-primary hover:underline"
              >
                Mailpit
              </a>
            </p>
          )}
        </div>
      )}
    </div>
  );
}
