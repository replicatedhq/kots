import { Route, Routes, Navigate, useMatch } from "react-router-dom";
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
import Loader from "@components/shared/Loader";
import { useGetUpgradeInfo } from "./hooks";
// react-query client
const queryClient = new QueryClient();

const UpgradeServiceBody = () => {
  const Crashz = () => {
    throw new Error("Crashz!");
  };
  const [currentStep, setCurrentStep] = useState(0);

  const { params } = useMatch("/upgrade-service/app/:slug/*");
  const {
    data: upgradeInfo,
    error: getUpgradeInfoError,
    isError,
    isLoading,
    isSuccess,
  } = useGetUpgradeInfo({ slug: params.slug });

  return (
    <UpgradeServiceProvider>
      <ToastProvider>
        <div className="flex1 flex-column u-overflow--auto tw-relative">
          <KotsPageTitle pageName={`Deploy`} showAppSlug />{" "}
          <StepIndicator
            items={["Config", "Preflight", "Confirm"]}
            value={currentStep}
            className="tw-my-8"
          />
          {isError && (
            <div className="ErrorWrapper flex-auto flex alignItems--center u-marginBottom--20">
              <div className="icon redWarningIcon u-marginRight--10" />
              <div>
                <p className="title">Encountered an error</p>
                <p className="error">{getUpgradeInfoError.message}</p>
              </div>
            </div>
          )}
          {isLoading && (
            <div className="tw-absolute tw-top-[44.3%] tw-w-full flex-column flex1 alignItems--center justifyContent--center tw-gap-4">
              <span className="u-fontWeight--bold">
                Checking required steps...
              </span>
              <Loader size="60" />
            </div>
          )}
          {isSuccess && (
            <Routes>
              <Route path="/crashz" element={<Crashz />} />
              <Route path="/app/:slug/*">
                <Route index element={<Navigate to="config" />} />
                <Route
                  path="config"
                  element={
                    upgradeInfo?.isConfigurable ? (
                      <AppConfig setCurrentStep={setCurrentStep} />
                    ) : (
                      <Navigate to="../preflight" />
                    )
                  }
                />
                <Route
                  path="preflight"
                  element={
                    upgradeInfo?.hasPreflight ? (
                      <PreflightChecks
                        setCurrentStep={setCurrentStep}
                        isConfigurable={upgradeInfo?.isConfigurable}
                      />
                    ) : (
                      <Navigate to="../deploy" />
                    )
                  }
                />
                <Route
                  path="deploy"
                  element={
                    <ConfirmAndDeploy
                      isConfigurable={upgradeInfo?.isConfigurable}
                      hasPreflight={upgradeInfo?.hasPreflight}
                      setCurrentStep={setCurrentStep}
                      isEC2Install={upgradeInfo?.isEC2Install}
                    />
                  }
                />
              </Route>
              <Route path="*" element={<NotFound />} />
            </Routes>
          )}
        </div>
      </ToastProvider>
    </UpgradeServiceProvider>
  );
};

const UpgradeService = () => {
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
      <UpgradeServiceBody />
    </QueryClientProvider>
  );
};

export { UpgradeService };
