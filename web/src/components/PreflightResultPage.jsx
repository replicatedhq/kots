import React, { Component } from "react";
import { Link } from "react-router-dom";
import { Helmet } from "react-helmet";
import { withRouter } from "react-router-dom";
import Modal from "react-modal";
import Loader from "./shared/Loader";
import PreflightRenderer from "./PreflightRenderer";
import { Repeater } from "../utilities/repeater";
import { getPreflightResultState, Utilities } from "../utilities/utilities";
import "../scss/components/PreflightCheckPage.scss";
import PreflightResultErrors from "./PreflightResultErrors";
import size from "lodash/size";

class PreflightResultPage extends Component {
  state = {
    showSkipModal: false,
    showWarningModal: false,
    getKotsPreflightResultJob: new Repeater(),
    preflightResultData: null,
    errorMessage: ""
  };

  componentDidMount() {
    this.state.getKotsPreflightResultJob.start(this.getKotsPreflightResult, 2000);
  }

  async componentWillUnmount() {
    this.state.getKotsPreflightResultJob.stop();

    if (this.props.fromLicenseFlow && this.props.refetchAppsList) {
      await this.props.refetchAppsList();
    }
  }

  deployKotsDownstream = async (force = false) => {
    this.setState({ errorMessage: "" });
    try {
      const { history, match } = this.props;
      const { slug } = match.params;
      const { preflightResultData } = this.state;

      const preflightResults = JSON.parse(preflightResultData?.result);
      const preflightState = getPreflightResultState(preflightResults);
      if (preflightState !== "pass") {
        if (!force) {
          this.showWarningModal();
          return;
        }
        const sequence = match.params.sequence ? parseInt(match.params.sequence, 10) : 0;
        await this.deployKotsVersion(slug, sequence, force);
      }

      history.push(`/app/${slug}`);
    } catch (err) {
      console.log(err);
      this.setState({
        errorMessage: err ? `Encountered an error while trying to deploy downstream version: ${err.message}` : "Something went wrong, please try again."
      });
    }
  }

