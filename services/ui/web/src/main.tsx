import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import "./styles.css";
import { AppProviders } from "./app/providers";
import { AppRoutes } from "./app/routes";

const container = document.getElementById("root");

if (!container) {
  throw new Error("Root container not found");
}

const root = createRoot(container);

root.render(
  <StrictMode>
    <AppProviders>
      <AppRoutes />
    </AppProviders>
  </StrictMode>,
);
