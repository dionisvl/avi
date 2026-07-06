"use client";

import {
  ArrowLeft,
  Check,
  ChevronDown,
  ChevronRight,
  Heart,
  MapPin,
  Menu,
  MessageCircle,
  User,
  Zap,
} from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { LanguageSwitcher } from "@/components/language-switcher";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { useLocale, useT } from "@/i18n";
import { useAuth } from "@/lib/auth/context";
import { isDemoMode } from "@/lib/demo";
import { cn } from "@/lib/utils";
import { useCities } from "./use-catalog";

export function Header() {
  const t = useT();
  const locale = useLocale();
  const router = useRouter();
  const { data: cities = [] } = useCities();
  const [selectedCity, setSelectedCity] = useState(cities.length > 0 ? cities[0] : null);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isCityMenuOpen, setIsCityMenuOpen] = useState(false);
  const { user, isAuthenticated, logout } = useAuth();

  const handleLogout = async () => {
    await logout();
    router.push("/");
  };

  const cityLabel = selectedCity
    ? selectedCity.names?.[locale] || selectedCity.names?.en || selectedCity.slug
    : t.header.selectCity;

  const navLinks = [
    { label: t.header.categories, href: "/items" },
    { label: t.header.favorites, href: "/favorites" },
    { label: t.header.myListings, href: "/profile/items" },
    { label: t.header.messages, href: "/messages" },
    { label: t.header.business, href: "/items/new" },
    { label: t.header.help, href: "/help" },
  ];

  return (
    <header className="sticky top-0 z-40 bg-white/70 backdrop-blur-xl">
      <div className="container-avi flex h-[72px] items-center justify-between gap-2 sm:gap-5">
        <Link href="/" className="flex shrink-0 items-center gap-3">
          <div className="flex size-8 rotate-12 items-center justify-center rounded-[10px] bg-primary">
            <Zap className="size-5 -rotate-12 text-white" />
          </div>
          <span className="text-[26px] font-extrabold leading-none tracking-normal text-text-primary">
            Avi
          </span>
          {isDemoMode() && (
            <span className="rounded-full bg-primary/10 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-primary">
              <span className="sm:hidden">{t.demo.badgeShort}</span>
              <span className="hidden sm:inline">{t.demo.badge}</span>
            </span>
          )}
        </Link>

        <nav className="hidden items-center gap-9 2xl:flex">
          {navLinks.map((link) => (
            <Link
              key={link.label}
              href={link.href}
              className="text-[13px] font-semibold text-text-primary/85 transition-colors hover:text-primary"
            >
              {link.label}
            </Link>
          ))}
        </nav>

        <div className="ml-auto hidden 2xl:block" />

        <div className="hidden 2xl:block">
          <LanguageSwitcher />
        </div>

        <div className="hidden 2xl:block">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-10 gap-1.5 px-2 text-primary hover:bg-primary/5">
                <MapPin className="size-4" />
                <span className="text-[13px] font-semibold">{cityLabel}</span>
                <ChevronDown className="size-3.5 text-text-primary" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {cities.map((city) => (
                <DropdownMenuItem key={city.id} onClick={() => setSelectedCity(city)}>
                  {city.names?.[locale] || city.names?.en || city.slug}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div className="hidden items-center gap-4 2xl:flex">
          <Link href="/favorites">
            <Button variant="ghost" size="icon" className="size-10" aria-label={t.header.favorites}>
              <Heart className="size-5 text-text-primary" />
            </Button>
          </Link>
          <Link href="/messages">
            <Button variant="ghost" size="icon" className="size-10" aria-label={t.header.messages}>
              <MessageCircle className="size-5 text-text-primary" />
            </Button>
          </Link>
        </div>

        <Link href="/items/new" className="hidden 2xl:block">
          <Button className="h-11 rounded-[12px] px-6 text-[13px] font-bold shadow-[0_10px_24px_rgba(91,69,245,0.24)]">
            {t.header.postListing}
          </Button>
        </Link>

        <div className="hidden items-center gap-2 2xl:flex">
          {isAuthenticated && user ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-10 gap-2 px-2 hover:bg-transparent">
                  <div className="flex size-10 items-center justify-center rounded-full bg-[#f4ecdf] ring-2 ring-white">
                    <User className="size-4 text-text-primary" />
                  </div>
                  <ChevronDown className="size-3.5 text-text-primary" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <div className="px-2 py-1.5 text-sm text-text-secondary">{user.email}</div>
                <Link href="/profile">
                  <DropdownMenuItem>{t.header.profile}</DropdownMenuItem>
                </Link>
                <Link href="/profile/items">
                  <DropdownMenuItem>{t.header.myListings}</DropdownMenuItem>
                </Link>
                <DropdownMenuItem onClick={handleLogout} variant="destructive">
                  {t.auth.logOut}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <Link href="/login">
              <Button variant="default">{t.auth.logIn}</Button>
            </Link>
          )}
        </div>

        <div className="flex items-center gap-2 2xl:hidden">
          <LanguageSwitcher className="h-9 min-w-12" />
        </div>

        <Sheet
          modal={false}
          open={isMenuOpen}
          onOpenChange={(open) => {
            setIsMenuOpen(open);
            if (!open) {
              setIsCityMenuOpen(false);
            }
          }}
        >
          <SheetTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="ml-0 size-10 rounded-[14px] border-border bg-white/92 text-text-primary shadow-[0_8px_22px_rgba(44,42,84,0.08)] hover:border-primary/30 hover:bg-primary/5 hover:text-primary 2xl:hidden"
              aria-label={t.header.menu}
            >
              <Menu className="size-5" />
            </Button>
          </SheetTrigger>
          <SheetContent side="right" className="w-full p-0 sm:w-[380px]">
            <SheetHeader>
              <SheetTitle className="sr-only">{t.header.menu}</SheetTitle>
            </SheetHeader>
            {isCityMenuOpen ? (
              <div className="flex min-h-0 flex-1 flex-col px-5 pb-5 pt-4">
                <button
                  type="button"
                  onClick={() => setIsCityMenuOpen(false)}
                  className="mb-4 inline-flex w-fit items-center gap-2 rounded-lg px-1 py-2 text-sm font-semibold text-text-secondary transition-colors hover:text-primary"
                >
                  <ArrowLeft className="size-4" />
                  {t.header.menu}
                </button>
                <div className="mb-4">
                  <h2 className="text-section text-text-primary">{t.header.selectCity}</h2>
                  <p className="mt-1 text-meta text-text-secondary">{cityLabel}</p>
                </div>
                <div className="min-h-0 flex-1 overflow-y-scroll pr-2 [scrollbar-gutter:stable]">
                  <div className="space-y-2 pb-2">
                    {cities.map((city) => {
                      const name = city.names?.[locale] || city.names?.en || city.slug;
                      const isSelected = selectedCity?.id === city.id;

                      return (
                        <button
                          type="button"
                          key={city.id}
                          onClick={() => {
                            setSelectedCity(city);
                            setIsCityMenuOpen(false);
                          }}
                          className={cn(
                            "flex w-full items-center justify-between gap-3 rounded-control px-3 py-3 text-left text-body transition-colors",
                            isSelected
                              ? "bg-primary text-primary-foreground"
                              : "bg-surface-soft text-text-primary hover:bg-primary/5 hover:text-primary",
                          )}
                        >
                          <span className="min-w-0 truncate">{name}</span>
                          {isSelected && <Check className="size-4 shrink-0" />}
                        </button>
                      );
                    })}
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex flex-col gap-6 px-5 pb-6 pt-12">
                <Link href="/items/new" className="block">
                  <Button variant="default" className="w-full">
                    {t.header.postListing}
                  </Button>
                </Link>

                <button
                  type="button"
                  onClick={() => setIsCityMenuOpen(true)}
                  className="flex w-full items-center justify-between gap-3 rounded-control border border-border bg-surface-soft px-3 py-3 text-left transition-colors hover:border-primary/30 hover:bg-primary/5"
                >
                  <span className="flex min-w-0 items-center gap-3">
                    <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-white text-primary shadow-card">
                      <MapPin className="size-4" />
                    </span>
                    <span className="min-w-0">
                      <span className="block text-meta font-semibold text-text-secondary">
                        {t.header.selectCity}
                      </span>
                      <span className="block truncate text-body font-semibold text-text-primary">
                        {cityLabel}
                      </span>
                    </span>
                  </span>
                  <ChevronRight className="size-4 shrink-0 text-text-muted" />
                </button>

                <nav className="space-y-3 border-t border-border pt-6">
                  {navLinks.map((link) => (
                    <Link
                      key={link.label}
                      href={link.href}
                      className="block text-body text-text-primary transition-colors hover:text-primary"
                    >
                      {link.label}
                    </Link>
                  ))}
                </nav>

                <div className="border-t border-border pt-6">
                  {isAuthenticated && user ? (
                    <div className="flex flex-col gap-4">
                      <div className="flex items-center gap-3">
                        <div className="flex size-8 items-center justify-center rounded-full bg-surface-soft">
                          <User className="size-4" />
                        </div>
                        <span className="text-body text-text-primary break-all">{user.email}</span>
                      </div>
                      <Link
                        href="/profile"
                        className="text-body text-text-primary transition-colors hover:text-primary"
                      >
                        {t.header.profile}
                      </Link>
                      <Link
                        href="/profile/items"
                        className="text-body text-text-primary transition-colors hover:text-primary"
                      >
                        {t.header.myListings}
                      </Link>
                      <button
                        type="button"
                        onClick={handleLogout}
                        className="w-full text-left text-body text-destructive transition-colors hover:text-destructive/80"
                      >
                        {t.auth.logOut}
                      </button>
                    </div>
                  ) : (
                    <Link
                      href="/login"
                      className="flex w-full items-center gap-3 text-body text-text-primary transition-colors hover:text-primary"
                    >
                      <div className="flex size-8 items-center justify-center rounded-full bg-surface-soft">
                        <User className="size-4" />
                      </div>
                      <span>{t.auth.logIn}</span>
                    </Link>
                  )}
                </div>
              </div>
            )}
          </SheetContent>
        </Sheet>
      </div>
    </header>
  );
}
