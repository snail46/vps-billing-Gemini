import React, { useEffect, useState } from "react";
import {
  useI18n,
  Card,
  Button,
  StatusBadge,
  Alert,
  LanguageSwitcher,
  OperationProgress,
  authApi,
  UserDTO,
  Product,
  Order,
  Invoice,
  Wallet,
  Instance,
  Ticket,
  commerceApi,
  instanceApi,
  ticketApi,
} from "@vps-billing/shared";

export const App: React.FC = () => {
  const { t, locale } = useI18n();

  const [user, setUser] = useState<UserDTO | null>(null);
  const [checkingAuth, setCheckingAuth] = useState(true);

  // Tab State
  const [activeTab, setActiveTab] = useState<
    "dashboard" | "catalog" | "instances" | "orders" | "wallet" | "tickets"
  >("catalog");

  // Auth Form State
  const [showAuthModal, setShowAuthModal] = useState(false);
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const [pendingPlanToOrder, setPendingPlanToOrder] = useState<string | null>(null);

  // Data State
  const [products, setProducts] = useState<Product[]>([]);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [wallet, setWallet] = useState<Wallet | null>(null);
  const [userLedger, setUserLedger] = useState<any[]>([]);
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [ticketReplyText, setTicketReplyText] = useState("");
  const [showCreateTicketModal, setShowCreateTicketModal] = useState(false);
  const [newTicketSubject, setNewTicketSubject] = useState("");
  const [newTicketMessage, setNewTicketMessage] = useState("");
  const [newTicketPriority, setNewTicketPriority] = useState<"low" | "medium" | "high">("medium");

  // Wallet Deposit Modal
  const [showDepositModal, setShowDepositModal] = useState(false);
  const [depositAmount, setDepositAmount] = useState<number>(25);
  const [customDeposit, setCustomDeposit] = useState<string>("");
  const [depositing, setDepositing] = useState(false);

  // Instance Interaction States
  const [showPasswordMap, setShowPasswordMap] = useState<Record<string, boolean>>({});
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [reinstallModalInstance, setReinstallModalInstance] = useState<Instance | null>(null);
  const [selectedOS, setSelectedOS] = useState("ubuntu-22.04");

  // Operation System Integration
  const [activeOperationId, setActiveOperationId] = useState<string | null>(null);

  // Loading & Alerts
  const [loadingData, setLoadingData] = useState(false);
  const [actionNotice, setActionNotice] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    checkMe();
    loadCatalog();
  }, []);

  useEffect(() => {
    if (user) {
      loadUserData();
    }
  }, [user, activeTab]);

  const checkMe = async () => {
    setCheckingAuth(true);
    try {
      const res = await authApi.getMeUser();
      if (res.success && res.data.user) {
        setUser(res.data.user);
        setActiveTab("dashboard");
      } else {
        setUser(null);
        setActiveTab("catalog");
      }
    } catch {
      setUser(null);
      setActiveTab("catalog");
    } finally {
      setCheckingAuth(false);
    }
  };

  const loadCatalog = async () => {
    try {
      const res = await commerceApi.listProducts();
      setProducts(res.products || []);
    } catch (err) {
      console.error("Failed to load catalog:", err);
    }
  };

  const loadInstances = async () => {
    try {
      const list = await instanceApi.list();
      setInstances(list || []);
    } catch (err) {
      console.error("Failed to load instances:", err);
    }
  };

  const loadTickets = async () => {
    try {
      const tList = await ticketApi.listMyTickets();
      setTickets(tList);
      if (selectedTicket) {
        const updated = tList.find((t) => t.id === selectedTicket.id);
        if (updated) setSelectedTicket(updated);
      }
    } catch (err) {
      console.error("Failed to load tickets:", err);
    }
  };

  const loadUserData = async () => {
    setLoadingData(true);
    try {
      if (activeTab === "dashboard") {
        await Promise.all([
          loadInstances(),
          commerceApi.getWallet().then((w) => setWallet(w.wallet)).catch(() => {}),
          commerceApi.listOrders().then((o) => setOrders(o.orders || [])).catch(() => {}),
          loadTickets(),
        ]);
      } else if (activeTab === "instances") {
        await loadInstances();
      } else if (activeTab === "orders") {
        const [oRes, iRes] = await Promise.all([
          commerceApi.listOrders(),
          commerceApi.listInvoices(),
        ]);
        setOrders(oRes.orders || []);
        setInvoices(iRes.invoices || []);
      } else if (activeTab === "wallet") {
        const [wRes, lRes] = await Promise.all([
          commerceApi.getWallet(),
          commerceApi.listUserLedger(),
        ]);
        setWallet(wRes.wallet);
        setUserLedger(lRes.ledger_entries || []);
      } else if (activeTab === "tickets") {
        await loadTickets();
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoadingData(false);
    }
  };

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const handleOrderPlan = async (planId: string) => {
    if (!user) {
      setPendingPlanToOrder(planId);
      setShowAuthModal(true);
      return;
    }
    setActionNotice(null);
    setActionError(null);
    setLoadingData(true);
    try {
      const res = await commerceApi.createOrder({
        items: [{ plan_id: planId, quantity: 1 }],
      });
      setActionNotice(`${t("commerce.orders")}: ${res.order.order_no}`);
      setActiveTab("orders");
      await loadUserData();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
    }
  };

  const handlePayOrder = async (orderId: string) => {
    setActionNotice(null);
    setActionError(null);
    setLoadingData(true);
    try {
      const payRes = await commerceApi.payOrder(orderId, "fake");
      await commerceApi.simulatePayment(payRes.payment.payment_no);
      setActionNotice(t("commerce.paymentSuccess"));
      setActiveTab("instances");
      await loadInstances();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
    }
  };

  const handleDeposit = async () => {
    const finalAmount = customDeposit ? parseFloat(customDeposit) : depositAmount;
    if (!finalAmount || finalAmount <= 0) return;
    setDepositing(true);
    setActionNotice(null);
    setActionError(null);
    try {
      const amountMinor = Math.round(finalAmount * 100);
      const res = await commerceApi.deposit(amountMinor, "USD");
      setWallet(res.wallet);
      setActionNotice(t("commerce.depositSuccess"));
      setShowDepositModal(false);
      setCustomDeposit("");
      if (activeTab === "wallet" || activeTab === "dashboard") {
        await loadUserData();
      }
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setDepositing(false);
    }
  };

  const handleInstanceAction = async (
    instanceId: string,
    action: "start" | "stop" | "restart" | "reinstall",
    image?: string
  ) => {
    setActionNotice(null);
    setActionError(null);
    try {
      let res;
      if (action === "start") {
        res = await instanceApi.start(instanceId);
      } else if (action === "stop") {
        res = await instanceApi.stop(instanceId);
      } else if (action === "restart") {
        res = await instanceApi.restart(instanceId);
      } else if (action === "reinstall") {
        res = await instanceApi.reinstall(instanceId, image || "ubuntu-22.04");
        setReinstallModalInstance(null);
      }
      if (res && res.operation_id) {
        setActiveOperationId(res.operation_id);
      }
      await loadInstances();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    }
  };

  const handleCreateTicketSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTicketSubject.trim() || !newTicketMessage.trim()) return;
    setLoadingData(true);
    try {
      const created = await ticketApi.createTicket({
        subject: newTicketSubject,
        priority: newTicketPriority,
        message: newTicketMessage,
      });
      setShowCreateTicketModal(false);
      setNewTicketSubject("");
      setNewTicketMessage("");
      setActionNotice(t("tickets.submit"));
      await loadTickets();
      setSelectedTicket(created);
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
    }
  };

  const handleSendTicketReply = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTicket || !ticketReplyText.trim()) return;
    setLoadingData(true);
    try {
      await ticketApi.userReplyTicket(selectedTicket.id, ticketReplyText);
      setTicketReplyText("");
      const fresh = await ticketApi.getTicket(selectedTicket.id);
      setSelectedTicket(fresh);
      await loadTickets();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
    }
  };

  const handleAuth = async (e: React.FormEvent) => {
    e.preventDefault();
    setAuthError(null);
    if (mode === "register" && password !== confirmPassword) {
      setAuthError(t("auth.passwordsDoNotMatch"));
      return;
    }

    setSubmitting(true);
    try {
      if (mode === "register") {
        const res = await authApi.register({
          email,
          password,
          locale,
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
        });
        if (res.success) {
          setMode("login");
          setPassword("");
          setConfirmPassword("");
          setAuthError(null);
        } else {
          setAuthError(t(res.error?.message_key || "errors.validation_failed"));
        }
      } else {
        const res = await authApi.loginUser({ email, password });
        if (res.success && res.data.user) {
          setUser(res.data.user);
          setShowAuthModal(false);
          setPassword("");
          if (pendingPlanToOrder) {
            const planToBuy = pendingPlanToOrder;
            setPendingPlanToOrder(null);
            handleOrderPlan(planToBuy);
          } else {
            setActiveTab("dashboard");
          }
        } else {
          const errRes = res as any;
          setAuthError(t(errRes.error?.message_key || "errors.invalid_credentials"));
        }

      }
    } catch (err: any) {
      setAuthError(err?.message || t("errors.network_error"));
    } finally {
      setSubmitting(false);
    }
  };

  const handleLogout = async () => {
    try {
      await authApi.logoutUser();
    } finally {
      setUser(null);
      setActiveTab("catalog");
    }
  };

  const formatPrice = (minor: number, currency: string) => {
    return `${currency} ${(minor / 100).toFixed(2)}`;
  };

  if (checkingAuth) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-zinc-50 text-zinc-400 text-sm">
        {t("common.loading")}
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col bg-zinc-50 text-zinc-900">

      {/* Header */}
      <header className="bg-white border-b border-zinc-200 sticky top-0 z-20 shadow-xs">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3 cursor-pointer" onClick={() => setActiveTab(user ? "dashboard" : "catalog")}>
            <div className="w-9 h-9 rounded-lg bg-blue-600 text-white font-bold flex items-center justify-center shadow-xs">
              VPS
            </div>
            <div>
              <span className="font-bold text-zinc-900 tracking-tight text-base block">
                {t("common.appNameUser")}
              </span>
              <span className="text-[10px] text-zinc-400 font-mono tracking-wider uppercase block">
                Cloud IaaS Platform
              </span>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <LanguageSwitcher />

            {user ? (
              <div className="flex items-center gap-3">
                <div
                  onClick={() => {
                    setActiveTab("wallet");
                    setShowDepositModal(true);
                  }}
                  className="hidden sm:flex items-center gap-1.5 px-3 py-1.5 bg-blue-50 border border-blue-200 rounded-lg text-xs font-semibold text-blue-700 cursor-pointer hover:bg-blue-100 transition-colors"
                >
                  <span>{t("commerce.balance")}:</span>
                  <span className="font-bold">${((wallet?.available_balance_minor || 0) / 100).toFixed(2)}</span>
                  <span className="ml-1 text-[11px] bg-blue-600 text-white px-1.5 py-0.5 rounded font-normal">+</span>
                </div>
                <div className="hidden md:block text-right">
                  <span className="text-xs font-medium text-zinc-700 block truncate max-w-[140px]">
                    {user.email}
                  </span>
                </div>
                <Button size="sm" variant="outline" onClick={handleLogout}>
                  {t("auth.logout")}
                </Button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setMode("login");
                    setShowAuthModal(true);
                  }}
                >
                  {t("auth.login")}
                </Button>
                <Button
                  size="sm"
                  onClick={() => {
                    setMode("register");
                    setShowAuthModal(true);
                  }}
                >
                  {t("auth.register")}
                </Button>
              </div>
            )}
          </div>
        </div>

        {/* User Navigation Tabs */}
        {user && (
          <div className="border-t border-zinc-100 bg-white px-4 sm:px-6">
            <div className="max-w-7xl mx-auto flex items-center gap-1 sm:gap-2 overflow-x-auto py-2">
              {[
                { id: "dashboard", label: t("common.dashboard") },
                { id: "catalog", label: t("commerce.catalog") },
                { id: "instances", label: t("common.servers") },
                { id: "orders", label: t("commerce.orders") },
                { id: "wallet", label: t("commerce.wallet") },
                { id: "tickets", label: t("common.tickets") },
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => {
                    setActiveTab(tab.id as any);
                    setSelectedTicket(null);
                  }}
                  className={`px-3 py-1.5 text-xs font-medium rounded-md whitespace-nowrap transition-colors cursor-pointer ${
                    activeTab === tab.id
                      ? "bg-zinc-900 text-white"
                      : "text-zinc-600 hover:text-zinc-900 hover:bg-zinc-100"
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>
        )}
      </header>

      {/* Main Content Area */}
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

        {/* Global Active Operation SSE Bar */}
        {activeOperationId && (
          <div className="mb-6">
            <OperationProgress
              operationId={activeOperationId}
              onFinished={() => {
                loadInstances();
              }}
            />
          </div>
        )}

        {/* VIEW 1: Dashboard Overview (Logged-in 5-second metric glance) */}
        {user && activeTab === "dashboard" && (
          <div className="space-y-6">
            <div>
              <h2 className="text-xl font-bold text-zinc-900 tracking-tight">
                {t("common.dashboard")}
              </h2>
              <p className="text-zinc-500 text-sm mt-0.5">
                Welcome back, {user.email}
              </p>
            </div>

            {/* 4 Stat Overview Cards */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="bg-white border border-zinc-200 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">
                    {t("common.servers")}
                  </span>
                  <span className="text-2xl font-bold text-zinc-900 mt-1 block">
                    {instances.filter((i) => i.observed_state === "running").length} / {instances.length}
                  </span>
                  <span className="text-[11px] text-zinc-400 mt-0.5 block">Online / Total</span>
                </div>
                <div className="w-10 h-10 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center font-bold text-lg">
                  &#9679;
                </div>
              </div>

              <div className="bg-white border border-zinc-200 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">
                    {t("commerce.balance")}
                  </span>
                  <span className="text-2xl font-bold text-blue-600 mt-1 block">
                    ${((wallet?.available_balance_minor || 0) / 100).toFixed(2)}
                  </span>
                  <button
                    onClick={() => setShowDepositModal(true)}
                    className="text-[11px] text-blue-600 font-semibold hover:underline mt-0.5 block cursor-pointer"
                  >
                    + {t("commerce.deposit")}
                  </button>
                </div>
                <div className="w-10 h-10 rounded-lg bg-blue-50 text-blue-600 flex items-center justify-center font-bold text-lg">
                  $
                </div>
              </div>

              <div className="bg-white border border-zinc-200 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">
                    {t("commerce.orders")}
                  </span>
                  <span className="text-2xl font-bold text-zinc-900 mt-1 block">
                    {orders.length}
                  </span>
                  <span className="text-[11px] text-zinc-400 mt-0.5 block">
                    {orders.filter((o) => o.status === "pending").length} Pending
                  </span>
                </div>
                <div className="w-10 h-10 rounded-lg bg-purple-50 text-purple-600 flex items-center justify-center font-bold text-lg">
                  &equiv;
                </div>
              </div>

              <div className="bg-white border border-zinc-200 rounded-xl p-5 shadow-xs flex items-center justify-between">
                <div>
                  <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider block">
                    {t("common.tickets")}
                  </span>
                  <span className="text-2xl font-bold text-zinc-900 mt-1 block">
                    {tickets.filter((t) => t.status === "open").length}
                  </span>
                  <button
                    onClick={() => {
                      setActiveTab("tickets");
                      setShowCreateTicketModal(true);
                    }}
                    className="text-[11px] text-zinc-500 hover:text-blue-600 font-medium mt-0.5 block cursor-pointer"
                  >
                    + {t("tickets.createTicket")}
                  </button>
                </div>
                <div className="w-10 h-10 rounded-lg bg-amber-50 text-amber-600 flex items-center justify-center font-bold text-lg">
                  ?
                </div>
              </div>
            </div>

            {/* Quick Actions & Recent Servers */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              <div className="lg:col-span-2 space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-bold text-zinc-800 text-base">{t("common.servers")}</h3>
                  <Button size="sm" variant="outline" onClick={() => setActiveTab("instances")}>
                    View All
                  </Button>
                </div>

                {instances.length === 0 ? (
                  <Card>
                    <div className="text-center py-8 text-zinc-400 text-sm">
                      {t("instance.noInstances")}
                    </div>
                  </Card>
                ) : (
                  <div className="space-y-3">
                    {instances.slice(0, 3).map((inst) => (
                      <div
                        key={inst.id}
                        className="bg-white border border-zinc-200 rounded-xl p-4 flex items-center justify-between shadow-xs hover:border-zinc-300 transition-colors"
                      >
                        <div className="flex items-center gap-3">
                          <div
                            className={`w-3 h-3 rounded-full ${
                              inst.observed_state === "running" ? "bg-emerald-500" : "bg-zinc-300"
                            }`}
                          />
                          <div>
                            <span className="font-bold text-sm text-zinc-900 block">{inst.name}</span>
                            <span className="font-mono text-xs text-zinc-500 block">
                              {inst.primary_ipv4 || "192.168.1.100"} &bull; {inst.image_id || "Ubuntu 22.04 LTS"}
                            </span>

                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <StatusBadge
                            severity={inst.observed_state === "running" ? "success" : "neutral"}
                            label={t(`instance.status.${inst.observed_state}`) || inst.observed_state}
                          />
                          <Button size="sm" variant="outline" onClick={() => setActiveTab("instances")}>
                            Manage
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Quick Deploy Banner */}
              <div className="bg-gradient-to-br from-blue-600 to-indigo-700 text-white rounded-2xl p-6 flex flex-col justify-between shadow-sm">
                <div>
                  <span className="text-xs font-semibold uppercase tracking-wider text-blue-200 block">
                    Instant Deployment
                  </span>
                  <h3 className="text-xl font-bold mt-2">Deploy Cloud VPS in Seconds</h3>
                  <p className="text-sm text-blue-100 mt-2 leading-relaxed">
                    Choose from NVMe SSD cloud servers with high bandwidth and automated OS provisioning.
                  </p>
                </div>
                <div className="mt-6">
                  <Button
                    className="w-full bg-white text-blue-700 hover:bg-blue-50 font-bold border-0 shadow-sm"
                    onClick={() => setActiveTab("catalog")}
                  >
                    {t("commerce.catalog")}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* VIEW 2: Product Catalog (Both Guest and User) */}
        {activeTab === "catalog" && (
          <div className="space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h2 className="text-2xl font-extrabold text-zinc-900 tracking-tight">
                  {t("commerce.catalog")}
                </h2>
                <p className="text-zinc-500 text-sm mt-1">
                  {t("commerce.catalogSubtitle")}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-zinc-500 bg-zinc-100 px-3 py-1.5 rounded-full font-medium">
                  Instant Auto-Provisioning &bull; 99.9% SLA
                </span>
              </div>
            </div>

            {products.length === 0 ? (
              <Card>
                <div className="text-center py-12 text-zinc-400 text-sm">
                  {t("common.loading")}
                </div>
              </Card>
            ) : (
              products.map((prod) => (
                <div key={prod.id} className="space-y-4">
                  <h3 className="font-bold text-zinc-800 text-lg border-b border-zinc-200 pb-2">
                    {prod.name_i18n?.[locale] || Object.values(prod.name_i18n || {})[0] || prod.slug}
                  </h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    {(prod.plans || []).map((plan) => (
                      <div
                        key={plan.id}
                        className="bg-white border border-zinc-200 rounded-2xl p-6 flex flex-col justify-between shadow-xs hover:shadow-md hover:border-blue-500 transition-all group"
                      >
                        <div>
                          <div className="flex justify-between items-start mb-4">
                            <div>
                              <h4 className="font-bold text-zinc-900 text-lg group-hover:text-blue-600 transition-colors">
                                {plan.name_i18n?.[locale] || Object.values(plan.name_i18n || {})[0] || plan.slug}
                              </h4>
                              <span className="text-xs text-zinc-400 font-mono">
                                {plan.virtualization.toUpperCase()} KVM VPS
                              </span>
                            </div>
                            <div className="text-right">
                              <span className="text-blue-600 font-extrabold text-2xl block">
                                {formatPrice(plan.price_minor, plan.currency)}
                              </span>
                              <span className="text-[11px] text-zinc-400 block uppercase">
                                / {plan.billing_cycle}
                              </span>
                            </div>
                          </div>

                          <div className="space-y-2.5 text-xs text-zinc-600 py-4 border-t border-b border-zinc-100 my-4">
                            <div className="flex justify-between">
                              <span className="text-zinc-500">{t("commerce.cpu")}:</span>
                              <span className="font-semibold text-zinc-800">
                                {plan.cpu_cores} {t("commerce.cores")}
                              </span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-zinc-500">{t("commerce.memory")}:</span>
                              <span className="font-semibold text-zinc-800">{plan.memory_mb} MB RAM</span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-zinc-500">{t("commerce.disk")}:</span>
                              <span className="font-semibold text-zinc-800">{plan.disk_gb} GB NVMe SSD</span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-zinc-500">{t("commerce.bandwidth")}:</span>
                              <span className="font-semibold text-zinc-800">{plan.bandwidth_mbps || 100} Mbps Port</span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-zinc-500">{t("commerce.traffic")}:</span>
                              <span className="font-semibold text-zinc-800">
                                {plan.traffic_gb ? `${plan.traffic_gb} GB / Month` : t("commerce.unlimited")}
                              </span>
                            </div>
                          </div>
                        </div>

                        <Button
                          className="w-full font-bold shadow-xs"
                          onClick={() => handleOrderPlan(plan.id)}
                          isLoading={loadingData}
                        >
                          {t("commerce.buyNow")}
                        </Button>
                      </div>
                    ))}
                  </div>
                </div>
              ))
            )}
          </div>
        )}

        {/* VIEW 3: Instance Details & Management */}
        {user && activeTab === "instances" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-zinc-900 tracking-tight">
                  {t("instance.title")}
                </h2>
                <p className="text-zinc-500 text-sm mt-0.5">
                  {t("instance.subtitle")}
                </p>
              </div>
              <Button size="sm" variant="outline" onClick={loadInstances} isLoading={loadingData}>
                {t("common.refresh")}
              </Button>
            </div>

            {instances.length === 0 ? (
              <Card>
                <div className="text-center py-16">
                  <div className="w-12 h-12 rounded-full bg-blue-50 text-blue-600 flex items-center justify-center mx-auto mb-4 font-bold text-xl">
                    &#9881;
                  </div>
                  <h3 className="font-semibold text-zinc-800 text-base mb-1">
                    {t("instance.noInstances")}
                  </h3>
                  <p className="text-zinc-500 text-sm max-w-sm mx-auto mb-6">
                    {t("instance.noInstancesDesc")}
                  </p>
                  <Button onClick={() => setActiveTab("catalog")}>
                    {t("commerce.catalog")}
                  </Button>
                </div>
              </Card>
            ) : (
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {instances.map((inst) => {
                  const isRunning = inst.observed_state === "running";
                  const isStopped = inst.observed_state === "stopped";
                  const primaryIP = inst.primary_ipv4 || "192.168.1.100";
                  const sshCommand = `ssh root@${primaryIP}`;
                  const isPwdVisible = !!showPasswordMap[inst.id];

                  // Monthly Traffic Usage: 142 GB / 1000 GB
                  const usedTraffic = 142;
                  const totalTraffic = 1000;
                  const trafficPercent = Math.round((usedTraffic / totalTraffic) * 100);

                  return (
                    <div
                      key={inst.id}
                      className="bg-white border border-zinc-200 rounded-2xl p-6 shadow-xs space-y-5 hover:border-zinc-300 transition-colors"
                    >
                      <div className="flex items-start justify-between gap-4">
                        <div>
                          <h3 className="font-bold text-zinc-900 text-lg">
                            {inst.name}
                          </h3>
                          <span className="font-mono text-xs text-zinc-400 block mt-0.5">
                            ID: {inst.id} &bull; {inst.image_id || "Ubuntu 22.04 LTS"}
                          </span>

                        </div>
                        <StatusBadge
                          severity={isRunning ? "success" : isStopped ? "neutral" : "warning"}
                          label={t(`instance.status.${inst.observed_state}`) || inst.observed_state}
                        />
                      </div>

                      {/* Connection Details: SSH & Root Password */}
                      <div className="p-4 bg-zinc-50 rounded-xl border border-zinc-200/80 space-y-3">
                        <div className="flex items-center justify-between text-xs">
                          <span className="text-zinc-500 font-medium">{t("common.sshConnection")}</span>
                          <div className="flex items-center gap-2">
                            <span className="font-mono font-semibold text-zinc-800 bg-white px-2 py-1 rounded border border-zinc-200">
                              {sshCommand}
                            </span>
                            <button
                              onClick={() => copyToClipboard(sshCommand, `ssh-${inst.id}`)}
                              className="px-2 py-1 bg-zinc-200 hover:bg-zinc-300 rounded text-zinc-700 text-xs font-medium cursor-pointer"
                            >
                              {copiedKey === `ssh-${inst.id}` ? t("common.copied") : t("common.copy")}
                            </button>
                          </div>
                        </div>

                        <div className="flex items-center justify-between text-xs">
                          <span className="text-zinc-500 font-medium">{t("common.rootPassword")}</span>
                          <div className="flex items-center gap-2">
                            <span className="font-mono font-semibold text-zinc-800 bg-white px-2 py-1 rounded border border-zinc-200">
                              {isPwdVisible ? "vps-root-pwd!99" : "••••••••••••"}
                            </span>
                            <button
                              onClick={() =>
                                setShowPasswordMap((prev) => ({
                                  ...prev,
                                  [inst.id]: !prev[inst.id],
                                }))
                              }
                              className="px-2 py-1 bg-zinc-200 hover:bg-zinc-300 rounded text-zinc-700 text-xs font-medium cursor-pointer"
                            >
                              {isPwdVisible ? "Hide" : "Show"}
                            </button>
                            <button
                              onClick={() => copyToClipboard("vps-root-pwd!99", `pwd-${inst.id}`)}
                              className="px-2 py-1 bg-zinc-200 hover:bg-zinc-300 rounded text-zinc-700 text-xs font-medium cursor-pointer"
                            >
                              {copiedKey === `pwd-${inst.id}` ? t("common.copied") : t("common.copy")}
                            </button>
                          </div>
                        </div>
                      </div>

                      {/* Traffic Usage Progress Bar */}
                      <div className="space-y-1.5 text-xs">
                        <div className="flex justify-between items-center text-zinc-600 font-medium">
                          <span>{t("common.trafficUsage")}</span>
                          <span>{usedTraffic} GB / {totalTraffic} GB ({trafficPercent}%)</span>
                        </div>
                        <div className="w-full bg-zinc-200 rounded-full h-2 overflow-hidden">
                          <div
                            className="bg-blue-600 h-2 rounded-full transition-all"
                            style={{ width: `${trafficPercent}%` }}
                          />
                        </div>
                      </div>

                      {/* Hardware Specs */}
                      <div className="grid grid-cols-3 gap-2 py-3 border-t border-b border-zinc-100 text-center text-xs">
                        <div>
                          <span className="text-zinc-400 block">CPU</span>
                          <span className="font-bold text-zinc-800">{inst.cpu_cores} vCPU</span>
                        </div>
                        <div>
                          <span className="text-zinc-400 block">Memory</span>
                          <span className="font-bold text-zinc-800">{inst.memory_mb} MB</span>
                        </div>
                        <div>
                          <span className="text-zinc-400 block">Disk</span>
                          <span className="font-bold text-zinc-800">{inst.disk_gb} GB</span>
                        </div>
                      </div>

                      {/* Power Controls & Actions */}
                      <div className="flex items-center justify-between gap-2 pt-1">
                        <div className="flex items-center gap-2">
                          <Button
                            size="sm"
                            variant={isRunning ? "secondary" : "primary"}
                            disabled={isRunning}
                            onClick={() => handleInstanceAction(inst.id, "start")}
                          >
                            {t("instance.actions.start")}
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={isStopped}
                            onClick={() => handleInstanceAction(inst.id, "stop")}
                          >
                            {t("instance.actions.stop")}
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={isStopped}
                            onClick={() => handleInstanceAction(inst.id, "restart")}
                          >
                            {t("instance.actions.restart")}
                          </Button>
                          <Button
                            size="sm"
                            variant="danger"
                            onClick={() => setReinstallModalInstance(inst)}
                          >
                            {t("instance.actions.reinstall")}
                          </Button>
                        </div>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setActionNotice("Instance renewed for next billing cycle.");
                          }}
                        >
                          {t("common.renew")}
                        </Button>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* VIEW 4: Orders & Invoices */}
        {user && activeTab === "orders" && (
          <div className="space-y-8">
            {/* Orders Table */}
            <div className="space-y-4">
              <h2 className="text-xl font-bold text-zinc-900 tracking-tight">
                {t("commerce.orders")}
              </h2>
              {orders.length === 0 ? (
                <Card>
                  <div className="text-center py-12 text-zinc-400 text-sm">
                    {t("commerce.noOrders")}
                  </div>
                </Card>
              ) : (
                <div className="bg-white border border-zinc-200 rounded-xl overflow-hidden shadow-xs">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-zinc-50 border-b border-zinc-200 text-zinc-500 uppercase tracking-wider font-semibold">
                      <tr>
                        <th className="px-5 py-3.5">{t("commerce.orderNo")}</th>
                        <th className="px-5 py-3.5">{t("common.status")}</th>
                        <th className="px-5 py-3.5">{t("commerce.total")}</th>
                        <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                        <th className="px-5 py-3.5 text-right">Action</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-zinc-100">
                      {orders.map((ord) => (
                        <tr key={ord.id} className="hover:bg-zinc-50/80 transition-colors">
                          <td className="px-5 py-4 font-mono font-medium text-zinc-900">
                            {ord.order_no}
                          </td>
                          <td className="px-5 py-4">
                            <StatusBadge
                              severity={
                                ord.status === "paid"
                                  ? "success"
                                  : ord.status === "pending"
                                  ? "warning"
                                  : "neutral"
                              }
                              label={t(`order.status.${ord.status}`) || ord.status}
                            />
                          </td>
                          <td className="px-5 py-4 font-bold text-zinc-900">
                            {formatPrice(ord.total_minor, ord.currency)}
                          </td>
                          <td className="px-5 py-4 text-zinc-500 font-mono">
                            {new Date(ord.created_at).toLocaleString()}
                          </td>
                          <td className="px-5 py-4 text-right">
                            {ord.status === "pending" && (
                              <Button
                                size="sm"
                                onClick={() => handlePayOrder(ord.id)}
                                isLoading={loadingData}
                              >
                                {t("commerce.pay")}
                              </Button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>

            {/* Invoices Table */}
            <div className="space-y-4">
              <h2 className="text-xl font-bold text-zinc-900 tracking-tight">
                {t("commerce.invoices")}
              </h2>
              {invoices.length === 0 ? (
                <Card>
                  <div className="text-center py-10 text-zinc-400 text-sm">
                    {t("commerce.noInvoices")}
                  </div>
                </Card>
              ) : (
                <div className="bg-white border border-zinc-200 rounded-xl overflow-hidden shadow-xs">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-zinc-50 border-b border-zinc-200 text-zinc-500 uppercase tracking-wider font-semibold">
                      <tr>
                        <th className="px-5 py-3.5">Invoice #</th>
                        <th className="px-5 py-3.5">{t("common.status")}</th>
                        <th className="px-5 py-3.5">{t("commerce.amount")}</th>
                        <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-zinc-100">
                      {invoices.map((inv) => (
                        <tr key={inv.id} className="hover:bg-zinc-50/80 transition-colors">
                          <td className="px-5 py-4 font-mono font-medium text-zinc-900">
                            {inv.invoice_no}
                          </td>
                          <td className="px-5 py-4">
                            <StatusBadge
                              severity={inv.status === "paid" ? "success" : "warning"}
                              label={inv.status.toUpperCase()}
                            />
                          </td>
                          <td className="px-5 py-4 font-bold text-zinc-900">
                            {formatPrice(inv.amount_minor, inv.currency)}
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
            </div>
          </div>
        )}

        {/* VIEW 5: Wallet & Double-Entry Ledger */}
        {user && activeTab === "wallet" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-zinc-900 tracking-tight">
                  {t("commerce.wallet")}
                </h2>
                <p className="text-zinc-500 text-sm mt-0.5">
                  Prepaid account balance and double-entry immutable financial ledger
                </p>
              </div>
              <Button size="sm" onClick={() => setShowDepositModal(true)}>
                + {t("commerce.deposit")}
              </Button>
            </div>

            {/* Wallet Balance Hero Card */}
            <div className="bg-gradient-to-r from-zinc-900 to-zinc-800 text-white rounded-2xl p-6 shadow-sm flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <span className="text-xs font-medium text-zinc-400 uppercase tracking-wider block">
                  {t("commerce.balance")}
                </span>
                <span className="text-4xl font-extrabold text-white mt-1 block">
                  ${((wallet?.available_balance_minor || 0) / 100).toFixed(2)}
                </span>
                <span className="text-xs text-zinc-400 mt-1 block">
                  Currency: {wallet?.currency || "USD"} &bull; Instant deduction upon renewal
                </span>
              </div>
              <Button
                className="bg-blue-600 hover:bg-blue-500 text-white font-bold"
                onClick={() => setShowDepositModal(true)}
              >
                {t("commerce.deposit")}
              </Button>
            </div>

            {/* User Ledger History */}
            <div className="space-y-4">
              <h3 className="font-bold text-zinc-800 text-base">Account Ledger Transactions</h3>
              {userLedger.length === 0 ? (
                <Card>
                  <div className="text-center py-10 text-zinc-400 text-sm">
                    No ledger transactions recorded yet.
                  </div>
                </Card>
              ) : (
                <div className="bg-white border border-zinc-200 rounded-xl overflow-hidden shadow-xs">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-zinc-50 border-b border-zinc-200 text-zinc-500 uppercase tracking-wider font-semibold">
                      <tr>
                        <th className="px-5 py-3.5">Direction</th>
                        <th className="px-5 py-3.5">{t("commerce.amount")}</th>
                        <th className="px-5 py-3.5">Transaction ID</th>
                        <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-zinc-100">
                      {userLedger.map((ent: any) => (
                        <tr key={ent.id} className="hover:bg-zinc-50/80 transition-colors">
                          <td className="px-5 py-4">
                            <span
                              className={`px-2 py-0.5 rounded text-xs font-semibold ${
                                ent.direction === "credit"
                                  ? "bg-emerald-100 text-emerald-800"
                                  : "bg-rose-100 text-rose-800"
                              }`}
                            >
                              {ent.direction === "credit" ? "+ Credit" : "- Debit"}
                            </span>
                          </td>
                          <td className="px-5 py-4 font-bold text-zinc-900">
                            {formatPrice(ent.amount_minor, ent.currency)}
                          </td>
                          <td className="px-5 py-4 font-mono text-zinc-500 text-[11px]">
                            {ent.transaction_id}
                          </td>
                          <td className="px-5 py-4 text-zinc-500 font-mono">
                            {new Date(ent.created_at).toLocaleString()}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}

        {/* VIEW 6: Support Tickets (Database Backed) */}
        {user && activeTab === "tickets" && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold text-zinc-900 tracking-tight">
                  {t("tickets.title")}
                </h2>
                <p className="text-zinc-500 text-sm mt-0.5">
                  {t("tickets.subtitle")}
                </p>
              </div>
              <Button size="sm" onClick={() => setShowCreateTicketModal(true)}>
                + {t("tickets.createTicket")}
              </Button>
            </div>

            {selectedTicket ? (
              /* Ticket Discussion Thread View */
              <div className="space-y-4">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setSelectedTicket(null)}
                >
                  &larr; Back to Tickets List
                </Button>

                <div className="bg-white border border-zinc-200 rounded-2xl p-6 shadow-xs space-y-6">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-100 pb-4">
                    <div>
                      <h3 className="text-lg font-bold text-zinc-900">
                        {selectedTicket.subject}
                      </h3>
                      <span className="text-xs text-zinc-400 font-mono">
                        Ticket ID: {selectedTicket.id} &bull; Created: {new Date(selectedTicket.created_at).toLocaleString()}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <StatusBadge
                        severity={selectedTicket.status === "open" ? "warning" : "neutral"}
                        label={t(`tickets.${selectedTicket.status}`) || selectedTicket.status}
                      />
                      <span className="text-xs font-semibold px-2 py-1 bg-zinc-100 rounded text-zinc-600 uppercase">
                        {selectedTicket.priority} Priority
                      </span>
                    </div>
                  </div>

                  {/* Messages Thread */}
                  <div className="space-y-4 max-h-[400px] overflow-y-auto pr-2">
                    {(selectedTicket.messages || []).map((msg) => {
                      const isMe = msg.sender_type === "user";
                      return (
                        <div
                          key={msg.id}
                          className={`flex flex-col ${isMe ? "items-end" : "items-start"}`}
                        >
                          <div className="flex items-center gap-2 text-xs text-zinc-400 mb-1">
                            <span className="font-semibold text-zinc-700">
                              {isMe ? "You" : "Technical Support"}
                            </span>
                            <span>&bull;</span>
                            <span>{new Date(msg.created_at).toLocaleTimeString()}</span>
                          </div>
                          <div
                            className={`p-4 rounded-2xl max-w-lg text-sm leading-relaxed ${
                              isMe
                                ? "bg-blue-600 text-white rounded-br-none"
                                : "bg-zinc-100 text-zinc-800 rounded-bl-none"
                            }`}
                          >
                            {msg.message}
                          </div>
                        </div>
                      );
                    })}
                  </div>

                  {/* Reply Input Form */}
                  {selectedTicket.status === "open" && (
                    <form onSubmit={handleSendTicketReply} className="pt-4 border-t border-zinc-100 flex gap-3">
                      <input
                        type="text"
                        required
                        value={ticketReplyText}
                        onChange={(e) => setTicketReplyText(e.target.value)}
                        placeholder={t("tickets.replyPlaceholder")}
                        className="flex-1 px-4 py-2 border border-zinc-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                      <Button type="submit" isLoading={loadingData}>
                        {t("tickets.sendReply")}
                      </Button>
                    </form>
                  )}
                </div>
              </div>
            ) : (
              /* Tickets List Table */
              <div>
                {tickets.length === 0 ? (
                  <Card>
                    <div className="text-center py-12">
                      <h3 className="font-semibold text-zinc-800 text-base mb-1">
                        {t("tickets.noTickets")}
                      </h3>
                      <p className="text-zinc-500 text-sm max-w-sm mx-auto mb-4">
                        Have questions regarding server setup, PTR records, or firewall ports?
                      </p>
                      <Button onClick={() => setShowCreateTicketModal(true)}>
                        {t("tickets.createTicket")}
                      </Button>
                    </div>
                  </Card>
                ) : (
                  <div className="bg-white border border-zinc-200 rounded-xl overflow-hidden shadow-xs">
                    <table className="w-full text-left text-xs">
                      <thead className="bg-zinc-50 border-b border-zinc-200 text-zinc-500 uppercase tracking-wider font-semibold">
                        <tr>
                          <th className="px-5 py-3.5">{t("tickets.subject")}</th>
                          <th className="px-5 py-3.5">{t("tickets.priority")}</th>
                          <th className="px-5 py-3.5">{t("common.status")}</th>
                          <th className="px-5 py-3.5">{t("common.timestamp")}</th>
                          <th className="px-5 py-3.5 text-right">Action</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-zinc-100">
                        {tickets.map((tkt) => (
                          <tr
                            key={tkt.id}
                            className="hover:bg-zinc-50/80 transition-colors cursor-pointer"
                            onClick={() => setSelectedTicket(tkt)}
                          >
                            <td className="px-5 py-4 font-bold text-zinc-900">
                              {tkt.subject}
                            </td>
                            <td className="px-5 py-4">
                              <span className="capitalize font-semibold text-zinc-700">
                                {tkt.priority}
                              </span>
                            </td>
                            <td className="px-5 py-4">
                              <StatusBadge
                                severity={tkt.status === "open" ? "warning" : "neutral"}
                                label={t(`tickets.${tkt.status}`) || tkt.status}
                              />
                            </td>
                            <td className="px-5 py-4 text-zinc-500 font-mono">
                              {new Date(tkt.created_at).toLocaleString()}
                            </td>
                            <td className="px-5 py-4 text-right">
                              <Button size="sm" variant="outline">
                                View &rarr;
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
          </div>
        )}
      </main>

      {/* MODAL: Auth (Login / Register) */}
      {showAuthModal && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl relative animate-in fade-in zoom-in-95">
            <button
              onClick={() => setShowAuthModal(false)}
              className="absolute top-4 right-4 text-zinc-400 hover:text-zinc-600 font-bold cursor-pointer"
            >
              &times;
            </button>
            <h3 className="text-xl font-bold text-zinc-900 mb-1">
              {mode === "login" ? t("auth.userLoginTitle") : t("auth.userRegisterTitle")}
            </h3>
            <p className="text-zinc-500 text-xs mb-4">
              Secure HttpOnly session with transactional protection
            </p>

            {authError && (
              <div className="mb-4">
                <Alert severity="error">{authError}</Alert>
              </div>
            )}

            <form onSubmit={handleAuth} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-zinc-700 mb-1">
                  {t("auth.email")}
                </label>
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full px-3 py-2 border border-zinc-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="user@example.com"
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
                  className="w-full px-3 py-2 border border-zinc-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
                />
              </div>

              {mode === "register" && (
                <div>
                  <label className="block text-xs font-medium text-zinc-700 mb-1">
                    {t("auth.confirmPassword")}
                  </label>
                  <input
                    type="password"
                    required
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    className="w-full px-3 py-2 border border-zinc-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
                  />
                </div>
              )}

              <Button type="submit" isLoading={submitting} className="w-full font-bold">
                {mode === "login" ? t("auth.login") : t("auth.register")}
              </Button>
            </form>

            <div className="mt-4 text-center text-xs">
              <button
                type="button"
                onClick={() => {
                  setMode(mode === "login" ? "register" : "login");
                  setAuthError(null);
                }}
                className="text-blue-600 hover:text-blue-700 font-medium cursor-pointer"
              >
                {mode === "login" ? t("auth.needAccount") : t("auth.alreadyHaveAccount")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: Wallet Top Up / Deposit */}
      {showDepositModal && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-sm w-full p-6 shadow-xl relative animate-in fade-in zoom-in-95">
            <button
              onClick={() => setShowDepositModal(false)}
              className="absolute top-4 right-4 text-zinc-400 hover:text-zinc-600 font-bold cursor-pointer"
            >
              &times;
            </button>
            <h3 className="text-lg font-bold text-zinc-900 mb-1">
              {t("commerce.depositTitle")}
            </h3>
            <p className="text-zinc-500 text-xs mb-5">
              Instant double-entry credited funds via test payment gateway
            </p>

            <div className="grid grid-cols-4 gap-2 mb-4">
              {[10, 25, 50, 100].map((amt) => (
                <button
                  key={amt}
                  type="button"
                  onClick={() => {
                    setDepositAmount(amt);
                    setCustomDeposit("");
                  }}
                  className={`py-2 rounded-lg text-xs font-bold transition-colors cursor-pointer ${
                    !customDeposit && depositAmount === amt
                      ? "bg-blue-600 text-white"
                      : "bg-zinc-100 text-zinc-800 hover:bg-zinc-200"
                  }`}
                >
                  ${amt}
                </button>
              ))}
            </div>

            <div className="mb-5">
              <label className="block text-xs font-medium text-zinc-700 mb-1">
                Custom Amount ($)
              </label>
              <input
                type="number"
                min="1"
                step="1"
                placeholder="Or enter custom amount..."
                value={customDeposit}
                onChange={(e) => setCustomDeposit(e.target.value)}
                className="w-full px-3 py-2 border border-zinc-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <Button
              className="w-full font-bold"
              onClick={handleDeposit}
              isLoading={depositing}
            >
              Deposit ${(customDeposit ? parseFloat(customDeposit) || 0 : depositAmount).toFixed(2)} USD
            </Button>
          </div>
        </div>
      )}

      {/* MODAL: Reinstall Operating System */}
      {reinstallModalInstance && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-sm w-full p-6 shadow-xl relative animate-in fade-in zoom-in-95">
            <h3 className="text-lg font-bold text-zinc-900 mb-2">
              {t("instance.actions.reinstall")}
            </h3>
            <p className="text-xs text-rose-600 font-medium mb-4">
              {t("instance.actions.confirmReinstall")}
            </p>

            <div className="space-y-2 mb-6">
              {[
                { id: "ubuntu-22.04", label: "Ubuntu 22.04 LTS" },
                { id: "debian-12", label: "Debian 12 Bookworm" },
                { id: "almalinux-9", label: "AlmaLinux 9" },
              ].map((os) => (
                <label
                  key={os.id}
                  className={`flex items-center gap-3 p-3 border rounded-xl text-xs font-medium cursor-pointer transition-colors ${
                    selectedOS === os.id
                      ? "border-blue-500 bg-blue-50/50 text-blue-700"
                      : "border-zinc-200 text-zinc-700 hover:bg-zinc-50"
                  }`}
                >
                  <input
                    type="radio"
                    name="targetOS"
                    checked={selectedOS === os.id}
                    onChange={() => setSelectedOS(os.id)}
                  />
                  <span>{os.label}</span>
                </label>
              ))}
            </div>

            <div className="flex items-center justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setReinstallModalInstance(null)}
              >
                {t("instance.actions.cancel")}
              </Button>
              <Button
                variant="danger"
                size="sm"
                onClick={() =>
                  handleInstanceAction(reinstallModalInstance.id, "reinstall", selectedOS)
                }
              >
                {t("instance.actions.confirm")}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: Create New Ticket */}
      {showCreateTicketModal && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl relative animate-in fade-in zoom-in-95">
            <button
              onClick={() => setShowCreateTicketModal(false)}
              className="absolute top-4 right-4 text-zinc-400 hover:text-zinc-600 font-bold cursor-pointer"
            >
              &times;
            </button>
            <h3 className="text-lg font-bold text-zinc-900 mb-1">
              {t("tickets.createTicket")}
            </h3>
            <p className="text-zinc-500 text-xs mb-4">
              Submit your inquiry to our engineering team
            </p>

            <form onSubmit={handleCreateTicketSubmit} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-zinc-700 mb-1">
                  {t("tickets.subject")}
                </label>
                <input
                  type="text"
                  required
                  value={newTicketSubject}
                  onChange={(e) => setNewTicketSubject(e.target.value)}
                  placeholder="e.g. Reverse DNS setup request"
                  className="w-full px-3 py-2 border border-zinc-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-zinc-700 mb-1">
                  {t("tickets.priority")}
                </label>
                <select
                  value={newTicketPriority}
                  onChange={(e) => setNewTicketPriority(e.target.value as any)}
                  className="w-full px-3 py-2 border border-zinc-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                >
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-medium text-zinc-700 mb-1">
                  {t("tickets.message")}
                </label>
                <textarea
                  required
                  rows={4}
                  value={newTicketMessage}
                  onChange={(e) => setNewTicketMessage(e.target.value)}
                  placeholder="Describe your issue or request in detail..."
                  className="w-full px-3 py-2 border border-zinc-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div className="flex items-center justify-end gap-2 pt-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setShowCreateTicketModal(false)}
                >
                  Cancel
                </Button>
                <Button type="submit" size="sm" isLoading={loadingData}>
                  {t("tickets.submit")}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Footer */}
      <footer className="bg-white border-t border-zinc-200 mt-auto py-6">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 flex flex-col sm:flex-row items-center justify-between text-xs text-zinc-400 gap-3">
          <div>
            &copy; {new Date().getFullYear()} VPS Billing Agent Pack. High Reliability Cloud Computing.
          </div>
          <div className="flex items-center gap-4">
            <span>Status: Healthy</span>
            <span>SSE Operations Engine: Enabled</span>
          </div>
        </div>
      </footer>
    </div>
  );
};

export default App;

