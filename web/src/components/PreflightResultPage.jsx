import React, { Component } from "react";
import { Link } from "react-router-dom";
import { Helmet } from "react-helmet";
import { withRouter } from "react-router-dom";
import Modal from "react-modal";
import size from "lodash/size";
import ReactTooltip from "react-tooltip";

import PreflightRenderer from "./PreflightRenderer";
import PreflightResultErrors from "./PreflightResultErrors";
import SkipPreflightsModal from "./shared/modals/SkipPreflightsModal";

import { getPreflightResultState, Utilities } from "../utilities/utilities";
import { Repeater } from "../utilities/repeater";
import "../scss/components/PreflightCheckPage.scss";
import PreflightsProgress from "./troubleshoot/PreflightsProgress";

class PreflightResultPage extends Component {
  state = {
    showSkipModal: false,
    showWarningModal: false,
    getKotsPreflightResultJob: new Repeater(),
    preflightResultData: null,
    errorMessage: "",
    preflightResultCheckCount: 0,
  };

  componentDidMount() {
    this.state.getKotsPreflightResultJob.start(
      this.getKotsPreflightResult,
      1000,
    );
  }

  async componentWillUnmount() {
    this.state.getKotsPreflightResultJob.stop();
    if (this.props.fromLicenseFlow && this.props.refetchAppsList) {
      await this.props.refetchAppsList();
    }
  }

