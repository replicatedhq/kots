import React from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import ReplicatedErrorBoundary from "./components/shared/ErrorBoundary";
import { Root } from "./Root";

const container = document.getElementById("app");
if (!container) {
  throw new Error("No container found");
}

const root = createRoot(container); // createRoot(container!) if you use TypeScript

root.render(
  <ReplicatedErrorBoundary>
    <BrowserRouter>
      <Root />
    </BrowserRouter>
  </ReplicatedErrorBoundary>
);
