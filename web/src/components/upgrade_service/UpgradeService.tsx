import { Route, Routes, Navigate } from "react-router-dom";
import { Helmet } from "react-helmet";
import NotFound from "@components/static/NotFound";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import AppConfig from "@components/upgrade_service/AppConfig";

// types
import { ToastProvider } from "@src/context/ToastContext";

// react-query client
const queryClient = new QueryClient();

const Upgrader = () => {
  const Crashz = () => {
    throw new Error("Crashz!");
  };

  return (
    <QueryClientProvider client={queryClient}>
      <Helmet>
        <meta
          httpEquiv="Cache-Control"
          content="no-cache, no-store, must-revalidate"
        />
        <meta httpEquiv="Pragma" content="no-cache" />
        <meta httpEquiv="Expires" content="0" />
      </Helmet>
      <ToastProvider>
        <div className="flex1 flex-column u-overflow--auto tw-relative">
          <Routes>
            <Route path="/crashz" element={<Crashz />} />{" "}
            <Route path="/app/:slug/*">
              <Route index element={<Navigate to="config" />} />
              <Route path="config" element={<AppConfig />} />
            </Route>
            <Route path="*" element={<NotFound />} />
          </Routes>
        </div>
      </ToastProvider>
    </QueryClientProvider>
  );
};

export { Upgrader };