  deployKotsVersion = async (appSlug, sequence, force) => {
    this.setState({ errorMessage: "" });
    try {
      await fetch(`${window.env.API_ENDPOINT}/app/${appSlug}/sequence/${sequence}/deploy`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "POST",
        body: JSON.stringify({
          isSkipPreflights: false,
          continueWithFailedPreflights: force ? true : false
        }),
      });
    } catch (err) {
      console.log(err);
      this.setState({
        errorMessage: err ? `Encountered an error while trying to deploy version: ${err.message}` : "Something went wrong, please try again."
      });
    }
  }

  showSkipModal = () => {
    this.setState({
      showSkipModal: true
    })
  }

  hideSkipModal = () => {
    this.setState({
      showSkipModal: false
    });
  }

  showWarningModal = () => {
    this.setState({
      showWarningModal: true
    })
  }

  hideWarningModal = () => {
    this.setState({
      showWarningModal: false
    });
  }

  ignorePermissionErrors = () => {
    this.setState({ errorMessage: "" });

    const { slug } = this.props.match.params;
    const sequence = this.props.match.params.sequence ? parseInt(this.props.match.params.sequence, 10) : 0;

    fetch(`${window.env.API_ENDPOINT}/app/${slug}/sequence/${sequence}/preflight/ignore-rbac`, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
        "Authorization": Utilities.getToken(),
      },
      method: "POST",
    })
      .then(async (res) => {
        this.setState({
          preflightResultData: null,
        });
        this.state.getKotsPreflightResultJob.start(this.getKotsPreflightResult, 2000);
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          errorMessage: err ? `Encountered an error while trying to ignore permissions: ${err.message}` : "Something went wrong, please try again."
        });
      });
  }

  rerunPreflights = () => {
    const { slug } = this.props.match.params;

    this.setState({ errorMessage: "" });
    const sequence = this.props.match.params.sequence ? parseInt(this.props.match.params.sequence, 10) : 0;

    fetch(`${window.env.API_ENDPOINT}/app/${slug}/sequence/${sequence}/preflight/run`, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
        "Authorization": Utilities.getToken(),
      },
      method: "POST",
    })
      .then((res) => {
        if (res.status === 200) {
          this.setState({
            preflightResultData: null,
          });
          this.state.getKotsPreflightResultJob.start(this.getKotsPreflightResult, 2000);
        } else {
          this.setState({
            errorMessage: `Encountered an error while trying to re-run preflight checks: Status ${res.status}`
          });
        }
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          errorMessage: err ? `Encountered an error while trying to re-run preflight checks: ${err.message}` : "Something went wrong, please try again."
        });
      });
  }

  renderErrors = (errors) => {
    const { preflightResultData } = this.state;

    // TODO: why start polling here?
    // this.state.getKotsPreflightResultJob.start(this.getKotsPreflightResult, 2000);

    const valueFromAPI = errors.map(error => {
      return error.error;
    }).join("\n");

    return (
      <PreflightResultErrors
        valueFromAPI={valueFromAPI}
        ignorePermissionErrors={this.ignorePermissionErrors}
        logo={this.props.logo}
        preflightResultData={preflightResultData}
      />
    );
  }

  getKotsPreflightResult = async () => {
    this.setState({ errorMessage: "" });
    const { match } = this.props;
    if (match.params.downstreamSlug) { // why?
      const sequence = match.params.sequence ? parseInt(match.params.sequence, 10) : 0;
      return this.getKotsPreflightResultForSequence(match.params.slug, sequence);
    }
    return this.getLatestKotsPreflightResult();
  }

  getKotsPreflightResultForSequence = async (slug, sequence) => {
    try {
      const res = await fetch(`${window.env.API_ENDPOINT}/app/${slug}/sequence/${sequence}/preflight/result`, {
        method: "GET",
        headers: {
          "Authorization": Utilities.getToken(),
        }
      });
      if (!res.ok) {
        this.state.getKotsPreflightResultJob.stop();
        this.setState({
          errorMessage: `Encountered an error while fetching preflight results: Unexpected status code: ${res.status}`,
        });
        return;
      }
      const response = await res.json();
      if (response.preflightResult?.result) {
        this.state.getKotsPreflightResultJob.stop();
      }
      this.setState({
        preflightResultData: response.preflightResult,
      });
    } catch (err) {
      console.log(err);
      this.setState({
        errorMessage: err ? `Encountered an error while fetching preflight results: ${err.message}` : "Something went wrong, please try again."
      });
    }
  }

  getLatestKotsPreflightResult = async () => {
    const { slug } = this.props.match.params;

    try {
      const res = await fetch(`${window.env.API_ENDPOINT}/app/${slug}/preflight/result`, {
        method: "GET",
        headers: {
          "Authorization": Utilities.getToken(),
        }
      });
      if (!res.ok) {
        this.state.getKotsPreflightResultJob.stop();
        this.setState({
          errorMessage: `Encountered an error while fetching preflight results: Unexpected status code: ${res.status}`,
        });
        return;
      }
      const response = await res.json();
      if (response.preflightResult?.result) {
        this.state.getKotsPreflightResultJob.stop();
      }
      this.setState({
        preflightResultData: response.preflightResult,
      });
    } catch (err) {
      console.log(err);
      this.setState({
        errorMessage: err ? `Encountered an error while fetching preflight results: ${err.message}` : "Something went wrong, please try again."
      });
    }
  }

  sendPreflightsReport = async () => {
    const { slug } = this.props.match.params;

    fetch(`${window.env.API_ENDPOINT}/app/${slug}/preflight/report`, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
        "Authorization": Utilities.getToken(),
      },
      method: "POST",
    })
      this.props.history.push(`/app/${slug}`)
  }

  render() {
    const { slug } = this.props.match.params;
    const { showSkipModal, showWarningModal, preflightResultData, errorMessage } = this.state;

    const stopPolling = !!preflightResultData?.result;
    let preflightJSON = {};
    if (stopPolling) {
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
          <title>{`${this.props.appName ? `${this.props.appName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <div className="flex1 flex u-overflow--auto">
          <div className="PreflightChecks--wrapper flex1 flex-column u-paddingTop--30">
            {this.props.history.location.pathname.includes("version-history") &&
              <div className="u-fontWeight--bold u-color--royalBlue u-cursor--pointer" onClick={() => this.props.history.goBack()}>
                <span className="icon clickable backArrow-icon u-marginRight--10" style={{ verticalAlign: "0" }} />
                Back
            </div>}
            <div className="u-minWidth--full u-marginTop--20 flex-column flex1 u-position--relative">
              {errorMessage && errorMessage.length > 0 ?
                <div className="ErrorWrapper flex-auto flex alignItems--center u-marginBottom--20">
                  <div className="icon redWarningIcon u-marginRight--10" />
                  <div>
                    <p className="title">Encountered an error</p>
                    <p className="error">{errorMessage}</p>
                  </div>
                </div>
                : null}
              <p className="u-fontSize--header u-color--tuna u-fontWeight--bold">
                Preflight checks
              </p>
              <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--10">
                Preflight checks validate that your cluster will meet the minimum requirements. If your cluster does not meet the requirements you can still proceed, but understand that things might not work properly.
              </p>
              {!stopPolling && (
                <div className="flex-column justifyContent--center alignItems--center flex1 u-minWidth--full">
                  <Loader size="60" />
                </div>
              )}
              {hasErrors && this.renderErrors(preflightJSON?.errors)}
              {stopPolling && !hasErrors &&
                <div className="flex-column">
                  <PreflightRenderer
                    className="u-marginTop--20"
                    results={preflightResultData.result}
                  />
                </div>
              }
            </div>
          </div>
        </div>

        {this.props.fromLicenseFlow ?
          <div className="flex-auto flex justifyContent--flexEnd u-marginBottom--15">
            {stopPolling && hasResult && preflightState !== "pass" &&
              <div className="flex">
                <Link to={`/app/${slug}`}>
                  <button type="button" className="btn secondary u-marginRight--10">Cancel</button>
                </Link>
                <button type="button" className="btn secondary blue u-marginRight--10" onClick={this.rerunPreflights}>Re-run</button>
              </div>
            }
            <button
              type="button"
              className="btn primary blue"
              onClick={stopPolling ? () => this.deployKotsDownstream(false) : this.showSkipModal}
            >
              {stopPolling ? "Continue" : "Skip"}
            </button>
          </div>
          : stopPolling ?
            <div className="flex-auto flex justifyContent--flexEnd u-marginBottom--15">
              <button type="button" className="btn primary blue" onClick={this.rerunPreflights}>Re-run</button>
            </div>
            :
            null
        }

        <Modal
          isOpen={showSkipModal}
          onRequestClose={this.hideSkipModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Skip preflight checks"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body">
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Skipping preflight checks will not cancel them. They will continue to run in the background. Do you want to continue to the {slug} dashboard? </p>
            <div className="u-marginTop--10 flex justifyContent--flexEnd">
              <button type="button" className="btn secondary" onClick={this.hideSkipModal}>Close</button>
              <button type="button" className="btn blue primary u-marginLeft--10" onClick={this.sendPreflightsReport}>Go to Dashboard</button>
            </div>
          </div>
        </Modal>

        <Modal
          isOpen={showWarningModal}
          onRequestClose={this.hideWarningModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Preflight shows some issues"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body">
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Preflight is showing some issues, are you sure you want to continue?</p>
            <div className="u-marginTop--10 flex justifyContent--flexEnd">
              <button type="button" className="btn secondary" onClick={this.hideWarningModal}>Close</button>
              <button type="button" className="btn blue primary u-marginLeft--10" onClick={() => this.deployKotsDownstream(true)}>
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
