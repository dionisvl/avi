import type { Metadata } from "next";
import { LanguageSwitcher } from "@/components/language-switcher";
import { LoginForm } from "./login-form";

export const metadata: Metadata = {
  title: "Log in — Avi",
  description: "Sign in to your Avi account",
};

export default function LoginPage() {
  return (
    <main className="min-h-screen flex items-center justify-center bg-background px-4">
      <div className="absolute right-4 top-4">
        <LanguageSwitcher />
      </div>
      <div className="w-full max-w-md">
        <LoginForm />
      </div>
    </main>
  );
}
