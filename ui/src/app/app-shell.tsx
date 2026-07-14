import { LogOutIcon, PanelLeftIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { NavLink, Outlet, useLocation } from "react-router-dom"
import { toast } from "sonner"

import { Separator } from "@/components/ui/separator"
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
import { useAuth } from "@/features/auth/auth-context"
import { footerItems, navigationGroups, primaryItems, type NavigationItem } from "@/app/navigation"

function NavItems({ items }: { items: NavigationItem[] }) {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  return (
    <SidebarMenu>
      {items.map((item) => (
        <SidebarMenuItem key={item.to}>
          <SidebarMenuButton render={<NavLink to={item.to} />} isActive={pathname === item.to} tooltip={t(item.label)}>
            <item.icon />
            <span>{t(item.label)}</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      ))}
    </SidebarMenu>
  )
}

function AppSidebar() {
  const { t } = useTranslation()
  const auth = useAuth()
  return (
    <Sidebar collapsible="icon">
      <SidebarHeader><NavItems items={primaryItems} /></SidebarHeader>
      <SidebarContent>
        {navigationGroups.map((group) => (
          <SidebarGroup key={group.label}>
            <SidebarGroupLabel>{t(group.label)}</SidebarGroupLabel>
            <SidebarGroupContent><NavItems items={group.items} /></SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <NavItems items={footerItems} />
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton onClick={() => { void auth.logout().catch((error: Error) => toast.error(error.message)) }}>
              <LogOutIcon />
              <span>{t("nav.logout")}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  )
}

export function AppShell() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-14 items-center gap-3 px-4">
          <SidebarTrigger><PanelLeftIcon /></SidebarTrigger>
          <Separator orientation="vertical" className="h-5" />
          <span className="font-medium">BoxUI</span>
        </header>
        <main className="flex flex-1 flex-col gap-4 p-4"><Outlet /></main>
      </SidebarInset>
    </SidebarProvider>
  )
}
