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
  commerceApi,
  instanceApi,
} from "@vps-billing/shared";

interface Ticket {
  id: string;
  subject: string;
  message: string;
  priority: "low" | "medium" | "high";
  status: "open" | "closed";
  created_at: string;
}

export const App: React.FC = () => {
  const { t, locale } = useI18n();

  const [user, setUser] = useState<UserDTO | null>(null);
  const [checkingAuth, setCheckingAuth] = useState(true);

  // Active tab
  const [activeTab, setActiveTab] = useState<
    "catalog" | "instances" | "orders" | "billing" | "tickets" | "account"
  >("catalog");

  // Auth Form State
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const [successNotice, setSuccessNotice] = useState<string | null>(null);

  // Commerce & Infra state
  const [products, setProducts] = useState<Product[]>([]);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [wallet, setWallet] = useState<Wallet | null>(null);
  const [loadingData, setLoadingData] = useState(false);
  const [actionNotice, setActionNotice] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  // Operation System Integration
  const [activeOperationId, setActiveOperationId] = useState<string | null>(null);

  // Reinstall modal state
  const [reinstallModalInstance, setReinstallModalInstance] = useState<Instance | null>(null);
  const [selectedOS, setSelectedOS] = useState("ubuntu-22.04");

  // Tickets state
  const [tickets, setTickets] = useState<Ticket[]>([
    {
      id: "t-1001",
      subject: "Welcome to VPS Platform Support",
      message: "Feel free to submit tickets for reverse DNS, IP routing, or capacity requests.",
      priority: "low",
      status: "closed",
      created_at: new Date(Date.now() - 86400000).toISOString(),
    },
  ]);
  const [ticketSubject, setTicketSubject] = useState("");
  const [ticketMessage, setTicketMessage] = useState("");
  const [ticketPriority, setTicketPriority] = useState<"low" | "medium" | "high">("medium");

  // Check current session on boot
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
      if (res.success) {
        setUser(res.data.user);
      } else {
        setUser(null);
      }
    } catch {
      setUser(null);
    } finally {
      setCheckingAuth(false);
    }
  };

  const loadCatalog = async () => {
    try {
      const res = await commerceApi.listProducts();
      setProducts(res.products || []);
    } catch {
      // Ignore initial public load error
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

  const loadUserData = async () => {
    setLoadingData(true);
    try {
      if (activeTab === "instances") {
        await loadInstances();
      } else if (activeTab === "orders" || activeTab === "catalog") {
        const oRes = await commerceApi.listOrders();
        setOrders(oRes.orders || []);
      } else if (activeTab === "billing") {
        const [wRes, iRes] = await Promise.all([
          commerceApi.getWallet(),
          commerceApi.listInvoices(),
        ]);
        setWallet(wRes.wallet);
        setInvoices(iRes.invoices || []);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoadingData(false);
    }
  };

  const handleOrderPlan = async (planId: string) => {
    if (!user) {
      setActiveTab("account");
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
      // Trigger simulation which immediately kicks off fulfillment provision workflow
      await commerceApi.simulatePayment(payRes.payment.payment_no);
      setActionNotice(t("commerce.paymentSuccess"));
      // Switch directly to instances tab to view the live provisioning!
      setActiveTab("instances");
      await loadInstances();
    } catch (err: any) {
      setActionError(err?.message || t("errors.internal_error"));
    } finally {
      setLoadingData(false);
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

  const handleCreateTicket = (e: React.FormEvent) => {
    e.preventDefault();
    if (!ticketSubject.trim() || !ticketMessage.trim()) return;

    const newTicket: Ticket = {
      id: `t-${Date.now().toString().slice(-4)}`,
      subject: ticketSubject,
      message: ticketMessage,
      priority: ticketPriority,
      status: "open",
      created_at: new Date().toISOString(),
    };
    setTickets([newTicket, ...tickets]);
    setTicketSubject("");
    setTicketMessage("");
    setActionNotice(t("tickets.submit"));
  };

  const handleAuth = async (e: React.FormEvent) => {
    e.preventDefault();
    setAuthError(null);
    setSuccessNotice(null);

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
          setSuccessNotice(t("auth.alreadyHaveAccount"));
          setMode("login");
          setPassword("");
          setConfirmPassword("");
        } else {
          setAuthError(t(res.error?.message_key || "errors.validation_failed"));
        }
      } else {
        const res = await authApi.loginUser({ email, password });
        if (res.success) {
          if (res.data.user) {
            setUser(res.data.user);
            setPassword("");
          }
        } else {
          setAuthError(t(res.error?.message_key || "errors.invalid_credentials"));
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
    }
  };

  const formatPrice = (minor: number, currency: string) => {
    return `${currency} ${(minor / 100).toFixed(2)}`;
  };

  return (
    <div className="min-h-screen flex flex-col bg-zinc-50 text-zinc-900">
      {/* Header */}
      <header className="bg-white border-b border-zinc-200 sticky top-0 z-10 shadow-xs">
        <div className="max-w-6xl mx-auto px-4 h-16 flex items-center justify-between">
          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-lg bg-blue-600 text-white flex items-center justify-center font-bold text-lg shadow-sm">
                V
              </div>
              <span className="font-semibold text-zinc-900 tracking-tight text-base">
                {t("common.appNameUser")}
              </span>
            </div>

            {user && (
              <nav className="hidden md:flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => setActiveTab("catalog")}
                  className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                    activeTab === "catalog"
                      ? "bg-zinc-100 text-zinc-900"
                      : "text-zinc-600 hover:text-zinc-900"
                  }`}
                >
                  {t("commerce.catalog")}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("instances")}
                  className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                    activeTab === "instances"
                      ? "bg-zinc-100 text-zinc-900"
                      : "text-zinc-600 hover:text-zinc-900"
                  }`}
                >
                  {t("common.servers")}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("orders")}
                  className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                    activeTab === "orders"
                      ? "bg-zinc-100 text-zinc-900"
                      : "text-zinc-600 hover:text-zinc-900"
                  }`}
                >
                  {t("commerce.orders")}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("billing")}
                  className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                    activeTab === "billing"
                      ? "bg-zinc-100 text-zinc-900"
                      : "text-zinc-600 hover:text-zinc-900"
                  }`}
                >
                  {t("common.billing")}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("tickets")}
                  className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                    activeTab === "tickets"
                      ? "bg-zinc-100 text-zinc-900"
                      : "text-zinc-600 hover:text-zinc-900"
                  }`}
                >
                  {t("common.tickets")}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("account")}
                  className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                    activeTab === "account"
                      ? "bg-zinc-100 text-zinc-900"
                      : "text-zinc-600 hover:text-zinc-900"
                  }`}
                >
                  {t("common.dashboard")}
                </button>
              </nav>
            )}
          </div>
          <div className="flex items-center gap-4">
            <LanguageSwitcher />
            {user && (
              <Button size="sm" variant="outline" onClick={handleLogout}>
                {t("auth.logout")}
              </Button>
            )}
          </div>
        </div>
      </header>

      {/* Body */}
      <main className="flex-1 max-w-6xl w-full mx-auto px-4 py-8">
        {checkingAuth ? (
          <div className="py-20 text-center text-zinc-400 text-sm">{t("common.loading")}</div>
        ) : !user ? (
          /* Authentication Screen */
          <div className="max-w-md mx-auto py-12">
            <Card
              title={mode === "login" ? t("auth.userLoginTitle") : t("auth.userRegisterTitle")}
              subtitle="Secure HttpOnly Cookie Session &bull; CSRF Protected"
            >
              {authError && (
                <div className="mb-4">
                  <Alert severity="error">{authError}</Alert>
                </div>
              )}

              {successNotice && (
                <div className="mb-4">
                  <Alert severity="success">{successNotice}</Alert>
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
                    className="w-full px-3 py-2 border border-zinc-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
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
                    className="w-full px-3 py-2 border border-zinc-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
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
                      className="w-full px-3 py-2 border border-zinc-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
                    />
                  </div>
                )}

                <Button type="submit" isLoading={submitting} className="w-full">
                  {mode === "login" ? t("auth.login") : t("auth.register")}
                </Button>
              </form>

              <div className="mt-6 text-center text-xs">
                <button
                  type="button"
                  onClick={() => {
                    setMode(mode === "login" ? "register" : "login");
                    setAuthError(null);
                    setSuccessNotice(null);
                  }}
                  className="text-blue-600 hover:text-blue-700 font-medium cursor-pointer"
                >
                  {mode === "login" ? t("auth.needAccount") : t("auth.alreadyHaveAccount")}
                </button>
              </div>
            </Card>
          </div>
        ) : (
          /* User Dashboard Views */
          <div className="space-y-6">
            {actionNotice && (
              <Alert severity="success">{actionNotice}</Alert>
            )}
            {actionError && (
              <Alert severity="error">{actionError}</Alert>
            )}

            {/* Active Operation Progress Display */}
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

            {/* TAB 1: Product Catalog */}
            {activeTab === "catalog" && (
              <div className="space-y-6">
                <div>
                  <h2 className="text-xl font-bold text-zinc-900 tracking-tight">
                    {t("commerce.catalog")}
                  </h2>
                  <p className="text-zinc-500 text-sm mt-1">
                    {t("commerce.catalogSubtitle")}
                  </p>
                </div>

                {products.map((prod) => (
                  <div key={prod.id} className="space-y-4">
                    <h3 className="font-semibold text-zinc-800 text-base">
                      {prod.name_i18n?.[locale] || Object.values(prod.name_i18n || {})[0] || prod.slug}
                    </h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                      {(prod.plans || []).map((plan) => (
                        <div
                          key={plan.id}
                          className="bg-white border border-zinc-200 rounded-xl p-5 flex flex-col justify-between shadow-sm hover:border-blue-500 transition-colors"
                        >
                          <div>
                            <div className="flex justify-between items-start mb-3">
                              <h4 className="font-bold text-zinc-900 text-base">
                                {plan.name_i18n?.[locale] || Object.values(plan.name_i18n || {})[0] || plan.slug}
                              </h4>
                              <span className="text-blue-600 font-bold text-lg">
                                {formatPrice(plan.price_minor, plan.currency)}
                              </span>
                            </div>

                            <div className="space-y-2 text-xs text-zinc-600 py-3 border-t border-b border-zinc-100 my-3">
                              <div className="flex justify-between">
                                <span className="text-zinc-400">{t("commerce.cpu")}:</span>
                                <span className="font-medium text-zinc-800">
                                  {plan.cpu_cores} {t("commerce.cores")}
                                </span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-zinc-400">{t("commerce.memory")}:</span>
                                <span className="font-medium text-zinc-800">{plan.memory_mb} MB</span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-zinc-400">{t("commerce.disk")}:</span>
                                <span className="font-medium text-zinc-800">{plan.disk_gb} GB NVMe</span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-zinc-400">{t("commerce.traffic")}:</span>
                                <span className="font-medium text-zinc-800">
                                  {plan.traffic_gb ? `${plan.traffic_gb} GB` : t("commerce.unlimited")}
                                </span>
                              </div>
                            </div>
                          </div>

                          <Button
                            className="w-full mt-4"
                            onClick={() => handleOrderPlan(plan.id)}
                            isLoading={loadingData}
                          >
                            {t("commerce.buyNow")}
                          </Button>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* TAB 2: Instances Management */}
            {activeTab === "instances" && (
              <div className="space-y-6">
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="text-xl font-bold text-zinc-900 tracking-tight">
                      {t("instance.title")}
                    </h2>
                    <p className="text-zinc-500 text-sm mt-1">
                      {t("instance.subtitle")}
                    </p>
                  </div>
                  <Button size="sm" variant="outline" onClick={loadInstances} isLoading={loadingData}>
                    {t("common.refresh")}
                  </Button>
                </div>

                {loadingData && instances.length === 0 ? (
                  <div className="text-center py-16 text-zinc-400 text-sm">
                    {t("common.loading")}
                  </div>
                ) : instances.length === 0 ? (
                  <Card>
                    <div className="text-center py-16">
                      <div className="w-12 h-12 rounded-full bg-blue-50 text-blue-600 flex items-center justify-center mx-auto mb-4 font-bold text-xl">
                        &hearts;
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
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    {instances.map((inst) => {
                      const isRunning = inst.observed_state === "running";
                      const isStopped = inst.observed_state === "stopped";
                      return (
                        <div
                          key={inst.id}
                          className="bg-white border border-zinc-200 rounded-xl p-5 shadow-xs space-y-4 hover:border-zinc-300 transition-colors"
                        >
                          <div className="flex items-start justify-between gap-4">
                            <div>
                              <h3 className="font-bold text-zinc-900 text-base">
                                {inst.name}
                              </h3>
                              <span className="font-mono text-xs text-zinc-400 block mt-0.5">
                                ID: {inst.id}
                              </span>
                            </div>
                            <StatusBadge
                              severity={
                                isRunning
                                  ? "success"
                                  : isStopped
                                  ? "neutral"
                                  : "warning"
                              }
                              label={t(`instance.status.${inst.observed_state}`) || inst.observed_state}
                            />
                          </div>

                          {/* Network info */}
                          <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-100 space-y-1.5 text-xs">
                            <div className="flex justify-between items-center">
                              <span className="text-zinc-500">{t("instance.ipv4")}:</span>
                              <span className="font-mono font-medium text-zinc-800">
                                {inst.primary_ipv4 || "192.168.1.100"}
                              </span>
                            </div>
                            {inst.primary_ipv6 && (
                              <div className="flex justify-between items-center">
                                <span className="text-zinc-500">{t("instance.ipv6")}:</span>
                                <span className="font-mono text-zinc-700 truncate max-w-[200px]">
                                  {inst.primary_ipv6}
                                </span>
                              </div>
                            )}
                          </div>

                          {/* Specs */}
                          <div className="grid grid-cols-3 gap-2 text-center text-xs">
                            <div className="p-2 bg-zinc-50 rounded-md border border-zinc-100">
                              <span className="text-zinc-400 block text-[10px] uppercase font-semibold">
                                {t("commerce.cpu")}
                              </span>
                              <span className="font-bold text-zinc-800">
                                {inst.cpu_cores} {t("commerce.cores")}
                              </span>
                            </div>
                            <div className="p-2 bg-zinc-50 rounded-md border border-zinc-100">
                              <span className="text-zinc-400 block text-[10px] uppercase font-semibold">
                                {t("commerce.memory")}
                              </span>
                              <span className="font-bold text-zinc-800">
                                {inst.memory_mb} MB
                              </span>
                            </div>
                            <div className="p-2 bg-zinc-50 rounded-md border border-zinc-100">
                              <span className="text-zinc-400 block text-[10px] uppercase font-semibold">
                                {t("commerce.disk")}
                              </span>
                              <span className="font-bold text-zinc-800">
                                {inst.disk_gb} GB
                              </span>
                            </div>
                          </div>

                          {/* Actions */}
                          <div className="pt-2 border-t border-zinc-100 flex items-center justify-end gap-2">
                            {isStopped && (
                              <Button
                                size="sm"
                                variant="primary"
                                onClick={() => handleInstanceAction(inst.id, "start")}
                              >
                                {t("instance.actions.start")}
                              </Button>
                            )}

                            {isRunning && (
                              <>
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => handleInstanceAction(inst.id, "restart")}
                                >
                                  {t("instance.actions.restart")}
                                </Button>
                                <Button
                                  size="sm"
                                  variant="danger"
                                  onClick={() => {
                                    if (window.confirm(t("instance.actions.confirmStop"))) {
                                      handleInstanceAction(inst.id, "stop");
                                    }
                                  }}
                                >
                                  {t("instance.actions.stop")}
                                </Button>
                              </>
                            )}

                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => setReinstallModalInstance(inst)}
                            >
                              {t("instance.actions.reinstall")}
                            </Button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            )}

            {/* Reinstall OS Modal */}
            {reinstallModalInstance && (
              <div className="fixed inset-0 bg-black/40 backdrop-blur-xs flex items-center justify-center z-50 p-4">
                <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6 space-y-4">
                  <h3 className="font-bold text-lg text-zinc-900">
                    {t("instance.actions.reinstall")} &bull; {reinstallModalInstance.name}
                  </h3>
                  <p className="text-sm text-amber-700 bg-amber-50 p-3 rounded-lg border border-amber-200">
                    {t("instance.actions.confirmReinstall")}
                  </p>

                  <div>
                    <label className="block text-xs font-semibold text-zinc-700 mb-2">
                      Target OS Image
                    </label>
                    <select
                      value={selectedOS}
                      onChange={(e) => setSelectedOS(e.target.value)}
                      className="w-full px-3 py-2 border border-zinc-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="ubuntu-22.04">Ubuntu 22.04 LTS</option>
                      <option value="debian-12">Debian 12 Bookworm</option>
                      <option value="almalinux-9">AlmaLinux 9</option>
                    </select>
                  </div>

                  <div className="flex items-center justify-end gap-3 pt-3">
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

            {/* TAB 3: Orders */}
            {activeTab === "orders" && (
              <Card title={t("commerce.orders")}>
                {orders.length === 0 ? (
                  <div className="text-center py-12 text-zinc-400 text-sm">
                    {t("commerce.noOrders")}
                  </div>
                ) : (
                  <div className="divide-y divide-zinc-100">
                    {orders.map((ord) => (
                      <div key={ord.id} className="py-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                        <div className="space-y-1">
                          <div className="flex items-center gap-3">
                            <span className="font-mono text-sm font-semibold text-zinc-900">
                              {ord.order_no}
                            </span>
                            <StatusBadge
                              severity={
                                ord.status === "paid" || ord.status === "fulfilled"
                                  ? "success"
                                  : ord.status === "pending"
                                  ? "warning"
                                  : "neutral"
                              }
                              label={t(`order.status.${ord.status}`) || ord.status}
                            />
                          </div>
                          <span className="text-xs text-zinc-400 block font-mono">
                            {new Date(ord.created_at).toLocaleString()} &bull;{" "}
                            {formatPrice(ord.total_minor, ord.currency)}
                          </span>
                        </div>

                        {ord.status === "pending" && (
                          <div className="flex items-center gap-2">
                            <Button
                              size="sm"
                              onClick={() => handlePayOrder(ord.id)}
                              isLoading={loadingData}
                            >
                              {t("commerce.simulatePay")}
                            </Button>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </Card>
            )}

            {/* TAB 4: Invoices & Wallet */}
            {activeTab === "billing" && (
              <div className="space-y-6">
                <Card title={t("commerce.wallet")}>
                  <div className="p-4 bg-zinc-50 rounded-xl border border-zinc-100 flex items-center justify-between">
                    <div>
                      <span className="text-xs text-zinc-500 block">{t("commerce.balance")}</span>
                      <span className="text-2xl font-bold text-zinc-900">
                        {wallet ? formatPrice(wallet.available_balance_minor, wallet.currency) : "$ 0.00"}
                      </span>
                    </div>
                  </div>
                </Card>

                <Card title={t("commerce.invoices")}>
                  {invoices.length === 0 ? (
                    <div className="text-center py-12 text-zinc-400 text-sm">
                      {t("commerce.noInvoices")}
                    </div>
                  ) : (
                    <div className="divide-y divide-zinc-100">
                      {invoices.map((inv) => (
                        <div key={inv.id} className="py-4 flex items-center justify-between">
                          <div>
                            <span className="font-mono text-sm font-semibold text-zinc-900 block">
                              {inv.invoice_no}
                            </span>
                            <span className="text-xs text-zinc-400 font-mono">
                              {new Date(inv.created_at).toLocaleString()}
                            </span>
                          </div>
                          <div className="flex items-center gap-4">
                            <span className="font-semibold text-zinc-900 text-sm">
                              {formatPrice(inv.amount_minor, inv.currency)}
                            </span>
                            <StatusBadge
                              severity={inv.status === "paid" ? "success" : "warning"}
                              label={inv.status}
                            />
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </Card>
              </div>
            )}

            {/* TAB 5: Support Tickets */}
            {activeTab === "tickets" && (
              <div className="space-y-6">
                <div>
                  <h2 className="text-xl font-bold text-zinc-900 tracking-tight">
                    {t("tickets.title")}
                  </h2>
                  <p className="text-zinc-500 text-sm mt-1">
                    {t("tickets.subtitle")}
                  </p>
                </div>

                <Card title={t("tickets.createTicket")}>
                  <form onSubmit={handleCreateTicket} className="space-y-4">
                    <div>
                      <label className="block text-xs font-medium text-zinc-700 mb-1">
                        {t("tickets.subject")}
                      </label>
                      <input
                        type="text"
                        required
                        value={ticketSubject}
                        onChange={(e) => setTicketSubject(e.target.value)}
                        placeholder="Reverse DNS request / IPv6 setup..."
                        className="w-full px-3 py-2 border border-zinc-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </div>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs font-medium text-zinc-700 mb-1">
                          {t("tickets.priority")}
                        </label>
                        <select
                          value={ticketPriority}
                          onChange={(e) => setTicketPriority(e.target.value as any)}
                          className="w-full px-3 py-2 border border-zinc-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        >
                          <option value="low">Low</option>
                          <option value="medium">Medium</option>
                          <option value="high">High</option>
                        </select>
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-zinc-700 mb-1">
                        {t("tickets.message")}
                      </label>
                      <textarea
                        required
                        rows={3}
                        value={ticketMessage}
                        onChange={(e) => setTicketMessage(e.target.value)}
                        placeholder="Detailed request information..."
                        className="w-full px-3 py-2 border border-zinc-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </div>
                    <Button type="submit" size="sm">
                      {t("tickets.submit")}
                    </Button>
                  </form>
                </Card>

                <Card title={t("tickets.title")}>
                  <div className="divide-y divide-zinc-100">
                    {tickets.map((tkt) => (
                      <div key={tkt.id} className="py-4 space-y-1">
                        <div className="flex items-center justify-between">
                          <span className="font-semibold text-zinc-900 text-sm">
                            {tkt.subject}
                          </span>
                          <StatusBadge
                            severity={tkt.status === "open" ? "warning" : "neutral"}
                            label={tkt.status === "open" ? t("tickets.open") : t("tickets.closed")}
                          />
                        </div>
                        <p className="text-xs text-zinc-600">
                          {tkt.message}
                        </p>
                        <span className="text-[11px] text-zinc-400 font-mono block">
                          {new Date(tkt.created_at).toLocaleString()} &bull; Priority: {tkt.priority}
                        </span>
                      </div>
                    ))}
                  </div>
                </Card>
              </div>
            )}

            {/* TAB 6: Account */}
            {activeTab === "account" && (
              <Card title={t("common.dashboard")} subtitle="Account Profile & Security">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
                  <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-100">
                    <span className="text-zinc-500 block text-xs">{t("auth.email")}</span>
                    <span className="font-medium text-zinc-900">{user.email}</span>
                  </div>
                  <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-100">
                    <span className="text-zinc-500 block text-xs">{t("common.status")}</span>
                    <StatusBadge
                      severity={user.status === "active" ? "success" : "warning"}
                      label={user.status}
                    />
                  </div>
                  <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-100">
                    <span className="text-zinc-500 block text-xs">{t("common.language")}</span>
                    <span className="font-medium text-zinc-900">{user.locale}</span>
                  </div>
                  <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-100">
                    <span className="text-zinc-500 block text-xs">{t("common.timestamp")}</span>
                    <span className="font-medium text-zinc-900 font-mono text-xs">
                      {user.last_login_at ? new Date(user.last_login_at).toLocaleString() : "-"}
                    </span>
                  </div>
                </div>
              </Card>
            )}
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="border-t border-zinc-200 py-6 text-center text-xs text-zinc-400">
        VPS Billing Platform &copy; 2026. Phase 8 User Web.
      </footer>
    </div>
  );
};

export default App;
