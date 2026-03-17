import { Outlet, NavLink, useParams, Link } from "react-router";
import { useProject } from "@/hooks/use-projects";
import { useAuthStore } from "@/stores/auth";
import { useLogout } from "@/hooks/use-auth";
import InfoIcon from "@/components/info-icon";
import * as help from "@/lib/help-content";

const navItems = [
  { to: "alerts", label: "Alerts", help: help.navAlerts },
  { to: "keywords", label: "Keywords", help: help.navKeywords },
  { to: "sources", label: "Sources", help: help.navSources },
  { to: "processing", label: "Processing", help: help.navProcessing },
  { to: "rules", label: "Rules", help: help.navRules },
  { to: "connections", label: "Connections", help: help.navConnections },
  { to: "posts", label: "Posts", help: help.navPosts },
  { to: "members", label: "Members", help: help.navMembers },
  { to: "settings", label: "Settings", help: help.navSettings },
];

export default function Layout() {
  const { id } = useParams<{ id: string }>();
  const { data: project } = useProject(id!);
  const user = useAuthStore((s) => s.user);
  const logout = useLogout();

  return (
    <div className="flex h-screen flex-col">
      {/* Topbar */}
      <header className="flex h-14 shrink-0 items-center justify-between border-b bg-white px-4">
        <div className="flex items-center gap-3">
          <Link to="/projects" className="text-sm text-gray-500 hover:text-gray-700">
            &larr; Projects
          </Link>
          <span className="text-gray-300">/</span>
          <span className="font-semibold">{project?.name ?? "..."}</span>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-gray-500">{user?.email}</span>
          <button
            onClick={logout}
            className="rounded border px-3 py-1 text-sm text-gray-600 hover:bg-gray-50"
          >
            Logout
          </button>
        </div>
      </header>

      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <nav className="w-52 shrink-0 overflow-y-auto border-r bg-gray-50 p-3">
          <ul className="space-y-1">
            {navItems.map((item) => (
              <li key={item.to}>
                <div className="flex items-center">
                  <NavLink
                    to={item.to}
                    className={({ isActive }) =>
                      `flex-1 rounded px-3 py-2 text-sm ${
                        isActive
                          ? "bg-blue-50 font-medium text-blue-700"
                          : "text-gray-700 hover:bg-gray-100"
                      }`
                    }
                  >
                    {item.label}
                  </NavLink>
                  <InfoIcon tooltip={item.help.tooltip}>{item.help.modal}</InfoIcon>
                </div>
              </li>
            ))}
          </ul>
        </nav>

        {/* Content */}
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
