import { Route, Routes, Navigate, useMatch } from "react-router-dom";
import { Helmet } from "react-helmet";
import NotFound from "@components/static/NotFound";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import AppConfig from "@components/upgrade_service/AppConfig";

// types
import { ToastProvider } from "@src/context/ToastContext";
import StepIndicator from "./StepIndicator";
import { useEffect, useState } from "react";

import PreflightChecks from "./PreflightChecks";
import ConfirmAndDeploy from "./ConfirmAndDeploy";
import { KotsPageTitle } from "@components/Head";
import { UpgradeServiceProvider } from "./UpgradeServiceContext";
import Loader from "@components/shared/Loader";
// react-query client
const queryClient = new QueryClient();

const UpgradeService = () => {
  const Crashz = () => {
    throw new Error("Crashz!");
  };
  const [currentStep, setCurrentStep] = useState(0);
  const [isConfigurable, setIsConfigurable] = useState(false);
  const [hasPreflight, setHasPreflight] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const { params } = useMatch("/upgrade-service/app/:slug/*");

  useEffect(() => {
    fetch(`${process.env.API_ENDPOINT}/upgrade-service/app/${params.slug}`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
    })
      .then(async (response) => {
        if (!response.ok) {
          const res = await response.json();
          throw new Error(res.error);
        }
        const data = (await response.json()) as {
          isConfigurable: boolean;
          hasPreflight: boolean;
        };
        console.log("Configuration check");
        console.log(data);
        setHasPreflight(data.hasPreflight);
        setIsConfigurable(data.isConfigurable);
        setIsLoading(false);
      })
      .catch((err) => {
        // TODO handle error
      });
  }, []);

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
            {isLoading ? (
              <div className="tw-absolute tw-top-[44.3%] tw-w-full flex-column flex1 alignItems--center justifyContent--center tw-gap-4">
                <span className="u-fontWeight--bold">
                  Checking required steps...
                </span>
                <Loader size="60" />
              </div>
            ) : (
              <Routes>
                <Route path="/crashz" element={<Crashz />} />
                <Route path="/app/:slug/*">
                  <Route index element={<Navigate to="config" />} />
                  <Route
                    path="config"
                    element={
                      isConfigurable ? (
                        <AppConfig setCurrentStep={setCurrentStep} />
                      ) : (
                        <Navigate to="../preflight" />
                      )
                    }
                  />
                  <Route
                    path="preflight"
                    element={
                      hasPreflight ? (
                        <PreflightChecks setCurrentStep={setCurrentStep} />
                      ) : (
                        <Navigate to="../deploy" />
                      )
                    }
                  />
                  <Route
                    path="deploy"
                    element={
                      <ConfirmAndDeploy setCurrentStep={setCurrentStep} />
                    }
                  />
                </Route>
                <Route path="*" element={<NotFound />} />
              </Routes>
            )}
          </div>
        </ToastProvider>
      </UpgradeServiceProvider>
    </QueryClientProvider>
  );
};

export { UpgradeService };
