import { BoxIcon, LogOutIcon, PanelLeftIcon } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { NavLink, Outlet, useLocation } from "react-router-dom"
import { reportSettingsRequestError } from "@/features/settings/settings-request-error-actions"

import { Separator } from "@/components/ui/separator"
import { Spinner } from "@/components/ui/spinner"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { footerItems, isNavigationItemActive, navigationGroups, primaryItems, type NavigationItem } from "@/app/navigation"
import { useAuth } from "@/features/auth/auth-context"
import { KernelStatusBadge } from "@/features/dashboard/kernel-status-badge"
import { AppearanceMenu } from "@/features/preferences/appearance-menu"
import { DefaultPasswordGate } from "@/features/settings/default-password-gate"
import { isDesktop } from "@/lib/api/desktop"

function navigationContext(pathname: string) {
  const primary = primaryItems.find((item) => isNavigationItemActive(item, pathname))
  if (primary) return { group: "", item: primary.label }
  for (const group of navigationGroups) {
    const item = group.items.find((candidate) => isNavigationItemActive(candidate, pathname))
    if (item) return { group: group.label, item: item.label }
  }
  const footer = footerItems.find((item) => isNavigationItemActive(item, pathname))
  return footer ? { group: "", item: footer.label } : { group: "", item: "" }
}

function NavItems({ items }: { items: NavigationItem[] }) {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  return (
    <SidebarMenu>
      {items.map((item) => (
        <SidebarMenuItem key={item.to}>
          <SidebarMenuButton
            render={<NavLink to={item.to} aria-label={t(item.label)} />}
            isActive={isNavigationItemActive(item, pathname)}
            tooltip={t(item.label)}
            className="h-9 rounded-xl px-2.5 text-sidebar-foreground/70 transition-colors duration-200 hover:text-sidebar-foreground data-active:bg-sidebar-accent data-active:text-sidebar-accent-foreground data-active:shadow-[inset_2px_0_0_var(--sidebar-primary)]"
          >
            <item.icon />
            <span className="group-data-[collapsible=icon]:hidden">{t(item.label)}</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      ))}
    </SidebarMenu>
  )
}

function AppSidebar() {
  const { t } = useTranslation()
  const auth = useAuth()
  const [loggingOut, setLoggingOut] = useState(false)
  const logout = async () => {
    setLoggingOut(true)
    try {
      await auth.logout()
    } catch (error) {
      reportSettingsRequestError(error, t, {
        scope: "logout",
        fallback: t("nav.logoutFailed"),
      })
    } finally {
      setLoggingOut(false)
    }
  }
  return (
    <Sidebar collapsible="icon" className="border-r border-sidebar-border/70">
      <SidebarHeader className="p-3">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size="lg"
              className="h-14 gap-3 rounded-2xl px-2 hover:bg-sidebar-accent/70"
              render={<NavLink to="/dashboard" aria-label="boxd" />}
              tooltip="boxd"
            >
              <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-sidebar-primary to-[oklch(0.64_0.16_166)] text-sidebar-primary-foreground shadow-[0_8px_24px_-10px_var(--sidebar-primary)]">
                <BoxIcon />
              </span>
              <span className="flex min-w-0 flex-col items-start group-data-[collapsible=icon]:hidden">
                <span className="truncate text-[0.95rem] font-semibold tracking-tight">boxd</span>
                <span className="truncate text-[0.68rem] tracking-[0.08em] text-sidebar-foreground/50 uppercase">sing-box control plane</span>
              </span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent className="gap-1 px-1 pb-2">
        <SidebarGroup className="py-1"><SidebarGroupContent><NavItems items={primaryItems} /></SidebarGroupContent></SidebarGroup>
        {navigationGroups.map((group) => (
          <SidebarGroup key={group.label} className="gap-1 py-1.5">
            <SidebarGroupLabel className="h-7 px-2.5 text-[0.62rem] font-semibold tracking-[0.16em] text-sidebar-foreground/40 uppercase">{t(group.label)}</SidebarGroupLabel>
            <SidebarGroupContent><NavItems items={group.items} /></SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter className="gap-2 border-t border-sidebar-border/70 p-3">
        <NavItems items={footerItems} />
        {isDesktop() ? null : (
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                disabled={loggingOut}
                tooltip={t("nav.logout")}
                aria-label={t("nav.logout")}
                className="h-9 rounded-xl px-2.5 text-sidebar-foreground/60 hover:text-sidebar-foreground"
                onClick={() => { void logout() }}
              >
                {loggingOut ? <Spinner aria-hidden="true" /> : <LogOutIcon />}
                <span className="group-data-[collapsible=icon]:hidden">{t("nav.logout")}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        )}
      </SidebarFooter>
    </Sidebar>
  )
}

export function AppShell() {
  const { pathname } = useLocation()
  const { t } = useTranslation()
  const context = navigationContext(pathname)
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className="min-w-0 overflow-hidden">
        <header className="sticky top-0 z-10 flex h-16 items-center gap-3 border-b border-border/70 bg-background/75 px-4 shadow-[0_1px_0_oklch(0.18_0.04_140/0.03)] backdrop-blur-xl md:px-6">
          <SidebarTrigger className="rounded-lg"><PanelLeftIcon /></SidebarTrigger>
          <Separator orientation="vertical" className="h-6 bg-border/70" />
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="flex items-center gap-1.5 truncate text-[0.65rem] font-semibold tracking-[0.15em] text-muted-foreground uppercase">
              <span className="size-1.5 rounded-full bg-primary" aria-hidden="true" />
              {context.group ? t(context.group) : "boxd"}
            </span>
            <span className="truncate text-sm font-semibold tracking-tight">{context.item ? t(context.item) : "boxd"}</span>
          </div>
          <KernelStatusBadge />
          <Separator orientation="vertical" className="hidden h-6 bg-border/70 sm:block" />
          <AppearanceMenu />
        </header>
        <main className="app-shell-main mx-auto flex min-w-0 w-full max-w-[1600px] flex-1 flex-col gap-4 p-4 md:gap-5 md:p-6 lg:p-8">
          {isDesktop() ? <Outlet /> : <DefaultPasswordGate><Outlet /></DefaultPasswordGate>}
        </main>
      </SidebarInset>
    </SidebarProvider>
  )
}
