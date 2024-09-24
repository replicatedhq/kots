import { KotsPageTitle } from "@components/Head";
import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";

import PreflightRenderer from "@components/PreflightRenderer";
import SkipPreflightsModal from "@components/shared/modals/SkipPreflightsModal";

import PreflightsProgress from "@components/troubleshoot/PreflightsProgress";
import "../../scss/components/PreflightCheckPage.scss";

import { useGetPrelightResults, useRunPreflights } from "./hooks/index";

import { KotsParams } from "@types";
import { useUpgradeServiceContext } from "./UpgradeServiceContext";
import { isEqual } from "lodash";

const PreflightCheck = ({
  setCurrentStep,
  isConfigurable,
}: {
  setCurrentStep: (step: number) => void;
  isConfigurable: boolean;
}) => {
  const navigate = useNavigate();

  const [
    showConfirmIgnorePreflightsModal,
    setShowConfirmIgnorePreflightsModal,
  ] = useState(false);

  const {
    setIsSkipPreflights,
    setContinueWithFailedPreflights,
    prevConfig,
    config,
  } = useUpgradeServiceContext();

  const { sequence = "0", slug } = useParams<keyof KotsParams>() as KotsParams;

  const {
    mutate: runPreflights,
    error: runPreflightsError,
    isSuccess: runPreflightsSuccess,
  } = useRunPreflights({ slug, sequence });
  const { data: preflightCheck, error: getPreflightResultsError } =
    useGetPrelightResults({ slug, sequence, enabled: runPreflightsSuccess });

  if (!preflightCheck?.showPreflightCheckPending) {
    if (showConfirmIgnorePreflightsModal) {
      setShowConfirmIgnorePreflightsModal(false);
    }
  }

  useEffect(() => {
    setCurrentStep(1);
    // Config changed so we'll re-run the preflights
    if (!isEqual(prevConfig, config)) {
      runPreflights();
      return;
    }
    // No preflight results means we haven't run them yet,  let's do that
    if (
      !preflightCheck?.preflightResults ||
      preflightCheck.preflightResults.length === 0
    ) {
      runPreflights();
    }
  }, []);

  const handleIgnorePreflights = () => {
    setContinueWithFailedPreflights(false);
    setIsSkipPreflights(true);
    navigate(`/upgrade-service/app/${slug}/deploy`);
  };

  return (
    <div className="flex-column flex1 container">
      <KotsPageTitle pageName="Preflight Checks" showAppSlug />
      <div
        data-testid="preflight-check-area"
        className="PreflightChecks--wrapper flex-column u-paddingTop--30 flex1 flex tw-max-h-[60%]"
      >
        <div
          className={`u-maxWidth--full u-marginTop--20 flex-column u-position--relative card-bg ${
            preflightCheck?.showPreflightCheckPending ? "flex1" : ""
          }`}
        >
          {getPreflightResultsError?.message && (
            <div className="ErrorWrapper flex-auto flex alignItems--center u-marginBottom--20">
              <div className="icon redWarningIcon u-marginRight--10" />
              <div>
                <p className="title">Encountered an error</p>
                <p className="error">{getPreflightResultsError.message}</p>
              </div>
            </div>
          )}

          {runPreflightsError?.message && (
            <div className="ErrorWrapper flex-auto flex alignItems--center u-marginBottom--20">
              <div className="icon redWarningIcon u-marginRight--10" />
              <div>
                <p className="title">Encountered an error</p>
                <p className="error">{runPreflightsError.message}</p>
              </div>
            </div>
          )}
          <p className="u-fontSize--jumbo2 u-textColor--primary u-fontWeight--bold">
            Preflight checks
          </p>
          <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--15">
            Preflight checks validate that your cluster meets the minimum
            requirements. Required checks must pass in order to deploy the
            application. Optional checks are recommended to ensure that the
            application will work as intended.
          </p>

          {preflightCheck?.showPreflightCheckPending && (
            <div className="flex-column justifyContent--center alignItems--center flex1 u-minWidth--full tw-mt-4">
              <PreflightsProgress
                pendingPreflightCheckName={
                  preflightCheck?.pendingPreflightCheckName || ""
                }
                percentage={
                  Math.round(
                    preflightCheck?.pendingPreflightChecksPercentage
                  ) || 0
                }
              />
            </div>
          )}

          {preflightCheck?.showPreflightResults && (
            <div className="tw-mt-6">
              <div className="flex flex1 tw-justify-between tw-items-end">
                <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">
                  Results
                </p>
                {preflightCheck?.shouldShowRerunPreflight && (
                  <button
                    type="button"
                    className="btn primary blue"
                    onClick={() => runPreflights()}
                  >
                    Re-run
                  </button>
                )}
              </div>
              <div className="flex-column">
                <PreflightRenderer
                  results={preflightCheck?.preflightResults}
                  skipped={preflightCheck?.showPreflightSkipped}
                />
              </div>
            </div>
          )}

          {preflightCheck?.showIgnorePreflight && (
            <div className="flex flex0 justifyContent--center alignItems--center">
              <span
                className="u-fontSize--normal u-fontWeight--medium u-textDecoration--underline u-textColor--bodyCopy u-marginTop--15 u-cursor--pointer"
                onClick={() => setShowConfirmIgnorePreflightsModal(true)}
              >
                Ignore Preflights{" "}
              </span>
            </div>
          )}
        </div>
        <div className="tw-flex tw-justify-between tw-mt-4">
          <button
            className="btn secondary blue"
            onClick={() => navigate(`/upgrade-service/app/${slug}/config`)}
            disabled={!isConfigurable}
          >
            Back: Config
          </button>
          {!preflightCheck?.showPreflightCheckPending && (
            <button
              className="btn primary blue"
              disabled={preflightCheck?.showDeploymentBlocked}
              onClick={() => navigate(`/upgrade-service/app/${slug}/deploy`)}
            >
              Next: Confirm and deploy
            </button>
          )}
        </div>
      </div>

      {showConfirmIgnorePreflightsModal && (
        <SkipPreflightsModal
          hideSkipModal={() => setShowConfirmIgnorePreflightsModal(false)}
          onIgnorePreflightsAndDeployClick={() => {
            handleIgnorePreflights();
          }}
          showSkipModal={showConfirmIgnorePreflightsModal}
          isEmbeddedCluster={true}
        />
      )}
    </div>
  );
};

export default PreflightCheck;