  deployKotsDownstream = async (
    continueWithFailedPreflights = false,
    isSkipPreflights = false,
  ) => {
    this.setState({ errorMessage: "" });
    try {
      const { history, match } = this.props;
      const { slug } = match.params;
      const { preflightResultData } = this.state;

      if (!isSkipPreflights) {
        const preflightResults = JSON.parse(preflightResultData?.result);
        const preflightState = getPreflightResultState(preflightResults);
        if (preflightState !== "pass") {
          if (!continueWithFailedPreflights) {
            this.showWarningModal();
            return;
          }
        }
      }

      const sequence = match.params.sequence
        ? parseInt(match.params.sequence, 10)
        : 0;
      await fetch(
        `${process.env.API_ENDPOINT}/app/${slug}/sequence/${sequence}/deploy`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "POST",
          body: JSON.stringify({
            isSkipPreflights: isSkipPreflights,
            continueWithFailedPreflights: !!continueWithFailedPreflights,
          }),
        },
      );

      history.push(`/app/${slug}`);
    } catch (err) {
      console.log(err);
      this.setState({
        errorMessage: err
          ? `Encountered an error while trying to deploy downstream version: ${err.message}`
          : "Something went wrong, please try again.",
      });
    }
  };

  showSkipModal = () => {
    this.setState({
      showSkipModal: true,
    });
  };

  hideSkipModal = () => {
    this.setState({
      showSkipModal: false,
    });
  };

  showWarningModal = () => {
    this.setState({
      showWarningModal: true,
    });
  };

  hideWarningModal = () => {
    this.setState({
      showWarningModal: false,
    });
  };

  ignorePermissionErrors = () => {
    this.setState({ errorMessage: "" });

    const { slug } = this.props.match.params;
    const sequence = this.props.match.params.sequence
      ? parseInt(this.props.match.params.sequence, 10)
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
      },
    )
      .then(async (res) => {
        this.setState({
          preflightResultData: null,
        });
        this.state.getKotsPreflightResultJob.start(
          this.getKotsPreflightResult,
          1000,
        );
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          errorMessage: err
            ? `Encountered an error while trying to ignore permissions: ${err.message}`
            : "Something went wrong, please try again.",
        });
      });
  };

  rerunPreflights = () => {
    const { slug } = this.props.match.params;

    this.setState({ errorMessage: "" });
    const sequence = this.props.match.params.sequence
      ? parseInt(this.props.match.params.sequence, 10)
      : 0;

    fetch(
      `${process.env.API_ENDPOINT}/app/${slug}/sequence/${sequence}/preflight/run`,
      {
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
          Authorization: Utilities.getToken(),
        },
        method: "POST",
      },
    )
      .then((res) => {
        if (res.status === 200) {
          this.setState({
            preflightResultData: null,
          });
          this.state.getKotsPreflightResultJob.start(
            this.getKotsPreflightResult,
            1000,
          );
        } else {
          this.setState({
            errorMessage: `Encountered an error while trying to re-run preflight checks: Status ${res.status}`,
          });
        }
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          errorMessage: err
            ? `Encountered an error while trying to re-run preflight checks: ${err.message}`
            : "Something went wrong, please try again.",
        });
      });
  };

  renderErrors = (errors) => {
    const { preflightResultData } = this.state;

    // TODO: why start polling here?
    // this.state.getKotsPreflightResultJob.start(this.getKotsPreflightResult, 2000);

    const valueFromAPI = errors
      .map((error) => {
        return error.error;
      })
      .join("\n");

    return (
      <PreflightResultErrors
        valueFromAPI={valueFromAPI}
        ignorePermissionErrors={this.ignorePermissionErrors}
        logo={this.props.logo}
        preflightResultData={preflightResultData}
        appSlug={this.props.match.params.slug}
      />
    );
  };

  getKotsPreflightResult = async () => {
    this.setState({ errorMessage: "" });
    const { match } = this.props;
    if (match.params.downstreamSlug) {
      // why?
      const sequence = match.params.sequence
        ? parseInt(match.params.sequence, 10)
        : 0;
      return this.getKotsPreflightResultForSequence(
        match.params.slug,
        sequence,
      );
    }
    return this.getLatestKotsPreflightResult();
  };

  getKotsPreflightResultForSequence = async (slug, sequence) => {
    try {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${slug}/sequence/${sequence}/preflight/result`,
        {
          method: "GET",
          headers: {
            Authorization: Utilities.getToken(),
          },
        },
      );
      if (!res.ok) {
        this.state.getKotsPreflightResultJob.stop();
        this.setState({
          errorMessage: `Encountered an error while fetching preflight results: Unexpected status code: ${res.status}`,
          preflightResultCheckCount: 0,
        });
        return;
      }
      const response = await res.json();
      if (response.preflightResult?.result) {
        this.state.getKotsPreflightResultJob.stop();
        this.setState({ preflightResultCheckCount: 0 });
      }
      let parsedStatusResults = {};
      try {
        parsedStatusResults = JSON.parse(response.preflightProgress);
      } catch {
        // empty
      }
      this.setState({
        preflightCurrentStatus: parsedStatusResults,
        preflightResultData: response.preflightResult,
        preflightResultCheckCount: this.state.preflightResultCheckCount + 1,
      });
    } catch (err) {
      console.log(err);
      this.setState({
        errorMessage: err
          ? `Encountered an error while fetching preflight results: ${err.message}`
          : "Something went wrong, please try again.",
        preflightResultCheckCount: 0,
      });
    }
  };

  getLatestKotsPreflightResult = async () => {
    const { slug } = this.props.match.params;

    try {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${slug}/preflight/result`,
        {
          method: "GET",
          headers: {
            Authorization: Utilities.getToken(),
          },
        },
      );
      if (!res.ok) {
        this.state.getKotsPreflightResultJob.stop();
        this.setState({
          errorMessage: `Encountered an error while fetching preflight results: Unexpected status code: ${res.status}`,
          preflightResultCheckCount: 0,
        });
        return;
      }
      const response = await res.json();
      if (
        response.preflightResult?.result ||
        response.preflightResult?.skipped
      ) {
        this.state.getKotsPreflightResultJob.stop();
        this.setState({ preflightResultCheckCount: 0 });
      }
      let parsedStatusResults = {};
      try {
        parsedStatusResults = JSON.parse(response.preflightProgress);
      } catch {
        // empty
      }
      this.setState({
        preflightCurrentStatus: parsedStatusResults,
        preflightResultData: response.preflightResult,
        preflightResultCheckCount: this.state.preflightResultCheckCount + 1,
      });
    } catch (err) {
      console.log(err);
      this.setState({
        errorMessage: err
          ? `Encountered an error while fetching preflight results: ${err.message}`
          : "Something went wrong, please try again.",
        preflightResultCheckCount: 0,
      });
    }
  };

  render() {
    const { slug } = this.props.match.params;
    const {
      showSkipModal,
      showWarningModal,
      preflightResultData,
      errorMessage,
    } = this.state;

    const preflightSkipped = preflightResultData?.skipped;
    const stopPolling = preflightResultData?.result || preflightSkipped;
    const blockDeployment = preflightResultData?.hasFailingStrictPreflights;
    let preflightJSON = {};
    if (preflightResultData?.result) {
      if (showSkipModal) {
        this.hideSkipModal();
      }
      preflightJSON = JSON.parse(preflightResultData?.result);
    }
    const hasResult = size(preflightJSON.results) > 0;
    const hasErrors = size(preflightJSON.errors) > 0;
    const preflightState = getPreflightResultState(preflightJSON);

    return (
      <div className="flex-column flex1 container">
        <Helmet>
          <title>{`${
            this.props.appName
              ? `${this.props.appName} Admin Console`
              : "Admin Console"
          }`}</title>
        </Helmet>
        <div className="flex1 flex u-overflow--auto">
          <div className="PreflightChecks--wrapper flex1 flex-column u-paddingTop--30">
            {this.props.history.location.pathname.includes(
              "version-history",
            ) && (
              <div
                className="u-fontWeight--bold u-linkColor u-cursor--pointer"
                onClick={() => this.props.history.goBack()}
              >
                <span
                  className="icon clickable backArrow-icon u-marginRight--10"
                  style={{ verticalAlign: "0" }}
                />
                Back
              </div>
            )}
            <div className="u-minWidth--full u-marginTop--20 flex-column flex1 u-position--relative">
              {errorMessage && errorMessage.length > 0 ? (
                <div className="ErrorWrapper flex-auto flex alignItems--center u-marginBottom--20">
                  <div className="icon redWarningIcon u-marginRight--10" />
                  <div>
                    <p className="title">Encountered an error</p>
                    <p className="error">{errorMessage}</p>
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
              {!stopPolling && (
                <div className="flex-column justifyContent--center alignItems--center flex1 u-minWidth--full">
                  <PreflightsProgress
                    progressData={this.state.preflightCurrentStatus}
                    preflightResultCheckCount={
                      this.state.preflightResultCheckCount
                    }
                  />
                </div>
              )}
              {hasErrors && this.renderErrors(preflightJSON?.errors)}
              {stopPolling && !hasErrors && (
                <div className="dashboard-card">
                  <div className="flex flex1 justifyContent--spaceBetween alignItems--center">
                    <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">
                      Results from your preflight checks
                    </p>
                    <div className="flex alignItems--center">
                      {this.props.fromLicenseFlow &&
                      stopPolling &&
                      hasResult &&
                      preflightState !== "pass" ? (
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
                      results={preflightResultData.result}
                      skipped={preflightSkipped}
                    />
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {this.props.fromLicenseFlow ? (
          <div className="flex-auto flex justifyContent--flexEnd u-marginBottom--15">
            {stopPolling ? (
              <div>
                <button
                  type="button"
                  className="btn primary blue"
                  disabled={blockDeployment}
                  onClick={() => this.deployKotsDownstream()}
                >
                  <span
                    data-tip-disable={!blockDeployment}
                    data-tip="Deployment is disabled as a strict analyzer in this version's preflight checks has failed or has not been run"
                    data-for="disable-deployment-tooltip"
                  >
                    Continue
                  </span>
                </button>
                <ReactTooltip effect="solid" id="disable-deployment-tooltip" />
              </div>
            ) : !blockDeployment ? (
              <div className="flex flex1 justifyContent--center alignItems--center">
                <span
                  className="u-fontSize--normal u-fontWeight--medium u-textDecoration--underline u-textColor--bodyCopy u-marginTop--15 u-cursor--pointer"
                  onClick={this.showSkipModal}
                >
                  Ignore Preflights{" "}
                </span>
              </div>
            ) : null}
          </div>
        ) : stopPolling ? (
          <div className="flex-auto flex justifyContent--flexEnd u-marginBottom--15">
            <button
              type="button"
              className="btn primary blue"
              onClick={this.rerunPreflights}
            >
              Re-run
            </button>
          </div>
        ) : null}

        {showSkipModal && (
          <SkipPreflightsModal
            showSkipModal={showSkipModal}
            hideSkipModal={this.hideSkipModal}
            deployKotsDownstream={this.deployKotsDownstream}
          />
        )}

        <Modal
          isOpen={showWarningModal}
          onRequestClose={this.hideWarningModal}
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
                onClick={this.hideWarningModal}
              >
                Close
              </button>
              <button
                type="button"
                className="btn blue primary u-marginLeft--10"
                onClick={() => this.deployKotsDownstream(true)}
              >
                Deploy and continue
              </button>
            </div>
          </div>
        </Modal>
      </div>
    );
  }
}

export default withRouter(PreflightResultPage);
