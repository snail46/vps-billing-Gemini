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
  const [providerType, setProviderType] = useState("lxd");

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
      if (res.success && res.data.admin) {
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
      if (res.success && res.data.admin) {
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
      setActionNotice("Node provisioned successfully.");
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || "Failed to create node");
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
      setActionNotice("Infrastructure Provider connected.");
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || "Failed to create provider");
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
      setActionNotice("Product created successfully.");
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || "Failed to create product");
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
      setActionNotice("Plan created and published.");
      await loadTabData();
    } catch (err: any) {
      setActionError(err?.message || "Failed to create plan");
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
      setActionError(err?.message || "Failed to reply");
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
      setActionError(err?.message || "Failed to update ticket status");
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
      setTwoFAMsg(err?.message || "Failed to init 2FA");
    }
  };

  const handleEnable2FA = async () => {
    if (!twoFASecret || !twoFACode) return;
    try {
      await adminApi.enable2FA(twoFASecret, twoFACode);
      setTwoFAMsg("2FA successfully enabled!");
      setTwoFACode("");
    } catch (err: any) {
      setTwoFAMsg(err?.message || "Invalid verification code");
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
              Privileged infrastructure and billing control plane
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
                <span className="font-semibold text-zinc-300 block">Default Credentials:</span>
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
                Control Plane &bull; Super Admin
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
              { id: "infrastructure", label: "Infrastructure" },
              { id: "instances", label: t("common.servers") },
              { id: "commerce", label: "Commerce & Billing" },
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
                <h2 className="text-xl font-bold text-white tracking-tight">System Telemetry & Status</h2>
                <p className="text-xs text-zinc-400 mt-0.5">Real-time health of cluster nodes and microservices</p>
              </div>
              <Button size="sm" variant="outline" onClick={loadTabData} isLoading={loadingData}>
                {t("common.refresh")}
              </Button>
            </div>

            {/* Health Indicators */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">Nodes Online</span>
                  <span className="text-2xl font-bold text-white mt-1 block">
                    {nodes.filter((n) => n.status === "active").length} / {nodes.length}
                  </span>
                  <span className="text-[11px] text-emerald-400 mt-0.5 block">100% Operational</span>
                </div>
                <div className="w-10 h-10 rounded-lg bg-emerald-950 text-emerald-400 flex items-center justify-center font-bold text-lg">
                  &#9679;
                </div>
              </div>

              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">Live Instances</span>
                  <span className="text-2xl font-bold text-blue-400 mt-1 block">
                    {instances.filter((i) => i.observed_state === "running").length}
                  </span>
                  <span className="text-[11px] text-zinc-400 mt-0.5 block">{instances.length} Provisioned</span>
                </div>
                <div className="w-10 h-10 rounded-lg bg-blue-950 text-blue-400 flex items-center justify-center font-bold text-lg">
                  &bull;
                </div>
              </div>

              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">Total Orders</span>
                  <span className="text-2xl font-bold text-purple-400 mt-1 block">
                    {orders.length}
                  </span>
                  <span className="text-[11px] text-zinc-400 mt-0.5 block">
                    {orders.filter((o) => o.status === "paid").length} Settled
                  </span>
                </div>
                <div className="w-10 h-10 rounded-lg bg-purple-950 text-purple-400 flex items-center justify-center font-bold text-lg">
                  $
                </div>
              </div>

              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">Platform Health</span>
                  <span className="text-2xl font-bold text-emerald-400 mt-1 block">
                    {health?.status ? health.status.toUpperCase() : "READY"}
                  </span>
                  <span className="text-[11px] text-zinc-400 mt-0.5 block">Postgres + Redis</span>
                </div>
                <div className="w-10 h-10 rounded-lg bg-emerald-950 text-emerald-400 flex items-center justify-center font-bold text-lg">
                  &#10003;
                </div>
              </div>
            </div>

            {/* Quick Links & Cluster Nodes */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-bold text-white text-base">Compute Nodes</h3>
                  <Button size="sm" onClick={() => setShowCreateNodeModal(true)}>
                    + {t("common.createNode")}
                  </Button>
                </div>
                {nodes.length === 0 ? (
                  <div className="text-center py-8 text-zinc-500 text-xs">No active nodes registered</div>
                ) : (
                  <div className="space-y-2">
                    {nodes.map((n) => (
                      <div
                        key={n.id}
                        className="bg-zinc-950 border border-zinc-800 rounded-lg p-3 flex items-center justify-between text-xs"
                      >
                        <div>
                          <span className="font-bold text-white block">{n.name}</span>
                          <span className="text-zinc-500 font-mono">{n.region} &bull; {n.cpu_total} Cores &bull; {n.memory_total_mb} MB</span>
                        </div>
                        <StatusBadge severity={n.status === "active" ? "success" : "neutral"} label={n.status} />
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-bold text-white text-base">Infrastructure Providers</h3>
                  <Button size="sm" onClick={() => setShowCreateProviderModal(true)}>
                    + {t("common.createProvider")}
                  </Button>
                </div>
                {providers.length === 0 ? (
                  <div className="text-center py-8 text-zinc-500 text-xs">No providers attached</div>
                ) : (
                  <div className="space-y-2">
                    {providers.map((p) => (
                      <div
                        key={p.id}
                        className="bg-zinc-950 border border-zinc-800 rounded-lg p-3 flex items-center justify-between text-xs"
                      >
                        <div>
                          <span className="font-bold text-white block">{p.name}</span>
                          <span className="text-zinc-500 uppercase font-mono">{p.provider_type} Driver</span>
                        </div>
                        <StatusBadge severity="success" label={p.status} />
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* TAB 2: Infrastructure Management (Nodes & Providers) */}
        {activeTab === "infrastructure" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">Infrastructure Topology</h2>
                <p className="text-xs text-zinc-400 mt-0.5">Physical hypervisors, cloud endpoints, and capability drivers</p>
              </div>
              <div className="flex items-center gap-2">
                <Button size="sm" onClick={() => setShowCreateNodeModal(true)}>
                  + {t("common.createNode")}
                </Button>
                <Button size="sm" variant="outline" onClick={() => setShowCreateProviderModal(true)}>
                  + {t("common.createProvider")}
                </Button>
              </div>
            </div>

            {/* Nodes Table */}
            <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
              <div className="px-5 py-3.5 border-b border-zinc-800 font-bold text-sm text-white">
                Hypervisor Nodes
              </div>
              <table className="w-full text-left text-xs">
                <thead className="bg-zinc-950 text-zinc-400 uppercase tracking-wider font-semibold border-b border-zinc-800">
                  <tr>
                    <th className="px-5 py-3.5">Node Name</th>
                    <th className="px-5 py-3.5">Region</th>
                    <th className="px-5 py-3.5">CPU Capacity</th>
                    <th className="px-5 py-3.5">Memory Capacity</th>
                    <th className="px-5 py-3.5">Disk Capacity</th>
                    <th className="px-5 py-3.5">{t("common.status")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-800/80">
                  {nodes.map((node) => (
                    <tr key={node.id} className="hover:bg-zinc-800/40 transition-colors">
                      <td className="px-5 py-4 font-mono font-medium text-white">{node.name}</td>
                      <td className="px-5 py-4 text-zinc-400 font-mono uppercase">{node.region}</td>
                      <td className="px-5 py-4 text-zinc-300 font-semibold">{node.cpu_total} Cores</td>
                      <td className="px-5 py-4 text-zinc-300 font-semibold">{(node.memory_total_mb / 1024).toFixed(0)} GB</td>
                      <td className="px-5 py-4 text-zinc-300 font-semibold">{node.disk_total_gb} GB</td>
                      <td className="px-5 py-4">
                        <StatusBadge severity={node.status === "active" ? "success" : "neutral"} label={node.status} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Providers Table */}
            <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
              <div className="px-5 py-3.5 border-b border-zinc-800 font-bold text-sm text-white">
                Cloud Provider Drivers
              </div>
              <table className="w-full text-left text-xs">
                <thead className="bg-zinc-950 text-zinc-400 uppercase tracking-wider font-semibold border-b border-zinc-800">
                  <tr>
                    <th className="px-5 py-3.5">Provider Name</th>
                    <th className="px-5 py-3.5">Driver Type</th>
                    <th className="px-5 py-3.5">{t("common.status")}</th>
                    <th className="px-5 py-3.5">Capabilities</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-800/80">
                  {providers.map((p) => (
                    <tr key={p.id} className="hover:bg-zinc-800/40 transition-colors">
                      <td className="px-5 py-4 font-bold text-white">{p.name}</td>
                      <td className="px-5 py-4 font-mono text-blue-400 uppercase">{p.provider_type}</td>
                      <td className="px-5 py-4">
                        <StatusBadge severity="success" label={p.status} />
                      </td>
                      <td className="px-5 py-4 text-zinc-400 font-mono text-[11px]">
                        create, start, stop, restart, reinstall
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* TAB 3: Instances Management */}
        {activeTab === "instances" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">{t("common.servers")}</h2>
                <p className="text-xs text-zinc-400 mt-0.5">All customer virtual servers across hypervisors</p>
              </div>
              <Button size="sm" variant="outline" onClick={loadTabData} isLoading={loadingData}>
                {t("common.refresh")}
              </Button>
            </div>

            {instances.length === 0 ? (
              <Card>
                <div className="text-center py-12 text-zinc-500 text-sm">No instances deployed yet</div>
              </Card>
            ) : (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 text-zinc-400 uppercase tracking-wider font-semibold border-b border-zinc-800">
                    <tr>
                      <th className="px-5 py-3.5">Instance Name</th>
                      <th className="px-5 py-3.5">IP Address</th>
                      <th className="px-5 py-3.5">Specs</th>
                      <th className="px-5 py-3.5">{t("common.status")}</th>
                      <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/80">
                    {instances.map((inst) => (
                      <tr key={inst.id} className="hover:bg-zinc-800/40 transition-colors">
                        <td className="px-5 py-4">
                          <span className="font-bold text-white block">{inst.name}</span>
                          <span className="font-mono text-zinc-500 text-[11px]">{inst.id}</span>
                        </td>
                        <td className="px-5 py-4 font-mono font-medium text-blue-400">
                          {inst.primary_ipv4 || "192.168.1.100"}
                        </td>
                        <td className="px-5 py-4 text-zinc-300">
                          {inst.cpu_cores}C / {inst.memory_mb}MB / {inst.disk_gb}GB
                        </td>
                        <td className="px-5 py-4">
                          <StatusBadge
                            severity={inst.observed_state === "running" ? "success" : "neutral"}
                            label={t(`instance.status.${inst.observed_state}`) || inst.observed_state}
                          />
                        </td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">
                          {new Date(inst.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* TAB 4: Commerce (Orders, Invoices, Ledger, Products) */}
        {activeTab === "commerce" && (
          <div className="space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">Commerce & Billing Engine</h2>
                <p className="text-xs text-zinc-400 mt-0.5">Double-entry accounting, order fulfillment, and product catalog</p>
              </div>

              {/* Subtabs */}
              <div className="flex items-center gap-1 bg-zinc-900 border border-zinc-800 p-1 rounded-xl">
                {[
                  { id: "orders", label: t("commerce.orders") },
                  { id: "invoices", label: t("commerce.invoices") },
                  { id: "ledger", label: "Ledger" },
                  { id: "products", label: "Products & Plans" },
                ].map((st) => (
                  <button
                    key={st.id}
                    onClick={() => setCommerceSubTab(st.id as any)}
                    className={`px-3 py-1.5 rounded-lg text-xs font-semibold cursor-pointer transition-colors ${
                      commerceSubTab === st.id ? "bg-blue-600 text-white" : "text-zinc-400 hover:text-white"
                    }`}
                  >
                    {st.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Subtab 1: Orders */}
            {commerceSubTab === "orders" && (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 text-zinc-400 uppercase tracking-wider font-semibold border-b border-zinc-800">
                    <tr>
                      <th className="px-5 py-3.5">{t("commerce.orderNo")}</th>
                      <th className="px-5 py-3.5">User ID</th>
                      <th className="px-5 py-3.5">{t("commerce.total")}</th>
                      <th className="px-5 py-3.5">{t("common.status")}</th>
                      <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/80">
                    {orders.map((ord) => (
                      <tr key={ord.id} className="hover:bg-zinc-800/40 transition-colors">
                        <td className="px-5 py-4 font-mono font-medium text-white">{ord.order_no}</td>
                        <td className="px-5 py-4 font-mono text-zinc-400 text-[11px]">{ord.user_id}</td>
                        <td className="px-5 py-4 font-bold text-white">{formatPrice(ord.total_minor, ord.currency)}</td>
                        <td className="px-5 py-4">
                          <StatusBadge
                            severity={ord.status === "paid" ? "success" : "warning"}
                            label={ord.status.toUpperCase()}
                          />
                        </td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">
                          {new Date(ord.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Subtab 2: Invoices */}
            {commerceSubTab === "invoices" && (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 text-zinc-400 uppercase tracking-wider font-semibold border-b border-zinc-800">
                    <tr>
                      <th className="px-5 py-3.5">Invoice #</th>
                      <th className="px-5 py-3.5">User ID</th>
                      <th className="px-5 py-3.5">{t("commerce.amount")}</th>
                      <th className="px-5 py-3.5">{t("common.status")}</th>
                      <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/80">
                    {invoices.map((inv) => (
                      <tr key={inv.id} className="hover:bg-zinc-800/40 transition-colors">
                        <td className="px-5 py-4 font-mono font-medium text-white">{inv.invoice_no}</td>
                        <td className="px-5 py-4 font-mono text-zinc-400 text-[11px]">{inv.user_id}</td>
                        <td className="px-5 py-4 font-bold text-white">{formatPrice(inv.amount_minor, inv.currency)}</td>
                        <td className="px-5 py-4">
                          <StatusBadge
                            severity={inv.status === "paid" ? "success" : "warning"}
                            label={inv.status.toUpperCase()}
                          />
                        </td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">
                          {new Date(inv.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Subtab 3: Double-Entry Ledger */}
            {commerceSubTab === "ledger" && (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 text-zinc-400 uppercase tracking-wider font-semibold border-b border-zinc-800">
                    <tr>
                      <th className="px-5 py-3.5">Transaction Type</th>
                      <th className="px-5 py-3.5">Reference ID</th>
                      <th className="px-5 py-3.5">Description</th>
                      <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/80">
                    {ledgerTxs.map((tx) => (
                      <tr key={tx.id} className="hover:bg-zinc-800/40 transition-colors">
                        <td className="px-5 py-4 font-bold text-blue-400 uppercase">{tx.type}</td>
                        <td className="px-5 py-4 font-mono text-zinc-400 text-[11px]">{tx.reference_id || tx.id}</td>
                        <td className="px-5 py-4 text-zinc-300">{tx.description || "-"}</td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">
                          {new Date(tx.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Subtab 4: Product Catalog Management */}
            {commerceSubTab === "products" && (
              <div className="space-y-6">
                <div className="flex items-center justify-between">
                  <h3 className="font-bold text-white text-base">VPS Products & Plans</h3>
                  <div className="flex items-center gap-2">
                    <Button size="sm" onClick={() => setShowCreateProductModal(true)}>
                      + {t("common.createProduct")}
                    </Button>
                    <Button size="sm" variant="outline" onClick={() => setShowCreatePlanModal(true)}>
                      + {t("common.createPlan")}
                    </Button>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  {products.map((prod) => (
                    <div key={prod.id} className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 space-y-4">
                      <div className="flex items-start justify-between">
                        <div>
                          <h4 className="font-bold text-white text-base">
                            {prod.name_i18n?.[locale] || Object.values(prod.name_i18n || {})[0] || prod.slug}
                          </h4>
                          <span className="font-mono text-xs text-zinc-500">Slug: {prod.slug}</span>
                        </div>
                        <StatusBadge severity="success" label="Active" />
                      </div>

                      <div className="space-y-2 pt-2 border-t border-zinc-800">
                        <span className="text-xs font-semibold text-zinc-400 block">Available Plans:</span>
                        {(prod.plans || []).map((pl) => (
                          <div
                            key={pl.id}
                            className="bg-zinc-950 p-3 rounded-lg border border-zinc-800/80 flex items-center justify-between text-xs"
                          >
                            <div>
                              <span className="font-bold text-zinc-200 block">
                                {pl.name_i18n?.[locale] || Object.values(pl.name_i18n || {})[0] || pl.slug}
                              </span>
                              <span className="text-zinc-500 font-mono">
                                {pl.cpu_cores}C / {pl.memory_mb}MB / {pl.disk_gb}GB
                              </span>
                            </div>
                            <span className="font-bold text-blue-400 text-sm">
                              {formatPrice(pl.price_minor, pl.currency)}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
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
                <p className="text-xs text-zinc-400 mt-0.5">Asynchronous operation event timeline and diagnostics</p>
              </div>
              <Button size="sm" variant="outline" onClick={loadTabData} isLoading={loadingData}>
                {t("common.refresh")}
              </Button>
            </div>

            <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
              <table className="w-full text-left text-xs">
                <thead className="bg-zinc-950 text-zinc-400 uppercase tracking-wider font-semibold border-b border-zinc-800">
                  <tr>
                    <th className="px-5 py-3.5">Operation Type</th>
                    <th className="px-5 py-3.5">Trace ID</th>
                    <th className="px-5 py-3.5">State</th>
                    <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                    <th className="px-5 py-3.5 text-right">Diagnostic</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-800/80">
                  {operations.map((op) => (
                    <tr
                      key={op.id}
                      className="hover:bg-zinc-800/40 transition-colors cursor-pointer"
                      onClick={() => setInspectOperation(op)}
                    >
                      <td className="px-5 py-4 font-bold text-white uppercase">{op.type}</td>
                      <td className="px-5 py-4 font-mono text-zinc-400 text-[11px]">{op.trace_id || op.id}</td>
                      <td className="px-5 py-4">
                        <StatusBadge
                          severity={op.status === "succeeded" ? "success" : op.status === "failed" ? "error" : "warning"}
                          label={op.status.toUpperCase()}
                        />

                      </td>
                      <td className="px-5 py-4 text-zinc-500 font-mono">
                        {new Date(op.created_at).toLocaleString()}
                      </td>
                      <td className="px-5 py-4 text-right">
                        <Button size="sm" variant="outline">
                          Inspect &rarr;
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* TAB 6: Support Tickets Desk */}
        {activeTab === "tickets" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">{t("common.tickets")}</h2>
                <p className="text-xs text-zinc-400 mt-0.5">Customer technical inquiry desk and resolution threads</p>
              </div>
              <Button size="sm" variant="outline" onClick={loadTabData} isLoading={loadingData}>
                {t("common.refresh")}
              </Button>
            </div>

            {selectedTicket ? (
              /* Ticket Discussion Thread for Admin */
              <div className="space-y-4">
                <Button size="sm" variant="outline" onClick={() => setSelectedTicket(null)}>
                  &larr; Back to Tickets List
                </Button>

                <div className="bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-xs space-y-6">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800 pb-4">
                    <div>
                      <h3 className="text-lg font-bold text-white">{selectedTicket.subject}</h3>
                      <span className="text-xs text-zinc-400 font-mono">
                        User ID: {selectedTicket.user_id} &bull; Created: {new Date(selectedTicket.created_at).toLocaleString()}
                      </span>
                    </div>
                    <div className="flex items-center gap-3">
                      <StatusBadge
                        severity={selectedTicket.status === "open" ? "warning" : "neutral"}
                        label={selectedTicket.status.toUpperCase()}
                      />
                      <Button
                        size="sm"
                        variant={selectedTicket.status === "open" ? "danger" : "secondary"}
                        onClick={handleToggleTicketStatus}
                      >
                        {selectedTicket.status === "open" ? t("tickets.closeTicket") : t("tickets.reopenTicket")}
                      </Button>
                    </div>
                  </div>

                  {/* Messages Thread */}
                  <div className="space-y-4 max-h-[400px] overflow-y-auto pr-2">
                    {(selectedTicket.messages || []).map((msg) => {
                      const isSupport = msg.sender_type === "admin";
                      return (
                        <div
                          key={msg.id}
                          className={`flex flex-col ${isSupport ? "items-end" : "items-start"}`}
                        >
                          <div className="flex items-center gap-2 text-xs text-zinc-400 mb-1">
                            <span className="font-semibold text-zinc-300">
                              {isSupport ? "Technical Support (You)" : "Customer"}
                            </span>
                            <span>&bull;</span>
                            <span>{new Date(msg.created_at).toLocaleTimeString()}</span>
                          </div>
                          <div
                            className={`p-4 rounded-2xl max-w-lg text-sm leading-relaxed ${
                              isSupport
                                ? "bg-blue-600 text-white rounded-br-none"
                                : "bg-zinc-800 text-zinc-200 rounded-bl-none border border-zinc-700"
                            }`}
                          >
                            {msg.message}
                          </div>
                        </div>
                      );
                    })}
                  </div>

                  {/* Admin Reply Form */}
                  <form onSubmit={handleAdminReplyTicket} className="pt-4 border-t border-zinc-800 flex gap-3">
                    <input
                      type="text"
                      required
                      value={adminReplyText}
                      onChange={(e) => setAdminReplyText(e.target.value)}
                      placeholder={t("tickets.replyPlaceholder")}
                      className="flex-1 px-4 py-2 bg-zinc-950 border border-zinc-700 rounded-xl text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                    <Button type="submit" isLoading={loadingData}>
                      {t("tickets.sendReply")}
                    </Button>
                  </form>
                </div>
              </div>
            ) : (
              /* Tickets List Table */
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 text-zinc-400 uppercase tracking-wider font-semibold border-b border-zinc-800">
                    <tr>
                      <th className="px-5 py-3.5">Subject</th>
                      <th className="px-5 py-3.5">Priority</th>
                      <th className="px-5 py-3.5">Status</th>
                      <th className="px-5 py-3.5">User ID</th>
                      <th className="px-5 py-3.5">Created</th>
                      <th className="px-5 py-3.5 text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/80">
                    {tickets.map((tkt) => (
                      <tr
                        key={tkt.id}
                        className="hover:bg-zinc-800/40 transition-colors cursor-pointer"
                        onClick={() => setSelectedTicket(tkt)}
                      >
                        <td className="px-5 py-4 font-bold text-white">{tkt.subject}</td>
                        <td className="px-5 py-4">
                          <span className="capitalize font-semibold text-zinc-300">{tkt.priority}</span>
                        </td>
                        <td className="px-5 py-4">
                          <StatusBadge
                            severity={tkt.status === "open" ? "warning" : "neutral"}
                            label={tkt.status.toUpperCase()}
                          />
                        </td>
                        <td className="px-5 py-4 font-mono text-zinc-500 text-[11px]">{tkt.user_id}</td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">
                          {new Date(tkt.created_at).toLocaleString()}
                        </td>
                        <td className="px-5 py-4 text-right">
                          <Button size="sm" variant="outline">
                            Reply &rarr;
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

        {/* TAB 7: Users & Admins */}
        {activeTab === "users" && (
          <div className="space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-white tracking-tight">{t("common.users")}</h2>
                <p className="text-xs text-zinc-400 mt-0.5">Directory of registered users and privileged administrators</p>
              </div>

              <div className="flex items-center gap-1 bg-zinc-900 border border-zinc-800 p-1 rounded-xl">
                {[
                  { id: "users", label: t("common.users") },
                  { id: "admins", label: t("common.admins") },
                ].map((st) => (
                  <button
                    key={st.id}
                    onClick={() => setUsersSubTab(st.id as any)}
                    className={`px-3 py-1.5 rounded-lg text-xs font-semibold cursor-pointer transition-colors ${
                      usersSubTab === st.id ? "bg-blue-600 text-white" : "text-zinc-400 hover:text-white"
                    }`}
                  >
                    {st.label}
                  </button>
                ))}
              </div>
            </div>

            {usersSubTab === "users" ? (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 text-zinc-400 uppercase tracking-wider font-semibold border-b border-zinc-800">
                    <tr>
                      <th className="px-5 py-3.5">Email</th>
                      <th className="px-5 py-3.5">User ID</th>
                      <th className="px-5 py-3.5">Locale</th>
                      <th className="px-5 py-3.5">Registered</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/80">
                    {users.map((u) => (
                      <tr key={u.id} className="hover:bg-zinc-800/40 transition-colors">
                        <td className="px-5 py-4 font-bold text-white">{u.email}</td>
                        <td className="px-5 py-4 font-mono text-zinc-400 text-[11px]">{u.id}</td>
                        <td className="px-5 py-4 font-mono text-zinc-400">{u.locale}</td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">
                          {new Date(u.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xs">
                <table className="w-full text-left text-xs">
                  <thead className="bg-zinc-950 text-zinc-400 uppercase tracking-wider font-semibold border-b border-zinc-800">
                    <tr>
                      <th className="px-5 py-3.5">Admin Email</th>
                      <th className="px-5 py-3.5">Admin ID</th>
                      <th className="px-5 py-3.5">Status</th>
                      <th className="px-5 py-3.5">Created</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/80">
                    {admins.map((a) => (
                      <tr key={a.id} className="hover:bg-zinc-800/40 transition-colors">
                        <td className="px-5 py-4 font-bold text-white">{a.email}</td>
                        <td className="px-5 py-4 font-mono text-zinc-400 text-[11px]">{a.id}</td>
                        <td className="px-5 py-4">
                          <StatusBadge severity="success" label={a.status || "ACTIVE"} />
                        </td>
                        <td className="px-5 py-4 text-zinc-500 font-mono">
                          {new Date(a.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </main>

      {/* MODAL: Operation Inspector Drawer */}
      {inspectOperation && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-2xl w-full p-6 shadow-2xl relative space-y-4">
            <button
              onClick={() => setInspectOperation(null)}
              className="absolute top-4 right-4 text-zinc-400 hover:text-white font-bold cursor-pointer"
            >
              &times;
            </button>
            <h3 className="text-lg font-bold text-white">{t("common.operationInspector")}</h3>
            <div className="grid grid-cols-2 gap-4 bg-zinc-950 p-4 rounded-xl border border-zinc-800 text-xs">
              <div>
                <span className="text-zinc-500 block">Operation ID</span>
                <span className="font-mono text-white block mt-0.5">{inspectOperation.id}</span>
              </div>
              <div>
                <span className="text-zinc-500 block">Trace ID</span>
                <span className="font-mono text-white block mt-0.5">{inspectOperation.trace_id || "-"}</span>
              </div>
              <div>
                <span className="text-zinc-500 block">Status</span>
                <span className="font-bold text-blue-400 block mt-0.5 uppercase">{inspectOperation.status}</span>
              </div>
              <div>
                <span className="text-zinc-500 block">Phase / Step</span>
                <span className="font-mono text-zinc-300 block mt-0.5">{inspectOperation.phase || inspectOperation.status}</span>
              </div>

            </div>

            <div>
              <span className="text-xs font-semibold text-zinc-400 mb-2 block">{t("common.rawDiagnostic")}</span>
              <pre className="bg-zinc-950 p-4 rounded-xl border border-zinc-800 text-[11px] font-mono text-zinc-300 max-h-60 overflow-y-auto">
                {JSON.stringify(inspectOperation, null, 2)}
              </pre>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: Add Node */}
      {showCreateNodeModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 shadow-2xl relative">
            <button
              onClick={() => setShowCreateNodeModal(false)}
              className="absolute top-4 right-4 text-zinc-400 hover:text-white font-bold cursor-pointer"
            >
              &times;
            </button>
            <h3 className="text-lg font-bold text-white mb-4">{t("common.createNode")}</h3>
            <form onSubmit={handleCreateNode} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">Node Name</label>
                <input
                  type="text"
                  required
                  value={nodeName}
                  onChange={(e) => setNodeName(e.target.value)}
                  placeholder="node-us-west-2"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">Region</label>
                <input
                  type="text"
                  required
                  value={nodeRegion}
                  onChange={(e) => setNodeRegion(e.target.value)}
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">CPU Cores</label>
                  <input
                    type="number"
                    required
                    value={nodeCPU}
                    onChange={(e) => setNodeCPU(Number(e.target.value))}
                    className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">RAM (MB)</label>
                  <input
                    type="number"
                    required
                    value={nodeRAM}
                    onChange={(e) => setNodeRAM(Number(e.target.value))}
                    className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">Disk (GB)</label>
                  <input
                    type="number"
                    required
                    value={nodeDisk}
                    onChange={(e) => setNodeDisk(Number(e.target.value))}
                    className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2">
                <Button type="button" variant="outline" size="sm" onClick={() => setShowCreateNodeModal(false)}>
                  Cancel
                </Button>
                <Button type="submit" size="sm" isLoading={loadingData}>
                  Create Node
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Add Provider */}
      {showCreateProviderModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-sm w-full p-6 shadow-2xl relative">
            <button
              onClick={() => setShowCreateProviderModal(false)}
              className="absolute top-4 right-4 text-zinc-400 hover:text-white font-bold cursor-pointer"
            >
              &times;
            </button>
            <h3 className="text-lg font-bold text-white mb-4">{t("common.createProvider")}</h3>
            <form onSubmit={handleCreateProvider} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">Provider Name</label>
                <input
                  type="text"
                  required
                  value={providerName}
                  onChange={(e) => setProviderName(e.target.value)}
                  placeholder="Primary LXD Cluster"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">Driver Type</label>
                <select
                  value={providerType}
                  onChange={(e) => setProviderType(e.target.value)}
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="lxd">LXD / Incus API</option>
                  <option value="runman">Runman Host Agent</option>
                  <option value="mock">Mock Simulator</option>
                </select>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2">
                <Button type="button" variant="outline" size="sm" onClick={() => setShowCreateProviderModal(false)}>
                  Cancel
                </Button>
                <Button type="submit" size="sm" isLoading={loadingData}>
                  Connect Provider
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Add Product */}
      {showCreateProductModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-sm w-full p-6 shadow-2xl relative">
            <button
              onClick={() => setShowCreateProductModal(false)}
              className="absolute top-4 right-4 text-zinc-400 hover:text-white font-bold cursor-pointer"
            >
              &times;
            </button>
            <h3 className="text-lg font-bold text-white mb-4">{t("common.createProduct")}</h3>
            <form onSubmit={handleCreateProduct} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">Slug</label>
                <input
                  type="text"
                  required
                  value={productSlug}
                  onChange={(e) => setProductSlug(e.target.value)}
                  placeholder="cloud-vps-ssd"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">Name (en-US)</label>
                <input
                  type="text"
                  required
                  value={productNameEn}
                  onChange={(e) => setProductNameEn(e.target.value)}
                  placeholder="NVMe Cloud VPS"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">Name (zh-CN)</label>
                <input
                  type="text"
                  required
                  value={productNameZh}
                  onChange={(e) => setProductNameZh(e.target.value)}
                  placeholder="NVMe 高性能云服务器"
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div className="flex items-center justify-end gap-2 pt-2">
                <Button type="button" variant="outline" size="sm" onClick={() => setShowCreateProductModal(false)}>
                  Cancel
                </Button>
                <Button type="submit" size="sm" isLoading={loadingData}>
                  Create Product
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Add Plan */}
      {showCreatePlanModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-md w-full p-6 shadow-2xl relative">
            <button
              onClick={() => setShowCreatePlanModal(false)}
              className="absolute top-4 right-4 text-zinc-400 hover:text-white font-bold cursor-pointer"
            >
              &times;
            </button>
            <h3 className="text-lg font-bold text-white mb-4">{t("common.createPlan")}</h3>
            <form onSubmit={handleCreatePlan} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-zinc-400 mb-1">Parent Product</label>
                <select
                  value={planProductID}
                  onChange={(e) => setPlanProductID(e.target.value)}
                  className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {products.map((p) => (
                    <option key={p.id} value={p.id}>{p.slug}</option>
                  ))}
                </select>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">Plan Slug</label>
                  <input
                    type="text"
                    required
                    value={planSlug}
                    onChange={(e) => setPlanSlug(e.target.value)}
                    placeholder="starter-2c"
                    className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">Monthly Price ($)</label>
                  <input
                    type="number"
                    step="0.01"
                    required
                    value={planPrice}
                    onChange={(e) => setPlanPrice(Number(e.target.value))}
                    className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">Name (en-US)</label>
                  <input
                    type="text"
                    required
                    value={planNameEn}
                    onChange={(e) => setPlanNameEn(e.target.value)}
                    placeholder="Standard VPS"
                    className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-zinc-400 mb-1">Name (zh-CN)</label>
                  <input
                    type="text"
                    required
                    value={planNameZh}
                    onChange={(e) => setPlanNameZh(e.target.value)}
                    placeholder="标准型 VPS"
                    className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-white"
                  />
                </div>
              </div>

              <div className="grid grid-cols-4 gap-2">
                <div>
                  <label className="block text-[11px] font-medium text-zinc-400 mb-1">CPU Cores</label>
                  <input
                    type="number"
                    required
                    value={planCPU}
                    onChange={(e) => setPlanCPU(Number(e.target.value))}
                    className="w-full px-2 py-1.5 bg-zinc-950 border border-zinc-700 rounded text-xs text-white"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-zinc-400 mb-1">RAM (MB)</label>
                  <input
                    type="number"
                    required
                    value={planRAM}
                    onChange={(e) => setPlanRAM(Number(e.target.value))}
                    className="w-full px-2 py-1.5 bg-zinc-950 border border-zinc-700 rounded text-xs text-white"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-zinc-400 mb-1">Disk (GB)</label>
                  <input
                    type="number"
                    required
                    value={planDisk}
                    onChange={(e) => setPlanDisk(Number(e.target.value))}
                    className="w-full px-2 py-1.5 bg-zinc-950 border border-zinc-700 rounded text-xs text-white"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-zinc-400 mb-1">Traffic (GB)</label>
                  <input
                    type="number"
                    required
                    value={planTraffic}
                    onChange={(e) => setPlanTraffic(Number(e.target.value))}
                    className="w-full px-2 py-1.5 bg-zinc-950 border border-zinc-700 rounded text-xs text-white"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2">
                <Button type="button" variant="outline" size="sm" onClick={() => setShowCreatePlanModal(false)}>
                  Cancel
                </Button>
                <Button type="submit" size="sm" isLoading={loadingData}>
                  Publish Plan
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: 2FA TOTP Setup */}
      {show2FAModal && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-2xl max-w-sm w-full p-6 shadow-2xl relative space-y-4">
            <button
              onClick={() => setShow2FAModal(false)}
              className="absolute top-4 right-4 text-zinc-400 hover:text-white font-bold cursor-pointer"
            >
              &times;
            </button>
            <h3 className="text-lg font-bold text-white">RFC 6238 TOTP 2FA</h3>
            <p className="text-xs text-zinc-400">
              Bind your Google Authenticator or 1Password client
            </p>

            {twoFAMsg && <Alert severity="info">{twoFAMsg}</Alert>}

            {twoFASecret && (
              <div className="bg-zinc-950 p-3 rounded-xl border border-zinc-800 text-center space-y-2">
                <span className="text-[11px] text-zinc-500 uppercase block font-semibold">Secret Key</span>
                <span className="font-mono text-sm text-blue-400 select-all tracking-wider font-bold block">
                  {twoFASecret}
                </span>
                <span className="text-[10px] text-zinc-500 block truncate">
                  URI: {twoFAUrl}
                </span>
              </div>
            )}

            <div className="space-y-2">
              <label className="block text-xs font-medium text-zinc-400">6-Digit Authenticator Code</label>
              <input
                type="text"
                maxLength={6}
                value={twoFACode}
                onChange={(e) => setTwoFACode(e.target.value)}
                placeholder="123456"
                className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-sm text-center text-white tracking-widest font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <Button className="w-full font-bold" onClick={handleEnable2FA}>
              Verify & Enable 2FA
            </Button>
          </div>
        </div>
      )}

      {/* Footer */}
      <footer className="bg-zinc-900 border-t border-zinc-800 mt-auto py-6">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 flex flex-col sm:flex-row items-center justify-between text-xs text-zinc-500 gap-3">
          <div>
            &copy; {new Date().getFullYear()} VPS Billing Admin Console. Confidential & Proprietary.
          </div>
          <div className="flex items-center gap-4 font-mono text-[11px]">
            <span>Audit: Enabled</span>
            <span>Ledger: Immutable</span>
          </div>
        </div>
      </footer>
    </div>
  );
};

export default App;
