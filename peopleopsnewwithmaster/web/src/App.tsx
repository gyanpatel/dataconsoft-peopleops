import * as React from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Activity,
  ArrowDownRight,
  ArrowRight,
  ArrowUpRight,
  Bell,
  CalendarDays,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  CircleHelp,
  Clock3,
  Download,
  FileCheck2,
  FileText,
  LayoutDashboard,
  LogOut,
  Menu,
  MoreHorizontal,
  Search,
  Settings,
  ShieldCheck,
  Users,
  Wallet,
  X,
  BriefcaseBusiness,
  CalendarCheck,
  CircleDollarSign,
  Building2,
  Plus,
  CheckCircle2,
  XCircle,
  PanelLeftClose,
  UserRound,
  Sparkles,
} from "lucide-react";

type User = {
  id: string;
  email: string;
  name: string;
  role: string;
  employeeId?: string;
  pages?: string[];
  permissions?: string[];
};
type Employee = {
  id: string;
  employeeNumber: string;
  firstName: string;
  lastName: string;
  email: string;
  department: string;
  jobTitle: string;
  location: string;
  employmentType: string;
  status: string;
  startDate: string;
  salaryMinor: number;
  currency: string;
  customFields?: Record<string, any>;
  photoId?: string;
  roleName?: string;
  managerId?: string;
};
type Leave = {
  id: string;
  employeeId: string;
  employeeName: string;
  leaveType: string;
  startDate: string;
  endDate: string;
  days: number;
  reason: string;
  status: string;
  submittedAt: string;
  decisionNote: string;
};
type Metrics = {
  employees: number;
  activeEmployees: number;
  pendingApprovals: number;
  departments: number;
  payrollNetMinor: number;
  payrollPeriod: string;
  payrollStatus: string;
};
const API = "/api/v1";
async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (!headers.has("Content-Type"))
    headers.set("Content-Type", "application/json");
  const res = await fetch(API + path, {
    ...init,
    credentials: "include",
    headers,
  });
  if (!res.ok) {
    let msg = `Request failed (${res.status})`;
    try {
      const j = await res.json();
      msg = j.error?.message || msg;
    } catch {}
    throw new Error(msg);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}
const money = (minor: number) =>
  new Intl.NumberFormat("en-GB", {
    style: "currency",
    currency: "GBP",
    maximumFractionDigits: 0,
  }).format(minor / 100);
