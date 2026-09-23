import React from "react";
import ReactDOM from "react-dom/client";
import { I18nProvider } from "@vps-billing/shared";
import App from "./App";
import "./index.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <I18nProvider storageKey="admin_web_locale">
      <App />
    </I18nProvider>
  </React.StrictMode>
);
