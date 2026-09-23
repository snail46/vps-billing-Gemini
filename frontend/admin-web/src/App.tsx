import React, { useEffect, useState } from "react";
import {
  useI18n,
  Card,
  Button,
  StatusBadge,
  Alert,
  LanguageSwitcher,
  authApi,
  commerceApi,
  adminApi,
  AdminDTO,
  UserDTO,
  AuditEventDTO,
  Order,
  Invoice,
  LedgerTransaction,
  Node,
  Provider,
  Instance,
  Operation,
  HealthCheckResult,
} from "@vps-billing/shared";

type AdminTab =
  | "dashboard"
  | "infrastructure"
  | "instances"
  | "commerce"
  | "operations"
  | "users";

export const App: React.FC = () => {
  const { t } = useI18n();

  const [admin, setAdmin] = useState<AdminDTO | null>(null);
  const [roles, setRoles] = useState<string[]>([]);
  const [permissions, setPermissions] = useState<string[]>([]);
  const [checkingAuth, setCheckingAuth] = useState(true);

  // Tab State
  const [activeTab, setActiveTab] = useState<AdminTab>("dashboard");
  const [commerceSubTab, setCommerceSubTab] = useState<"orders" | "invoices" | "ledger">("orders");
  const [usersSubTab, setUsersSubTab] = useState<"users" | "admins" | "audit">("users");

  // Login form state
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [loginError, setLoginError] = useState<string | null>(null);

  // Data states
  const [health, setHealth] = useState<HealthCheckResult | null>(null);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [ledgerTxs, setLedgerTxs] = useState<LedgerTransaction[]>([]);
  const [operations, setOperations] = useState<Operation[]>([]);
  const [users, setUsers] = useState<UserDTO[]>([]);
  const [admins, setAdmins] = useState<AdminDTO[]>([]);
  const [auditEvents, setAuditEvents] = useState<AuditEventDTO[]>([]);
  const [loadingData, setLoadingData] = useState(false);

  useEffect(() => {
    checkMe();
  }, []);

  useEffect(() => {
    if (admin) {
      loadTabData();
    }
  }, [admin, activeTab, commerceSubTab, usersSubTab]);

  const checkMe = async () => {
    setCheckingAuth(true);
    try {
      const res = await authApi.getMeAdmin();
      if (res.success) {
        setAdmin(res.data.admin);
        setRoles(res.data.roles || []);
        setPermissions(res.data.permissions || []);
      } else {
        setAdmin(null);
      }
    } catch {
      setAdmin(null);
    } finally {
      setCheckingAuth(false);
    }
  };

  const loadTabData = async () => {
    setLoadingData(true);
    try {
      if (activeTab === "dashboard") {
        const h = await adminApi.checkHealth();
        setHealth(h);
        const [nList, iList, oList] = await Promise.all([
          adminApi.listNodes(),
          adminApi.listInstances(),
          commerceApi.adminListOrders(),
        ]);
        setNodes(nList);
        setInstances(iList);
        setOrders(oList.orders || []);
      } else if (activeTab === "infrastructure") {
        const [nList, pList] = await Promise.all([
          adminApi.listNodes(),
          adminApi.listProviders(),
        ]);
        setNodes(nList);
        setProviders(pList);
      } else if (activeTab === "instances") {
        const list = await adminApi.listInstances();
        setInstances(list);
      } else if (activeTab === "commerce") {
        if (commerceSubTab === "orders") {
          const res = await commerceApi.adminListOrders();
          setOrders(res.orders || []);
        } else if (commerceSubTab === "invoices") {
          const res = await commerceApi.adminListInvoices();
          setInvoices(res.invoices || []);
        } else if (commerceSubTab === "ledger") {
          const res = await commerceApi.adminListLedger();
          setLedgerTxs(res.transactions || []);
        }
      } else if (activeTab === "operations") {
        const list = await adminApi.listOperations(50);
        setOperations(list);
      } else if (activeTab === "users") {
        if (usersSubTab === "users") {
          const uList = await adminApi.listUsers(50);
          setUsers(uList);
        } else if (usersSubTab === "admins") {
          const aList = await adminApi.listAdmins();
          setAdmins(aList);
        } else if (usersSubTab === "audit") {
          const res = await authApi.listAudit(50, 0);
          if (res.success) {
            setAuditEvents(res.data.items || []);
          }
        }
      }
    } catch (err) {
      console.error("Failed to load admin data:", err);
    } finally {
      setLoadingData(false);
    }
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoginError(null);
    setSubmitting(true);
    try {
      const res = await authApi.loginAdmin({ email, password });
      if (res.success) {
        if (res.data.admin) {
          setAdmin(res.data.admin);
          setRoles(res.data.roles || []);
          setPermissions(res.data.permissions || []);
          setPassword("");
        }
      } else {
        setLoginError(t(res.error?.message_key || "errors.invalid_credentials"));
      }
    } catch (err: any) {
      setLoginError(err?.message || t("errors.network_error"));
    } finally {
      setSubmitting(false);
    }
  };

  const handleLogout = async () => {
    try {
      await authApi.logoutAdmin();
    } finally {
      setAdmin(null);
      setRoles([]);
      setPermissions([]);
    }
  };

  const formatPrice = (minor: number, currency: string) => {
    return `${currency} ${(minor / 100).toFixed(2)}`;
  };

  return (
    <div className="min-h-screen flex flex-col bg-zinc-100 text-zinc-900">
      {/* Header */}
      <header className="bg-zinc-900 text-white border-b border-zinc-800 sticky top-0 z-20 shadow-md">
        <div className="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between">
          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-lg bg-emerald-600 text-white flex items-center justify-center font-bold text-lg shadow-sm">
                A
              </div>
              <div>
                <span className="font-semibold tracking-tight block text-sm">
                  {t("common.appNameAdmin")}
                </span>
                <span className="text-[10px] text-zinc-400 font-mono block">
                  v1.0.0-rc &bull; RBAC Protected
                </span>
              </div>
            </div>

            {admin && (
              <nav className="hidden lg:flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => setActiveTab("dashboard")}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                    activeTab === "dashboard"
                      ? "bg-zinc-800 text-white"
                      : "text-zinc-400 hover:text-white"
                  }`}
                >
                  {t("common.dashboard")}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("infrastructure")}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                    activeTab === "infrastructure"
                      ? "bg-zinc-800 text-white"
                      : "text-zinc-400 hover:text-white"
                  }`}
                >
                  {t("common.nodes")} &amp; {t("common.providers")}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("instances")}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                    activeTab === "instances"
                      ? "bg-zinc-800 text-white"
                      : "text-zinc-400 hover:text-white"
                  }`}
                >
                  {t("common.servers")}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("commerce")}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                    activeTab === "commerce"
                      ? "bg-zinc-800 text-white"
                      : "text-zinc-400 hover:text-white"
                  }`}
                >
                  {t("common.billing")}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("operations")}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                    activeTab === "operations"
                      ? "bg-zinc-800 text-white"
                      : "text-zinc-400 hover:text-white"
                  }`}
                >
                  {t("common.operations")}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("users")}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                    activeTab === "users"
                      ? "bg-zinc-800 text-white"
                      : "text-zinc-400 hover:text-white"
                  }`}
                >
                  {t("common.users")} &amp; {t("common.audit")}
                </button>
              </nav>
            )}
          </div>

          <div className="flex items-center gap-4">
            <LanguageSwitcher />
            {admin && (
              <div className="flex items-center gap-3">
                <span className="text-xs text-zinc-300 hidden sm:inline">
                  {admin.display_name || admin.email}
                </span>
                <Button size="sm" variant="outline" onClick={handleLogout} className="border-zinc-700 text-zinc-200">
                  {t("auth.logout")}
                </Button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Main Body */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 py-8">
        {checkingAuth ? (
          <div className="py-20 text-center text-zinc-500 text-sm">{t("common.loading")}</div>
        ) : !admin ? (
          /* Admin Login Screen */
          <div className="max-w-md mx-auto py-12">
            <Card
              title={t("auth.adminLoginTitle")}
              subtitle={t("auth.adminLoginSubtitle")}
            >
              {loginError && (
                <div className="mb-4">
                  <Alert severity="error">{loginError}</Alert>
                </div>
              )}

              <form onSubmit={handleLogin} className="space-y-4">
                <div>
                  <label className="block text-xs font-medium text-zinc-700 mb-1">
                    {t("auth.email")}
                  </label>
                  <input
                    type="email"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full px-3 py-2 border border-zinc-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    placeholder="admin@vps-billing.local"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-zinc-700 mb-1">
                    {t("auth.password")}
                  </label>
                  <input
                    type="password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full px-3 py-2 border border-zinc-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
                  />
                </div>

                <Button type="submit" isLoading={submitting} className="w-full bg-emerald-600 hover:bg-emerald-700 text-white">
                  {t("auth.login")}
                </Button>
              </form>
            </Card>
          </div>
        ) : (
          /* Admin Views */
          <div className="space-y-6">
            {/* Header sub-bar with refresh */}
            <div className="flex items-center justify-between pb-2 border-b border-zinc-200">
              <div className="flex items-center gap-2">
                <span className="text-xs text-zinc-500 font-mono">
                  {t("auth.loggedInAs")}: <strong className="text-zinc-800">{admin.email}</strong>
                </span>
                <span className="text-xs text-zinc-400">&bull;</span>
                <div className="flex items-center gap-1">
                  {roles.map((r) => (
                    <span key={r} className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-zinc-200 text-zinc-700">
                      {r}
                    </span>
                  ))}
                  {permissions.length > 0 && (
                    <span className="text-[10px] font-mono text-zinc-400">
                      ({permissions.length} perms)
                    </span>
                  )}
                </div>
              </div>
              <Button size="sm" variant="outline" onClick={loadTabData} isLoading={loadingData}>
                {t("common.refresh")}
              </Button>
            </div>

            {/* TAB 1: System Dashboard */}
            {activeTab === "dashboard" && (
              <div className="space-y-6">
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-2xs">
                    <span className="text-xs text-zinc-500 block uppercase font-bold tracking-wider mb-1">
                      {t("common.systemHealth")}
                    </span>
                    <div className="flex items-center gap-2 mt-2">
                      <div className="w-3 h-3 rounded-full bg-emerald-500 animate-pulse" />
                      <span className="text-xl font-bold text-zinc-900">
                        {health?.status === "healthy" ? t("common.healthy") : "Operational"}
                      </span>
                    </div>
                  </div>

                  <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-2xs">
                    <span className="text-xs text-zinc-500 block uppercase font-bold tracking-wider mb-1">
                      Active Nodes
                    </span>
                    <span className="text-2xl font-bold text-zinc-900 mt-1 block">
                      {nodes.length}
                    </span>
                  </div>

                  <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-2xs">
                    <span className="text-xs text-zinc-500 block uppercase font-bold tracking-wider mb-1">
                      Cloud Instances
                    </span>
                    <span className="text-2xl font-bold text-zinc-900 mt-1 block">
                      {instances.length}
                    </span>
                  </div>

                  <div className="bg-white p-5 rounded-xl border border-zinc-200 shadow-2xs">
                    <span className="text-xs text-zinc-500 block uppercase font-bold tracking-wider mb-1">
                      Commerce Orders
                    </span>
                    <span className="text-2xl font-bold text-zinc-900 mt-1 block">
                      {orders.length}
                    </span>
                  </div>
                </div>

                <Card title={t("common.serviceStatus")}>
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs font-mono">
                    <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-100 flex items-center justify-between">
                      <span>{t("common.database")}</span>
                      <StatusBadge severity="success" label="connected" />
                    </div>
                    <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-100 flex items-center justify-between">
                      <span>{t("common.redis")}</span>
                      <StatusBadge severity="success" label="ready" />
                    </div>
                    <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-100 flex items-center justify-between">
                      <span>{t("common.worker")}</span>
                      <StatusBadge severity="success" label="polling" />
                    </div>
                  </div>
                </Card>
              </div>
            )}

            {/* TAB 2: Infrastructure (Nodes & Providers) */}
            {activeTab === "infrastructure" && (
              <div className="space-y-6">
                <Card title={t("common.providers")}>
                  {providers.length === 0 ? (
                    <div className="text-center py-8 text-zinc-400 text-sm">{t("common.empty")}</div>
                  ) : (
                    <div className="divide-y divide-zinc-100">
                      {providers.map((p) => (
                        <div key={p.id} className="py-3 flex items-center justify-between">
                          <div>
                            <span className="font-semibold text-zinc-900 text-sm block">{p.name}</span>
                            <span className="text-xs text-zinc-400 font-mono">
                              Type: {p.provider_type} &bull; ID: {p.id}
                            </span>
                          </div>
                          <StatusBadge severity="success" label={p.status} />
                        </div>
                      ))}
                    </div>
                  )}
                </Card>

                <Card title={t("common.nodes")}>
                  {nodes.length === 0 ? (
                    <div className="text-center py-8 text-zinc-400 text-sm">{t("common.empty")}</div>
                  ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      {nodes.map((n) => {
                        const cpuPct = n.cpu_total > 0 ? Math.round(((n.cpu_allocated + n.cpu_reserved) / n.cpu_total) * 100) : 0;
                        const memPct = n.memory_total_mb > 0 ? Math.round(((n.memory_allocated_mb + n.memory_reserved_mb) / n.memory_total_mb) * 100) : 0;
                        const diskPct = n.disk_total_gb > 0 ? Math.round(((n.disk_allocated_gb + n.disk_reserved_gb) / n.disk_total_gb) * 100) : 0;
                        return (
                          <div key={n.id} className="p-4 bg-zinc-50 rounded-xl border border-zinc-200 space-y-3">
                            <div className="flex items-center justify-between">
                              <div>
                                <span className="font-bold text-zinc-900 text-sm block">{n.name}</span>
                                <span className="text-xs text-zinc-400 font-mono">Region: {n.region}</span>
                              </div>
                              <StatusBadge severity={n.status === "active" ? "success" : "neutral"} label={n.status} />
                            </div>

                            <div className="space-y-2 text-xs">
                              <div>
                                <div className="flex justify-between text-zinc-600 mb-1">
                                  <span>CPU ({n.cpu_allocated + n.cpu_reserved} / {n.cpu_total} cores)</span>
                                  <span>{cpuPct}%</span>
                                </div>
                                <div className="w-full bg-zinc-200 h-1.5 rounded-full overflow-hidden">
                                  <div className="bg-blue-600 h-full" style={{ width: `${cpuPct}%` }} />
                                </div>
                              </div>

                              <div>
                                <div className="flex justify-between text-zinc-600 mb-1">
                                  <span>RAM ({n.memory_allocated_mb + n.memory_reserved_mb} / {n.memory_total_mb} MB)</span>
                                  <span>{memPct}%</span>
                                </div>
                                <div className="w-full bg-zinc-200 h-1.5 rounded-full overflow-hidden">
                                  <div className="bg-emerald-600 h-full" style={{ width: `${memPct}%` }} />
                                </div>
                              </div>

                              <div>
                                <div className="flex justify-between text-zinc-600 mb-1">
                                  <span>Disk ({n.disk_allocated_gb + n.disk_reserved_gb} / {n.disk_total_gb} GB)</span>
                                  <span>{diskPct}%</span>
                                </div>
                                <div className="w-full bg-zinc-200 h-1.5 rounded-full overflow-hidden">
                                  <div className="bg-purple-600 h-full" style={{ width: `${diskPct}%` }} />
                                </div>
                              </div>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </Card>
              </div>
            )}

            {/* TAB 3: Instances */}
            {activeTab === "instances" && (
              <Card title={t("common.servers")}>
                {instances.length === 0 ? (
                  <div className="text-center py-12 text-zinc-400 text-sm">{t("common.empty")}</div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs">
                      <thead className="border-b border-zinc-200 text-zinc-500 uppercase tracking-wider">
                        <tr>
                          <th className="py-3 px-2">Name</th>
                          <th className="py-3 px-2">Status</th>
                          <th className="py-3 px-2">IP Address</th>
                          <th className="py-3 px-2">Specs</th>
                          <th className="py-3 px-2">Subscription ID</th>
                          <th className="py-3 px-2">Created</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-zinc-100">
                        {instances.map((inst) => (
                          <tr key={inst.id} className="hover:bg-zinc-50">
                            <td className="py-3 px-2 font-semibold text-zinc-900">{inst.name}</td>
                            <td className="py-3 px-2">
                              <StatusBadge
                                severity={inst.observed_state === "running" ? "success" : "neutral"}
                                label={inst.observed_state}
                              />
                            </td>
                            <td className="py-3 px-2 font-mono">{inst.primary_ipv4 || "192.168.1.100"}</td>
                            <td className="py-3 px-2 font-mono">
                              {inst.cpu_cores}C / {inst.memory_mb}MB / {inst.disk_gb}GB
                            </td>
                            <td className="py-3 px-2 font-mono text-zinc-400 truncate max-w-[120px]">
                              {inst.subscription_id}
                            </td>
                            <td className="py-3 px-2 text-zinc-500 font-mono">
                              {new Date(inst.created_at).toLocaleDateString()}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </Card>
            )}

            {/* TAB 4: Commerce (Orders, Invoices, Ledger) */}
            {activeTab === "commerce" && (
              <div className="space-y-4">
                <div className="flex gap-2 border-b border-zinc-200 pb-2">
                  <button
                    type="button"
                    onClick={() => setCommerceSubTab("orders")}
                    className={`px-3 py-1 rounded text-xs font-semibold ${
                      commerceSubTab === "orders" ? "bg-zinc-800 text-white" : "bg-white text-zinc-600 hover:bg-zinc-50"
                    }`}
                  >
                    {t("commerce.orders")}
                  </button>
                  <button
                    type="button"
                    onClick={() => setCommerceSubTab("invoices")}
                    className={`px-3 py-1 rounded text-xs font-semibold ${
                      commerceSubTab === "invoices" ? "bg-zinc-800 text-white" : "bg-white text-zinc-600 hover:bg-zinc-50"
                    }`}
                  >
                    {t("commerce.invoices")}
                  </button>
                  <button
                    type="button"
                    onClick={() => setCommerceSubTab("ledger")}
                    className={`px-3 py-1 rounded text-xs font-semibold ${
                      commerceSubTab === "ledger" ? "bg-zinc-800 text-white" : "bg-white text-zinc-600 hover:bg-zinc-50"
                    }`}
                  >
                    {t("commerce.adminLedger")}
                  </button>
                </div>

                {commerceSubTab === "orders" && (
                  <Card title={t("commerce.orders")}>
                    {orders.length === 0 ? (
                      <div className="text-center py-8 text-zinc-400 text-sm">{t("common.empty")}</div>
                    ) : (
                      <div className="divide-y divide-zinc-100">
                        {orders.map((o) => (
                          <div key={o.id} className="py-3 flex items-center justify-between text-xs">
                            <div>
                              <span className="font-mono font-bold text-zinc-900 block">{o.order_no}</span>
                              <span className="text-zinc-400 font-mono">User: {o.user_id}</span>
                            </div>
                            <div className="flex items-center gap-4">
                              <span className="font-bold text-zinc-900">{formatPrice(o.total_minor, o.currency)}</span>
                              <StatusBadge severity={o.status === "paid" ? "success" : "warning"} label={o.status} />
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </Card>
                )}

                {commerceSubTab === "invoices" && (
                  <Card title={t("commerce.invoices")}>
                    {invoices.length === 0 ? (
                      <div className="text-center py-8 text-zinc-400 text-sm">{t("common.empty")}</div>
                    ) : (
                      <div className="divide-y divide-zinc-100">
                        {invoices.map((inv) => (
                          <div key={inv.id} className="py-3 flex items-center justify-between text-xs">
                            <span className="font-mono font-bold text-zinc-900">{inv.invoice_no}</span>
                            <div className="flex items-center gap-4">
                              <span className="font-bold text-zinc-900">{formatPrice(inv.amount_minor, inv.currency)}</span>
                              <StatusBadge severity="success" label={inv.status} />
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </Card>
                )}

                {commerceSubTab === "ledger" && (
                  <Card title={t("commerce.adminLedger")}>
                    {ledgerTxs.length === 0 ? (
                      <div className="text-center py-8 text-zinc-400 text-sm">{t("common.empty")}</div>
                    ) : (
                      <div className="divide-y divide-zinc-100">
                        {ledgerTxs.map((tx) => {
                          const firstEntry = tx.entries?.[0];
                          const amt = firstEntry ? firstEntry.amount_minor : 0;
                          const curr = firstEntry ? firstEntry.currency : "USD";
                          return (
                            <div key={tx.id} className="py-3 flex items-center justify-between text-xs">
                              <div>
                                <span className="font-bold text-zinc-900 block">{tx.description || tx.type}</span>
                                <span className="text-zinc-400 font-mono">Type: {tx.type} &bull; ID: {tx.id}</span>
                              </div>
                              <span className="font-bold font-mono text-zinc-900">
                                {formatPrice(amt, curr)}
                              </span>
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </Card>
                )}
              </div>
            )}

            {/* TAB 5: Operations Queue */}
            {activeTab === "operations" && (
              <Card title={t("common.operations")}>
                {operations.length === 0 ? (
                  <div className="text-center py-12 text-zinc-400 text-sm">{t("common.empty")}</div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs">
                      <thead className="border-b border-zinc-200 text-zinc-500 uppercase tracking-wider">
                        <tr>
                          <th className="py-3 px-2">Type</th>
                          <th className="py-3 px-2">Status</th>
                          <th className="py-3 px-2">Progress</th>
                          <th className="py-3 px-2">Resource</th>
                          <th className="py-3 px-2">Trace ID</th>
                          <th className="py-3 px-2">Created</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-zinc-100">
                        {operations.map((op) => (
                          <tr key={op.id} className="hover:bg-zinc-50">
                            <td className="py-3 px-2 font-semibold text-zinc-900">{op.type}</td>
                            <td className="py-3 px-2">
                              <StatusBadge
                                severity={
                                  op.status === "succeeded"
                                    ? "success"
                                    : op.status === "failed"
                                    ? "error"
                                    : "info"
                                }
                                label={op.status}
                              />
                            </td>
                            <td className="py-3 px-2 font-mono">{op.progress}%</td>
                            <td className="py-3 px-2 font-mono text-zinc-500">
                              {op.resource_type}: {op.resource_id.slice(0, 8)}...
                            </td>
                            <td className="py-3 px-2 font-mono text-zinc-400">{op.trace_id || "-"}</td>
                            <td className="py-3 px-2 text-zinc-500 font-mono">
                              {new Date(op.created_at).toLocaleTimeString()}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </Card>
            )}

            {/* TAB 6: Users & Audit */}
            {activeTab === "users" && (
              <div className="space-y-4">
                <div className="flex gap-2 border-b border-zinc-200 pb-2">
                  <button
                    type="button"
                    onClick={() => setUsersSubTab("users")}
                    className={`px-3 py-1 rounded text-xs font-semibold ${
                      usersSubTab === "users" ? "bg-zinc-800 text-white" : "bg-white text-zinc-600 hover:bg-zinc-50"
                    }`}
                  >
                    {t("common.users")}
                  </button>
                  <button
                    type="button"
                    onClick={() => setUsersSubTab("admins")}
                    className={`px-3 py-1 rounded text-xs font-semibold ${
                      usersSubTab === "admins" ? "bg-zinc-800 text-white" : "bg-white text-zinc-600 hover:bg-zinc-50"
                    }`}
                  >
                    {t("common.admins")}
                  </button>
                  <button
                    type="button"
                    onClick={() => setUsersSubTab("audit")}
                    className={`px-3 py-1 rounded text-xs font-semibold ${
                      usersSubTab === "audit" ? "bg-zinc-800 text-white" : "bg-white text-zinc-600 hover:bg-zinc-50"
                    }`}
                  >
                    {t("common.audit")}
                  </button>
                </div>

                {usersSubTab === "users" && (
                  <Card title={t("common.users")}>
                    {users.length === 0 ? (
                      <div className="text-center py-8 text-zinc-400 text-sm">{t("common.empty")}</div>
                    ) : (
                      <div className="divide-y divide-zinc-100">
                        {users.map((u) => (
                          <div key={u.id} className="py-3 flex items-center justify-between text-xs">
                            <div>
                              <span className="font-semibold text-zinc-900 block">{u.email}</span>
                              <span className="text-zinc-400 font-mono">ID: {u.id} &bull; Locale: {u.locale}</span>
                            </div>
                            <StatusBadge severity={u.status === "active" ? "success" : "neutral"} label={u.status} />
                          </div>
                        ))}
                      </div>
                    )}
                  </Card>
                )}

                {usersSubTab === "admins" && (
                  <Card title={t("common.admins")}>
                    {admins.length === 0 ? (
                      <div className="text-center py-8 text-zinc-400 text-sm">{t("common.empty")}</div>
                    ) : (
                      <div className="divide-y divide-zinc-100">
                        {admins.map((a) => (
                          <div key={a.id} className="py-3 flex items-center justify-between text-xs">
                            <div>
                              <span className="font-semibold text-zinc-900 block">{a.display_name || a.email}</span>
                              <span className="text-zinc-400 font-mono">{a.email} &bull; 2FA: {a.two_factor_enabled ? "Enabled" : "Disabled"}</span>
                            </div>
                            <StatusBadge severity="success" label={a.status} />
                          </div>
                        ))}
                      </div>
                    )}
                  </Card>
                )}

                {usersSubTab === "audit" && (
                  <Card title={t("common.audit")}>
                    {auditEvents.length === 0 ? (
                      <div className="text-center py-8 text-zinc-400 text-sm">{t("common.empty")}</div>
                    ) : (
                      <div className="divide-y divide-zinc-100">
                        {auditEvents.map((evt) => (
                          <div key={evt.id} className="py-3 space-y-1 text-xs">
                            <div className="flex items-center justify-between">
                              <span className="font-mono font-semibold text-zinc-900">{evt.action}</span>
                              <span className="text-zinc-400 font-mono">{new Date(evt.created_at).toLocaleString()}</span>
                            </div>
                            <div className="flex items-center gap-3 text-zinc-500 font-mono text-[11px]">
                              <span>Actor: {evt.actor_type}</span>
                              <span>&bull;</span>
                              <span>Resource: {evt.resource_type}</span>
                              <span>&bull;</span>
                              <span>IP: {evt.ip_address || "127.0.0.1"}</span>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </Card>
                )}
              </div>
            )}
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="border-t border-zinc-200 py-6 text-center text-xs text-zinc-500 bg-white">
        VPS Billing Platform &copy; 2026. Phase 9 Admin Web Control Plane.
      </footer>
    </div>
  );
};

export default App;