const initials = (name: string) =>
  name
    .split(" ")
    .map((x) => x[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();
const nav = [
  {
    label: "OVERVIEW",
    items: [
      ["Dashboard", LayoutDashboard],
      ["Employees", Users],
      ["Employee hierarchy", Users],
      ["Attendance", Clock3],
      ["Shift management", CalendarCheck],
      ["Leave management", CalendarDays],
      ["Leave Balance", CalendarDays],
    ],
  },
  {
    label: "OPERATIONS",
    items: [
      ["Payroll", Wallet],
      ["Documents", FileText],
      ["Approvals", FileCheck2],
      ["Reports & analytics", Activity],
    ],
  },
  {
    label: "SYSTEM",
    items: [
      ["Audit log", ShieldCheck],
      ["Settings", Settings],
    ],
  },
] as const;
export default function App() {
  const [user, setUser] = useState<User | null>(null),
    [mustChangePassword, setMustChangePassword] = useState(false),
    [loginEmail, setLoginEmail] = useState(""),
    [password, setPassword] = useState(""),
    [loginBusy, setLoginBusy] = useState(false),
    [loginError, setLoginError] = useState("");
  const [page, setPage] = useState("Dashboard"),
    [sidebarOpen, setSidebarOpen] = useState(true),
    [search, setSearch] = useState(""),
    [employees, setEmployees] = useState<Employee[]>([]),
    [leaves, setLeaves] = useState<Leave[]>([]),
    [metrics, setMetrics] = useState<Metrics | null>(null),
    [payroll, setPayroll] = useState<any[]>([]),
    [attendance, setAttendance] = useState<any[]>([]),
    [audit, setAudit] = useState<any[]>([]),
    [shifts, setShifts] = useState<any[]>([]),
    [shiftAssignments, setShiftAssignments] = useState<any[]>([]),
    [actualHours, setActualHours] = useState<any[]>([]),
    [shiftForm, setShiftForm] = useState({
      name: "",
      startTime: "09:00",
      endTime: "17:00",
      breakMinutes: 30,
      graceMinutes: 5,
      workDays: "Mon,Tue,Wed,Thu,Fri",
    }),
    [assignmentForm, setAssignmentForm] = useState({
      shiftId: "",
      employeeId: "",
      date: new Date().toISOString().slice(0, 10),
      note: "",
    }),
    [employeeFields, setEmployeeFields] = useState<any[]>([]),
    [documents, setDocuments] = useState<any[]>([]),
    [loading, setLoading] = useState(false),
    [error, setError] = useState(""),
    [toast, setToast] = useState(""),
    [modal, setModal] = useState(""),
    [decisionNote, setDecisionNote] = useState(""),
    [selectedLeave, setSelectedLeave] = useState<Leave | null>(null),
    [selectedEmployee, setSelectedEmployee] = useState<Employee | null>(null),
    [mobileMenu, setMobileMenu] = useState(false);
  const load = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    setError("");
    try {
      const d = await request<any>("/dashboard");
      const l = user.pages?.includes("leave-management")
        ? await request<any>("/leave-requests")
        : { items: [] };
      const e = user.pages?.includes("employees")
        ? await request<any>("/employees")
        : { items: [] };
      const cfg = user.pages?.includes("employees")
        ? await request<any>("/app-master?page=employees")
        : { items: [] };
      setMetrics(d.metrics);
      setEmployees(e.items || []);
      setLeaves(l.items || []);
      if (cfg.items?.length) setEmployeeFields(cfg.items);
      if (page === "Documents") {
        const docs = await request<any>("/documents");
        setDocuments(docs.items || []);
      }
      if (["Payroll"].includes(page)) {
        const p = await request<any>("/payroll");
        setPayroll(p.items);
      }
      if (page === "Attendance") {
        const a = await request<any>("/attendance");
        setAttendance(a.items);
      }
      if (page === "Shift management") {
        const [sh, as, ah] = await Promise.all([
          request<any>("/shifts"),
          request<any>("/shift-assignments"),
          request<any>("/actual-hours"),
        ]);
        setShifts(sh.items);
        setShiftAssignments(as.items);
        setActualHours(ah.items || []);
        if (sh.items.length)
          setAssignmentForm((f) => ({
            ...f,
            shiftId: f.shiftId || sh.items[0].id,
          }));
        if (e.items.length)
          setAssignmentForm((f) => ({
            ...f,
            employeeId: f.employeeId || e.items[0].id,
          }));
      }
      if (page === "Audit log") {
        const a = await request<any>("/audit");
        setAudit(a.items);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unable to load data");
    } finally {
      setLoading(false);
    }
  }, [user, page]);
  useEffect(() => {
    request<{ user: User; mustChangePassword?: boolean }>("/auth/me")
      .then((x) => {
        setUser(x.user);
        setMustChangePassword(!!x.mustChangePassword);
      })
      .catch(() => setUser(null));
  }, []);
  useEffect(() => {
    load();
  }, [load]);
  useEffect(() => {
    if (!toast) return;
    const id = window.setTimeout(() => setToast(""), 3500);
    return () => window.clearTimeout(id);
  }, [toast]);
  async function login(e: React.FormEvent) {
    e.preventDefault();
    setLoginBusy(true);
    setLoginError("");
    try {
      const x = await request<{ user: User; mustChangePassword?: boolean }>(
        "/auth/login",
        {
          method: "POST",
          body: JSON.stringify({ email: loginEmail, password }),
        }
      );
      setUser(x.user);
      setMustChangePassword(!!x.mustChangePassword);
      setPage("Dashboard");
    } catch (e) {
      setLoginError(e instanceof Error ? e.message : "Could not sign in");
    } finally {
      setLoginBusy(false);
    }
  }
  async function logout() {
    try {
      await request("/auth/logout", { method: "POST" });
    } catch {}
    setUser(null);
    setPassword("");
  }
  async function punch(type: "check-in" | "check-out") {
    try {
      await request(`/attendance/${type}`, { method: "POST", body: "{}" });
      setToast(
        type === "check-in" ? "You are checked in" : "You are checked out"
      );
      await load();
    } catch (e) {
      setToast(e instanceof Error ? e.message : "Attendance action failed");
    }
  }
  async function decide(item: Leave, decision: "Approved" | "Rejected") {
    try {
      await request(`/leave-requests/${item.id}/decision`, {
        method: "POST",
        body: JSON.stringify({ decision, note: decisionNote }),
      });
      setToast(`Request ${decision.toLowerCase()}`);
      setModal("");
      setSelectedLeave(null);
      setDecisionNote("");
      await load();
    } catch (e) {
      setToast(e instanceof Error ? e.message : "Decision failed");
    }
  }
  const filtered = useMemo(
    () =>
      employees.filter((e) =>
        `${e.firstName} ${e.lastName} ${e.email} ${e.department} ${e.employeeNumber}`
          .toLowerCase()
          .includes(search.toLowerCase())
      ),
    [employees, search]
  );
  const isApprover = !!user?.permissions?.includes("leave.review");
  if (!user)
    return (
      <div className="login-shell">
        <div className="login-art">
          <div className="brand">
            <span className="brand-mark">
              <Building2 size={20} />
            </span>
            <span>
              people<span className="brand-light">ops</span>
            </span>
          </div>
          <div className="login-message">
            <div className="eyebrow">PEOPLE FIRST. OPERATIONS SIMPLIFIED.</div>
            <h1>
              Make work
              <br />
              <span>work better.</span>
            </h1>
            <p>
              A connected home for your people, payroll and everyday HR
              operations.
            </p>
            <div className="login-mini">
              <div className="mini-avatars">
                <i>AM</i>
                <i>PS</i>
                <i>JL</i>
                <i>+8</i>
              </div>
              <div>
                <strong>Your people, in sync.</strong>
                <small>One workspace. Less busywork.</small>
              </div>
            </div>
          </div>
          <div className="login-foot">
            PEOPLEOPS HR PLATFORM <span>•</span> BUILT FOR PEOPLE
          </div>
        </div>
        <div className="login-panel">
          <form className="login-form" onSubmit={login}>
            <div className="mobile-brand brand">
              <span className="brand-mark">
                <Building2 size={20} />
              </span>
              <span>
                people<span className="brand-light">ops</span>
              </span>
            </div>
            <div className="login-welcome">
              <h2>Welcome back</h2>
              <p>Enter your details to access your workspace.</p>
            </div>
            {loginError && <div className="error-banner">{loginError}</div>}
            <label>
              Email address
              <input
                type="email"
                autoComplete="username"
                value={loginEmail}
                onChange={(e) => setLoginEmail(e.target.value)}
                required
              />
            </label>
            <label>
              Password
              <div className="password-wrap">
                <input
                  type="password"
                  autoComplete="current-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
                <ShieldCheck size={17} />
              </div>
            </label>
            <div className="login-options">
              <label className="check-label">
                <input type="checkbox" defaultChecked /> Remember me
              </label>
              <button
                type="button"
                className="text-button"
                onClick={() =>
                  setLoginError(
                    "Contact your PeopleOps administrator to reset your password."
                  )
                }
              >
                Forgot password?
              </button>
            </div>
            <button className="primary full" disabled={loginBusy}>
              {loginBusy ? "Signing in…" : "Sign in"} <ArrowRight size={16} />
            </button>
            <div className="login-legal">
              Protected workspace · Access is monitored and audited.
            </div>
          </form>
        </div>
      </div>
    );
  const setNav = (p: string) => {
    setPage(p);
    setMobileMenu(false);
    setSearch("");
  };
  if (user && mustChangePassword)
    return (
      <div className="login-shell">
        <div className="login-panel">
          <form
            className="login-form"
            onSubmit={async (e) => {
              e.preventDefault();
              const f = e.currentTarget;
              const data = new FormData(f);
              try {
                await request("/auth/change-password", {
                  method: "POST",
                  body: JSON.stringify({
                    currentPassword: String(data.get("currentPassword") || ""),
                    newPassword: String(data.get("newPassword") || ""),
                  }),
                });
                setMustChangePassword(false);
                setToast("Password changed successfully");
              } catch (err) {
                setLoginError(
                  err instanceof Error
                    ? err.message
                    : "Could not change password"
                );
              }
            }}
          >
            <h1>Change your password</h1>
            <p>
              Your temporary password must be changed before using PeopleOps.
            </p>
            {loginError && <div className="error-banner">{loginError}</div>}
            <label>
              Temporary password
              <input
                name="currentPassword"
                type="password"
                required
                autoComplete="current-password"
              />
            </label>
            <label>
              New password (at least 12 characters)
              <input
                name="newPassword"
                type="password"
                minLength={12}
                required
                autoComplete="new-password"
              />
            </label>
            <button className="primary">Update password</button>
          </form>
        </div>
      </div>
    );
  return (
    <div className="app-shell">
      <aside
        className={`sidebar ${sidebarOpen ? "" : "collapsed"} ${
          mobileMenu ? "mobile-open" : ""
        }`}
      >
        <div className="side-brand">
          <span className="brand-mark">
            <Building2 size={20} />
          </span>
          {sidebarOpen && (
            <span>
              people<span className="brand-light">ops</span>
            </span>
          )}
          <button
            className="collapse-btn"
            onClick={() => setSidebarOpen(!sidebarOpen)}
            title="Collapse navigation"
          >
            <PanelLeftClose size={17} />
          </button>
        </div>
        <div className="workspace-label">{sidebarOpen ? "WORKSPACE" : ""}</div>
        <nav>
          {nav.map((group) => (
            <div className="nav-group" key={group.label}>
              <div className="nav-label">{sidebarOpen ? group.label : ""}</div>
              {group.items
                .filter(
                  ([label]) =>
                    label === "Employee hierarchy" ||
                    !user.pages ||
                    user.pages.includes(pageIdForLabel(label))
                )
                .map(([label, Icon]) => (
                  <button
                    key={label}
                    className={`nav-item ${page === label ? "active" : ""}`}
                    onClick={() => setNav(label)}
                    title={label}
                  >
                    <Icon size={18} />
                    {sidebarOpen && <span>{label}</span>}
                    {sidebarOpen &&
                    label === "Approvals" &&
                    metrics?.pendingApprovals ? (
                      <span className="nav-count">
                        {metrics.pendingApprovals}
                      </span>
                    ) : null}
                  </button>
                ))}
            </div>
          ))}
        </nav>
        <div className="side-bottom">
          <div className="help-card">
            <div className="help-icon">
              <CircleHelp size={19} />
            </div>
            {sidebarOpen && (
              <>
                <strong>Need a hand?</strong>
                <p>We're here if you need help.</p>
                <button
                  onClick={() =>
                    setToast(
                      "Help centre: contact your PeopleOps administrator."
                    )
                  }
                >
                  Visit help centre <ArrowRight size={13} />
                </button>
              </>
            )}
          </div>
          <button className="user-card" onClick={() => setModal("profile")}>
            <div className="avatar avatar-purple">{initials(user.name)}</div>
            {sidebarOpen && (
              <>
                <span className="user-details">
                  <b>{user.name}</b>
                  <small>{user.role}</small>
                </span>
                <MoreHorizontal size={17} />
              </>
            )}
          </button>
        </div>
      </aside>
      <main className="main-area">
        <header className="topbar">
          <div className="topbar-left">
            <button
              className="mobile-menu"
              onClick={() => setMobileMenu(!mobileMenu)}
            >
              <Menu size={20} />
            </button>
            <span className="breadcrumb-muted">Workspace</span>
            <ChevronRight size={14} />
            <strong>{page}</strong>
          </div>
          <div className="topbar-right">
            <div className="global-search">
              <Search size={16} />
              <input
                placeholder="Search people, reports, settings…"
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  if (page !== "Employees") setPage("Employees");
                }}
              />
              <kbd>⌘ K</kbd>
            </div>
            <button
              className="icon-btn notification"
              title="Notifications"
              onClick={() => setToast("You’re all caught up.")}
            >
              <Bell size={18} />
              <i />
            </button>
            <button
              className="icon-btn"
              title="Settings"
              onClick={() => setNav("Settings")}
            >
              <Settings size={18} />
            </button>
            <div className="top-user">
              <div className="avatar avatar-photo">{initials(user.name)}</div>
              <div>
                <b>{user.name}</b>
                <small>{user.role}</small>
              </div>
              <ChevronDown size={14} />
            </div>
            <button className="logout-btn" title="Sign out" onClick={logout}>
              <LogOut size={17} />
            </button>
          </div>
        </header>
        <div className="page-content">
          {error && (
            <div className="error-banner page-error">
              {error}
              <button onClick={load}>Retry</button>
            </div>
          )}
          {loading && <div className="loading-line" />}
          {page === "Dashboard" && (
            <>
              <div className="page-heading">
                <div>
                  <div className="eyebrow green-eyebrow">
                    WEDNESDAY, 14 OCTOBER 2026
                  </div>
                  <h1>
                    Good morning, {user.name.split(" ")[0]}{" "}
                    <span className="wave">✦</span>
                  </h1>
                  <p>Here’s what’s happening across your workspace today.</p>
                </div>
                <div className="heading-actions">
                  <button
                    className="secondary"
                    onClick={() => setNav("Reports & analytics")}
                  >
                    <Activity size={16} /> View reports
                  </button>
                  {user.permissions?.includes("employees.manage") && (
                    <button
                      className="primary"
                      onClick={() => setModal("employee")}
                    >
                      <Plus size={17} /> Add employee
                    </button>
                  )}
                </div>
              </div>
              {user.permissions?.includes("employees.manage") && (
                <div className="metrics-grid">
                  <Metric
                    icon={<Users />}
                    label="Total employees"
                    value={metrics?.employees ?? "—"}
                    note={`${metrics?.activeEmployees ?? "—"} active employees`}
                    tone="green"
                    trend="+4.8%"
                  />
                  <Metric
                    icon={<CalendarCheck />}
                    label="Pending approvals"
                    value={metrics?.pendingApprovals ?? "—"}
                    note="Requests need your attention"
                    tone="orange"
                    trend="Review"
                  />
                  <Metric
                    icon={<Building2 />}
                    label="Departments"
                    value={metrics?.departments ?? "—"}
                    note="Across your organisation"
                    tone="blue"
                    trend="Stable"
                  />
                  <Metric
                    icon={<Wallet />}
                    label="Latest net payroll"
                    value={metrics ? money(metrics.payrollNetMinor) : "—"}
                    note={metrics?.payrollPeriod || "Latest payroll period"}
                    tone="purple"
                    trend={metrics?.payrollStatus || "—"}
                  />
                </div>
              )}
              <div className="dashboard-grid">
                <section className="panel overview-panel">
                  <div className="panel-heading">
                    <div>
                      <h3>Workforce overview</h3>
                      <p>Employee distribution by department</p>
                    </div>
                    <button
                      className="subtle-btn"
                      onClick={() => setNav("Employees")}
                    >
                      View directory <ArrowRight size={14} />
                    </button>
                  </div>
                  <div className="chart-layout">
                    <div className="donut-wrap">
                      <div className="donut">
                        <div className="donut-center">
                          <strong>{metrics?.employees ?? "—"}</strong>
                          <span>Employees</span>
                        </div>
                      </div>
                    </div>
                    <div className="dept-legend">
                      {departmentRows(employees)
                        .slice(0, 5)
                        .map((d, i) => (
                          <div className="legend-row" key={d.name}>
                            <span className={`legend-dot dot-${i % 5}`} />
                            <span>{d.name}</span>
                            <b>{d.count}</b>
                            <small>
                              {metrics?.employees
                                ? Math.round(
                                    (d.count / metrics.employees) * 100
                                  )
                                : 0}
                              %
                            </small>
                          </div>
                        ))}
                    </div>
                  </div>
                  <div className="panel-foot green-foot">
                    <span>
                      <ArrowUpRight size={16} /> Team overview updated
                    </span>
                    <button onClick={() => setNav("Reports & analytics")}>
                      Explore analytics <ArrowRight size={14} />
                    </button>
                  </div>
                </section>
                <section className="panel activity-panel">
                  <div className="panel-heading">
                    <div>
                      <h3>Quick actions</h3>
                      <p>Common tasks, right where you need them</p>
                    </div>
                    <MoreHorizontal size={19} />
                  </div>
                  <div className="quick-actions">
                    {user.permissions?.includes("employees.manage") && (
                      <Quick
                        icon={<UserRound />}
                        label="Add employee"
                        desc="Create a profile"
                        onClick={() => setModal("employee")}
                      />
                    )}
                    <Quick
                      icon={<CalendarDays />}
                      label="Review leave"
                      desc={`${metrics?.pendingApprovals ?? 0} awaiting review`}
                      onClick={() => setNav("Approvals")}
                    />
                    <Quick
                      icon={<Clock3 />}
                      label="Attendance"
                      desc="View time records"
                      onClick={() => setNav("Attendance")}
                    />
                    <Quick
                      icon={<FileText />}
                      label="Payroll"
                      desc="View pay periods"
                      onClick={() => setNav("Payroll")}
                    />
                  </div>
                  <div className="assistant-strip">
                    <div className="assistant-icon">
                      <Sparkles size={17} />
                    </div>
                    <div>
                      <b>PeopleOps assistant</b>
                      <p>Need a hand with an HR task?</p>
                    </div>
                    <button
                      onClick={() =>
                        setToast(
                          "Assistant is not configured. Add an approved AI provider to enable it."
                        )
                      }
                      aria-label="Open assistant"
                    >
                      <ArrowRight size={17} />
                    </button>
                  </div>
                </section>
                <section className="panel employees-panel">
                  <div className="panel-heading">
                    <div>
                      <h3>People at a glance</h3>
                      <p>Your latest employee records</p>
                    </div>
                    <button
                      className="subtle-btn"
                      onClick={() => setNav("Employees")}
                    >
                      View all <ArrowRight size={14} />
                    </button>
                  </div>
                  <EmployeeTable
                    items={employees.slice(0, 5)}
                    compact
                    onSelect={() => setNav("Employees")}
                  />
                </section>
                <section className="panel approvals-panel">
                  <div className="panel-heading">
                    <div>
                      <h3>Needs your attention</h3>
                      <p>Leave requests awaiting a decision</p>
                    </div>
                    <span className="count-pill">
                      {leaves.filter((l) => l.status === "Pending").length}{" "}
                      pending
                    </span>
                  </div>
                  {leaves
                    .filter((l) => l.status === "Pending")
                    .slice(0, 3)
                    .map((l) => (
                      <div className="approval-row" key={l.id}>
                        <div className="avatar avatar-small">
                          {initials(l.employeeName)}
                        </div>
                        <div className="approval-main">
                          <b>{l.employeeName}</b>
                          <span>
                            {l.leaveType} · {l.days}{" "}
                            {l.days === 1 ? "day" : "days"}
                          </span>
                          <small>
                            {l.startDate} – {l.endDate}
                          </small>
                        </div>
                        {isApprover ? (
                          <button
                            className="review-btn"
                            onClick={() => {
                              setSelectedLeave(l);
                              setModal("decision");
                            }}
                          >
                            Review
                          </button>
                        ) : (
                          <span className="status status-pending">Pending</span>
                        )}
                      </div>
                    ))}
                  {!leaves.some((l) => l.status === "Pending") && (
                    <div className="empty-state">
                      <CheckCircle2 size={24} />
                      <span>All caught up. No pending requests.</span>
                    </div>
                  )}
                  <button
                    className="panel-bottom-link"
                    onClick={() => setNav("Approvals")}
                  >
                    Go to approvals inbox <ArrowRight size={14} />
                  </button>
                </section>
              </div>
            </>
          )}
          {page === "Employee hierarchy" && (
            <EmployeeHierarchy employees={employees} />
          )}
          {page === "Employees" && user.pages?.includes("employees") && (
            <>
              <div className="page-heading">
                <div>
                  <div className="eyebrow green-eyebrow">PEOPLE DIRECTORY</div>
                  <h1>Employees</h1>
                  <p>
                    Manage your people, profiles and organisational details.
                  </p>
                </div>
                {user.permissions?.includes("employees.manage") && (
                  <button
                    className="primary"
                    onClick={() => setModal("employee")}
                  >
                    <Plus size={17} /> Add employee
                  </button>
                )}
              </div>
              <section className="panel directory-panel">
                <div className="directory-tools">
                  <div className="directory-count">
                    <strong>All employees</strong>
                    <span>{filtered.length} records</span>
                  </div>
                  <div className="table-tools">
                    <div className="search-field">
                      <Search size={16} />
                      <input
                        placeholder="Search employees…"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                      />
                    </div>
                    <button
                      className="secondary"
                      onClick={() =>
                        (window.location.href = API + "/reports/headcount.csv")
                      }
                    >
                      <Download size={15} /> Export CSV
                    </button>
                  </div>
                </div>
                <EmployeeTable
                  items={filtered}
                  onSelect={(e) => {
                    setSelectedEmployee(e);
                    setModal("employee-details");
                  }}
                />
              </section>
            </>
          )}
          {page === "Attendance" && (
            <>
              <div className="page-heading">
                <div>
                  <div className="eyebrow green-eyebrow">TIME & ATTENDANCE</div>
                  <h1>Attendance</h1>
                  <p>Record attendance and review recent time events.</p>
                </div>
                <div className="heading-actions">
                  <button
                    className="secondary"
                    onClick={() => punch("check-out")}
                  >
                    <Clock3 size={16} /> Check out
                  </button>
                  <button className="primary" onClick={() => punch("check-in")}>
                    <Check size={16} /> Check in
                  </button>
                </div>
              </div>
              <div className="metrics-grid">
                <Metric
                  icon={<Clock3 />}
                  label="Events loaded"
                  value={attendance.length}
                  note="Most recent 100 events"
                  tone="green"
                />
                <Metric
                  icon={<CalendarDays />}
                  label="Today"
                  value={new Date().toLocaleDateString("en-GB", {
                    day: "2-digit",
                    month: "short",
                  })}
                  note="Local date"
                  tone="blue"
                />
                <Metric
                  icon={<ShieldCheck />}
                  label="Source"
                  value="Web"
                  note="Server timestamp recorded"
                  tone="purple"
                />
              </div>
              <section className="panel directory-panel">
                <div className="panel-heading">
                  <div>
                    <h3>Recent attendance events</h3>
                    <p>Newest events first</p>
                  </div>
                  <button className="subtle-btn" onClick={load}>
                    Refresh <Activity size={14} />
                  </button>
                </div>
                <div className="table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Employee</th>
                        <th>Event</th>
                        <th>Date & time (UTC)</th>
                        <th>Source</th>
                        <th>Note</th>
                      </tr>
                    </thead>
                    <tbody>
                      {attendance.map((a) => (
                        <tr key={a.id}>
                          <td>
                            <b>{a.employeeName}</b>
                          </td>
                          <td>
                            <span
                              className={`status ${
                                a.type === "check_in"
                                  ? "status-active"
                                  : "status-neutral"
                              }`}
                            >
                              {a.type === "check_in" ? "Check in" : "Check out"}
                            </span>
                          </td>
                          <td>
                            {new Date(a.occurredAt).toLocaleString("en-GB", {
                              timeZone: "UTC",
                            })}
                          </td>
                          <td>{a.source}</td>
                          <td>{a.note || "—"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {attendance.length === 0 && (
                    <div className="empty-state">
                      No attendance events recorded yet.
                    </div>
                  )}
                </div>
              </section>
              <p className="privacy-note">
                <ShieldCheck size={15} /> Attendance timestamps are
                server-recorded. GPS and geofence validation are not enabled in
                this starter build.
              </p>
            </>
          )}
          {page === "Shift management" && (
            <>
              <div className="page-heading">
                <div>
                  <div className="eyebrow green-eyebrow">
                    ROSTERS & SCHEDULING
                  </div>
                  <h1>Shift management</h1>
                  <p>
                    Create reusable shift templates and assign employees to
                    specific work dates.
                  </p>
                </div>
                <span className="read-only-tag">
                  <CalendarCheck size={14} /> Scheduling workspace
                </span>
              </div>
              <div className="metrics-grid">
                <Metric
                  icon={<CalendarCheck />}
                  label="Shift templates"
                  value={shifts.length}
                  note="Configured schedules"
                  tone="green"
                />
                <Metric
                  icon={<Users />}
                  label="Assignments"
                  value={shiftAssignments.length}
                  note="Scheduled employee shifts"
                  tone="blue"
                />
                <Metric
                  icon={<Clock3 />}
                  label="Upcoming shifts"
                  value={
                    shiftAssignments.filter(
                      (a: any) =>
                        a.date >= new Date().toISOString().slice(0, 10)
                    ).length
                  }
                  note="Today and future dates"
                  tone="purple"
                />
              </div>
              <div className="shift-layout">
                <section className="panel shift-form-panel">
                  <div className="panel-heading">
                    <div>
                      <h3>Create shift template</h3>
                      <p>Define standard hours and attendance grace periods.</p>
                    </div>
                  </div>
                  <form
                    className="modal-form shift-form"
                    onSubmit={async (ev) => {
                      ev.preventDefault();
                      try {
                        await request("/shifts", {
                          method: "POST",
                          body: JSON.stringify({
                            ...shiftForm,
                            breakMinutes: Number(shiftForm.breakMinutes),
                            graceMinutes: Number(shiftForm.graceMinutes),
                          }),
                        });
                        setToast("Shift template created");
                        setShiftForm({
                          name: "",
                          startTime: "09:00",
                          endTime: "17:00",
                          breakMinutes: 30,
                          graceMinutes: 5,
                          workDays: "Mon,Tue,Wed,Thu,Fri",
                        });
                        await load();
                      } catch (err) {
                        setToast(
                          err instanceof Error
                            ? err.message
                            : "Could not create shift"
                        );
                      }
                    }}
                  >
                    <label className="field-label">
                      Shift name
                      <input
                        required
                        maxLength={80}
                        value={shiftForm.name}
                        onChange={(e) =>
                          setShiftForm((f) => ({ ...f, name: e.target.value }))
                        }
                        placeholder="e.g. Weekend day shift"
                      />
                    </label>
                    <div className="form-grid">
                      <label className="field-label">
                        Start time
                        <input
                          type="time"
                          required
                          value={shiftForm.startTime}
                          onChange={(e) =>
                            setShiftForm((f) => ({
                              ...f,
                              startTime: e.target.value,
                            }))
                          }
                        />
                      </label>
                      <label className="field-label">
                        End time
                        <input
                          type="time"
                          required
                          value={shiftForm.endTime}
                          onChange={(e) =>
                            setShiftForm((f) => ({
                              ...f,
                              endTime: e.target.value,
                            }))
                          }
                        />
                      </label>
                      <label className="field-label">
                        Break (minutes)
                        <input
                          type="number"
                          min="0"
                          max="360"
                          value={shiftForm.breakMinutes}
                          onChange={(e) =>
                            setShiftForm((f) => ({
                              ...f,
                              breakMinutes: Number(e.target.value),
                            }))
                          }
                        />
                      </label>
                      <label className="field-label">
                        Grace period (minutes)
                        <input
                          type="number"
                          min="0"
                          max="180"
                          value={shiftForm.graceMinutes}
                          onChange={(e) =>
                            setShiftForm((f) => ({
                              ...f,
                              graceMinutes: Number(e.target.value),
                            }))
                          }
                        />
                      </label>
                    </div>
                    <label className="field-label">
                      Working days
                      <input
                        required
                        value={shiftForm.workDays}
                        onChange={(e) =>
                          setShiftForm((f) => ({
                            ...f,
                            workDays: e.target.value,
                          }))
                        }
                        placeholder="Mon,Tue,Wed,Thu,Fri"
                      />
                      <small className="form-hint">
                        Use comma-separated days, for example
                        Mon,Tue,Wed,Thu,Fri.
                      </small>
                    </label>
                    <button className="primary" type="submit">
                      <Plus size={16} /> Save shift template
                    </button>
                  </form>
                </section>
                <section className="panel shift-form-panel">
                  <div className="panel-heading">
                    <div>
                      <h3>Assign a shift</h3>
                      <p>Schedule an active employee for a date.</p>
                    </div>
                  </div>
                  <form
                    className="modal-form shift-form"
                    onSubmit={async (ev) => {
                      ev.preventDefault();
                      try {
                        await request("/shift-assignments", {
                          method: "POST",
                          body: JSON.stringify({
                            ...assignmentForm,
                            note: assignmentForm.note.trim(),
                          }),
                        });
                        setToast("Shift assigned successfully");
                        await load();
                      } catch (err) {
                        setToast(
                          err instanceof Error
                            ? err.message
                            : "Could not assign shift"
                        );
                      }
                    }}
                  >
                    <label className="field-label">
                      Employee
                      <select
                        required
                        value={assignmentForm.employeeId}
                        onChange={(e) =>
                          setAssignmentForm((f) => ({
                            ...f,
                            employeeId: e.target.value,
                          }))
                        }
                      >
                        <option value="">Select employee</option>
                        {employees
                          .filter((e) => e.status === "Active")
                          .map((e) => (
                            <option key={e.id} value={e.id}>
                              {e.firstName} {e.lastName} · {e.employeeNumber}
                            </option>
                          ))}
                      </select>
                    </label>
                    <label className="field-label">
                      Shift template
                      <select
                        required
                        value={assignmentForm.shiftId}
                        onChange={(e) =>
                          setAssignmentForm((f) => ({
                            ...f,
                            shiftId: e.target.value,
                          }))
                        }
                      >
                        <option value="">Select shift</option>
                        {shifts
                          .filter((s) => s.active)
                          .map((s) => (
                            <option key={s.id} value={s.id}>
                              {s.name} ({s.startTime}–{s.endTime})
                            </option>
                          ))}
                      </select>
                    </label>
                    <label className="field-label">
                      Shift date
                      <input
                        type="date"
                        required
                        value={assignmentForm.date}
                        onChange={(e) =>
                          setAssignmentForm((f) => ({
                            ...f,
                            date: e.target.value,
                          }))
                        }
                      />
                    </label>
                    <label className="field-label">
                      Notes <span className="optional">Optional</span>
                      <input
                        maxLength={250}
                        value={assignmentForm.note}
                        onChange={(e) =>
                          setAssignmentForm((f) => ({
                            ...f,
                            note: e.target.value,
                          }))
                        }
                        placeholder="Coverage, handover, etc."
                      />
                    </label>
                    <button
                      className="primary"
                      type="submit"
                      disabled={
                        !shifts.some((s) => s.active) ||
                        !employees.some((e) => e.status === "Active")
                      }
                    >
                      <CalendarCheck size={16} /> Assign shift
                    </button>
                    <p className="form-hint">
                      An employee can have one shift assignment per date.
                      Duplicate assignments are rejected.
                    </p>
                  </form>
                </section>
              </div>
              <section className="panel directory-panel shift-templates">
                <div className="panel-heading">
                  <div>
                    <h3>Shift templates</h3>
                    <p>Reusable schedules configured for your organisation.</p>
                  </div>
                  <span className="count-pill">{shifts.length} templates</span>
                </div>
                <div className="shift-template-grid">
                  {shifts.map((sh: any) => (
                    <div className="shift-template-card" key={sh.id}>
                      <div className="shift-template-top">
                        <span className="shift-icon">
                          <Clock3 size={17} />
                        </span>
                        <span
                          className={`status ${
                            sh.active ? "status-active" : "status-neutral"
                          }`}
                        >
                          {sh.active ? "Active" : "Inactive"}
                        </span>
                      </div>
                      <h4>{sh.name}</h4>
                      <div className="shift-time">
                        {sh.startTime} <ArrowRight size={14} /> {sh.endTime}
                      </div>
                      <div className="shift-meta">
                        <span>
                          <Clock3 size={13} /> {sh.breakMinutes} min break
                        </span>
                        <span>
                          <CheckCircle2 size={13} /> {sh.graceMinutes} min grace
                        </span>
                      </div>
                      <p>{sh.workDays}</p>
                    </div>
                  ))}
                  {!shifts.length && (
                    <div className="empty-state">
                      No shift templates yet. Create one above.
                    </div>
                  )}
                </div>
              </section>
              <section className="panel directory-panel">
                <div className="panel-heading">
                  <div>
                    <h3>Shift assignments</h3>
                    <p>Employee roster by date, newest first.</p>
                  </div>
                  <button className="subtle-btn" onClick={load}>
                    Refresh <Activity size={14} />
                  </button>
                </div>
                <div className="table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Employee</th>
                        <th>Shift</th>
                        <th>Date</th>
                        <th>Hours</th>
                        <th>Status</th>
                        <th>Notes</th>
                        <th>Action</th>
                      </tr>
                    </thead>
                    <tbody>
                      {shiftAssignments.map((a: any) => {
                        const sh = shifts.find((x: any) => x.id === a.shiftId);
                        return (
                          <tr key={a.id}>
                            <td>
                              <b>{a.employeeName}</b>
                            </td>
                            <td>{a.shiftName}</td>
                            <td>{a.date}</td>
                            <td>
                              {sh ? `${sh.startTime}–${sh.endTime}` : "—"}
                            </td>
                            <td>
                              <span className="status status-active">
                                {a.status}
                              </span>
                            </td>
                            <td>{a.note || "—"}</td>
                            <td>
                              <button
                                className="danger-btn"
                                onClick={async () => {
                                  if (
                                    !window.confirm(
                                      `Remove ${a.employeeName}'s shift on ${a.date}?`
                                    )
                                  )
                                    return;
                                  try {
                                    await request(
                                      `/shift-assignments/${a.id}`,
                                      { method: "DELETE" }
                                    );
                                    setToast("Shift assignment removed");
                                    await load();
                                  } catch (err) {
                                    setToast(
                                      err instanceof Error
                                        ? err.message
                                        : "Could not remove assignment"
                                    );
                                  }
                                }}
                              >
                                <XCircle size={14} /> Remove
                              </button>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                  {!shiftAssignments.length && (
                    <div className="empty-state">
                      No shift assignments yet. Use the form above to schedule
                      your team.
                    </div>
                  )}
                </div>
              </section>
              <section className="panel directory-panel">
                <div className="panel-heading">
                  <div>
                    <h3>Actual hours & overtime</h3>
                    <p>
                      Hours are calculated from check-in/check-out and assigned
                      shift duration. Overtime is only the time above the
                      assigned shift.
                    </p>
                  </div>
                  <button className="secondary" onClick={load}>
                    Refresh
                  </button>
                </div>
                <div className="table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Employee</th>
                        <th>Date</th>
                        <th>Check in</th>
                        <th>Check out</th>
                        <th>Shift hrs</th>
                        <th>Actual hrs</th>
                        <th>Normal hrs</th>
                        <th>Overtime hrs</th>
                        <th>Approval</th>
                        <th>Payroll</th>
                      </tr>
                    </thead>
                    <tbody>
                      {actualHours.map((h: any) => (
                        <tr key={h.employeeId + h.workDate}>
                          <td>
                            <b>{h.employeeName}</b>
                          </td>
                          <td>{h.workDate}</td>
                          <td>
                            {h.checkInAt
                              ? new Date(h.checkInAt).toLocaleTimeString(
                                  "en-GB",
                                  { hour: "2-digit", minute: "2-digit" }
                                )
                              : "—"}
                          </td>
                          <td>
                            {h.checkOutAt
                              ? new Date(h.checkOutAt).toLocaleTimeString(
                                  "en-GB",
                                  { hour: "2-digit", minute: "2-digit" }
                                )
                              : "—"}
                          </td>
                          <td>{Number(h.assignedHours).toFixed(2)}</td>
                          <td>{Number(h.actualHours).toFixed(2)}</td>
                          <td>{Number(h.normalHours).toFixed(2)}</td>
                          <td>{Number(h.overtimeHours).toFixed(2)}</td>
                          <td>
                            {h.approved ? (
                              <span
                                className="status status-active"
                                title={
                                  h.approvedAt
                                    ? `Approved ${new Date(
                                        h.approvedAt
                                      ).toLocaleString()} by ${h.approvedBy}`
                                    : ""
                                }
                              >
                                Approved
                              </span>
                            ) : (
                              <>
                                {h.canApprove && (
                                  <button
                                    className="review-btn"
                                    disabled={
                                      Number(h.assignedHours) <= 0 ||
                                      !h.checkInAt ||
                                      !h.checkOutAt
                                    }
                                    title={
                                      Number(h.assignedHours) <= 0
                                        ? "Assign a shift before approval"
                                        : "Approve actual hours and overtime"
                                    }
                                    onClick={async () => {
                                      const note =
                                        window.prompt(
                                          "Approval note (optional):"
                                        ) ?? "";
                                      if (note.length > 500) {
                                        setToast(
                                          "Approval note must be 500 characters or fewer"
                                        );
                                        return;
                                      }
                                      try {
                                        await request("/actual-hours/approve", {
                                          method: "POST",
                                          body: JSON.stringify({
                                            employeeId: h.employeeId,
                                            workDate: h.workDate,
                                            note,
                                          }),
                                        });
                                        setToast(
                                          "Actual hours and overtime approved"
                                        );
                                        await load();
                                      } catch (err) {
                                        setToast(
                                          err instanceof Error
                                            ? err.message
                                            : "Could not approve hours"
                                        );
                                      }
                                    }}
                                  >
                                    Approve
                                  </button>
                                )}
                                {!h.canApprove && (
                                  <span className="status status-pending">
                                    Awaiting approval
                                  </span>
                                )}
                              </>
                            )}
                          </td>
                          <td>
                            <span
                              className={`status ${
                                h.paymentPending
                                  ? "status-pending"
                                  : "status-active"
                              }`}
                            >
                              {h.paymentPending ? "Pending" : "Processed"}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {!actualHours.length && (
                    <div className="empty-state">
                      No completed attendance hours yet. Hours are recorded
                      after employees check out.
                    </div>
                  )}
                </div>
              </section>
            </>
          )}
          {page === "Leave Balance" && <LeaveBalancePage user={user} />}
          {page === "Leave management" && (
            <LeavePage
              leaves={leaves}
              onNew={() => setModal("leave")}
              onReview={(l) => {
                setSelectedLeave(l);
                setModal("decision");
              }}
              isApprover={isApprover}
            />
          )}
          {page === "Approvals" && (
            <>
              <div className="page-heading">
                <div>
                  <div className="eyebrow green-eyebrow">WORKFLOW INBOX</div>
                  <h1>Approvals</h1>
                  <p>Review requests and keep decisions moving.</p>
                </div>
              </div>
              <section className="panel directory-panel">
                <div className="panel-heading">
                  <div>
                    <h3>Leave requests</h3>
                    <p>Requests currently visible to your role</p>
                  </div>
                  <span className="count-pill">
                    {leaves.filter((l) => l.status === "Pending").length}{" "}
                    pending
                  </span>
                </div>
                <LeaveTable
                  leaves={leaves}
                  onReview={(l) => {
                    setSelectedLeave(l);
                    setModal("decision");
                  }}
                  isApprover={isApprover}
                />
              </section>
            </>
          )}
          {page === "Payroll" && (
            <PayrollPage user={user} employees={employees} onToast={setToast} />
          )}
          {page === "Documents" && (
            <DocumentsPage
              documents={documents}
              employees={employees}
              user={user}
              onRefresh={load}
              onToast={setToast}
            />
          )}
          {page === "Reports & analytics" && (
            <>
              <div className="page-heading">
                <div>
                  <div className="eyebrow green-eyebrow">INSIGHTS</div>
                  <h1>Reports & analytics</h1>
                  <p>Explore headcount and operational reporting.</p>
                </div>
                <button
                  className="primary"
                  onClick={() =>
                    (window.location.href = API + "/reports/headcount.csv")
                  }
                >
                  <Download size={16} /> Export headcount CSV
                </button>
              </div>
              <div className="metrics-grid">
                <Metric
                  icon={<Users />}
                  label="Headcount"
                  value={metrics?.employees ?? "—"}
                  note="Current employee records"
                  tone="green"
                />
                <Metric
                  icon={<Building2 />}
                  label="Departments"
                  value={metrics?.departments ?? "—"}
                  note="Distinct departments"
                  tone="blue"
                />
                <Metric
                  icon={<CalendarCheck />}
                  label="Pending requests"
                  value={metrics?.pendingApprovals ?? "—"}
                  note="Leave approvals"
                  tone="orange"
                />
              </div>
              <section className="panel overview-panel">
                <div className="panel-heading">
                  <div>
                    <h3>Headcount by department</h3>
                    <p>Current employee distribution</p>
                  </div>
                </div>
                {departmentRows(employees).map((d, i) => (
                  <div className="bar-row" key={d.name}>
                    <span>{d.name}</span>
                    <div className="bar-track">
                      <div
                        className={`bar-fill fill-${i % 5}`}
                        style={{
                          width: `${Math.max(
                            5,
                            (d.count / Math.max(1, employees.length)) * 100
                          )}%`,
                        }}
                      />
                    </div>
                    <b>{d.count}</b>
                  </div>
                ))}
              </section>
            </>
          )}
          {page === "Audit log" && (
            <>
              <div className="page-heading">
                <div>
                  <div className="eyebrow green-eyebrow">
                    SECURITY & COMPLIANCE
                  </div>
                  <h1>Audit log</h1>
                  <p>Review recorded security-sensitive activity.</p>
                </div>
                <button className="secondary" onClick={load}>
                  <Activity size={16} /> Refresh log
                </button>
              </div>
              <section className="panel directory-panel">
                <div className="table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Timestamp</th>
                        <th>Action</th>
                        <th>Target</th>
                        <th>Summary</th>
                        <th>Actor ID</th>
                      </tr>
                    </thead>
                    <tbody>
                      {audit.map((a) => (
                        <tr key={a.id}>
                          <td>
                            {new Date(a.createdAt).toLocaleString("en-GB")}
                          </td>
                          <td>
                            <code>{a.action}</code>
                          </td>
                          <td>
                            {a.targetType} · {a.targetId.slice(0, 10)}
                          </td>
                          <td>{a.summary}</td>
                          <td>{a.actorUserId.slice(0, 10) || "System"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {audit.length === 0 && (
                    <div className="empty-state">
                      No audit events available.
                    </div>
                  )}
                </div>
              </section>
            </>
          )}
          {page === "Settings" &&
            (user.permissions?.includes("app.configure") ? (
              <>
                <MasterConfigPage onToast={setToast} />
                <RoleAccessPage onToast={setToast} />
                <SalaryAlgorithmPage />
              </>
            ) : (
              <>
                <div className="page-heading">
                  <div>
                    <div className="eyebrow green-eyebrow">WORKSPACE ADMIN</div>
                    <h1>Settings</h1>
                    <p>Workspace profile and platform configuration.</p>
                  </div>
                </div>
                <div className="settings-grid">
                  <div className="panel setting-card">
                    <div className="setting-icon">
                      <Building2 />
                    </div>
                    <h3>Organisation</h3>
                    <p>PeopleOps Demo Organisation</p>
                    <span className="status status-active">Configured</span>
                  </div>
                  <div className="panel setting-card">
                    <div className="setting-icon">
                      <ShieldCheck />
                    </div>
                    <h3>Access & permissions</h3>
                    <p>
                      Signed in as {user.role}. Server-side role checks are
                      active for supported routes.
                    </p>
                    <span className="status status-active">Active</span>
                  </div>
                  <div className="panel setting-card">
                    <div className="setting-icon">
                      <CircleHelp />
                    </div>
                    <h3>Integrations</h3>
                    <p>
                      Email, AppDynamics and AI provider integrations are not
                      configured.
                    </p>
                    <span className="status status-pending">Needs setup</span>
                  </div>
                </div>
                <div className="notice-card">
                  <ShieldCheck size={19} />
                  <div>
                    <b>Before production use</b>
                    <p>
                      Set a strong SESSION_SECRET, enable HTTPS, configure
                      backups and monitoring, review roles, and complete
                      security and payroll validation. See
                      docs/PRODUCTION_READINESS.md.
                    </p>
                  </div>
                </div>
              </>
            ))}
        </div>
        <footer className="app-footer">
          <span>
            © 2026 PeopleOps <span className="footer-dot">·</span> Employee & HR
            platform
          </span>
          <span>
            <i /> All systems operational <span className="footer-dot">·</span>{" "}
            v1.0.0
          </span>
        </footer>
      </main>
      {toast && (
        <div className="toast">
          <CheckCircle2 size={18} />
          {toast}
          <button onClick={() => setToast("")}>
            <X size={15} />
          </button>
        </div>
      )}
      {modal && (
        <div
          className="modal-backdrop"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setModal("");
          }}
        >
          <div className="modal">
            <div className="modal-heading">
              <div>
                <h2>
                  {modal === "employee"
                    ? "Add employee"
                    : modal === "employee-edit"
                    ? "Edit employee"
                    : modal === "employee-details"
                    ? "Employee profile"
                    : modal === "leave"
                    ? "Request leave"
                    : modal === "decision"
                    ? "Review request"
                    : "Your profile"}
                </h2>
                <p>
                  {modal === "employee"
                    ? "Create a new employee profile."
                    : modal === "employee-edit"
                    ? "Update employee details."
                    : modal === "employee-details"
                    ? "View the complete employee record and available actions."
                    : modal === "leave"
                    ? "Submit a request for time away."
                    : modal === "decision"
                    ? "Review the request and record a decision."
                    : "Signed-in account details."}
                </p>
              </div>
              <button className="icon-btn" onClick={() => setModal("")}>
                <X size={18} />
              </button>
            </div>
            {modal === "employee" ? (
              <EmployeeForm
                fields={employeeFields}
                employees={employees}
                onCancel={() => setModal("")}
                onSave={async (data) => {
                  try {
                    await request("/employees", {
                      method: "POST",
                      body: JSON.stringify(data),
                    });
                    setModal("");
                    setToast("Employee created successfully");
                    await load();
                  } catch (e) {
                    const message =
                      e instanceof Error
                        ? e.message
                        : "Could not create employee";
                    setToast(message);
                    throw e;
                  }
                }}
              />
            ) : modal === "employee-details" && selectedEmployee ? (
              <EmployeeDetails
                employee={selectedEmployee}
                fields={employeeFields}
                role={user.role}
                permissions={user.permissions || []}
                onToast={setToast}
                onEdit={() => setModal("employee-edit")}
                onClose={() => {
                  setModal("");
                  setSelectedEmployee(null);
                }}
              />
            ) : modal === "employee-edit" && selectedEmployee ? (
              <EmployeeForm
                fields={employeeFields}
                employees={employees}
                initial={selectedEmployee}
                onCancel={() => setModal("employee-details")}
                onSave={async (data) => {
                  try {
                    await request(`/employees/${selectedEmployee.id}`, {
                      method: "PUT",
                      body: JSON.stringify(data),
                    });
                    setModal("");
                    setSelectedEmployee(null);
                    setToast("Employee changes saved");
                    await load();
                  } catch (e) {
                    const message =
                      e instanceof Error
                        ? e.message
                        : "Could not update employee";
                    setToast(message);
                    throw e;
                  }
                }}
              />
            ) : modal === "leave" ? (
              <LeaveForm
                onCancel={() => setModal("")}
                onSave={async (data) => {
                  try {
                    await request("/leave-requests", {
                      method: "POST",
                      body: JSON.stringify(data),
                    });
                    setModal("");
                    setToast("Leave request submitted");
                    await load();
                  } catch (e) {
                    setToast(
                      e instanceof Error
                        ? e.message
                        : "Could not submit request"
                    );
                  }
                }}
              />
            ) : modal === "decision" && selectedLeave ? (
              <div className="decision-content">
                <div className="request-person">
                  <div className="avatar avatar-large">
                    {initials(selectedLeave.employeeName)}
                  </div>
                  <div>
                    <b>{selectedLeave.employeeName}</b>
                    <span>
                      {selectedLeave.leaveType} · {selectedLeave.days}{" "}
                      {selectedLeave.days === 1 ? "day" : "days"}
                    </span>
                  </div>
                  <span className="status status-pending">Pending</span>
                </div>
                <div className="decision-detail">
                  <span>Requested dates</span>
                  <b>
                    {selectedLeave.startDate} – {selectedLeave.endDate}
                  </b>
                </div>
                <div className="decision-detail">
                  <span>Reason</span>
                  <p>{selectedLeave.reason || "No reason supplied"}</p>
                </div>
                <label className="field-label">
                  Decision note <span className="optional">Optional</span>
                  <textarea
                    value={decisionNote}
                    onChange={(e) => setDecisionNote(e.target.value)}
                    placeholder="Add context for the employee…"
                    rows={3}
                  />
                </label>
                <div className="modal-actions">
                  <button className="secondary" onClick={() => setModal("")}>
                    Cancel
                  </button>
                  <div className="action-pair">
                    <button
                      className="danger-btn"
                      disabled={!isApprover}
                      onClick={() => decide(selectedLeave, "Rejected")}
                    >
                      <XCircle size={15} /> Reject
                    </button>
                    <button
                      className="primary"
                      disabled={!isApprover}
                      onClick={() => decide(selectedLeave, "Approved")}
                    >
                      <CheckCircle2 size={15} /> Approve
                    </button>
                  </div>
                </div>
                {!isApprover && (
                  <p className="form-hint">
                    Your role can view but cannot decide this request.
                  </p>
                )}
              </div>
            ) : (
              <div className="profile-content">
                <div className="profile-avatar">{initials(user.name)}</div>
                <h3>{user.name}</h3>
                <p>{user.email}</p>
                <span className="status status-active">{user.role}</span>
                <div className="modal-actions">
                  <button className="secondary" onClick={() => setModal("")}>
                    Close
                  </button>
                  <button
                    className="primary"
                    onClick={() => {
                      setModal("");
                      logout();
                    }}
                  >
                    Sign out <LogOut size={15} />
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
function Metric({
  icon,
  label,
  value,
  note,
  tone,
  trend = "Stable",
}: {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
  note: string;
  tone: string;
  trend?: string;
}) {
  return (
    <div className="metric-card">
      <div className="metric-top">
        <span className={`metric-icon tone-${tone}`}>{icon}</span>
        <span className={`trend trend-${tone}`}>
          {trend === "Review" ? (
            <ArrowDownRight size={13} />
          ) : trend === "Stable" ? (
            <span>•</span>
          ) : (
            <ArrowUpRight size={13} />
          )}{" "}
          {trend}
        </span>
      </div>
      <div className="metric-label">{label}</div>
      <div className="metric-value">{value}</div>
      <div className="metric-note">{note}</div>
      <div className={`metric-accent accent-${tone}`} />
    </div>
  );
}
function departmentRows(employees: Employee[]) {
  const map = new Map<string, number>();
  employees.forEach((e) =>
    map.set(e.department, (map.get(e.department) || 0) + 1)
  );
  return [...map.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count);
}
function Quick({
  icon,
  label,
  desc,
  onClick,
}: {
  icon: React.ReactNode;
  label: string;
  desc: string;
  onClick: () => void;
}) {
  return (
    <button className="quick-action" onClick={onClick}>
      <span className="quick-icon">{icon}</span>
      <span>
        <b>{label}</b>
        <small>{desc}</small>
      </span>
      <ChevronRight size={15} />
    </button>
  );
}
function EmployeeHierarchy({ employees }: { employees: Employee[] }) {
  const [items, setItems] = useState<any[]>([]);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState("");
  useEffect(() => {
    request<any>("/employee-hierarchy")
      .then((x) => {
        const list = x.items || [];
        setItems(list);
      })
      .catch((e) =>
        setError(e instanceof Error ? e.message : "Could not load hierarchy")
      );
  }, []);
  const matching = items.filter((x) =>
    `${x.name} ${x.employeeNumber} ${x.email} ${x.department}`
      .toLowerCase()
      .includes(search.toLowerCase())
  );
  const selected =
    items.find((x) => x.id === selectedId) ||
    items.find((x) => `${x.name} (${x.employeeNumber})` === search) ||
    null;
  const path: any[] = [];
  if (selected) {
    let current = selected;
    const seen = new Set<string>();
    while (current && !seen.has(current.id)) {
      path.push(current);
      seen.add(current.id);
      current = items.find((x) => x.id === current.managerId) || null;
    }
  }
  const chain = [...path].reverse();
  const selectEmployee = (value: string) => {
    setSearch(value);
    const found =
      items.find((x) => `${x.name} (${x.employeeNumber})` === value) ||
      items.find((x) => x.name.toLowerCase() === value.toLowerCase());
    setSelectedId(found?.id || "");
  };
  return (
    <>
      <div className="page-heading">
        <div>
          <div className="eyebrow green-eyebrow">ORGANISATION</div>
          <h1>Employee hierarchy</h1>
          <p>
            Search for an employee to view their reporting line from the
            employee up to the most senior manager.
          </p>
        </div>
      </div>
      {error && <div className="error-banner">{error}</div>}
      <section className="panel directory-panel hierarchy-search-panel">
        <label
          className="hierarchy-search-label"
          htmlFor="hierarchy-employee-search"
        >
          Search employee
        </label>
        <div className="hierarchy-search-field">
          <Search size={18} />
          <input
            id="hierarchy-employee-search"
            list="hierarchy-employee-options"
            placeholder="Type a name, employee number or email…"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              const exact = items.find(
                (x) => `${x.name} (${x.employeeNumber})` === e.target.value
              );
              setSelectedId(exact?.id || "");
            }}
            onBlur={() => {
              const exact =
                items.find(
                  (x) => `${x.name} (${x.employeeNumber})` === search
                ) ||
                items.find(
                  (x) => x.name.toLowerCase() === search.toLowerCase()
                );
              if (exact) setSelectedId(exact.id);
            }}
          />
          <datalist id="hierarchy-employee-options">
            {items.map((x) => (
              <option key={x.id} value={`${x.name} (${x.employeeNumber})`}>
                {x.email} · {x.department}
              </option>
            ))}
          </datalist>
        </div>
        <p className="hierarchy-hint">
          Choose an employee from the suggestions to display their manager
          chain.
        </p>
      </section>
      <section className="panel directory-panel hierarchy-panel hierarchy-results">
        <div className="hierarchy-results-heading">
          <div>
            <h3>{selected ? "Reporting hierarchy" : "Reporting hierarchy"}</h3>
            <p>
              {selected
                ? "Top-level manager at the top; selected employee at the bottom."
                : "Search and select an employee to view their reporting line."}
            </p>
          </div>
          {selected && (
            <button
              className="secondary"
              onClick={() => {
                setSelectedId("");
                setSearch("");
              }}
            >
              <X size={15} /> Clear
            </button>
          )}
        </div>
        {!items.length && !error && (
          <div className="empty-state">Loading employee records…</div>
        )}
        {items.length > 0 && !selected && (
          <div className="empty-state">No employee selected yet.</div>
        )}
        {selected && (
          <div className="hierarchy-chain">
            {chain.map((item: any, index: number) => (
              <div className="hierarchy-chain-step" key={item.id}>
                <div
                  className={`hierarchy-person-card ${
                    item.id === selected.id ? "hierarchy-person-selected" : ""
                  }`}
                >
                  <div className="avatar avatar-purple">
                    {initials(item.name)}
                  </div>
                  <div className="hierarchy-person-info">
                    <b>{item.name}</b>
                    <small>
                      {item.jobTitle || "Job title not set"} ·{" "}
                      {item.department || "No department"}
                    </small>
                    <small>
                      {item.employeeNumber} · {item.email}
                    </small>
                  </div>
                  <span className="hierarchy-level">
                    {item.id === selected.id
                      ? "Selected employee"
                      : index === 0
                      ? "Top-level manager"
                      : `Manager level ${index + 1}`}
                  </span>
                </div>
                {index < chain.length - 1 && (
                  <div className="hierarchy-connector" aria-label="Reports to">
                    <span>reports to</span>
                    <ArrowUpRight size={17} />
                  </div>
                )}
              </div>
            ))}
            {path.length === 1 && !selected.managerId && (
              <p className="hierarchy-hint">
                This employee has no manager assigned.
              </p>
            )}
          </div>
        )}
      </section>
    </>
  );
}

function EmployeeTable({
  items,
  compact = false,
  onSelect,
}: {
  items: Employee[];
  compact?: boolean;
  onSelect?: (e: Employee) => void;
}) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Employee</th>
            <th>Department</th>
            <th>Job title</th>
            <th>Status</th>
            {!compact && (
              <>
                <th>Location</th>
                <th>Employment type</th>
                <th>Start date</th>
              </>
            )}
            <th>Profile</th>
          </tr>
        </thead>
        <tbody>
          {items.map((e) => (
            <tr
              key={e.id}
              onClick={() => onSelect?.(e)}
              className={onSelect ? "clickable-row" : ""}
            >
              <td>
                <div className="employee-cell">
                  <div className="avatar avatar-photo">
                    {e.photoId ? (
                      <img
                        className="employee-photo"
                        src={`${API}/employees/${e.id}/photo`}
                        alt=""
                      />
                    ) : (
                      initials(`${e.firstName} ${e.lastName}`)
                    )}
                  </div>
                  <div>
                    <b>
                      {e.firstName} {e.lastName}
                    </b>
                    <small>
                      {e.employeeNumber} · {e.email}
                    </small>
                  </div>
                </div>
              </td>
              <td>
                <span className="department-label">{e.department || "—"}</span>
              </td>
              <td>{e.jobTitle || "—"}</td>
              <td>
                <span
                  className={`status ${
                    e.status === "Active"
                      ? "status-active"
                      : e.status === "On Leave"
                      ? "status-leave"
                      : "status-neutral"
                  }`}
                >
                  <i />
                  {e.status}
                </span>
              </td>
              {!compact && (
                <>
                  <td>{e.location || "—"}</td>
                  <td>{e.employmentType || "—"}</td>
                  <td>
                    {e.startDate
                      ? new Date(e.startDate).toLocaleDateString("en-GB")
                      : "—"}
                  </td>
                </>
              )}
              <td>
                {!compact && (
                  <button
                    className="secondary"
                    onClick={(ev) => {
                      ev.stopPropagation();
                      onSelect?.(e);
                    }}
                  >
                    View details <ArrowRight size={14} />
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {items.length === 0 && (
        <div className="empty-state">No employees match your search.</div>
      )}
    </div>
  );
}
function EmployeeDetails({
  employee: e,
  fields,
  role,
  permissions = [],
  onToast,
  onEdit,
  onClose,
}: {
  employee: Employee;
  fields: any[];
  role: string;
  permissions?: string[];
  onToast: (s: string) => void;
  onEdit: () => void;
  onClose: () => void;
}) {
  const builtins: Record<string, any> = {
    employeeNumber: e.employeeNumber,
    firstName: e.firstName,
    lastName: e.lastName,
    email: e.email,
    department: e.department,
    jobTitle: e.jobTitle,
    location: e.location,
    employmentType: e.employmentType,
    status: e.status,
    startDate: e.startDate
      ? new Date(e.startDate).toLocaleDateString("en-GB")
      : "",
    salary: e.salaryMinor != null ? money(e.salaryMinor) : "",
    currency: e.currency,
  };
  const configured = (fields || [])
    .filter((f) => f.visible)
    .sort((a, b) => a.displayOrder - b.displayOrder);
  const used = new Set(configured.map((f) => f.fieldKey));
  return (
    <div className="modal-form">
      <div className="employee-profile-heading">
        <div className="avatar avatar-large avatar-photo">
          {e.photoId ? (
            <img
              className="employee-photo"
              src={`${API}/employees/${e.id}/photo`}
              alt="Employee profile"
            />
          ) : (
            initials(`${e.firstName} ${e.lastName}`)
          )}
        </div>
        <div>
          <h3>
            {e.firstName} {e.lastName}
          </h3>
          <p>
            {e.jobTitle} · {e.department}
          </p>
          <span
            className={`status ${
              e.status === "Active" ? "status-active" : "status-neutral"
            }`}
          >
            {e.status}
          </span>
        </div>
      </div>
      <div className="employee-detail-grid">
        {configured.map((f: any) => (
          <div className="employee-detail-item" key={f.fieldKey}>
            <small>{f.label}</small>
            <b>
              {builtins[f.fieldKey] !== undefined
                ? String(builtins[f.fieldKey] || "—")
                : String(e.customFields?.[f.fieldKey] ?? "—")}
            </b>
          </div>
        ))}
        {Object.entries(builtins)
          .filter(([k]) => !used.has(k) && k !== "currency")
          .map(([k, v]) => (
            <div className="employee-detail-item" key={k}>
              <small>
                {k
                  .replace(/[A-Z]/g, (m) => " " + m)
                  .replace(/^./, (m) => m.toUpperCase())}
              </small>
              <b>{String(v || "—")}</b>
            </div>
          ))}
        {Object.entries(e.customFields || {})
          .filter(([k]) => !used.has(k))
          .map(([k, v]) => (
            <div className="employee-detail-item" key={k}>
              <small>{k}</small>
              <b>{String(v ?? "—")}</b>
            </div>
          ))}
      </div>
      <SalaryBreakupSection
        employeeId={e.id}
        role={role}
        permissions={permissions}
        onToast={onToast}
      />
      <BankDetailsSection
        employeeId={e.id}
        role={role}
        permissions={permissions}
        onToast={onToast}
      />
      <p className="form-hint">
        Employee changes are recorded in the audit log. Where your organisation
        requires approval, changes should be routed through its employee-change
        review process.
      </p>
      <div className="modal-actions">
        <button className="secondary" onClick={onClose}>
          Close
        </button>
        <button className="primary" onClick={onEdit}>
          Edit employee <ArrowRight size={15} />
        </button>
      </div>
    </div>
  );
}
function LeaveTable({
  leaves,
  onReview,
  isApprover,
}: {
  leaves: Leave[];
  onReview: (l: Leave) => void;
  isApprover: boolean;
}) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Employee</th>
            <th>Leave type</th>
            <th>Dates</th>
            <th>Duration</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {leaves.map((l) => (
            <tr key={l.id}>
              <td>
                <b>{l.employeeName}</b>
                <small className="subcell">{l.reason}</small>
              </td>
              <td>{l.leaveType}</td>
              <td>
                {l.startDate} – {l.endDate}
              </td>
              <td>
                {l.days} {l.days === 1 ? "day" : "days"}
              </td>
              <td>
                <span
                  className={`status ${
                    l.status === "Approved"
                      ? "status-active"
                      : l.status === "Rejected"
                      ? "status-rejected"
                      : "status-pending"
                  }`}
                >
                  {l.status}
                </span>
              </td>
              <td>
                {l.status === "Pending" && isApprover ? (
                  <button className="review-btn" onClick={() => onReview(l)}>
                    Review
                  </button>
                ) : (
                  <span className="muted-dash">—</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {leaves.length === 0 && (
        <div className="empty-state">No leave requests found.</div>
      )}
    </div>
  );
}
function LeavePage({
  leaves,
  onNew,
  onReview,
  isApprover,
}: {
  leaves: Leave[];
  onNew: () => void;
  onReview: (l: Leave) => void;
  isApprover: boolean;
}) {
  return (
    <>
      <div className="page-heading">
        <div>
          <div className="eyebrow green-eyebrow">TIME AWAY</div>
          <h1>Leave management</h1>
          <p>Manage leave requests and keep your team informed.</p>
        </div>
        <button className="primary" onClick={onNew}>
          <Plus size={16} /> Request leave
        </button>
      </div>
      <div className="metrics-grid">
        <Metric
          icon={<CalendarDays />}
          label="Total requests"
          value={leaves.length}
          note="Visible requests"
          tone="blue"
        />
        <Metric
          icon={<Clock3 />}
          label="Awaiting review"
          value={leaves.filter((l) => l.status === "Pending").length}
          note="Pending decisions"
          tone="orange"
        />
        <Metric
          icon={<CheckCircle2 />}
          label="Approved"
          value={leaves.filter((l) => l.status === "Approved").length}
          note="Requests approved"
          tone="green"
        />
      </div>
      <section className="panel directory-panel">
        <div className="panel-heading">
          <div>
            <h3>Leave requests</h3>
            <p>Requests and decision history</p>
          </div>
        </div>
        <LeaveTable
          leaves={leaves}
          onReview={onReview}
          isApprover={isApprover}
        />
      </section>
    </>
  );
}

function SalaryBreakupSection({
  employeeId,
  role,
  permissions = [],
  onToast,
}: {
  employeeId: string;
  role: string;
  permissions?: string[];
  onToast: (s: string) => void;
}) {
  const allowed =
    permissions.includes("hr.payroll.read") ||
    permissions.includes("payroll.manage");
  const [data, setData] = useState<any>({
    grossPayMinor: 0,
    hourlyRateMinor: 0,
    overtimeMultiplier: 1.5,
    variablePayMinor: 0,
    allowancesMinor: 0,
    deductionsMinor: 0,
    currency: "GBP",
    effectiveFrom: new Date().toISOString().slice(0, 10),
  });
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    request<any>(`/employees/${employeeId}/salary-breakup`)
      .then(setData)
      .catch(() => {});
  }, [employeeId]);
  async function save() {
    setBusy(true);
    try {
      await request(`/employees/${employeeId}/salary-breakup`, {
        method: "PUT",
        body: JSON.stringify(data),
      });
      onToast("Salary breakup saved");
    } catch (e) {
      onToast(e instanceof Error ? e.message : "Could not save salary breakup");
    } finally {
      setBusy(false);
    }
  }
  const field = (label: string, key: string) => (
    <label>
      {label}
      <input
        type="number"
        min="0"
        step="0.01"
        disabled={!allowed}
        value={Number(data[key] || 0) / 100}
        onChange={(e) =>
          setData({
            ...data,
            [key]: Math.round(Number(e.target.value || 0) * 100),
          })
        }
      />
    </label>
  );
  return (
    <section className="employee-bank">
      <h4>Salary breakup</h4>
      <p>
        Amounts are entered in currency units and stored in minor units. Hourly
        rate drives normal and overtime calculations.
      </p>
      <div className="form-grid">
        {field("Gross pay (annual)", "grossPayMinor")}
        {field("Hourly rate", "hourlyRateMinor")}
        {field("Variable pay", "variablePayMinor")}
        {field("Allowances", "allowancesMinor")}
        {field("Deductions", "deductionsMinor")}
        <label>
          Overtime multiplier
          <input
            type="number"
            min="1"
            max="5"
            step="0.25"
            disabled={!allowed}
            value={data.overtimeMultiplier ?? 1.5}
            onChange={(e) =>
              setData({ ...data, overtimeMultiplier: Number(e.target.value) })
            }
          />
        </label>
        <label>
          Currency
          <input
            maxLength={3}
            disabled={!allowed}
            value={data.currency || "GBP"}
            onChange={(e) =>
              setData({ ...data, currency: e.target.value.toUpperCase() })
            }
          />
        </label>
        <label>
          Effective from
          <input
            type="date"
            disabled={!allowed}
            value={String(data.effectiveFrom || "").slice(0, 10)}
            onChange={(e) =>
              setData({ ...data, effectiveFrom: e.target.value })
            }
          />
        </label>
      </div>
      {allowed && (
        <div className="modal-actions">
          <button className="primary" disabled={busy} onClick={save}>
            {busy ? "Saving…" : "Save salary breakup"}
          </button>
        </div>
      )}
    </section>
  );
}

function BankDetailsSection({
  employeeId,
  role,
  permissions = [],
  onToast,
}: {
  employeeId: string;
  role: string;
  permissions?: string[];
  onToast: (s: string) => void;
}) {
  const canEdit =
    permissions.includes("hr.payroll.read") ||
    permissions.includes("payroll.manage");
  const [data, setData] = useState<any>({
    accountName: "",
    bankName: "",
    sortCode: "",
    accountNumber: "",
    iban: "",
    bicSwift: "",
  });
  const [loaded, setLoaded] = useState(false),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    request<any>(`/employees/${employeeId}/bank-details`)
      .then((x) => setData({ ...data, ...x }))
      .catch(() => {})
      .finally(() => setLoaded(true));
  }, [employeeId]);
  async function save() {
    setBusy(true);
    try {
      const payload = { ...data };
      if (payload.accountNumber.includes("*")) delete payload.accountNumber;
      if (payload.iban.includes("*")) delete payload.iban;
      await request(`/employees/${employeeId}/bank-details`, {
        method: "PUT",
        body: JSON.stringify(payload),
      });
      onToast("Bank details saved securely");
      const fresh = await request<any>(`/employees/${employeeId}/bank-details`);
      setData(fresh);
    } catch (e) {
      onToast(e instanceof Error ? e.message : "Could not save bank details");
    } finally {
      setBusy(false);
    }
  }
  if (!loaded)
    return (
      <div className="employee-bank">
        <h4>Bank details</h4>
        <p>Loading secure bank details…</p>
      </div>
    );
  return (
    <section className="employee-bank">
      <h4>Bank details</h4>
      <p>
        Account number and IBAN are encrypted at rest. Only authorised HR and
        Accounts users can view the full values.
      </p>
      <div className="form-grid">
        <label>
          Account name
          <input
            disabled={!canEdit}
            value={data.accountName || ""}
            onChange={(e) => setData({ ...data, accountName: e.target.value })}
          />
        </label>
        <label>
          Bank name
          <input
            disabled={!canEdit}
            value={data.bankName || ""}
            onChange={(e) => setData({ ...data, bankName: e.target.value })}
          />
        </label>
        <label>
          Sort code
          <input
            disabled={!canEdit}
            value={data.sortCode || ""}
            onChange={(e) => setData({ ...data, sortCode: e.target.value })}
          />
        </label>
        <label>
          Account number
          <input
            disabled={!canEdit}
            value={data.accountNumber || ""}
            onChange={(e) =>
              setData({ ...data, accountNumber: e.target.value })
            }
            placeholder="Enter account number"
            autoComplete="off"
          />
        </label>
        <label>
          IBAN
          <input
            disabled={!canEdit}
            value={data.iban || ""}
            onChange={(e) => setData({ ...data, iban: e.target.value })}
            autoComplete="off"
          />
        </label>
        <label>
          BIC / SWIFT
          <input
            disabled={!canEdit}
            value={data.bicSwift || ""}
            onChange={(e) => setData({ ...data, bicSwift: e.target.value })}
          />
        </label>
      </div>
      {canEdit && (
        <div className="modal-actions">
          <button className="primary" disabled={busy} onClick={save}>
            {busy ? "Saving…" : "Save bank details"}
          </button>
        </div>
      )}
    </section>
  );
}

function EmployeeForm({
  fields,
  employees = [],
  onCancel,
  onSave,
  initial,
}: {
  fields: any[];
  employees?: Employee[];
  onCancel: () => void;
  onSave: (x: any) => void;
  initial?: Employee;
}) {
  const defaults: Record<string, any> = {
    firstName: "",
    lastName: "",
    email: "",
    department: "",
    jobTitle: "",
    location: "London",
    employmentType: "Full-time",
    startDate: new Date().toISOString().slice(0, 10),
    employeeNumber: "",
    salary: 0,
    managerId: "",
  };
  const standard = new Set([
    "firstName",
    "lastName",
    "email",
    "department",
    "jobTitle",
    "location",
    "employmentType",
    "startDate",
    "employeeNumber",
    "salary",
    "roleName",
  ]);
  const [v, setV] = useState<Record<string, any>>({
    ...defaults,
    ...(initial || {}),
    startDate: initial?.startDate
      ? String(initial.startDate).slice(0, 10)
      : defaults.startDate,
    salary: initial?.salaryMinor != null ? initial.salaryMinor / 100 : 0,
    managerId: (initial as any)?.managerId || "",
    ...(initial?.customFields || {}),
  });
  const [photoData, setPhotoData] = useState("");
  const [photoName, setPhotoName] = useState("");
  const [busy, setBusy] = useState(false);
  const [roleOptions, setRoleOptions] = useState<string[]>([]);
  const [rolesLoaded, setRolesLoaded] = useState(false);
  const [formError, setFormError] = useState("");
  const set = (k: string, x: any) => setV((p) => ({ ...p, [k]: x }));
  useEffect(() => {
    request<any>("/roles")
      .then((r) => {
        const opts = (r.items || [])
          .filter((x: any) => x.active)
          .map((x: any) => x.roleName)
          .filter(Boolean);
        setRoleOptions(opts);
        setV((prev) => ({
          ...prev,
          roleName: initial
            ? opts.includes(String(initial.roleName || ""))
              ? initial.roleName
              : ""
            : "",
        }));
        if (!opts.length)
          setFormError(
            "No active roles found in dcs_app_roles. Add or activate a role before creating an employee."
          );
      })
      .catch((e) => {
        setRoleOptions([]);
        setFormError(
          "Could not load roles from dcs_app_roles: " +
            (e instanceof Error ? e.message : "request failed")
        );
      })
      .finally(() => setRolesLoaded(true));
  }, []);
  const visible = (fields || [])
    .filter((f) => f.visible)
    .sort((a, b) => a.displayOrder - b.displayOrder);
  const fallback = [
    {
      fieldKey: "firstName",
      label: "First name",
      dataType: "text",
      required: true,
    },
    {
      fieldKey: "lastName",
      label: "Last name",
      dataType: "text",
      required: true,
    },
    {
      fieldKey: "email",
      label: "Email address",
      dataType: "email",
      required: true,
    },
    { fieldKey: "employeeNumber", label: "Employee number", dataType: "text" },
    {
      fieldKey: "department",
      label: "Department",
      dataType: "select",
      options: [
        "People",
        "Engineering",
        "Finance",
        "Operations",
        "Design",
        "Sales",
        "Marketing",
      ],
      required: true,
    },
    {
      fieldKey: "jobTitle",
      label: "Job title",
      dataType: "text",
      required: true,
    },
    { fieldKey: "location", label: "Location", dataType: "text" },
    {
      fieldKey: "employmentType",
      label: "Employment type",
      dataType: "select",
      options: ["Full-time", "Part-time", "Fixed-term", "Contractor"],
    },
    {
      fieldKey: "startDate",
      label: "Start date",
      dataType: "date",
      required: true,
    },
    { fieldKey: "salary", label: "Annual salary (GBP)", dataType: "number" },
  ];
  const renderFields = visible.length ? visible : fallback;
  return (
    <form
      className="modal-form"
      onSubmit={async (e) => {
        e.preventDefault();
        setFormError("");
        const missing = renderFields.filter(
          (f: any) =>
            f.required &&
            f.dataType !== "checkbox" &&
            (v[f.fieldKey] === undefined ||
              v[f.fieldKey] === null ||
              String(v[f.fieldKey]).trim() === "")
        );
        if (missing.length) {
          setFormError(
            "Complete these required fields: " +
              missing.map((f: any) => f.label).join(", ")
          );
          return;
        }
        setBusy(true);
        try {
          const customFields: Record<string, any> = {};
          for (const f of renderFields) {
            if (!standard.has(f.fieldKey))
              customFields[f.fieldKey] =
                f.dataType === "number"
                  ? v[f.fieldKey] === ""
                    ? null
                    : Number(v[f.fieldKey])
                  : f.dataType === "checkbox"
                  ? Boolean(v[f.fieldKey])
                  : v[f.fieldKey] ?? "";
          }
          await onSave({
            firstName: v.firstName,
            lastName: v.lastName,
            email: v.email,
            employeeNumber: v.employeeNumber,
            department: v.department,
            jobTitle: v.jobTitle,
            location: v.location,
            employmentType: v.employmentType,
            startDate:
              typeof v.startDate === "string"
                ? v.startDate
                : v.startDate?.slice?.(0, 10),
            status: v.status || "Active",
            managerId: v.managerId || "",
            salaryMinor: Math.round(Number(v.salary || 0) * 100),
            roleName: v.roleName,
            customFields,
            photoData,
            photoName,
          });
        } catch (err) {
          setFormError(
            err instanceof Error ? err.message : "Could not save employee"
          );
        } finally {
          setBusy(false);
        }
      }}
    >
      <div className="profile-photo-picker">
        <div className="avatar avatar-large avatar-photo">
          {photoData ? (
            <img
              className="employee-photo"
              src={photoData}
              alt="Selected profile"
            />
          ) : initial?.photoId ? (
            <img
              className="employee-photo"
              src={`${API}/employees/${initial.id}/photo`}
              alt="Current profile"
            />
          ) : (
            initials(`${v.firstName || "N"} ${v.lastName || "A"}`)
          )}
        </div>
        <label className="field-label">
          Profile photo{" "}
          <span className="optional">
            Optional · JPG, PNG or WebP, max 5 MB
          </span>
          <input
            type="file"
            accept="image/jpeg,image/png,image/webp"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (!f) return;
              if (f.size > 5 * 1024 * 1024) {
                e.currentTarget.value = "";
                return;
              }
              const reader = new FileReader();
              reader.onload = () => {
                setPhotoData(String(reader.result || ""));
                setPhotoName(f.name);
              };
              reader.readAsDataURL(f);
            }}
          />
        </label>
      </div>
      {formError && <div className="error-banner">{formError}</div>}
      <div className="form-grid">
        {renderFields.map((f: any) => (
          <React.Fragment key={f.id || f.fieldKey}>
            <label>
              {f.label}
              {f.required ? " *" : ""}
              {f.dataType === "select" ? (
                <select
                  value={v[f.fieldKey] ?? ""}
                  onChange={(e) => set(f.fieldKey, e.target.value)}
                  required={f.required}
                  disabled={false}
                >
                  <option value="">Select…</option>
                  {(f.options || []).map((o: string) => (
                    <option key={o} value={o}>
                      {o}
                    </option>
                  ))}
                </select>
              ) : f.dataType === "textarea" ? (
                <textarea
                  rows={2}
                  value={v[f.fieldKey] ?? ""}
                  onChange={(e) => set(f.fieldKey, e.target.value)}
                  required={f.required}
                />
              ) : f.dataType === "checkbox" ? (
                <input
                  type="checkbox"
                  checked={!!v[f.fieldKey]}
                  onChange={(e) => set(f.fieldKey, e.target.checked)}
                />
              ) : (
                <input
                  type={
                    ["text", "email", "number", "date", "url"].includes(
                      f.dataType
                    )
                      ? f.dataType
                      : "text"
                  }
                  min={f.dataType === "number" ? 0 : undefined}
                  value={v[f.fieldKey] ?? ""}
                  onChange={(e) => set(f.fieldKey, e.target.value)}
                  required={f.required}
                />
              )}
              {f.helpText && <small className="subcell">{f.helpText}</small>}
            </label>
            {f.fieldKey === "department" && (
              <label>
                Login role *
                <select
                  value={v.roleName || ""}
                  onChange={(e) => set("roleName", e.target.value)}
                  required
                  disabled={!rolesLoaded || roleOptions.length === 0}
                >
                  <option value="">
                    {rolesLoaded ? "Select role…" : "Loading roles…"}
                  </option>
                  {roleOptions.map((role: string) => (
                    <option key={role} value={role}>
                      {role}
                    </option>
                  ))}
                </select>
                <small className="subcell">
                  Controls this user’s application access.
                </small>
              </label>
            )}
          </React.Fragment>
        ))}
        {!renderFields.some((f: any) => f.fieldKey === "department") && (
          <label>
            Login role *
            <select
              value={v.roleName || ""}
              onChange={(e) => set("roleName", e.target.value)}
              required
              disabled={!rolesLoaded || roleOptions.length === 0}
            >
              <option value="">
                {rolesLoaded ? "Select role…" : "Loading roles…"}
              </option>
              {roleOptions.map((role: string) => (
                <option key={role} value={role}>
                  {role}
                </option>
              ))}
            </select>
            <small className="subcell">
              Controls this user’s application access.
            </small>
          </label>
        )}
      </div>
      <label>
        Manager <span className="optional">Optional</span>
        <input
          list="employee-manager-options"
          value={v.managerId || ""}
          onChange={(e) => {
            const chosen = employees.find(
              (x) =>
                x.id === e.target.value ||
                `${x.firstName} ${x.lastName} · ${x.employeeNumber}` ===
                  e.target.value
            );
            set("managerId", chosen?.id || e.target.value);
          }}
          placeholder="Search employee by name or employee number"
        />
        <datalist id="employee-manager-options">
          {employees
            .filter((e) => e.id !== initial?.id)
            .map((e) => (
              <option
                key={e.id}
                value={`${e.firstName} ${e.lastName} · ${e.employeeNumber}`}
              />
            ))}
        </datalist>
        <small className="subcell">
          Select an existing employee from the suggestions.
        </small>
      </label>
      <p className="form-hint">
        Fields are configured by an authorised administrator. Additional fields
        are stored with the employee record in PostgreSQL.
      </p>
      <div className="modal-actions">
        <button type="button" className="secondary" onClick={onCancel}>
          Cancel
        </button>
        <button className="primary" disabled={busy}>
          {busy ? "Saving…" : initial ? "Save changes" : "Create employee"}{" "}
          <ArrowRight size={15} />
        </button>
      </div>
    </form>
  );
}

function LeaveForm({
  onCancel,
  onSave,
}: {
  onCancel: () => void;
  onSave: (x: any) => void;
}) {
  const [v, setV] = useState({
    leaveType: "Annual leave",
    startDate: "",
    endDate: "",
    days: 1,
    reason: "",
  });
  const [busy, setBusy] = useState(false);
  return (
    <form
      className="modal-form"
      onSubmit={async (e) => {
        e.preventDefault();
        setBusy(true);
        try {
          await onSave(v);
        } finally {
          setBusy(false);
        }
      }}
    >
      <label>
        Leave type
        <select
          value={v.leaveType}
          onChange={(e) => setV({ ...v, leaveType: e.target.value })}
        >
          <option>Annual leave</option>
          <option>Sick leave</option>
          <option>Personal leave</option>
          <option>Unpaid leave</option>
          <option>Parental leave</option>
        </select>
      </label>
      <div className="form-grid">
        <label>
          Start date *
          <input
            type="date"
            value={v.startDate}
            onChange={(e) => setV({ ...v, startDate: e.target.value })}
            required
          />
        </label>
        <label>
          End date *
          <input
            type="date"
            value={v.endDate}
            min={v.startDate}
            onChange={(e) => setV({ ...v, endDate: e.target.value })}
            required
          />
        </label>
        <label>
          Working days *
          <input
            type="number"
            min="0.5"
            max="365"
            step="0.5"
            value={v.days}
            onChange={(e) => setV({ ...v, days: Number(e.target.value) })}
            required
          />
        </label>
      </div>
      <label>
        Reason
        <textarea
          rows={3}
          maxLength={1000}
          value={v.reason}
          onChange={(e) => setV({ ...v, reason: e.target.value })}
          placeholder="Share any helpful context…"
        />
      </label>
      <div className="modal-actions">
        <button type="button" className="secondary" onClick={onCancel}>
          Cancel
        </button>
        <button className="primary" disabled={busy}>
          {busy ? "Submitting…" : "Submit request"} <ArrowRight size={15} />
        </button>
      </div>
    </form>
  );
}
function MasterConfigPage({ onToast }: { onToast: (s: string) => void }) {
  const [items, setItems] = useState<any[]>([]),
    [page, setPage] = useState("employees"),
    [busy, setBusy] = useState(false),
    [editing, setEditing] = useState<any | null>(null),
    [error, setError] = useState("");
  const pages = [
    ["employees", "Employees"],
    ["attendance", "Attendance"],
    ["shift-management", "Shift management"],
    ["leave-management", "Leave management"],
    ["payroll", "Payroll"],
    ["documents", "Documents"],
    ["approvals", "Approvals"],
    ["reports-analytics", "Reports & analytics"],
    ["audit-log", "Audit log"],
    ["settings", "Settings"],
  ];
  const load = useCallback(async () => {
    setBusy(true);
    setError("");
    try {
      const x = await request<any>("/app-master");
      setItems(x.items || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not load configuration");
    } finally {
      setBusy(false);
    }
  }, []);
  useEffect(() => {
    load();
  }, [load]);
  const blank = {
    pageId: page,
    pageName: pages.find((p) => p[0] === page)?.[1] || page,
    pageItem: "",
    fieldKey: "",
    label: "",
    dataType: "text",
    required: false,
    visible: true,
    displayOrder: 100,
    options: [],
    helpText: "",
  };
  async function save() {
    if (!editing) return;
    try {
      if (editing.id)
        await request("/app-master", {
          method: "PUT",
          body: JSON.stringify(editing),
        });
      else
        await request("/app-master", {
          method: "POST",
          body: JSON.stringify(editing),
        });
      setEditing(null);
      onToast("Application configuration saved");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    }
  }
  async function remove(item: any) {
    if (!window.confirm(`Delete configurable field “${item.label}”?`)) return;
    try {
      await request(`/app-master?id=${item.id}`, { method: "DELETE" });
      onToast("Configuration field deleted");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Delete failed");
    }
  }
  const visible = items.filter((i) => i.pageId === page);
  return (
    <>
      <div className="page-heading">
        <div>
          <div className="eyebrow green-eyebrow">
            MASTER ADMIN · CONFIGURATION
          </div>
          <h1>Application configuration</h1>
          <p>
            Configure page fields, labels, data types, required state and
            display order. Changes are stored in PostgreSQL table{" "}
            <code>dcs_app_master</code>.
          </p>
        </div>
        <button className="primary" onClick={() => setEditing({ ...blank })}>
          <Plus size={16} /> Add field
        </button>
      </div>
      {error && <div className="error-banner">{error}</div>}
      <section className="panel directory-panel">
        <div className="panel-heading">
          <div>
            <h3>Page configuration</h3>
            <p>Choose a page to configure its fields.</p>
          </div>
          <select value={page} onChange={(e) => setPage(e.target.value)}>
            {pages.map((p) => (
              <option key={p[0]} value={p[0]}>
                {p[1]}
              </option>
            ))}
          </select>
        </div>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Order</th>
                <th>Label</th>
                <th>Field key</th>
                <th>Type</th>
                <th>Required</th>
                <th>Visible</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((i) => (
                <tr key={i.id}>
                  <td>{i.displayOrder}</td>
                  <td>
                    <b>{i.label}</b>
                    <small className="subcell">{i.helpText}</small>
                  </td>
                  <td>
                    <code>{i.fieldKey}</code>
                  </td>
                  <td>{i.dataType}</td>
                  <td>{i.required ? "Yes" : "No"}</td>
                  <td>{i.visible ? "Yes" : "No"}</td>
                  <td>
                    <button
                      className="review-btn"
                      onClick={() =>
                        setEditing({ ...i, options: i.options || [] })
                      }
                    >
                      Edit
                    </button>{" "}
                    <button className="danger-btn" onClick={() => remove(i)}>
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!visible.length && (
            <div className="empty-state">
              No fields configured for this page.
            </div>
          )}
        </div>
      </section>
      {editing && (
        <div
          className="modal-backdrop"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setEditing(null);
          }}
        >
          <section className="modal">
            <div className="modal-header">
              <div>
                <h2>{editing.id ? "Edit field" : "Add field"}</h2>
                <p>Configuration changes take effect on supported screens.</p>
              </div>
              <button className="icon-btn" onClick={() => setEditing(null)}>
                <X size={18} />
              </button>
            </div>
            <div className="modal-form">
              <div className="form-grid">
                <label>
                  Page
                  <select
                    value={editing.pageId}
                    onChange={(e) => {
                      const p = pages.find((x) => x[0] === e.target.value);
                      setEditing({
                        ...editing,
                        pageId: e.target.value,
                        pageName: p?.[1] || e.target.value,
                      });
                    }}
                  >
                    {pages.map((p) => (
                      <option key={p[0]} value={p[0]}>
                        {p[1]}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  Display label
                  <input
                    value={editing.label}
                    onChange={(e) =>
                      setEditing({
                        ...editing,
                        label: e.target.value,
                        pageItem: e.target.value,
                      })
                    }
                    required
                  />
                </label>
                <label>
                  Field key
                  <input
                    value={editing.fieldKey}
                    onChange={(e) =>
                      setEditing({ ...editing, fieldKey: e.target.value })
                    }
                    required
                    pattern="[A-Za-z][A-Za-z0-9_]*"
                  />
                </label>
                <label>
                  Data type
                  <select
                    value={editing.dataType}
                    onChange={(e) =>
                      setEditing({ ...editing, dataType: e.target.value })
                    }
                  >
                    {[
                      "text",
                      "textarea",
                      "number",
                      "date",
                      "email",
                      "select",
                      "checkbox",
                      "url",
                    ].map((x) => (
                      <option key={x}>{x}</option>
                    ))}
                  </select>
                </label>
                <label>
                  Display order
                  <input
                    type="number"
                    value={editing.displayOrder}
                    onChange={(e) =>
                      setEditing({
                        ...editing,
                        displayOrder: Number(e.target.value),
                      })
                    }
                  />
                </label>
                <label>
                  Options for select
                  <input
                    value={(editing.options || []).join(", ")}
                    onChange={(e) =>
                      setEditing({
                        ...editing,
                        options: e.target.value
                          .split(",")
                          .map((x: string) => x.trim())
                          .filter(Boolean),
                      })
                    }
                    placeholder="Option A, Option B"
                  />
                </label>
                <label className="check-label">
                  <input
                    type="checkbox"
                    checked={!!editing.required}
                    onChange={(e) =>
                      setEditing({ ...editing, required: e.target.checked })
                    }
                  />{" "}
                  Required field
                </label>
                <label className="check-label">
                  <input
                    type="checkbox"
                    checked={!!editing.visible}
                    onChange={(e) =>
                      setEditing({ ...editing, visible: e.target.checked })
                    }
                  />{" "}
                  Visible on page
                </label>
                <label>
                  Help text
                  <input
                    value={editing.helpText || ""}
                    onChange={(e) =>
                      setEditing({ ...editing, helpText: e.target.value })
                    }
                  />
                </label>
              </div>
              <div className="modal-actions">
                <button className="secondary" onClick={() => setEditing(null)}>
                  Cancel
                </button>
                <button className="primary" onClick={save}>
                  Save configuration
                </button>
              </div>
            </div>
          </section>
        </div>
      )}
    </>
  );
}
function DocumentsPage({
  documents,
  employees,
  user,
  onRefresh,
  onToast,
}: {
  documents: any[];
  employees: Employee[];
  user: User;
  onRefresh: () => Promise<void>;
  onToast: (s: string) => void;
}) {
  const [employeeId, setEmployeeId] = useState(user.employeeId || ""),
    [documentType, setDocumentType] = useState("Identity"),
    [issueDate, setIssueDate] = useState(""),
    [expiryDate, setExpiryDate] = useState(""),
    [notes, setNotes] = useState(""),
    [file, setFile] = useState<File | null>(null),
    [busy, setBusy] = useState(false);
  const canManage = !!user.permissions?.includes("documents.manage");
  const canUpload = canManage || !!user.employeeId;
  async function upload(e: React.FormEvent) {
    e.preventDefault();
    if (!file) {
      onToast("Choose a file first");
      return;
    }
    const form = new FormData();
    form.append("file", file);
    form.append("employeeId", employeeId || user.employeeId || "");
    form.append("documentType", documentType);
    form.append("issueDate", issueDate);
    form.append("expiryDate", expiryDate);
    form.append("notes", notes);
    setBusy(true);
    try {
      const res = await fetch("/api/v1/documents", {
        method: "POST",
        credentials: "include",
        body: form,
      });
      if (!res.ok) {
        const j = await res.json().catch(() => ({}));
        throw new Error(j.error?.message || `Upload failed (${res.status})`);
      }
      setFile(null);
      setNotes("");
      onToast("Document uploaded to PostgreSQL");
      await onRefresh();
    } catch (e) {
      onToast(e instanceof Error ? e.message : "Document upload failed");
    } finally {
      setBusy(false);
    }
  }
  async function verify(id: string, status: string) {
    try {
      await request(`/documents/${id}/decision`, {
        method: "POST",
        body: JSON.stringify({ status }),
      });
      onToast(`Document marked ${status.toLowerCase()}`);
      await onRefresh();
    } catch (e) {
      onToast(e instanceof Error ? e.message : "Could not update status");
    }
  }
  return (
    <>
      <div className="page-heading">
        <div>
          <div className="eyebrow green-eyebrow">DOCUMENT VAULT</div>
          <h1>Documents</h1>
          <p>
            Upload, review and securely download employee documents. Files are
            stored in PostgreSQL BYTEA.
          </p>
        </div>
        <span className="count-pill">{documents.length} documents</span>
      </div>
      {canUpload && (
        <section className="panel directory-panel">
          <div className="panel-heading">
            <div>
              <h3>Upload document</h3>
              <p>PDF, PNG, JPG, DOCX, XLSX or TXT · Maximum 15 MB.</p>
            </div>
          </div>
          <form className="modal-form" onSubmit={upload}>
            <div className="form-grid">
              <label>
                Employee
                <select
                  required
                  disabled={!canManage}
                  value={canManage ? employeeId : user.employeeId || ""}
                  onChange={(e) => setEmployeeId(e.target.value)}
                >
                  <option value="">Select employee</option>
                  {employees.map((e) => (
                    <option key={e.id} value={e.id}>
                      {e.firstName} {e.lastName} · {e.employeeNumber}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Document type
                <select
                  value={documentType}
                  onChange={(e) => setDocumentType(e.target.value)}
                >
                  {[
                    "Identity",
                    "Contract",
                    "Right to work",
                    "Qualification",
                    "Other",
                  ].map((x) => (
                    <option key={x}>{x}</option>
                  ))}
                </select>
              </label>
              <label>
                Issue date
                <input
                  type="date"
                  value={issueDate}
                  onChange={(e) => setIssueDate(e.target.value)}
                />
              </label>
              <label>
                Expiry date
                <input
                  type="date"
                  value={expiryDate}
                  min={issueDate || undefined}
                  onChange={(e) => setExpiryDate(e.target.value)}
                />
              </label>
              <label>
                File
                <input
                  type="file"
                  required
                  accept=".pdf,.png,.jpg,.jpeg,.docx,.xlsx,.txt"
                  onChange={(e) => setFile(e.target.files?.[0] || null)}
                />
              </label>
              <label>
                Notes
                <input
                  maxLength={1000}
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                />
              </label>
            </div>
            <div className="modal-actions">
              <button className="primary" disabled={busy || !file}>
                {busy ? "Uploading…" : "Upload document"} <FileText size={16} />
              </button>
            </div>
          </form>
        </section>
      )}
      <section className="panel directory-panel">
        <div className="panel-heading">
          <div>
            <h3>Stored documents</h3>
            <p>Authenticated download and verification status.</p>
          </div>
          <button className="secondary" onClick={onRefresh}>
            Refresh
          </button>
        </div>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Document</th>
                <th>Employee</th>
                <th>Type</th>
                <th>Expiry</th>
                <th>Size</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {documents.map((d) => (
                <tr key={d.id}>
                  <td>
                    <b>{d.documentName}</b>
                    <small className="subcell">
                      Uploaded{" "}
                      {new Date(d.uploadedAt).toLocaleDateString("en-GB")}
                    </small>
                  </td>
                  <td>{d.employeeName}</td>
                  <td>{d.documentType}</td>
                  <td>{d.expiryDate || "—"}</td>
                  <td>{(d.fileSize / 1024).toFixed(0)} KB</td>
                  <td>
                    <span
                      className={`status ${
                        d.verificationStatus === "Verified"
                          ? "status-active"
                          : d.verificationStatus === "Rejected"
                          ? "status-rejected"
                          : "status-pending"
                      }`}
                    >
                      {d.verificationStatus}
                    </span>
                  </td>
                  <td>
                    <a
                      className="review-btn"
                      href={`/api/v1/documents/${d.id}/download`}
                    >
                      Download
                    </a>{" "}
                    {canManage && d.verificationStatus === "Pending" && (
                      <>
                        <button
                          className="review-btn"
                          onClick={() => verify(d.id, "Verified")}
                        >
                          Verify
                        </button>
                        <button
                          className="danger-btn"
                          onClick={() => verify(d.id, "Rejected")}
                        >
                          Reject
                        </button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!documents.length && (
            <div className="empty-state">
              No documents stored yet. Upload an employee document to begin.
            </div>
          )}
        </div>
      </section>
    </>
  );
}

function LeaveBalancePage({ user }: { user: User | null }) {
  const [year, setYear] = useState(new Date().getFullYear());
  const [items, setItems] = useState<any[]>([]);
  const [settings, setSettings] = useState<any>({ roles: [], holidays: [] });
  const [role, setRole] = useState("");
  const [annualDays, setAnnualDays] = useState("25");
  const [holidayDate, setHolidayDate] = useState("");
  const [holidayName, setHolidayName] = useState("");
  const [weekendDays, setWeekendDays] = useState<number[]>([0,6]);
  const [message, setMessage] = useState("");
  const canConfigure = user?.role?.toLowerCase() === "master admin";
  const load = async () => {
    const b = await request<any>(`/leave-balances?year=${year}`); setItems(b.items || []);
    if (canConfigure) { const cfg = await request<any>("/leave-balances?settings=true"); setSettings(cfg); setWeekendDays(cfg.weekendDays || [0,6]); }
  };
  useEffect(() => { load().catch(e => setMessage(e.message)); }, [year, user?.role]);
  const save = async (payload: any) => { await request("/leave-balances", { method: "POST", body: JSON.stringify(payload) }); await load(); setMessage("Saved successfully"); };
  return <>
    <div className="page-heading"><div><div className="eyebrow green-eyebrow">ANNUAL ENTITLEMENT</div><h1>Leave Balance</h1><p>Annual leave entitlement is prorated from the employee's joining date. Approved annual leave excludes weekends and configured public holidays.</p></div><label>Calendar year <select value={year} onChange={e=>setYear(Number(e.target.value))}>{[year-1,year,year+1].map(y=><option key={y} value={y}>{y}</option>)}</select></label></div>
    {message && <p role="status">{message}</p>}
    <section className="panel directory-panel"><div className="panel-heading"><div><h3>Employee leave balances</h3><p>Total entitlement, approved leave taken, and remaining balance</p></div></div><div className="table-wrap"><table><thead><tr><th>Employee</th><th>Role</th><th>Join date</th><th>Total leaves</th><th>Leaves taken</th><th>Remaining</th></tr></thead><tbody>{items.map(x=><tr key={x.employeeId}><td><strong>{x.employeeName}</strong><div className="muted">{x.employeeNumber}</div></td><td>{x.role || "—"}</td><td>{x.joinDate}</td><td>{Number(x.totalLeaves).toFixed(2)}</td><td>{Number(x.leavesTaken).toFixed(2)}</td><td><strong>{Number(x.remainingLeaves).toFixed(2)}</strong></td></tr>)}</tbody></table>{!items.length&&<div className="empty-state">No leave balances found for this year.</div>}</div></section>
    {canConfigure && <section className="panel directory-panel"><div className="panel-heading"><div><h3>Master Admin configuration</h3><p>Set annual leave by role, public holidays, and non-working weekdays.</p></div></div><form className="form-grid" onSubmit={e=>{e.preventDefault();save({role,annualDays:Number(annualDays)}).catch(err=>setMessage(err.message));}}><label>Role name<input value={role} onChange={e=>setRole(e.target.value)} required placeholder="e.g. HR Admin" /></label><label>Annual leave days<input type="number" min="0" max="366" step="0.5" value={annualDays} onChange={e=>setAnnualDays(e.target.value)} required /></label><button className="primary" type="submit">Save role entitlement</button></form><form className="form-grid" onSubmit={e=>{e.preventDefault();save({date:holidayDate,name:holidayName}).then(()=>{setHolidayDate("");setHolidayName("");}).catch(err=>setMessage(err.message));}}><label>Public holiday date<input type="date" value={holidayDate} onChange={e=>setHolidayDate(e.target.value)} required /></label><label>Holiday name<input value={holidayName} onChange={e=>setHolidayName(e.target.value)} required /></label><button className="primary" type="submit">Add / update holiday</button></form><div><strong>Non-working weekdays</strong><div className="form-grid">{["Sunday","Monday","Tuesday","Wednesday","Thursday","Friday","Saturday"].map((d,i)=><label key={d}><input type="checkbox" checked={weekendDays.includes(i)} onChange={e=>setWeekendDays(old=>e.target.checked?[...old,i]:old.filter(x=>x!==i))} /> {d}</label>)}</div><button className="primary" onClick={()=>save({weekendDays}).catch(err=>setMessage(err.message))}>Save weekend days</button></div><div className="table-wrap"><table><thead><tr><th>Role</th><th>Annual days</th></tr></thead><tbody>{settings.roles.map((x:any)=><tr key={x.role}><td>{x.role}</td><td>{x.annualDays}</td></tr>)}</tbody></table></div><div className="table-wrap"><table><thead><tr><th>Holiday</th><th>Date</th><th></th></tr></thead><tbody>{settings.holidays.map((x:any)=><tr key={x.date}><td>{x.name}</td><td>{x.date}</td><td><button className="secondary" onClick={()=>save({deleteHoliday:x.date}).catch(err=>setMessage(err.message))}>Remove</button></td></tr>)}</tbody></table></div></section>}
  </>;
}

function pageIdForLabel(label: string): string {
  return (
    (
      {
        Dashboard: "dashboard",
        Employees: "employees",
        "Employee hierarchy": "employee-hierarchy",
        Attendance: "attendance",
        "Shift management": "shift-management",
        "Shift assignments": "shift-assignments",
        "Leave management": "leave-management",
        "Leave Balance": "leave-balance",
        Payroll: "payroll",
        Documents: "documents",
        Approvals: "approvals",
        "Reports & analytics": "reports-analytics",
        "Audit log": "audit-log",
        Settings: "settings",
        "My profile": "my-profile",
      } as Record<string, string>
    )[label] || label.toLowerCase().replace(/ /g, "-")
  );
}

function RoleAccessPage({ onToast }: { onToast: (s: string) => void }) {
  const [roles, setRoles] = useState<any[]>([]),
    [selected, setSelected] = useState(""),
    [pages, setPages] = useState<any[]>([]),
    [edit, setEdit] = useState<any | null>(null),
    [busy, setBusy] = useState(false),
    [err, setErr] = useState("");
  const pageList = [
    ["dashboard", "Dashboard"],
    ["employees", "Employees"],
    ["employee-hierarchy", "Employee hierarchy"],
    ["attendance", "Attendance"],
    ["shift-management", "Shift management"],
    ["shift-assignments", "Shift assignments"],
    ["leave-management", "Leave management"],
    ["payroll", "Payroll"],
    ["documents", "Documents"],
    ["approvals", "Approvals"],
    ["reports-analytics", "Reports & analytics"],
    ["audit-log", "Audit log"],
    ["settings", "Settings"],
    ["my-profile", "My profile"],
  ];
  async function loadRoles() {
    try {
      const r = await request<any>("/roles");
      setRoles(r.items || []);
      if (!selected && r.items?.length) setSelected(r.items[0].id);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Could not load roles");
    }
  }
  async function loadPages(roleId: string) {
    if (!roleId) return;
    try {
      const r = await request<any>(
        `/role-pages?roleId=${encodeURIComponent(roleId)}`
      );
      const existing = new Map((r.items || []).map((p: any) => [p.pageId, p]));
      setPages(
        pageList.map(([pageId, pageName]) => ({
          pageId,
          pageName,
          enabled: !!(existing.get(pageId) as any)?.enabled,
        }))
      );
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Could not load page access");
    }
  }
  useEffect(() => {
    loadRoles();
  }, []);
  useEffect(() => {
    loadPages(selected);
  }, [selected]);
  async function saveRole() {
    if (!edit) return;
    setBusy(true);
    setErr("");
    try {
      await request("/roles", {
        method: edit.id ? "PUT" : "POST",
        body: JSON.stringify({
          id: edit.id,
          roleName: edit.roleName,
          description: edit.description,
          active: edit.active,
        }),
      });
      setEdit(null);
      await loadRoles();
      onToast("Role saved");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Could not save role");
    } finally {
      setBusy(false);
    }
  }
  async function savePages() {
    setBusy(true);
    try {
      await request("/role-pages", {
        method: "PUT",
        body: JSON.stringify({ roleId: selected, pages }),
      });
      onToast("Page access saved. It applies on next sign-in.");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Could not save page access");
    } finally {
      setBusy(false);
    }
  }
  async function removeRole(r: any) {
    if (r.systemRole || !window.confirm(`Remove role ${r.roleName}?`)) return;
    try {
      await request(`/roles?id=${encodeURIComponent(r.id)}`, {
        method: "DELETE",
      });
      await loadRoles();
      onToast("Role removed");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Could not remove role");
    }
  }
  return (
    <>
      <div className="page-heading">
        <div>
          <div className="eyebrow green-eyebrow">ACCESS CONTROL</div>
          <h1>Roles & page access</h1>
          <p>
            Maintain available roles and configure which application pages each
            role can see after sign-in.
          </p>
        </div>
        <button
          className="primary"
          onClick={() =>
            setEdit({ roleName: "", description: "", active: true })
          }
        >
          <Plus size={16} /> Add role
        </button>
      </div>
      {err && <div className="error-banner">{err}</div>}
      <section className="panel directory-panel">
        <div className="panel-heading">
          <div>
            <h3>Available roles</h3>
            <p>
              Stored in <code>dcs_app_roles</code>. Built-in system roles are
              protected from removal.
            </p>
          </div>
        </div>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Role</th>
                <th>Description</th>
                <th>Status</th>
                <th>Access setup</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {roles.map((r) => (
                <tr key={r.id}>
                  <td>
                    <b>{r.roleName}</b>
                    {r.systemRole && (
                      <small className="subcell">System role</small>
                    )}
                  </td>
                  <td>{r.description || "—"}</td>
                  <td>{r.active ? "Active" : "Inactive"}</td>
                  <td>
                    <button
                      className="review-btn"
                      onClick={() => setSelected(r.id)}
                    >
                      Configure pages
                    </button>
                  </td>
                  <td>
                    <button
                      className="review-btn"
                      disabled={r.systemRole}
                      onClick={() => setEdit({ ...r })}
                    >
                      Edit
                    </button>{" "}
                    <button
                      className="danger-btn"
                      disabled={r.systemRole}
                      onClick={() => removeRole(r)}
                    >
                      Remove
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
      <section className="panel directory-panel">
        <div className="panel-heading">
          <div>
            <h3>Page availability</h3>
            <p>
              {roles.find((r) => r.id === selected)?.roleName ||
                "Select a role"}{" "}
              · changes apply to navigation on next sign-in
            </p>
          </div>
          <select
            value={selected}
            onChange={(e) => setSelected(e.target.value)}
          >
            {roles.map((r) => (
              <option value={r.id} key={r.id}>
                {r.roleName}
              </option>
            ))}
          </select>
        </div>
        <div className="module-items role-page-grid">
          {pages.map((p) => (
            <label className="check-label" key={p.pageId}>
              <input
                type="checkbox"
                checked={p.enabled}
                onChange={(e) =>
                  setPages((ps) =>
                    ps.map((x) =>
                      x.pageId === p.pageId
                        ? { ...x, enabled: e.target.checked }
                        : x
                    )
                  )
                }
              />
              {p.pageName}
            </label>
          ))}
        </div>
        <div className="modal-actions">
          <button
            className="primary"
            disabled={busy || !selected}
            onClick={savePages}
          >
            Save page access
          </button>
        </div>
      </section>
      {edit && (
        <div className="modal-backdrop">
          <section className="modal">
            <div className="modal-header">
              <div>
                <h2>{edit.id ? "Edit role" : "Add role"}</h2>
                <p>Use a clear role name and least-privilege description.</p>
              </div>
              <button className="icon-btn" onClick={() => setEdit(null)}>
                <X size={18} />
              </button>
            </div>
            <form
              className="modal-form"
              onSubmit={(e) => {
                e.preventDefault();
                saveRole();
              }}
            >
              <div className="form-grid">
                <label>
                  Role name
                  <input
                    required
                    maxLength={80}
                    value={edit.roleName}
                    onChange={(e) =>
                      setEdit({ ...edit, roleName: e.target.value })
                    }
                  />
                </label>
                <label>
                  Description
                  <input
                    maxLength={500}
                    value={edit.description || ""}
                    onChange={(e) =>
                      setEdit({ ...edit, description: e.target.value })
                    }
                  />
                </label>
                <label className="check-label">
                  <input
                    type="checkbox"
                    checked={!!edit.active}
                    onChange={(e) =>
                      setEdit({ ...edit, active: e.target.checked })
                    }
                  />{" "}
                  Role active
                </label>
              </div>
              <div className="modal-actions">
                <button
                  type="button"
                  className="secondary"
                  onClick={() => setEdit(null)}
                >
                  Cancel
                </button>
                <button className="primary" disabled={busy}>
                  {busy ? "Saving…" : "Save role"}
                </button>
              </div>
            </form>
          </section>
        </div>
      )}
    </>
  );
}

function SalaryAlgorithmPage() {
  const [steps, setSteps] = useState<any[]>([]);
  useEffect(() => {
    request<any>("/salary-algorithm")
      .then((x) => setSteps(x.items || []))
      .catch(() => {});
  }, []);
  return (
    <section className="panel directory-panel">
      <div className="panel-heading">
        <div>
          <h3>Salary calculation algorithm</h3>
          <p>
            Stored in <code>dcs_salary_calculation_algorithm</code>. Changes are
            migration-controlled in this release.
          </p>
        </div>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Step</th>
              <th>Name</th>
              <th>Formula</th>
              <th>Enabled</th>
            </tr>
          </thead>
          <tbody>
            {steps.map((s) => (
              <tr key={s.stepOrder}>
                <td>{s.stepOrder}</td>
                <td>
                  <b>{s.stepName}</b>
                </td>
                <td>
                  <code>{s.formula}</code>
                </td>
                <td>{s.enabled ? "Yes" : "No"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function PayrollPage({
  user,
  employees,
  onToast,
}: {
  user: User;
  employees: Employee[];
  onToast: (s: string) => void;
}) {
  const isAccounts = !!user.permissions?.includes("payroll.manage");
  const canIssue = !!user.permissions?.includes("payroll.issue");
  const canClear = !!user.permissions?.includes("payroll.approve");
  const [items, setItems] = useState<any[]>([]),
    [summaries, setSummaries] = useState<any[]>([]),
    [startDate, setStartDate] = useState(
      new Date().toISOString().slice(0, 7) + "-01"
    ),
    [endDate, setEndDate] = useState(
      new Date().toISOString().slice(0, 7) +
        "-" +
        new Date(new Date().getFullYear(), new Date().getMonth() + 1, 0)
          .getDate()
          .toString()
          .padStart(2, "0")
    ),
    [reference, setReference] = useState(
      "Payroll " + new Date().toISOString().slice(0, 7)
    ),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function load() {
    setError("");
    try {
      const p = await request<any>("/payroll");
      setItems(p.items || []);
      if (isAccounts) {
        const sm = await request<any>("/payroll/summaries");
        setSummaries(sm.items || []);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not load payroll");
    }
  }
  useEffect(() => {
    load();
  }, [user.id]);
  async function action(path: string, body: any, msg: string) {
    setBusy(true);
    try {
      const res = await request<any>(path, {
        method: "POST",
        body: JSON.stringify(body),
      });
      onToast(res.message || msg);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Payroll action failed");
    } finally {
      setBusy(false);
    }
  }
  const fmt = (minor: number, currency = "GBP") =>
    new Intl.NumberFormat("en-GB", {
      style: "currency",
      currency,
      maximumFractionDigits: 2,
    }).format((minor || 0) / 100);
  return (
    <>
      <div className="page-heading">
        <div>
          <div className="eyebrow green-eyebrow">PAY & COMPENSATION</div>
          <h1>Payroll</h1>
          <p>
            {isAccounts
              ? "Calculate salary from approved attendance hours and manage payment records."
              : "View your payment history and download individual salary slips."}
          </p>
        </div>
        <span className="read-only-tag">
          <ShieldCheck size={14} />
          {isAccounts ? "Accounts workspace" : "Employee payslips"}
        </span>
      </div>
      {error && <div className="error-banner">{error}</div>}
      {isAccounts && (
        <>
          <section className="panel directory-panel">
            <div className="panel-heading">
              <div>
                <h3>Payroll period</h3>
                <p>
                  Calculations use actual hours marked payment-pending and the
                  employee salary breakup.
                </p>
              </div>
            </div>
            <div className="modal-form">
              <div className="form-grid">
                <label>
                  Period start
                  <input
                    type="date"
                    value={startDate}
                    onChange={(e) => setStartDate(e.target.value)}
                  />
                </label>
                <label>
                  Period end
                  <input
                    type="date"
                    value={endDate}
                    min={startDate}
                    onChange={(e) => setEndDate(e.target.value)}
                  />
                </label>
                <label>
                  Payment reference
                  <input
                    value={reference}
                    onChange={(e) => setReference(e.target.value)}
                    maxLength={120}
                  />
                </label>
              </div>
              <div className="modal-actions">
                <button
                  className="secondary"
                  disabled={busy}
                  onClick={() =>
                    action(
                      "/payroll/calculate",
                      { startDate, endDate },
                      "Salary calculated"
                    )
                  }
                >
                  Calculate salary
                </button>
                {canIssue && (
                  <button
                    className="primary"
                    disabled={busy}
                    onClick={() =>
                      action(
                        "/payroll/issue",
                        { startDate, endDate, reference },
                        "Payments issued"
                      )
                    }
                  >
                    Issue payment debits
                  </button>
                )}
              </div>
            </div>
          </section>
          <section className="panel directory-panel">
            <div className="panel-heading">
              <div>
                <h3>Salary summaries</h3>
                <p>Review calculated values before issuing payments.</p>
              </div>
              <button className="secondary" onClick={load}>
                Refresh
              </button>
            </div>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Employee</th>
                    <th>Period</th>
                    <th>Normal hrs</th>
                    <th>Overtime hrs</th>
                    <th>Gross</th>
                    <th>Deductions</th>
                    <th>Net pay</th>
                    <th>Status</th>
                    <th>Slip</th>
                  </tr>
                </thead>
                <tbody>
                  {summaries.map((x) => (
                    <tr key={x.id}>
                      <td>
                        {x.employeeName ||
                          employees.find((e) => e.id === x.employeeId)
                            ?.firstName +
                            " " +
                            employees.find((e) => e.id === x.employeeId)
                              ?.lastName ||
                          x.employeeId}
                      </td>
                      <td>
                        {x.periodStart} – {x.periodEnd}
                      </td>
                      <td>{x.normalHours}</td>
                      <td>{x.overtimeHours}</td>
                      <td>{fmt(x.grossMinor, x.currency)}</td>
                      <td>{fmt(x.deductionsMinor, x.currency)}</td>
                      <td>
                        <b>{fmt(x.netMinor, x.currency)}</b>
                      </td>
                      <td>{x.status}</td>
                      <td>
                        <a
                          className="review-btn"
                          target="_blank"
                          rel="noreferrer"
                          href={`/api/v1/payroll/${x.id}/payslip.pdf?view=1`}
                        >
                          View
                        </a>{" "}
                        <a
                          className="review-btn"
                          href={`/api/v1/payroll/${x.id}/payslip.pdf`}
                        >
                          Download
                        </a>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {!summaries.length && (
                <div className="empty-state">
                  No salary summaries yet. Calculate a period after attendance
                  and salary breakup are configured.
                </div>
              )}
            </div>
          </section>
        </>
      )}
      {!isAccounts && (
        <section className="panel directory-panel">
          <div className="panel-heading">
            <div>
              <h3>Your payments and payslips</h3>
              <p>
                PDF salary slips are available for your own employee record.
              </p>
            </div>
            <button className="secondary" onClick={load}>
              Refresh
            </button>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Period</th>
                  <th>Normal hours</th>
                  <th>Overtime hours</th>
                  <th>Gross</th>
                  <th>Deductions</th>
                  <th>Net paid</th>
                  <th>Status</th>
                  <th>Salary slip</th>
                </tr>
              </thead>
              <tbody>
                {items.map((x) => (
                  <tr key={x.id}>
                    <td>
                      {x.periodStart} – {x.periodEnd}
                    </td>
                    <td>{x.normalHours}</td>
                    <td>{x.overtimeHours}</td>
                    <td>{fmt(x.grossMinor, x.currency)}</td>
                    <td>{fmt(x.deductionsMinor, x.currency)}</td>
                    <td>
                      <b>{fmt(x.netMinor, x.currency)}</b>
                    </td>
                    <td>{x.status}</td>
                    <td>
                      <a
                        className="review-btn"
                        target="_blank"
                        rel="noreferrer"
                        href={`/api/v1/payroll/${x.id}/payslip.pdf?view=1`}
                      >
                        View PDF
                      </a>{" "}
                      <a
                        className="review-btn"
                        href={`/api/v1/payroll/${x.id}/payslip.pdf`}
                      >
                        Download
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!items.length && (
              <div className="empty-state">No salary slips available yet.</div>
            )}
          </div>
        </section>
      )}
      {isAccounts && (
        <section className="panel directory-panel">
          <div className="panel-heading">
            <div>
              <h3>Payment ledger</h3>
              <p>
                Debit records represent issued payments; credit records are
                added only after clearance.
              </p>
            </div>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Employee</th>
                  <th>Period</th>
                  <th>Type</th>
                  <th>Amount</th>
                  <th>Status</th>
                  <th>Reference</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {items.map((x) => (
                  <tr key={x.id}>
                    <td>
                      {employees.find((e) => e.id === x.employeeId)
                        ?.firstName || x.employeeId}
                    </td>
                    <td>
                      {x.periodStart} – {x.periodEnd}
                    </td>
                    <td>{x.entryType}</td>
                    <td>{fmt(x.amountMinor, x.currency)}</td>
                    <td>{x.status}</td>
                    <td>{x.reference || "—"}</td>
                    <td>
                      {canClear &&
                        x.entryType === "debit" &&
                        x.status === "Pending" && (
                          <button
                            className="review-btn"
                            disabled={busy}
                            onClick={() =>
                              action(
                                `/payroll/payments/${x.id}/clear`,
                                {},
                                "Payment cleared"
                              )
                            }
                          >
                            Mark cleared
                          </button>
                        )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!items.length && (
              <div className="empty-state">No payment records yet.</div>
            )}
          </div>
        </section>
      )}
      <div className="notice-card">
        <ShieldCheck size={19} />
        <div>
          <b>Payroll control</b>
          <p>
            Configure salary breakup and bank details before processing. Tax,
            pension, statutory deductions, provider settlement and approval
            rules must be validated for your jurisdiction before production
            payroll.
          </p>
        </div>
      </div>
    </>
  );
}

function SimplePage({
  eyebrow,
  title,
  desc,
  icon,
  items,
  onClick,
}: {
  eyebrow: string;
  title: string;
  desc: string;
  icon: React.ReactNode;
  items: string[];
  onClick: () => void;
}) {
  return (
    <>
      <div className="page-heading">
        <div>
          <div className="eyebrow green-eyebrow">{eyebrow}</div>
          <h1>{title}</h1>
          <p>{desc}</p>
        </div>
        <button className="primary" onClick={onClick}>
          <Plus size={16} /> Configure module
        </button>
      </div>
      <div className="panel module-empty">
        <span className="module-icon">{icon}</span>
        <h2>{title} module setup</h2>
        <p>
          This module is scaffolded in the interface but requires private
          storage and policy configuration before it can be enabled for real
          employee data.
        </p>
        <div className="module-items">
          {items.map((x) => (
            <div key={x}>
              <CheckCircle2 size={16} />
              {x}
            </div>
          ))}
        </div>
        <button className="secondary" onClick={onClick}>
          View setup guidance <ArrowRight size={15} />
        </button>
      </div>
    </>
  );
}
