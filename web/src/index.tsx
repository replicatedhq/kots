import { createRoot } from "react-dom/client";
import { BrowserRouter, Route, Routes } from "react-router-dom";
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
      <Routes>
        <Route
          path="/upgrader"
          element={
            <div style={{ height: "100vh", width: "100vw", background: "white" }}>
              <h1 style={{ color: "black" }}>
                Hello from KOTS Upgrader!
              </h1>
            </div>
          }
        />
        <Route path="/*" element={<Root />}/>
      </Routes>
    </BrowserRouter>
  </ReplicatedErrorBoundary>
);
