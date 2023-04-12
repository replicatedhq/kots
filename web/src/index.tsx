import React from "react";
import { createRoot } from "react-dom/client";

import { App } from "@src/App";
// import ReplicatedErrorBoundary from "./components/shared/ErrorBoundary";
// import { Root } from "./Root";

// root.render(
//   <ReplicatedErrorBoundary>
//     <Root />
//   </ReplicatedErrorBoundary>
// );

const container = document.getElementById("app");
if (!container) {
  throw new Error("No container found");
}

const root = createRoot(container);

root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>);
