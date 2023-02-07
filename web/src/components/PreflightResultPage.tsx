import React, { useEffect, useReducer } from "react";
import { Link } from "react-router-dom";
import { KotsPageTitle } from "@components/Head";
import { useHistory, useParams } from "react-router-dom";
import Modal from "react-modal";
import ReactTooltip from "react-tooltip";

import PreflightRenderer from "./PreflightRenderer";
import PreflightResultErrors from "./PreflightResultErrors";
import SkipPreflightsModal from "./shared/modals/SkipPreflightsModal.tsx";

import { Utilities } from "../utilities/utilities";
import "../scss/components/PreflightCheckPage.scss";
import PreflightsProgress from "./troubleshoot/PreflightsProgress";
import Icon from "./Icon";

import {
  useDeployKotsDownsteam,
  useGetPrelightResults,
  useRerunPreflights
} from "@features/PreflightChecks/api";


import {
  KotsParams,
  PreflightProgress,
} from "@types";

interface Props {
  fromLicenseFlow?: boolean;
  logo: string;
  refetchAppsList?: () => void;
};

interface State {
  preflightCurrentStatus?: PreflightProgress | null;
  errorMessage?: string;
  preflightResultCheckCount: number;
  showSkipModal: boolean;
  showWarningModal: boolean;
};

// class PreflightResultPage extends Component<Props, State> {
function PreflightResultPage(props: Props) {

  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }), {
    preflightResultCheckCount: 0,
    showSkipModal: false,
    showWarningModal: false,
  });

  const history = useHistory();
  const {
    sequence = "0",
    slug
  } = useParams<KotsParams>();
  const { data: preflightCheck } = useGetPrelightResults({ slug, sequence });
  const { mutate: rerunPreflights } = useRerunPreflights({ slug, sequence });
  const { mutate: deployKotsDownstream } =
    useDeployKotsDownsteam({ slug, sequence });

  // TODO: remove this once everything is using react-query
  // componentWilUnmount
  useEffect(() => {
    return () => {
      if (props.fromLicenseFlow && props.refetchAppsList) {
        props.refetchAppsList();
      }
    };
  }, []);

  const showConfirmSkipPreflightsModal = () => {
    setState({
      showWarningModal: true,
    });
  }

  const showSkipModal = () => {
    setState({
      showSkipModal: true,
    });
  };

  const hideSkipModal = () => {
    setState({
      showSkipModal: false,
    });
  };

  const hideWarningModal = () => {
    setState({
      showWarningModal: false,
    });
  };

  const ignorePermissionErrors = () => {
    setState({ errorMessage: "" });

    const { slug } = props.match.params;
    const sequence = props.match.params.sequence
      ? parseInt(props.match.params.sequence, 10)
      : 0;

    fetch(
      `${process.env.API_ENDPOINT}/app/${slug}/sequence/${sequence}/preflight/ignore-rbac`,
      {
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
          Authorization: Utilities.getToken(),
        },
        method: "POST",
      }
    )
      .then(async () => {
        setState({
          preflightResultData: null,
        });
        // state.getKotsPreflightResultJob.start(
        //   this.getKotsPreflightResult,
        //   1000
        // );
      })
      .catch((err) => {
        console.log(err);
        setState({
          errorMessage: err
            ? `Encountered an error while trying to ignore permissions: ${err.message}`
            : "Something went wrong, please try again.",
        });
      });
  };

  if (preflightCheck?.preflightResults) {
    if (state.showSkipModal) {
      hideSkipModal();
    }
  }

  return (
    <div className="flex-column flex1 container">
      <KotsPageTitle pageName="Preflight Checks" showAppSlug />
      <div className="flex1 flex u-overflow--auto">
        <div className="PreflightChecks--wrapper flex1 flex-column u-paddingTop--30">
          {history.location.pathname.includes(
            "version-history"
          ) && (
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
            {state.errorMessage && state.errorMessage.length > 0 ? (
              <div className="ErrorWrapper flex-auto flex alignItems--center u-marginBottom--20">
                <div className="icon redWarningIcon u-marginRight--10" />
                <div>
                  <p className="title">Encountered an error</p>
                  <p className="error">{state.errorMessage}</p>
                </div>
              </div>
            ) : null}
            <p className="u-fontSize--jumbo2 u-textColor--primary u-fontWeight--bold">
              Preflight checks
            </p>
            <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--15">
              Preflight checks validate that your cluster will meet the
              minimum requirements. If your cluster does not meet the
              requirements your application might not work properly. Some
              checks may be required which means your application will not be
              able to be deployed until they pass. Optional checks are
              recommended to ensure that the application you are installing
              will work as intended.
            </p>
            {!preflightCheck?.showPreflightCheckPending && (
              <div className="flex-column justifyContent--center alignItems--center flex1 u-minWidth--full">
                <PreflightsProgress
                  progressData={state.preflightCurrentStatus}
                  preflightResultCheckCount={
                    state.preflightResultCheckCount
                  }
                />
              </div>
            )}
            {preflightCheck?.errors && (
              <PreflightResultErrors
                errors={preflightCheck.errors}
                ignorePermissionErrors={ignorePermissionErrors}
                logo={props.logo}
                preflightResultData={preflightCheck.preflightResults}
                showRbacError={preflightCheck.showRbacError}
              />
            )}
            {preflightCheck?.showPreflightCheckPending &&
              !preflightCheck?.errors && (
                <div className="dashboard-card">
                  <div className="flex flex1 justifyContent--spaceBetween alignItems--center">
                    <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">
                      Results from your preflight checks
                    </p>
                    <div className="flex alignItems--center">
                      {props.fromLicenseFlow &&
                        // stopPolling &&
                        // hasResult &&
                        // preflightState !== "pass" ? (
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
                      // results={state.preflightResultData?.result}
                      // skipped={preflightSkipped}
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
                onClick={() => showConfirmSkipPreflightsModal()}
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
          {preflightCheck?.showDeploymentBlocked && (
            <div className="flex flex1 justifyContent--center alignItems--center">
              <span
                className="u-fontSize--normal u-fontWeight--medium u-textDecoration--underline u-textColor--bodyCopy u-marginTop--15 u-cursor--pointer"
                onClick={showSkipModal}
              >
                Ignore Preflights{" "}
              </span>
            </div>
          )}
        </div>
      )}
      {!props.fromLicenseFlow && preflightCheck?.showPreflightCheckPending && (
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

      {state.showSkipModal && (
        <SkipPreflightsModal
          hideSkipModal={hideSkipModal}
          onIgnorePreflightsAndDeployClick={() => {
            deployKotsDownstream({
              continueWithFailedPreflights: false,
              isSkipPreflights: true,
            });
          }}
          showSkipModal={state.showSkipModal}
        />
      )}

      <Modal
        isOpen={state.showWarningModal}
        onRequestClose={hideWarningModal}
        shouldReturnFocusAfterClose={false}
        contentLabel="Preflight shows some issues"
        ariaHideApp={false}
        className="Modal"
      >
        <div className="Modal-body">
          <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
            Preflight is showing some issues, are you sure you want to
            continue?
          </p>
          <div className="u-marginTop--10 flex justifyContent--flexEnd">
            <button
              type="button"
              className="btn secondary"
              onClick={hideWarningModal}
            >
              Close
            </button>
            <button
              type="button"
              className="btn blue primary u-marginLeft--10"
              onClick={() => {
                hideWarningModal();
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

// export default withRouter(PreflightResultPage) as any;
export default PreflightResultPage;
