import { KotsPageTitle } from "@components/Head";
import { useEffect, useState } from "react";
import { useParams, useLocation, useNavigate } from "react-router-dom";
import Modal from "react-modal";

import PreflightRenderer from "@components/PreflightRenderer";
import PreflightResultErrors from "@components/PreflightResultErrors";
import SkipPreflightsModal from "@components/shared/modals/SkipPreflightsModal";

import PreflightsProgress from "@components/troubleshoot/PreflightsProgress";
import "../../scss/components/PreflightCheckPage.scss";

import { useApps } from "@features/App";

import {
  useGetPrelightResults,
  useIgnorePermissionErrors,
} from "@features/PreflightChecks/api";

import { useDeployAppVersion } from "@features/App/api";
import Icon from "@components/Icon";
import Markdown from "react-remarkable";
import { useUpgradeServiceContext } from "./UpgradeServiceContext";

const ConfirmAndDeploy = ({ setCurrentStep }) => {
  useEffect(() => {
    setCurrentStep(2);
  }, []);
  const navigate = useNavigate();
  const [
    showContinueWithFailedPreflightsModal,
    setShowContinueWithFailedPreflightsModal,
  ] = useState(false);
  const [
    showConfirmIgnorePreflightsModal,
    setShowConfirmIgnorePreflightsModal,
  ] = useState(false);

  const { sequence = "0", slug } = useParams();
  const { mutate: deployKotsDownstream } = useDeployAppVersion({
    slug,
    sequence,
  });
  const { mutate: ignorePermissionErrors, error: ignorePermissionError } =
    useIgnorePermissionErrors({ sequence, slug });
  const { data: preflightCheck, error: getPreflightResultsError } =
    useGetPrelightResults({ sequence, slug });

  if (!preflightCheck?.showPreflightCheckPending) {
    if (showConfirmIgnorePreflightsModal) {
      setShowConfirmIgnorePreflightsModal(false);
    }
  }

  const PreflightResult = ({ results }) => {
    console.log(results, "res");
    function hasAllPassed(data) {
      return data.every((item) => item.showPass);
    }

    function hasWarning(data) {
      return data.some((item) => item.showWarn);
    }
    function hasFailed(data) {
      return data.some((item) => item.showFail);
    }

    const warnings = results.filter((result) => result.showWarn);
    const errors = results.filter((result) => result.showFail);

    // go through and find out if there are warnings
    if (hasAllPassed(results)) {
      return (
        <div className="flex justifyContent--space-between preflight-check-row tw-my-2 tw-py-2">
          <Icon
            className="success-color"
            icon="check-circle-filled"
            size={16}
          />
          <div className="u-textColor--primary u-fontWeight--bold u-fontSize--large tw-ml-2">
            All preflight checks passed
          </div>
        </div>
      );
    } else if (hasFailed(results)) {
      return (
        <div>
          <div className="tw-flex tw-my-2 tw-py-2">
            <Icon
              className="error-color"
              icon="warning-circle-filled"
              size={16}
            />
            <div className="u-textColor--error u-fontWeight--bold u-fontSize--large tw-ml-2">
              Preflight checks failed
            </div>
          </div>
          {errors.map((error, i) => {
            return (
              <div className="flex justifyContent--space-between preflight-check-row tw-my-2 tw-py-2">
                <div className="flex1">
                  <p className="u-textColor--primary u-fontSize--large u-fontWeight--bold">
                    {error.title}
                  </p>
                  <div className="PreflightMessageRow u-marginTop--10">
                    <Markdown source={error.message} />
                  </div>
                  {error.showCannotFail && (
                    <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-marginTop--10">
                      To deploy the application, this check cannot fail.
                    </p>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      );
    } else if (hasWarning(results)) {
      return (
        <div>
          <div className="tw-flex tw-my-2 tw-py-2">
            <Icon className="warning-color" icon="warning" size={16} />
            <div className="u-textColor--warning u-fontWeight--bold u-fontSize--large tw-ml-2">
              Preflight checks passed with warnings
            </div>
          </div>
          {warnings.map((warning, i) => {
            return (
              <div className="flex justifyContent--space-between preflight-check-row tw-my-2 tw-py-2">
                <div className="flex1">
                  <p className="u-textColor--primary u-fontSize--large u-fontWeight--bold">
                    {warning.title}
                  </p>
                  <div className="PreflightMessageRow u-marginTop--10">
                    <Markdown source={warning.message} />
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      );
    }
    // go thorugh and find out if there are errors

    // if there aren't, then show the success message
    return <div></div>;
  };
  const location = useLocation();

  const { refetch: refetchApps } = useApps();
  const { numberOfConfigChanges } = useUpgradeServiceContext();

  return (
    <div className="flex-column flex1 container">
      <KotsPageTitle pageName="Confirm and Deploy" showAppSlug />
      <div className="PreflightChecks--wrapper flex-column u-paddingTop--30 flex1 flex u-overflow--auto">
        {location.pathname.includes("version-history") && (
          <div className="u-fontWeight--bold link" onClick={() => navigate(-1)}>
            <Icon
              icon="prev-arrow"
              size={12}
              className="clickable u-marginRight--10"
              style={{ verticalAlign: "0" }}
            />
            Back
          </div>
        )}
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
          {ignorePermissionError?.message && (
            <div className="ErrorWrapper flex-auto flex alignItems--center u-marginBottom--20">
              <div className="icon redWarningIcon u-marginRight--10" />
              <div>
                <p className="title">Encountered an error</p>
                <p className="error">{ignorePermissionError.message}</p>
              </div>
            </div>
          )}

          <p className="u-fontSize--jumbo2 u-textColor--primary u-fontWeight--bold">
            Confirm and Deploy
          </p>

          {preflightCheck?.showPreflightCheckPending && (
            <div className="flex-column justifyContent--center alignItems--center flex1 u-minWidth--full">
              <PreflightsProgress
                pendingPreflightCheckName={
                  preflightCheck?.pendingPreflightCheckName || ""
                }
                percentage={
                  preflightCheck?.pendingPreflightChecksPercentage || 0
                }
              />
            </div>
          )}
          <div className="tw-mt-6">
            <div className="flex flex1 tw-justify-between tw-items-end">
              <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">
                Config
              </p>
            </div>
            {/* TODO: WILL NEED TO UPDATE THIS BASED ON THE RESPONSE FROM API */}
            <div className="flex justifyContent--space-between preflight-check-row tw-my-2 tw-py-2">
              <Icon
                className="success-color"
                icon="check-circle-filled"
                size={16}
              />
              <div className="u-textColor--primary u-fontWeight--bold u-fontSize--large tw-ml-2">
                {numberOfConfigChanges} value changed. No errors detected.
              </div>
            </div>
          </div>

          {preflightCheck?.showPreflightResults && (
            <div className="tw-mt-6">
              <div className="flex flex1 tw-justify-between tw-items-end">
                <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">
                  Preflight checks
                </p>
              </div>
              <div className="flex-column">
                <PreflightResult results={preflightCheck?.preflightResults} />
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
            onClick={() => navigate(`/upgrade-service/app/${slug}/preflight`)}
          >
            Back: Preflight checks
          </button>
          <button
            className="btn primary blue"
            disabled={preflightCheck?.showDeploymentBlocked}
            onClick={() =>
              preflightCheck?.shouldShowConfirmContinueWithFailedPreflights
                ? setShowContinueWithFailedPreflightsModal(true)
                : deployKotsDownstream({
                    continueWithFailedPreflights: true,
                  })
            }
          >
            Deploy
          </button>
        </div>
      </div>

      {showConfirmIgnorePreflightsModal && (
        <SkipPreflightsModal
          hideSkipModal={() => setShowConfirmIgnorePreflightsModal(false)}
          onIgnorePreflightsAndDeployClick={() => {
            deployKotsDownstream({
              continueWithFailedPreflights: false,
              isSkipPreflights: true,
            });
          }}
          showSkipModal={showConfirmIgnorePreflightsModal}
        />
      )}

      <Modal
        isOpen={showContinueWithFailedPreflightsModal}
        onRequestClose={() => setShowContinueWithFailedPreflightsModal(false)}
        shouldReturnFocusAfterClose={false}
        contentLabel="Preflight shows some issues"
        ariaHideApp={false}
        className="Modal"
      >
        <div className="Modal-body tw-w-[300px]">
          <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20 tw-text-center">
            Some preflight checks did not pass. <br /> Are you sure you want to
            deploy?
          </p>
          <div className="u-marginTop--10 flex tw-justify-center">
            <button
              type="button"
              className="btn secondary"
              onClick={() => setShowContinueWithFailedPreflightsModal(false)}
            >
              Close
            </button>
            <button
              type="button"
              className="btn blue primary u-marginLeft--10"
              onClick={() => {
                setShowContinueWithFailedPreflightsModal(false);
                deployKotsDownstream({ continueWithFailedPreflights: true });
                refetchApps();
              }}
            >
              Deploy anyway
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default ConfirmAndDeploy;
