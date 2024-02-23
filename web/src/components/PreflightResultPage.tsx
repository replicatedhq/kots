import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { KotsPageTitle } from "@components/Head";
import { useParams, useLocation } from "react-router-dom";
import Modal from "react-modal";
import ReactTooltip from "react-tooltip";

import PreflightRenderer from "./PreflightRenderer";
import PreflightResultErrors from "./PreflightResultErrors";
import SkipPreflightsModal from "./shared/modals/SkipPreflightsModal";

import "../scss/components/PreflightCheckPage.scss";
import PreflightsProgress from "./troubleshoot/PreflightsProgress";

import {
  useGetPrelightResults,
  useIgnorePermissionErrors,
  useRerunPreflights,
} from "@features/PreflightChecks/api";

import { useDeployAppVersion } from "@features/App/api";

import { KotsParams } from "@types";
import Icon from "./Icon";
import { useApps } from "@features/App";

interface Props {
  fromLicenseFlow?: boolean;
  logo: string;
  refetchAppsList?: () => void;
}

function PreflightResultPage(props: Props) {
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
    useGetPrelightResults({ sequence, slug });
  const { mutate: rerunPreflights, error: rerunPreflightsError } =
    useRerunPreflights({ sequence, slug });

  // TODO: remove this once everything is using react-query
  // componentWilUnmount
  useEffect(() => {
    return () => {
      if (props.fromLicenseFlow && props.refetchAppsList) {
        props.refetchAppsList();
      }
    };
  }, []);

  if (!preflightCheck?.showPreflightCheckPending) {
    if (showConfirmIgnorePreflightsModal) {
      setShowConfirmIgnorePreflightsModal(false);
    }
  }

  const location = useLocation();
  const navigate = useNavigate();
  const { refetch: refetchApps } = useApps();

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
                logo={props.logo}
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
          {props.fromLicenseFlow && preflightCheck?.showPreflightResults && (
            <div className="flex-auto flex justifyContent--flexEnd tw-mt-6">
              <div className="flex tw-gap-6">
                {preflightCheck?.showCancelPreflight && (
                  <Link to={`/app/${slug}`}>
                    <button className="btn secondary blue">
                      Go to dashboard
                    </button>
                  </Link>
                )}
                {!preflightCheck?.showPreflightCheckPending && (
                  <>
                    <button
                      type="button"
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
                      <span
                        data-tip-disable={
                          !preflightCheck?.showDeploymentBlocked
                        }
                        data-tip="Deployment is disabled as a strict analyzer in this version's preflight checks has failed or has not been run"
                        data-for="disable-deployment-tooltip"
                      >
                        Deploy
                      </span>
                    </button>
                    <ReactTooltip
                      effect="solid"
                      id="disable-deployment-tooltip"
                    />
                  </>
                )}
              </div>
            </div>
          )}
          {props.fromLicenseFlow && preflightCheck?.showIgnorePreflight && (
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
}

export default PreflightResultPage;
