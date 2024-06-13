import { Route, Routes, Navigate } from "react-router-dom";
import { Helmet } from "react-helmet";
import NotFound from "@components/static/NotFound";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import AppConfig from "@components/upgrade_service/AppConfig";

// types
import { ToastProvider } from "@src/context/ToastContext";
import StepIndicator from "./StepIndicator";
import { useEffect, useState } from "react";

import { useLocation } from "react-router-dom";
import PreflightChecks from "./PreflightChecks";
import ConfirmAndDeploy from "./ConfirmAndDeploy";
// react-query client
const queryClient = new QueryClient();

const UpgradeService = () => {
  const Crashz = () => {
    throw new Error("Crashz!");
  };
  const [currentStep, setCurrentStep] = useState(0); // Initial step
  const location = useLocation();

  // Update currentStep based on route
  useEffect(() => {
    const newStep = {
      "/app/:slug/config": 0,
      "/app/:slug/preflight": 1,
      "/app/:slug/confirm": 2,
    }[location.pathname];

    if (typeof newStep !== "undefined") {
      setCurrentStep(newStep);
    }
    console.log(location, " location");
  }, [location]);

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
          <StepIndicator
            items={["Config", "Preflight", "Confirm"]}
            value={currentStep}
            className="tw-my-8"
          />
          <Routes>
            <Route path="/crashz" element={<Crashz />} />{" "}
            <Route path="/app/:slug/*">
              <Route index element={<Navigate to="config" />} />
              <Route path="config" element={<AppConfig />} />
              <Route path="preflight" element={<PreflightChecks />} />
              <Route path="deploy" element={<ConfirmAndDeploy />} />
            </Route>
            <Route path="*" element={<NotFound />} />
          </Routes>
        </div>
      </ToastProvider>
    </QueryClientProvider>
  );
};

export { UpgradeService };
