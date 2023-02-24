import React, { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { KotsPageTitle } from "@components/Head";
import { useHistory, useParams } from "react-router-dom";
import Modal from "react-modal";
import ReactTooltip from "react-tooltip";

import PreflightRenderer from "./PreflightRenderer";
import PreflightResultErrors from "./PreflightResultErrors";
import SkipPreflightsModal from "./shared/modals/SkipPreflightsModal";

import "../scss/components/PreflightCheckPage.scss";
import PreflightsProgress from "./troubleshoot/PreflightsProgress";
import Icon from "./Icon";

import {
  useGetPrelightResults,
  useIgnorePermissionErrors,
  useRerunPreflights,
} from "@features/PreflightChecks/api";

import { useDeployAppVersion } from "@features/App/api";

import { KotsParams } from "@types";

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

  const history = useHistory();
  const { sequence = "0", slug } = useParams<KotsParams>();
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

  return (
    <div className="flex-column flex1 container">
      <KotsPageTitle pageName="Preflight Checks" showAppSlug />
      <div className="flex1 flex u-overflow--auto">
        <div className="PreflightChecks--wrapper flex1 flex-column u-paddingTop--30">
          {history.location.pathname.includes("version-history") && (
            <div
              className="u-fontWeight--bold link"
              onClick={() => history.goBack()}
            >
              <Icon
                icon="prev-arrow"
                size={12}
                className="clickable u-marginRight--10"
                style={{ verticalAlign: "0" }}
              />
              Back
            </div>
          )}
          <div className="u-minWidth--full u-marginTop--20 flex-column flex1 u-position--relative">
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
              Preflight checks validate that your cluster will meet the minimum
              requirements. If your cluster does not meet the requirements your
              application might not work properly. Some checks may be required
              which means your application will not be able to be deployed until
              they pass. Optional checks are recommended to ensure that the
              application you are installing will work as intended.
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
              <PreflightResultErrors
                errors={preflightCheck.errors}
                ignorePermissionErrors={ignorePermissionErrors}
                logo={props.logo}
                preflightResultData={preflightCheck.preflightResults}
                showRbacError={preflightCheck.showRbacError}
              />
            )}
            {preflightCheck?.showPreflightResults && (
              <div className="dashboard-card">
                <div className="flex flex1 justifyContent--spaceBetween alignItems--center">
                  <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">
                    Results from your preflight checks
                  </p>
                  <div className="flex alignItems--center">
                    {props.fromLicenseFlow &&
                    preflightCheck?.showCancelPreflight ? (
                      <div className="flex alignItems--center">
                        <div className="flex alignItems--center u-marginRight--20">
                          <Link
                            to={`/app/${slug}`}
                            className="u-textColor--error u-textDecoration--underlineOnHover u-fontWeight--medium u-fontSize--small"
                          >
                            Cancel
                          </Link>
                        </div>
                      </div>
                    ) : null}
                  </div>
                </div>
                <div className="flex-column">
                  <PreflightRenderer
                    results={preflightCheck?.preflightResults}
                    skipped={preflightCheck?.showPreflightSkipped}
                  />
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {props.fromLicenseFlow && (
        <div className="flex-auto flex justifyContent--flexEnd u-marginBottom--15">
          {!preflightCheck?.showPreflightCheckPending && (
            <div>
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
                  data-tip-disable={!preflightCheck?.showDeploymentBlocked}
                  data-tip="Deployment is disabled as a strict analyzer in this version's preflight checks has failed or has not been run"
                  data-for="disable-deployment-tooltip"
                >
                  Continue
                </span>
              </button>
              <ReactTooltip effect="solid" id="disable-deployment-tooltip" />
            </div>
          )}
          {preflightCheck?.showIgnorePreflight && (
            <div className="flex flex1 justifyContent--center alignItems--center">
              <span
                className="u-fontSize--normal u-fontWeight--medium u-textDecoration--underline u-textColor--bodyCopy u-marginTop--15 u-cursor--pointer"
                onClick={() => setShowConfirmIgnorePreflightsModal(true)}
              >
                Ignore Preflights{" "}
              </span>
            </div>
          )}
        </div>
      )}
      {preflightCheck?.shouldShowRerunPreflight &&
        preflightCheck?.showPreflightCheckPending && (
          <div className="flex-auto flex justifyContent--flexEnd u-marginBottom--15">
            <button
              type="button"
              className="btn primary blue"
              onClick={() => rerunPreflights()}
            >
              Re-run
            </button>
          </div>
        )}

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
        <div className="Modal-body">
          <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
            Preflight is showing some issues, are you sure you want to continue?
          </p>
          <div className="u-marginTop--10 flex justifyContent--flexEnd">
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
              }}
            >
              Deploy and continue
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

export default PreflightResultPage;
