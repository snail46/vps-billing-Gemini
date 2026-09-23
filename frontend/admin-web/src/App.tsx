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
  ticketApi,
  AdminDTO,
  UserDTO,
  Order,
  Invoice,
  LedgerTransaction,
  Node,
  Provider,
  Instance,
  Operation,
  HealthCheckResult,
  Product,
  Ticket,
} from "@vps-billing/shared";

type AdminTab =
  | "dashboard"
  | "infrastructure"
  | "instances"
  | "commerce"
  | "operations"
  | "tickets"
  | "users";

export const App: React.FC = () => {
  const { t, locale } = useI18n();

  const [admin, setAdmin] = useState<AdminDTO | null>(null);
  const [checkingAuth, setCheckingAuth] = useState(true);

  // Active Tab
  const [activeTab, setActiveTab] = useState<AdminTab>("dashboard");
  const [commerceSubTab, setCommerceSubTab] = useState<"orders" | "invoices" | "ledger" | "products">("orders");
  const [usersSubTab, setUsersSubTab] = useState<"users" | "admins">("users");

  // Login Form
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [loginError, setLoginError] = useState<string | null>(null);

  // Data
  const [health, setHealth] = useState<HealthCheckResult | null>(null);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [ledgerTxs, setLedgerTxs] = useState<LedgerTransaction[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [operations, setOperations] = useState<Operation[]>([]);
  const [users, setUsers] = useState<UserDTO[]>([]);
  const [admins, setAdmins] = useState<AdminDTO[]>([]);
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [adminReplyText, setAdminReplyText] = useState("");

  // Modals
  const [showCreateNodeModal, setShowCreateNodeModal] = useState(false);
  const [nodeName, setNodeName] = useState("");
  const [nodeRegion, setNodeRegion] = useState("us-west");
  const [nodeCPU, setNodeCPU] = useState(32);
  const [nodeRAM, setNodeRAM] = useState(65536);
  const [nodeDisk, setNodeDisk] = useState(2000);
  const [nodeProviderID, setNodeProviderID] = useState("");

  const [showCreateProviderModal, setShowCreateProviderModal] = useState(false);
  const [providerName, setProviderName] = useState("");
  const [providerType, setProviderType] = useState("mock");

  const [showCreateProductModal, setShowCreateProductModal] = useState(false);
  const [productSlug, setProductSlug] = useState("");
  const [productNameEn, setProductNameEn] = useState("");
  const [productNameZh, setProductNameZh] = useState("");

  const [showCreatePlanModal, setShowCreatePlanModal] = useState(false);
  const [planProductID, setPlanProductID] = useState("");
  const [planSlug, setPlanSlug] = useState("");
  const [planNameEn, setPlanNameEn] = useState("");
  const [planNameZh, setPlanNameZh] = useState("");
  const [planCPU, setPlanCPU] = useState(2);
  const [planRAM, setPlanRAM] = useState(2048);
  const [planDisk, setPlanDisk] = useState(40);
  const [planTraffic, setPlanTraffic] = useState(1000);
  const [planPrice, setPlanPrice] = useState(6.00);

  const [inspectOperation, setInspectOperation] = useState<Operation | null>(null);

  const [show2FAModal, setShow2FAModal] = useState(false);
  const [twoFASecret, setTwoFASecret] = useState<string | null>(null);
  const [twoFAUrl, setTwoFAUrl] = useState<string | null>(null);
  const [twoFACode, setTwoFACode] = useState("");
  const [twoFAMsg, setTwoFAMsg] = useState<string | null>(null);

  // Reconcile and Balance Adjustment states
  const [reconciling, setReconciling] = useState(false);
  const [selectedUserForAdjust, setSelectedUserForAdjust] = useState<UserDTO | null>(null);
  const [adjustAmount, setAdjustAmount] = useState<string>("10.00");
  const [adjustReason, setAdjustReason] = useState<string>("Manual credit adjustment by administrator");
  const [adjustingBalance, setAdjustingBalance] = useState(false);

  const [loadingData, setLoadingData] = useState(false);
  const [actionNotice, setActionNotice] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

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
      if (res.success && res.data && res.data.admin) {
        setAdmin(res.data.admin);
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
    if (!admin) return;
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
        if (pList.length > 0 && !nodeProviderID) {
          setNodeProviderID(pList[0].id);
        }
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
        } else if (commerceSubTab === "products") {
          const res = await commerceApi.listProducts();
          setProducts(res.products || []);
          if (res.products && res.products.length > 0 && !planProductID) {
            setPlanProductID(res.products[0].id);
          }
        }
      } else if (activeTab === "operations") {
        const ops = await adminApi.listOperations(50);
        setOperations(ops);
      } else if (activeTab === "tickets") {
        const tList = await ticketApi.adminListTickets();
        setTickets(tList);
        if (selectedTicket) {
          const updated = tList.find((t) => t.id === selectedTicket.id);
          if (updated) setSelectedTicket(updated);
        }
      } else if (activeTab === "users") {
        if (usersSubTab === "users") {
          const uList = await adminApi.listUsers();
          setUsers(uList);
        } else if (usersSubTab === "admins") {
          const aList = await adminApi.listAdmins();
          setAdmins(aList);
        }
      }
    } catch (err: any) {
      console.error(err);
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
      if (res.success && res.data && res.data.admin) {
        setAdmin(res.data.admin);
        setPassword("");
      } else {
        const errRes = res as any;
        setLoginError(t(errRes.error?.message_key || "errors.invalid_credentials"));
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
    }
  };

  const handleCreateNode = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoadingData(true);
    try {
      await adminApi.createNode({
        name: nodeName,
        region: nodeRegion,
        cpu_total: Number(nodeCPU),
        memory_total_mb: Number(nodeRAM),
        disk_total_gb: Number(nodeDisk),
        provider_id: nodeProviderID || undefined,
      });
      setShowCreateNodeModal(false);
      setNodeName("");
      setActionNotice(t("admin.nodeSuccess"));
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
    }
  };

  const handleCreateProvider = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoadingData(true);
    try {
      await adminApi.createProvider({
        name: providerName,
        provider_type: providerType,
      });
      setShowCreateProviderModal(false);
      setProviderName("");
      setActionNotice(t("admin.provSuccess"));
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
    }
  };

  const handleCreateProduct = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoadingData(true);
    try {
      await adminApi.createProduct({
        slug: productSlug,
        name_i18n: { "en-US": productNameEn, "zh-CN": productNameZh },
      });
      setShowCreateProductModal(false);
      setProductSlug("");
      setProductNameEn("");
      setProductNameZh("");
      setActionNotice(t("admin.productSuccess"));
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
    }
  };

  const handleCreatePlan = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoadingData(true);
    try {
      await adminApi.createPlan({
        product_id: planProductID,
        slug: planSlug,
        name_i18n: { "en-US": planNameEn, "zh-CN": planNameZh },
        cpu_cores: Number(planCPU),
        memory_mb: Number(planRAM),
        disk_gb: Number(planDisk),
        traffic_gb: Number(planTraffic),
        price_minor: Math.round(planPrice * 100),
        currency: "USD",
      });
      setShowCreatePlanModal(false);
      setPlanSlug("");
      setPlanNameEn("");
      setPlanNameZh("");
      setActionNotice(t("admin.planSuccess"));
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
    }
  };

  const handleAdminReplyTicket = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTicket || !adminReplyText.trim()) return;
    setLoadingData(true);
    try {
      await ticketApi.adminReplyTicket(selectedTicket.id, adminReplyText);
      setAdminReplyText("");
      const fresh = await ticketApi.getTicket(selectedTicket.id);
      setSelectedTicket(fresh);
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
    }
  };

  const handleToggleTicketStatus = async () => {
    if (!selectedTicket) return;
    setLoadingData(true);
    try {
      const nextStatus = selectedTicket.status === "open" ? "closed" : "open";
      await ticketApi.adminUpdateTicketStatus(selectedTicket.id, nextStatus);
      const fresh = await ticketApi.getTicket(selectedTicket.id);
      setSelectedTicket(fresh);
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
    }
  };

  const handleSetup2FA = async () => {
    setShow2FAModal(true);
    setTwoFAMsg(null);
    try {
      const res = await adminApi.setup2FA();
      setTwoFASecret(res.secret);
      setTwoFAUrl(res.otpauth_url);
    } catch (err: any) {
      setTwoFAMsg(err?.message || t("errors.internal_error"));
    }
  };

  const handleEnable2FA = async () => {
    if (!twoFASecret || !twoFACode) return;
    try {
      await adminApi.enable2FA(twoFASecret, twoFACode);
      setTwoFAMsg(t("admin.twofaSuccess"));
      setTwoFACode("");
    } catch (err: any) {
      setTwoFAMsg(t("admin.twofaInvalidCode"));
    }
  };

  const handleTriggerReconcile = async () => {
    setReconciling(true);
    setActionNotice(null);
    setActionError(null);
    try {
      await adminApi.triggerReconcile();
      setActionNotice(t("admin.reconcileSuccess"));
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setReconciling(false);
    }
  };

  const handleAdjustBalance = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedUserForAdjust) return;
    const num = parseFloat(adjustAmount);
    if (isNaN(num)) return;
    const minorUnits = Math.round(num * 100);
    setAdjustingBalance(true);
    setActionNotice(null);
    setActionError(null);
    try {
      await adminApi.adjustUserBalance(
        selectedUserForAdjust.id,
        minorUnits,
        "USD",
        adjustReason || "Admin manual adjustment"
      );
      setActionNotice(t("admin.adjustBalanceSuccess"));
      setSelectedUserForAdjust(null);
      setAdjustAmount("10.00");
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setAdjustingBalance(false);
    }
  };

  const formatPrice = (minor: number, currency: string) => {
    return `${currency} ${(minor / 100).toFixed(2)}`;
  };

  if (checkingAuth) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-zinc-900 text-zinc-400 text-sm">
        {t("common.loading")}
      </div>
    );
  }

  // Unauthenticated Admin View: Clean privileged login
  if (!admin) {
    return (
      <div className="min-h-screen flex flex-col justify-center items-center bg-zinc-950 px-4 text-zinc-100">
        <div className="absolute top-4 right-4">
          <LanguageSwitcher />
        </div>

        <div className="w-full max-w-md space-y-6">
          <div className="text-center space-y-2">
            <div className="w-12 h-12 rounded-xl bg-blue-600 text-white font-bold text-xl flex items-center justify-center mx-auto shadow-lg">
              ADM
            </div>
            <h1 className="text-2xl font-extrabold tracking-tight text-white">
              {t("common.appNameAdmin")}
            </h1>
            <p className="text-xs text-zinc-400">
              {t("admin.loginDesc")}
            </p>
          </div>

          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-xl space-y-4">
            {loginError && <Alert severity="error">{loginError}</Alert>}

            <form onSubmit={handleLogin} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">
                  {t("auth.email")}
                </label>
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@vps-billing.local"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">
                  {t("auth.password")}
                </label>
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Admin123456!"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div className="p-3 bg-zinc-950/80 rounded-lg border border-zinc-800 text-[11px] text-zinc-400 space-y-0.5">
                <span className="font-semibold text-zinc-300 block">{t("admin.defaultCreds")}</span>
                <span>admin@vps-billing.local &bull; Admin123456!</span>
              </div>

              <Button type="submit" isLoading={submitting} className="w-full font-bold">
                {t("auth.login")}
              </Button>
            </form>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col bg-zinc-950 text-zinc-100">
      {/* Top Admin Header */}
      <header className="bg-zinc-900 border-b border-zinc-800 sticky top-0 z-20 shadow-md">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-blue-600 text-white font-bold flex items-center justify-center text-sm">
              ADM
            </div>
            <div>
              <span className="font-bold text-white tracking-tight text-sm sm:text-base block">
                {t("common.appNameAdmin")}
              </span>
              <span className="text-[10px] text-zinc-400 font-mono tracking-wider uppercase block">
                {t("admin.controlPlaneTag")}
              </span>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <LanguageSwitcher />

            <button
              onClick={handleSetup2FA}
              className="px-2.5 py-1.5 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg text-xs font-medium text-zinc-300 transition-colors cursor-pointer"
            >
              {t("common.setup2fa")}
            </button>

            <div className="hidden sm:block text-right">
              <span className="text-xs font-semibold text-zinc-300 block truncate max-w-[150px]">
                {admin.email}
              </span>
            </div>

            <Button size="sm" variant="outline" onClick={handleLogout}>
              {t("auth.logout")}
            </Button>
          </div>
        </div>

        {/* Admin Navigation Tabs */}
        <div className="border-t border-zinc-800/80 bg-zinc-900/60 px-4 sm:px-6">
          <div className="max-w-7xl mx-auto flex items-center gap-1 sm:gap-2 overflow-x-auto py-2">
            {[
              { id: "dashboard", label: t("common.dashboard") },
              { id: "infrastructure", label: t("admin.infraTitle") },
              { id: "instances", label: t("common.servers") },
              { id: "commerce", label: t("common.billing") },
              { id: "operations", label: t("common.operations") },
              { id: "tickets", label: t("common.tickets") },
              { id: "users", label: t("common.users") },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => {
                  setActiveTab(tab.id as AdminTab);
                  setSelectedTicket(null);
                }}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg whitespace-nowrap transition-colors cursor-pointer ${
                  activeTab === tab.id
                    ? "bg-blue-600 text-white"
                    : "text-zinc-400 hover:text-white hover:bg-zinc-800"
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>
      </header>

      {/* Main Container */}
      <main className="flex-1 max-w-7xl mx-auto px-4 sm:px-6 py-6 w-full">
        {actionNotice && (
          <div className="mb-4">
            <Alert severity="success">{actionNotice}</Alert>
          </div>
        )}
        {actionError && (
          <div className="mb-4">
            <Alert severity="error">{actionError}</Alert>
          </div>
        )}

        {/* TAB 1: Dashboard Overview */}
        {activeTab === "dashboard" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">{t("admin.telemetryTitle")}</h2>
                <p className="text-xs text-zinc-400 mt-0.5">{t("admin.telemetrySubtitle")}</p>
              </div>
              <Button size="sm" variant="outline" onClick={loadTabData} isLoading={loadingData}>
                {t("common.refresh")}
              </Button>
            </div>

            {/* Health Indicators */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">{t("admin.nodesOnline")}</span>
                  <span className="text-2xl font-bold text-white mt-1 block">
                    {nodes.filter((n) => n.status === "active").length} / {nodes.length}
                  </span>
                  <span className="text-[11px] text-emerald-400 mt-0.5 block">{t("admin.allOperational")}</span>
                </div>
                <div className="w-10 h-10 rounded-lg bg-emerald-950 text-emerald-400 flex items-center justify-center font-bold text-lg">
                  &#9679;
                </div>
              </div>

              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">{t("admin.liveInstances")}</span>
                  <span className="text-2xl font-bold text-blue-400 mt-1 block">
                    {instances.filter((i) => i.observed_state === "running").length}
                  </span>
                  <span className="text-[11px] text-zinc-400 mt-0.5 block">{instances.length} {t("admin.provisioned")}</span>
                </div>
                <div className="w-10 h-10 rounded-lg bg-blue-950 text-blue-400 flex items-center justify-center font-bold text-lg">
                  &bull;
                </div>
              </div>

              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">{t("admin.totalOrders")}</span>
                  <span className="text-2xl font-bold text-purple-400 mt-1 block">
                    {orders.length}
                  </span>
                  <span className="text-[11px] text-zinc-400 mt-0.5 block">
                    {orders.filter((o) => o.status === "paid").length} {t("admin.settled")}
                  </span>
                </div>
                <div className="w-10 h-10 rounded-lg bg-purple-950 text-purple-400 flex items-center justify-center font-bold text-lg">
                  $
                </div>
              </div>

              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">{t("common.tickets")}</span>
                  <span className="text-2xl font-bold text-amber-400 mt-1 block">
                    {tickets.filter((t) => t.status === "open").length}
                  </span>
                  <span className="text-[11px] text-zinc-400 mt-0.5 block">{t("tickets.open")}</span>
                </div>
                <div className="w-10 h-10 rounded-lg bg-amber-950 text-amber-400 flex items-center justify-center font-bold text-lg">
                  ?
                </div>
              </div>
            </div>

            {/* Health Table */}
            <div className="bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-xs space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="font-bold text-white text-base">{t("common.systemHealth")}</h3>
                <Button size="sm" variant="outline" onClick={handleTriggerReconcile} isLoading={reconciling}>
                  {t("admin.triggerReconcile")}
                </Button>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 text-xs">
                <div className="p-4 bg-zinc-950 rounded-xl border border-zinc-800 flex items-center justify-between">
                  <span className="text-zinc-400">{t("common.database")}</span>
                  <StatusBadge
                    severity={
                      health?.checks?.database === "healthy" ||
                      health?.checks?.database === "ok" ||
                      health?.checks?.postgres === "healthy" ||
                      health?.checks?.postgres === "ok"
                        ? "success"
                        : "error"
                    }
                    label={
                      health?.checks?.database === "healthy" ||
                      health?.checks?.database === "ok" ||
                      health?.checks?.postgres === "healthy" ||
                      health?.checks?.postgres === "ok"
                        ? t("common.healthy")
                        : t("common.unhealthy")
                    }
                  />
                </div>
                <div className="p-4 bg-zinc-950 rounded-xl border border-zinc-800 flex items-center justify-between">
                  <span className="text-zinc-400">{t("common.redis")}</span>
                  <StatusBadge
                    severity={
                      health?.checks?.redis === "healthy" ||
                      health?.checks?.redis === "ok" ||
                      health?.checks?.queue === "healthy"
                        ? "success"
                        : "error"
                    }
                    label={
                      health?.checks?.redis === "healthy" ||
                      health?.checks?.redis === "ok" ||
                      health?.checks?.queue === "healthy"
                        ? t("common.healthy")
                        : t("common.unhealthy")
                    }
                  />
                </div>
                <div className="p-4 bg-zinc-950 rounded-xl border border-zinc-800 flex items-center justify-between">
                  <span className="text-zinc-400">{t("common.apiLive")}</span>
                  <StatusBadge
                    severity={health?.status === "ready" || health?.status === "alive" ? "success" : "warning"}
                    label={health?.status === "ready" || health?.status === "alive" ? t("common.healthy") : t("common.unhealthy")}
                  />
                </div>
                <div className="p-4 bg-zinc-950 rounded-xl border border-zinc-800 flex items-center justify-between">
                  <span className="text-zinc-400">{t("common.apiReady")}</span>
                  <StatusBadge
                    severity={health?.status === "ready" ? "success" : "error"}
                    label={health?.status === "ready" ? t("common.healthy") : t("common.unhealthy")}
                  />
                </div>
              </div>
            </div>
          </div>
        )}

        {/* TAB 2: Infrastructure Management */}
        {activeTab === "infrastructure" && (
          <div className="space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">{t("admin.infraTitle")}</h2>
                <p className="text-xs text-zinc-400 mt-0.5">{t("admin.infraSubtitle")}</p>
              </div>
              <div className="flex items-center gap-2">
                <Button size="sm" onClick={() => setShowCreateNodeModal(true)}>
                  + {t("admin.addNode")}
                </Button>
                <Button size="sm" variant="outline" onClick={() => setShowCreateProviderModal(true)}>
                  + {t("admin.addProvider")}
                </Button>
              </div>
            </div>

            {/* Nodes List */}
            <div className="space-y-4">
              <h3 className="font-bold text-white text-base">{t("admin.nodesTab")}</h3>
              {nodes.length === 0 ? (
                <Card>
                  <div className="text-center py-10 text-zinc-500 text-sm">{t("common.empty")}</div>
                </Card>
              ) : (
                <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-zinc-950 border-b border-zinc-800 text-zinc-400 font-semibold uppercase">
                      <tr>
                        <th className="px-5 py-3.5">{t("admin.nodeName")}</th>
                        <th className="px-5 py-3.5">{t("admin.region")}</th>
                        <th className="px-5 py-3.5">{t("common.status")}</th>
                        <th className="px-5 py-3.5">{t("admin.totalCPU")}</th>
                        <th className="px-5 py-3.5">{t("admin.totalRAM")}</th>
                        <th className="px-5 py-3.5">{t("admin.totalDisk")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-zinc-800">
                      {nodes.map((n) => (
                        <tr key={n.id} className="hover:bg-zinc-800/50 transition-colors">
                          <td className="px-5 py-4 font-bold text-white">{n.name}</td>
                          <td className="px-5 py-4 font-mono text-zinc-400">{n.region}</td>
                          <td className="px-5 py-4">
                            <StatusBadge severity={n.status === "active" ? "success" : "error"} label={n.status} />
                          </td>
                          <td className="px-5 py-4 text-zinc-300">{n.cpu_total} vCPU</td>
                          <td className="px-5 py-4 text-zinc-300">{n.memory_total_mb} MB</td>
                          <td className="px-5 py-4 text-zinc-300">{n.disk_total_gb} GB</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>

            {/* Providers List */}
            <div className="space-y-4 pt-4">
              <h3 className="font-bold text-white text-base">{t("admin.providersTab")}</h3>
              {providers.length === 0 ? (
                <Card>
                  <div className="text-center py-10 text-zinc-500 text-sm">{t("common.empty")}</div>
                </Card>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  {providers.map((p) => (
                    <div key={p.id} className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 shadow-xs space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="font-bold text-white text-sm">{p.name}</span>
                        <StatusBadge severity="success" label={p.status} />
                      </div>
                      <span className="text-xs text-zinc-400 block font-mono">
                        {t("admin.provType")}: {p.provider_type}
                      </span>
                      <span className="text-[11px] text-zinc-500 block font-mono">
                        ID: {p.id}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* TAB 3: Instances Management */}
        {activeTab === "instances" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">{t("common.servers")}</h2>
                <p className="text-xs text-zinc-400 mt-0.5">{t("instance.subtitle")}</p>
              </div>
              <Button size="sm" variant="outline" onClick={loadTabData} isLoading={loadingData}>
                {t("common.refresh")}
              </Button>
            </div>

            {instances.length === 0 ? (
              <Card>
                <div className="text-center py-12 text-zinc-500 text-sm">{t("instance.noInstances")}</div>
              </Card>
            ) : (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 border-b border-zinc-800 text-zinc-400 font-semibold uppercase">
                    <tr>
                      <th className="px-5 py-3.5">{t("instance.title")}</th>
                      <th className="px-5 py-3.5">{t("instance.ipv4")}</th>
                      <th className="px-5 py-3.5">{t("common.status")}</th>
                      <th className="px-5 py-3.5">{t("instance.specs")}</th>
                      <th className="px-5 py-3.5">{t("instance.created")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800">
                    {instances.map((i) => (
                      <tr key={i.id} className="hover:bg-zinc-800/50 transition-colors">
                        <td className="px-5 py-4 font-bold text-white">{i.name}</td>
                        <td className="px-5 py-4 font-mono text-zinc-300">{i.primary_ipv4 || "192.168.1.100"}</td>
                        <td className="px-5 py-4">
                          <StatusBadge
                            severity={i.observed_state === "running" ? "success" : "neutral"}
                            label={t(`instance.status.${i.observed_state}`) || i.observed_state}
                          />
                        </td>
                        <td className="px-5 py-4 text-zinc-400">
                          {i.cpu_cores} vCPU &bull; {i.memory_mb} MB &bull; {i.disk_gb} GB
                        </td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">
                          {new Date(i.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* TAB 4: Commerce Management */}
        {activeTab === "commerce" && (
          <div className="space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">{t("admin.catalogTitle")}</h2>
                <p className="text-xs text-zinc-400 mt-0.5">{t("admin.catalogSubtitle")}</p>
              </div>
              <div className="flex items-center gap-2">
                <Button size="sm" onClick={() => setShowCreateProductModal(true)}>
                  + {t("admin.addProduct")}
                </Button>
                <Button size="sm" variant="outline" onClick={() => setShowCreatePlanModal(true)}>
                  + {t("admin.addPlan")}
                </Button>
              </div>
            </div>

            {/* Subtabs */}
            <div className="flex items-center gap-2 border-b border-zinc-800 pb-2">
              {[
                { id: "orders", label: t("commerce.orders") },
                { id: "invoices", label: t("commerce.invoices") },
                { id: "ledger", label: t("commerce.adminLedger") },
                { id: "products", label: t("commerce.catalog") },
              ].map((st) => (
                <button
                  key={st.id}
                  onClick={() => setCommerceSubTab(st.id as any)}
                  className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors cursor-pointer ${
                    commerceSubTab === st.id
                      ? "bg-blue-600 text-white"
                      : "text-zinc-400 hover:text-white hover:bg-zinc-800"
                  }`}
                >
                  {st.label}
                </button>
              ))}
            </div>

            {/* Orders Subtab */}
            {commerceSubTab === "orders" && (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 border-b border-zinc-800 text-zinc-400 font-semibold uppercase">
                    <tr>
                      <th className="px-5 py-3.5">{t("commerce.orderNo")}</th>
                      <th className="px-5 py-3.5">{t("common.status")}</th>
                      <th className="px-5 py-3.5">{t("commerce.total")}</th>
                      <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800">
                    {orders.map((o) => (
                      <tr key={o.id} className="hover:bg-zinc-800/50 transition-colors">
                        <td className="px-5 py-4 font-mono font-medium text-white">{o.order_no}</td>
                        <td className="px-5 py-4">
                          <StatusBadge severity={o.status === "paid" ? "success" : "warning"} label={t(`order.status.${o.status}`) || o.status} />
                        </td>
                        <td className="px-5 py-4 font-bold text-white">{formatPrice(o.total_minor, o.currency)}</td>
                        <td className="px-5 py-4 text-zinc-400 font-mono">{new Date(o.created_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Invoices Subtab */}
            {commerceSubTab === "invoices" && (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 border-b border-zinc-800 text-zinc-400 font-semibold uppercase">
                    <tr>
                      <th className="px-5 py-3.5">{t("commerce.invoiceNo")}</th>
                      <th className="px-5 py-3.5">{t("common.status")}</th>
                      <th className="px-5 py-3.5">{t("commerce.amount")}</th>
                      <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800">
                    {invoices.map((inv) => (
                      <tr key={inv.id} className="hover:bg-zinc-800/50 transition-colors">
                        <td className="px-5 py-4 font-mono font-medium text-white">{inv.invoice_no}</td>
                        <td className="px-5 py-4">
                          <StatusBadge severity={inv.status === "paid" ? "success" : "warning"} label={t(`order.status.${inv.status}`) || inv.status} />
                        </td>
                        <td className="px-5 py-4 font-bold text-white">{formatPrice(inv.amount_minor, inv.currency)}</td>
                        <td className="px-5 py-4 text-zinc-400 font-mono">{new Date(inv.created_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Ledger Subtab */}
            {commerceSubTab === "ledger" && (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 border-b border-zinc-800 text-zinc-400 font-semibold uppercase">
                    <tr>
                      <th className="px-5 py-3.5">{t("commerce.txId")}</th>
                      <th className="px-5 py-3.5">{t("commerce.desc")}</th>
                      <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800">
                    {ledgerTxs.map((lt) => (
                      <tr key={lt.id} className="hover:bg-zinc-800/50 transition-colors">
                        <td className="px-5 py-4 font-mono text-white text-[11px]">{lt.id}</td>
                        <td className="px-5 py-4 text-zinc-300">{lt.description}</td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">{new Date(lt.created_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Products Subtab */}
            {commerceSubTab === "products" && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {products.map((p) => (
                  <div key={p.id} className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 shadow-xs space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-white text-base">{p.name_i18n?.[locale] || p.slug}</span>
                      <StatusBadge severity="success" label={p.status} />
                    </div>
                    <span className="text-xs text-zinc-400 block font-mono">{t("admin.productSlug")}: {p.slug}</span>
                    <div className="pt-2 border-t border-zinc-800 space-y-1">
                      <span className="text-xs font-semibold text-zinc-300 block">{t("common.createPlan")}:</span>
                      {(p.plans || []).map((pl) => (
                        <div key={pl.id} className="flex justify-between text-xs text-zinc-400">
                          <span>{pl.name_i18n?.[locale] || pl.slug}</span>
                          <span className="text-blue-400 font-bold">{formatPrice(pl.price_minor, pl.currency)}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* TAB 5: Operations & Inspector */}
        {activeTab === "operations" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">{t("common.operations")}</h2>
                <p className="text-xs text-zinc-400 mt-0.5">{t("admin.opInspector")}</p>
              </div>
              <Button size="sm" variant="outline" onClick={loadTabData} isLoading={loadingData}>
                {t("common.refresh")}
              </Button>
            </div>

            {operations.length === 0 ? (
              <Card>
                <div className="text-center py-12 text-zinc-500 text-sm">{t("common.empty")}</div>
              </Card>
            ) : (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 border-b border-zinc-800 text-zinc-400 font-semibold uppercase">
                    <tr>
                      <th className="px-5 py-3.5">{t("common.operations")}</th>
                      <th className="px-5 py-3.5">{t("common.status")}</th>
                      <th className="px-5 py-3.5">{t("admin.traceId")}</th>
                      <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                      <th className="px-5 py-3.5 text-right">{t("common.actions")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800">
                    {operations.map((op) => (
                      <tr key={op.id} className="hover:bg-zinc-800/50 transition-colors">
                        <td className="px-5 py-4 font-mono font-medium text-white">{op.type}</td>
                        <td className="px-5 py-4">
                          <StatusBadge severity={op.status === "succeeded" ? "success" : "warning"} label={t(`operation.status.${op.status}`) || op.status} />
                        </td>
                        <td className="px-5 py-4 font-mono text-zinc-500 text-[11px]">{op.trace_id || op.id}</td>
                        <td className="px-5 py-4 text-zinc-400 font-mono">{new Date(op.created_at).toLocaleString()}</td>
                        <td className="px-5 py-4 text-right">
                          <Button size="sm" variant="outline" onClick={() => setInspectOperation(op)}>
                            {t("admin.opInspector")} &rarr;
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* TAB 6: Ticket Desk */}
        {activeTab === "tickets" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">{t("admin.ticketDesk")}</h2>
                <p className="text-xs text-zinc-400 mt-0.5">{t("admin.ticketDeskSubtitle")}</p>
              </div>
              <Button size="sm" variant="outline" onClick={loadTabData} isLoading={loadingData}>
                {t("common.refresh")}
              </Button>
            </div>

            {selectedTicket ? (
              <div className="space-y-4">
                <Button size="sm" variant="outline" onClick={() => setSelectedTicket(null)}>
                  &larr; {t("tickets.back")}
                </Button>

                <div className="bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-xs space-y-6">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800 pb-4">
                    <div>
                      <h3 className="text-lg font-bold text-white">{selectedTicket.subject}</h3>
                      <span className="text-xs text-zinc-400 font-mono">
                        {t("tickets.ticketId")}: {selectedTicket.id} &bull; {t("instance.created")}: {new Date(selectedTicket.created_at).toLocaleString()}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <StatusBadge severity={selectedTicket.status === "open" ? "warning" : "neutral"} label={t(`tickets.${selectedTicket.status}`) || selectedTicket.status} />
                      <Button size="sm" variant="outline" onClick={handleToggleTicketStatus}>
                        {selectedTicket.status === "open" ? t("admin.closeTicket") : t("admin.reopenTicket")}
                      </Button>
                    </div>
                  </div>

                  {/* Messages */}
                  <div className="space-y-4 max-h-[400px] overflow-y-auto pr-2">
                    {(selectedTicket.messages || []).map((msg) => {
                      const isStaff = msg.sender_type === "admin";
                      return (
                        <div key={msg.id} className={`flex flex-col ${isStaff ? "items-end" : "items-start"}`}>
                          <div className="flex items-center gap-2 text-xs text-zinc-400 mb-1">
                            <span className="font-semibold text-zinc-300">
                              {isStaff ? t("tickets.staff") : t("tickets.you")}
                            </span>
                            <span>&bull;</span>
                            <span>{new Date(msg.created_at).toLocaleTimeString()}</span>
                          </div>
                          <div className={`p-4 rounded-2xl max-w-lg text-sm leading-relaxed ${isStaff ? "bg-blue-600 text-white rounded-br-none" : "bg-zinc-800 text-zinc-200 rounded-bl-none"}`}>
                            {msg.message}
                          </div>
                        </div>
                      );
                    })}
                  </div>

                  {/* Admin Reply Input */}
                  <form onSubmit={handleAdminReplyTicket} className="pt-4 border-t border-zinc-800 flex gap-3">
                    <input
                      type="text"
                      required
                      value={adminReplyText}
                      onChange={(e) => setAdminReplyText(e.target.value)}
                      placeholder={t("admin.replyPlaceholder")}
                      className="flex-1 px-4 py-2 bg-zinc-950 border border-zinc-700 rounded-xl text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                    <Button type="submit" isLoading={loadingData}>
                      {t("admin.sendStaffReply")}
                    </Button>
                  </form>
                </div>
              </div>
            ) : (
              <div>
                {tickets.length === 0 ? (
                  <Card>
                    <div className="text-center py-12 text-zinc-500 text-sm">{t("tickets.noTickets")}</div>
                  </Card>
                ) : (
                  <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                    <table className="w-full text-left text-xs">
                      <thead className="bg-zinc-950 border-b border-zinc-800 text-zinc-400 font-semibold uppercase">
                        <tr>
                          <th className="px-5 py-3.5">{t("tickets.subject")}</th>
                          <th className="px-5 py-3.5">{t("tickets.priority")}</th>
                          <th className="px-5 py-3.5">{t("common.status")}</th>
                          <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                          <th className="px-5 py-3.5 text-right">{t("common.actions")}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-zinc-800">
                        {tickets.map((tkt) => (
                          <tr key={tkt.id} className="hover:bg-zinc-800/50 transition-colors cursor-pointer" onClick={() => setSelectedTicket(tkt)}>
                            <td className="px-5 py-4 font-bold text-white">{tkt.subject}</td>
                            <td className="px-5 py-4 font-semibold text-zinc-300">
                              {t(`tickets.priority${tkt.priority.charAt(0).toUpperCase() + tkt.priority.slice(1)}`)}
                            </td>
                            <td className="px-5 py-4">
                              <StatusBadge severity={tkt.status === "open" ? "warning" : "neutral"} label={t(`tickets.${tkt.status}`) || tkt.status} />
                            </td>
                            <td className="px-5 py-4 text-zinc-400 font-mono">{new Date(tkt.created_at).toLocaleString()}</td>
                            <td className="px-5 py-4 text-right">
                              <Button size="sm" variant="outline">{t("common.view")} &rarr;</Button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* TAB 7: Users & Admins Management */}
        {activeTab === "users" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">{t("common.users")}</h2>
                <p className="text-xs text-zinc-400 mt-0.5">{t("auth.adminPortal")}</p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setUsersSubTab("users")}
                  className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors cursor-pointer ${
                    usersSubTab === "users" ? "bg-blue-600 text-white" : "text-zinc-400 hover:text-white hover:bg-zinc-800"
                  }`}
                >
                  {t("common.users")}
                </button>
                <button
                  onClick={() => setUsersSubTab("admins")}
                  className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors cursor-pointer ${
                    usersSubTab === "admins" ? "bg-blue-600 text-white" : "text-zinc-400 hover:text-white hover:bg-zinc-800"
                  }`}
                >
                  {t("common.admins")}
                </button>
              </div>
            </div>

            {usersSubTab === "users" ? (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 border-b border-zinc-800 text-zinc-400 font-semibold uppercase">
                    <tr>
                      <th className="px-5 py-3.5">{t("auth.email")}</th>
                      <th className="px-5 py-3.5">{t("common.status")}</th>
                      <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                      <th className="px-5 py-3.5 text-right">{t("common.actions")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800">
                    {users.map((u) => (
                      <tr key={u.id} className="hover:bg-zinc-800/50 transition-colors">
                        <td className="px-5 py-4 font-bold text-white">{u.email}</td>
                        <td className="px-5 py-4"><StatusBadge severity="success" label={u.status} /></td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">{new Date(u.created_at).toLocaleString()}</td>
                        <td className="px-5 py-4 text-right">
                          <Button size="sm" variant="outline" onClick={() => setSelectedUserForAdjust(u)}>
                            {t("admin.adjustBalance")}
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 border-b border-zinc-800 text-zinc-400 font-semibold uppercase">
                    <tr>
                      <th className="px-5 py-3.5">{t("auth.email")}</th>
                      <th className="px-5 py-3.5">{t("common.status")}</th>
                      <th className="px-5 py-3.5">2FA</th>
                      <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800">
                    {admins.map((a) => (
                      <tr key={a.id} className="hover:bg-zinc-800/50 transition-colors">
                        <td className="px-5 py-4 font-bold text-white">{a.email}</td>
                        <td className="px-5 py-4"><StatusBadge severity="success" label={a.status} /></td>
                        <td className="px-5 py-4"><StatusBadge severity={a.two_factor_enabled ? "success" : "neutral"} label={a.two_factor_enabled ? "ON" : "OFF"} /></td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">{new Date(a.created_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </main>

      {/* DRAWER: Operation Inspector */}
      {inspectOperation && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex justify-end">
          <div className="bg-zinc-900 border-l border-zinc-800 w-full max-w-lg p-6 flex flex-col justify-between shadow-2xl animate-in slide-in-from-right">
            <div className="space-y-6 overflow-y-auto">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-4">
                <h3 className="text-lg font-bold text-white">{t("admin.opInspector")}</h3>
                <button
                  onClick={() => setInspectOperation(null)}
                  className="text-zinc-400 hover:text-white font-bold cursor-pointer text-xl"
                >
                  &times;
                </button>
              </div>

              <div className="space-y-2 text-xs">
                <div className="flex justify-between py-1 border-b border-zinc-800/60">
                  <span className="text-zinc-400">{t("admin.traceId")}</span>
                  <span className="font-mono text-zinc-200">{inspectOperation.trace_id || inspectOperation.id}</span>
                </div>
                <div className="flex justify-between py-1 border-b border-zinc-800/60">
                  <span className="text-zinc-400">{t("common.operations")}</span>
                  <span className="font-mono text-zinc-200">{inspectOperation.type}</span>
                </div>
                <div className="flex justify-between py-1 border-b border-zinc-800/60">
                  <span className="text-zinc-400">{t("common.status")}</span>
                  <StatusBadge severity={inspectOperation.status === "succeeded" ? "success" : "warning"} label={t(`operation.status.${inspectOperation.status}`) || inspectOperation.status} />
                </div>
              </div>

              {/* Step Timeline */}
              <div className="space-y-3">
                <h4 className="font-bold text-zinc-200 text-sm">{t("admin.stepTimeline")}</h4>
                <div className="space-y-2">
                  {(inspectOperation.steps || []).map((step, idx) => (
                    <div key={idx} className="p-3 bg-zinc-950 rounded-lg border border-zinc-800/80 flex items-center justify-between text-xs">
                      <span className="font-mono text-zinc-300">{step.step_key}</span>
                      <StatusBadge severity={step.status === "succeeded" ? "success" : "neutral"} label={step.status} />
                    </div>
                  ))}
                </div>
              </div>

              {/* Raw Diagnostics */}
              <div className="space-y-2">
                <h4 className="font-bold text-zinc-200 text-sm">{t("admin.rawDiagnostics")}</h4>
                <pre className="p-4 bg-zinc-950 rounded-xl border border-zinc-800 text-[11px] font-mono text-zinc-400 overflow-x-auto max-h-48">
                  {JSON.stringify(inspectOperation, null, 2)}
                </pre>
              </div>
            </div>

            <div className="pt-4 border-t border-zinc-800 flex justify-end">
              <Button size="sm" variant="outline" onClick={() => setInspectOperation(null)}>
                {t("common.close")}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: Add Node */}
      {showCreateNodeModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 shadow-xl space-y-4">
            <h3 className="text-lg font-bold text-white">{t("admin.addNode")}</h3>
            <form onSubmit={handleCreateNode} className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.nodeName")}</label>
                <input
                  type="text"
                  required
                  value={nodeName}
                  onChange={(e) => setNodeName(e.target.value)}
                  placeholder="node-01.dc1"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.region")}</label>
                <input
                  type="text"
                  required
                  value={nodeRegion}
                  onChange={(e) => setNodeRegion(e.target.value)}
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="grid grid-cols-3 gap-2">
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">{t("commerce.cpu")}</label>
                  <input
                    type="number"
                    value={nodeCPU}
                    onChange={(e) => setNodeCPU(Number(e.target.value))}
                    className="w-full px-2 py-1.5 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">{t("commerce.memory")}</label>
                  <input
                    type="number"
                    value={nodeRAM}
                    onChange={(e) => setNodeRAM(Number(e.target.value))}
                    className="w-full px-2 py-1.5 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">{t("commerce.disk")}</label>
                  <input
                    type="number"
                    value={nodeDisk}
                    onChange={(e) => setNodeDisk(Number(e.target.value))}
                    className="w-full px-2 py-1.5 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <Button type="button" variant="outline" size="sm" onClick={() => setShowCreateNodeModal(false)}>
                  {t("common.cancel")}
                </Button>
                <Button type="submit" size="sm" isLoading={loadingData}>
                  {t("common.confirm")}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Add Provider */}
      {showCreateProviderModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 shadow-xl space-y-4">
            <h3 className="text-lg font-bold text-white">{t("admin.addProvider")}</h3>
            <form onSubmit={handleCreateProvider} className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.provName")}</label>
                <input
                  type="text"
                  required
                  value={providerName}
                  onChange={(e) => setProviderName(e.target.value)}
                  placeholder="CLICD Cluster 1"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.provType")}</label>
                <select
                  value={providerType}
                  onChange={(e) => setProviderType(e.target.value)}
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="mock">Mock Provider</option>
                  <option value="clicd">CLICD Direct Provider</option>
                  <option value="runman">Runman gRPC Agent</option>
                </select>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <Button type="button" variant="outline" size="sm" onClick={() => setShowCreateProviderModal(false)}>
                  {t("common.cancel")}
                </Button>
                <Button type="submit" size="sm" isLoading={loadingData}>
                  {t("common.confirm")}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Add Product */}
      {showCreateProductModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 shadow-xl space-y-4">
            <h3 className="text-lg font-bold text-white">{t("admin.addProduct")}</h3>
            <form onSubmit={handleCreateProduct} className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.productSlug")}</label>
                <input
                  type="text"
                  required
                  value={productSlug}
                  onChange={(e) => setProductSlug(e.target.value)}
                  placeholder="standard-vps"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.prodNameEn")}</label>
                <input
                  type="text"
                  required
                  value={productNameEn}
                  onChange={(e) => setProductNameEn(e.target.value)}
                  placeholder="Standard Cloud VPS"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.prodNameZh")}</label>
                <input
                  type="text"
                  required
                  value={productNameZh}
                  onChange={(e) => setProductNameZh(e.target.value)}
                  placeholder="标准云服务器"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <Button type="button" variant="outline" size="sm" onClick={() => setShowCreateProductModal(false)}>
                  {t("common.cancel")}
                </Button>
                <Button type="submit" size="sm" isLoading={loadingData}>
                  {t("common.confirm")}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Add Plan */}
      {showCreatePlanModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 shadow-xl space-y-4">
            <h3 className="text-lg font-bold text-white">{t("admin.addPlan")}</h3>
            <form onSubmit={handleCreatePlan} className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.targetProduct")}</label>
                <select
                  value={planProductID}
                  onChange={(e) => setPlanProductID(e.target.value)}
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {products.map((p) => (
                    <option key={p.id} value={p.id}>{p.name_i18n?.[locale] || p.slug}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.planSlug")}</label>
                <input
                  type="text"
                  required
                  value={planSlug}
                  onChange={(e) => setPlanSlug(e.target.value)}
                  placeholder="vps-2c-4g"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.planNameEn")}</label>
                  <input
                    type="text"
                    required
                    value={planNameEn}
                    onChange={(e) => setPlanNameEn(e.target.value)}
                    placeholder="2 Core 4GB RAM"
                    className="w-full px-2 py-1.5 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.planNameZh")}</label>
                  <input
                    type="text"
                    required
                    value={planNameZh}
                    onChange={(e) => setPlanNameZh(e.target.value)}
                    placeholder="2核 4G内存"
                    className="w-full px-2 py-1.5 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
              </div>
              <div className="grid grid-cols-5 gap-2">
                <div>
                  <label className="block text-[11px] font-medium text-zinc-400 mb-1">{t("commerce.cpu")}</label>
                  <input
                    type="number"
                    value={planCPU}
                    onChange={(e) => setPlanCPU(Number(e.target.value))}
                    className="w-full px-2 py-1 bg-zinc-950 border border-zinc-700 rounded-lg text-xs text-white"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-zinc-400 mb-1">{t("commerce.memory")}</label>
                  <input
                    type="number"
                    value={planRAM}
                    onChange={(e) => setPlanRAM(Number(e.target.value))}
                    className="w-full px-2 py-1 bg-zinc-950 border border-zinc-700 rounded-lg text-xs text-white"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-zinc-400 mb-1">{t("commerce.disk")}</label>
                  <input
                    type="number"
                    value={planDisk}
                    onChange={(e) => setPlanDisk(Number(e.target.value))}
                    className="w-full px-2 py-1 bg-zinc-950 border border-zinc-700 rounded-lg text-xs text-white"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-zinc-400 mb-1">{t("commerce.traffic")}</label>
                  <input
                    type="number"
                    value={planTraffic}
                    onChange={(e) => setPlanTraffic(Number(e.target.value))}
                    className="w-full px-2 py-1 bg-zinc-950 border border-zinc-700 rounded-lg text-xs text-white"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-zinc-400 mb-1">{t("commerce.amount")}</label>
                  <input
                    type="number"
                    step="0.01"
                    value={planPrice}
                    onChange={(e) => setPlanPrice(Number(e.target.value))}
                    className="w-full px-2 py-1 bg-zinc-950 border border-zinc-700 rounded-lg text-xs text-white"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <Button type="button" variant="outline" size="sm" onClick={() => setShowCreatePlanModal(false)}>
                  {t("common.cancel")}
                </Button>
                <Button type="submit" size="sm" isLoading={loadingData}>
                  {t("common.confirm")}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: 2FA Setup */}
      {show2FAModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 shadow-xl space-y-4">
            <h3 className="text-lg font-bold text-white">{t("admin.twofaTitle")}</h3>
            <p className="text-xs text-zinc-400">{t("admin.twofaDesc")}</p>

            {twoFAMsg && <Alert severity="info">{twoFAMsg}</Alert>}

            {twoFASecret && (
              <div className="p-3 bg-zinc-950 rounded-xl border border-zinc-800 space-y-1">
                <span className="text-[10px] text-zinc-500 uppercase tracking-wider block font-semibold">TOTP Secret:</span>
                <span className="font-mono text-xs text-blue-400 select-all block break-all font-bold">{twoFASecret}</span>
              </div>
            )}

            {twoFAUrl && (
              <div className="p-2 bg-zinc-950 rounded-lg border border-zinc-800 text-[10px] font-mono text-zinc-500 break-all select-all">
                {twoFAUrl}
              </div>
            )}

            <div>
              <label className="block text-xs font-medium text-zinc-400 mb-1">{t("admin.twofaCodeLabel")}</label>
              <input
                type="text"
                maxLength={6}
                value={twoFACode}
                onChange={(e) => setTwoFACode(e.target.value)}
                placeholder="123456"
                className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-center font-mono tracking-widest text-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <Button type="button" variant="outline" size="sm" onClick={() => setShow2FAModal(false)}>
                {t("common.close")}
              </Button>
              <Button size="sm" onClick={handleEnable2FA}>
                {t("admin.verifyEnable2fa")}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: Adjust User Balance */}
      {selectedUserForAdjust && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 shadow-xl space-y-4">
            <h3 className="text-lg font-bold text-white">{t("admin.adjustBalance")}</h3>
            <p className="text-xs text-zinc-400">
              {selectedUserForAdjust.email} ({selectedUserForAdjust.id})
            </p>

            <form onSubmit={handleAdjustBalance} className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">
                  {t("admin.adjustmentAmount")}
                </label>
                <input
                  type="number"
                  step="0.01"
                  required
                  value={adjustAmount}
                  onChange={(e) => setAdjustAmount(e.target.value)}
                  placeholder="10.00"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">
                  {t("admin.adjustmentReason")}
                </label>
                <input
                  type="text"
                  required
                  value={adjustReason}
                  onChange={(e) => setAdjustReason(e.target.value)}
                  placeholder="Reason for adjustment"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setSelectedUserForAdjust(null)}
                >
                  {t("common.cancel")}
                </Button>
                <Button type="submit" size="sm" isLoading={adjustingBalance}>
                  {t("common.confirm")}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default App;
