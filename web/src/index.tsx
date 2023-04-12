import React from "react";
import ReactDOM from "react-dom";

import { App } from "./App";
// import { createRoot } from "react-dom/client";
// import ReplicatedErrorBoundary from "./components/shared/ErrorBoundary";
// import { Root } from "./Root";

// const container = document.getElementById("app");
// if (!container) {
//   throw new Error("No container found");
// }

// const root = createRoot(container); // createRoot(container!) if you use TypeScript

// root.render(
//   <ReplicatedErrorBoundary>
//     <Root />
//   </ReplicatedErrorBoundary>
// );

ReactDOM.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
  document.getElementById("app")
);
