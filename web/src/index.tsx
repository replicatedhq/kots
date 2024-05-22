import { createRoot } from "react-dom/client";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import ReplicatedErrorBoundary from "./components/shared/ErrorBoundary";
import { Upgrader } from "@components/upgrader/Upgrader";
import { Root } from "./Root";

// scss
import "./scss/index.scss";
// tailwind
import "./index.css";

const container = document.getElementById("app");
if (!container) {
  throw new Error("No container found");
}

const root = createRoot(container); // createRoot(container!) if you use TypeScript

root.render(
  <BrowserRouter>
    <Routes>
      <Route path="/upgrader/*" element={<Upgrader />} />
      <Route path="/*"
        element={
          <ReplicatedErrorBoundary>
            <Root />
          </ReplicatedErrorBoundary>
        }
      />
    </Routes>
  </BrowserRouter>
);
