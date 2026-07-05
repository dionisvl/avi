"use client";

import { Home, MessageCircle, PlusCircle, Search, User } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useT } from "@/i18n";
import { cn } from "@/lib/utils";

/**
 * Mobile bottom navigation bar.
 *
 * IMPORTANT: Pages using this component should reserve bottom space on mobile
 * by adding `pb-[56px] md:pb-0` or `pb-[64px] md:pb-0` (depending on height)
 * to the main content container to prevent content from being hidden behind the nav.
 *
 * This component is hidden on lg and above (desktop).
 */
export function BottomNav() {
  const t = useT();
  const pathname = usePathname();

  const items = [
    { key: "home", label: t.bottomNav.home, icon: Home, href: "/" },
    { key: "search", label: t.bottomNav.search, icon: Search, href: "/items" },
    {
      key: "post",
      label: t.bottomNav.post,
      icon: PlusCircle,
      href: "/items/new",
      isAccent: true,
    },
    {
      key: "messages",
      label: t.bottomNav.messages,
      icon: MessageCircle,
      href: "/messages",
    },
    { key: "profile", label: t.bottomNav.profile, icon: User, href: "/profile" },
  ];

  const isActive = (href: string): boolean => {
    if (href === "/") {
      return pathname === "/";
    }
    return pathname.startsWith(href);
  };

  return (
    <nav
      className={cn(
        "fixed bottom-0 inset-x-0 z-40 lg:hidden",
        "bg-surface border-t border-border shadow-card",
        "pb-[env(safe-area-inset-bottom)]",
      )}
      aria-label="Mobile navigation"
    >
      {/* Navigation items: 5 columns with equal width, centered vertically */}
      <div className="flex h-14 md:h-16 justify-around items-center px-2">
        {items.map((item) => {
          const active = isActive(item.href);
          const Icon = item.icon;

          return (
            <Link
              key={item.key}
              href={item.href}
              className={cn(
                "flex flex-col items-center justify-center gap-1 py-2 px-2 transition-colors",
                item.isAccent
                  ? "text-primary hover:text-primary/80"
                  : cn(active ? "text-primary" : "text-text-secondary", "hover:text-primary"),
              )}
              aria-current={active ? "page" : undefined}
            >
              <Icon className="size-[22px]" aria-hidden="true" />
              <span className="text-meta font-medium">{item.label}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
