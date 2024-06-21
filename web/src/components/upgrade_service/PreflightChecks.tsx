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
  useRerunPreflights,
} from "@features/PreflightChecks/api";

import { useDeployAppVersion } from "@features/App/api";
import Icon from "@components/Icon";
import { KotsParams } from "@types";

const PreflightCheck = ({
  setCurrentStep,
}: {
  setCurrentStep: (step: number) => void;
}) => {
  const navigate = useNavigate();
  const [
    showContinueWithFailedPreflightsModal,
    setShowContinueWithFailedPreflightsModal,
  ] = useState(false);
  const [
    showConfirmIgnorePreflightsModal,
    setShowConfirmIgnorePreflightsModal,
  ] = useState(false);

  const { sequence = "0", slug } = useParams<keyof KotsParams>() as KotsParams;
  const { mutate: deployKotsDownstream } = useDeployAppVersion({
    slug,
    sequence,
  });
  const { mutate: ignorePermissionErrors, error: ignorePermissionError } =
    useIgnorePermissionErrors({ sequence, slug });
  const { data: preflightCheck, error: getPreflightResultsError } =
    useGetPrelightResults({ slug, sequence, isUpgradeService: true });
  const { mutate: rerunPreflights, error: rerunPreflightsError } =
    useRerunPreflights({ slug, sequence, isUpgradeService: true });

  if (!preflightCheck?.showPreflightCheckPending) {
    if (showConfirmIgnorePreflightsModal) {
      setShowConfirmIgnorePreflightsModal(false);
    }
  }

  const { refetch: refetchApps } = useApps();
  const location = useLocation();

  useEffect(() => {
    setCurrentStep(1);
  }, []);

  return (
    <div className="flex-column flex1 container">
      <KotsPageTitle pageName="Preflight Checks" showAppSlug />
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
          {rerunPreflightsError?.message && (
            <div className="ErrorWrapper flex-auto flex alignItems--center u-marginBottom--20">
              <div className="icon redWarningIcon u-marginRight--10" />
              <div>
                <p className="title">Encountered an error</p>
                <p className="error">{rerunPreflightsError.message}</p>
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

          {preflightCheck?.showPreflightResultErrors && (
            <>
              <PreflightResultErrors
                errors={preflightCheck.errors}
                ignorePermissionErrors={ignorePermissionErrors}
                logo={""}
                preflightResultData={preflightCheck.preflightResults}
                showRbacError={preflightCheck.showRbacError}
              />
              <div className="flex justifyContent--flexEnd tw-gap-6">
                <button
                  className="btn primary blue"
                  onClick={() => ignorePermissionErrors()}
                >
                  {!location.pathname.includes("version-history")
                    ? "Proceed"
                    : "Re-run"}{" "}
                  with limited Preflights
                </button>
              </div>
            </>
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
                    onClick={() => rerunPreflights()}
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
          >
            Back: Config
          </button>
          {!preflightCheck?.showPreflightCheckPending && (
            <button
              className="btn primary blue"
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

export default PreflightCheck;
