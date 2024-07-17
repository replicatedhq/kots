import { Route, Routes, Navigate } from "react-router-dom";
import { Helmet } from "react-helmet";
import NotFound from "@components/static/NotFound";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import AppConfig from "@components/upgrade_service/AppConfig";

// types
import { ToastProvider } from "@src/context/ToastContext";
import StepIndicator from "./StepIndicator";
import { useState } from "react";

import PreflightChecks from "./PreflightChecks";
import ConfirmAndDeploy from "./ConfirmAndDeploy";
import { KotsPageTitle } from "@components/Head";
import { UpgradeServiceProvider } from "./UpgradeServiceContext";
// react-query client
const queryClient = new QueryClient();

const UpgradeService = () => {
  const Crashz = () => {
    throw new Error("Crashz!");
  };
  const [currentStep, setCurrentStep] = useState(0);

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
      <UpgradeServiceProvider>
        <ToastProvider>
          <div className="flex1 flex-column u-overflow--auto tw-relative">
            <KotsPageTitle pageName={`Deploy`} showAppSlug />{" "}
            <StepIndicator
              items={["Config", "Preflight", "Confirm"]}
              value={currentStep}
              className="tw-my-8"
            />
            <Routes>
              <Route path="/crashz" element={<Crashz />} />{" "}
              <Route path="/app/:slug/*">
                <Route index element={<Navigate to="config" />} />
                <Route
                  path="config"
                  element={<AppConfig setCurrentStep={setCurrentStep} />}
                />
                <Route
                  path="preflight"
                  element={<PreflightChecks setCurrentStep={setCurrentStep} />}
                />
                <Route
                  path="deploy"
                  element={<ConfirmAndDeploy setCurrentStep={setCurrentStep} />}
                />
              </Route>
              <Route path="*" element={<NotFound />} />
            </Routes>
          </div>
        </ToastProvider>
      </UpgradeServiceProvider>
    </QueryClientProvider>
  );
};

export { UpgradeService };
