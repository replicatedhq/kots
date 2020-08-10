import React, { Component } from "react";
import { Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { Helmet } from "react-helmet";
import { withRouter } from "react-router-dom";
import Modal from "react-modal";
import { getKotsPreflightResult, getLatestKotsPreflightResult } from "@src/queries/AppsQueries";
import Loader from "./shared/Loader";
import PreflightRenderer from "./PreflightRenderer";
import { getPreflightResultState, Utilities } from "../utilities/utilities";
import "../scss/components/PreflightCheckPage.scss";
import PreflightResultErrors from "./PreflightResultErrors";
import has from "lodash/has";
import size from "lodash/size";

class PreflightResultPage extends Component {
  state = {
    showSkipModal: false,
    showWarningModal: false,
    errorMessage: ""
  };

  async componentWillUnmount() {
    if (has(this.props.data, "stopPolling")) {
      this.props.data.stopPolling();
    }

    if (this.props.fromLicenseFlow && this.props.refetchAppsList) {
      await this.props.refetchAppsList();
    }
  }

  deployKotsDownstream = async (force = false) => {
    this.setState({ errorMessage: "" });
    try {
      const { data, history, match } = this.props;
      const preflightResultData = data.getKotsPreflightResult || data.getLatestKotsPreflightResult;

      const preflightResults = JSON.parse(preflightResultData?.result);
      const preflightState = getPreflightResultState(preflightResults);
      if (preflightState !== "pass") {
        if (!force) {
          this.showWarningModal();
          return;
        }
        const sequence = match.params.sequence ? parseInt(match.params.sequence, 10) : 0;
        await this.deployKotsVersion(preflightResultData.appSlug, sequence);
      }

      history.push(`/app/${preflightResultData.appSlug}/version-history`);
    } catch (err) {
      console.log(err);
      this.setState({
        errorMessage: err ? `Encountered an error while trying to deploy downstream version: ${err.message}` : "Something went wrong, please try again."
      });
    }
  }

  deployKotsVersion = async (appSlug, sequence) => {
    this.setState({ errorMessage: "" });
    try {
      await fetch(`${window.env.API_ENDPOINT}/app/${appSlug}/sequence/${sequence}/deploy`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "POST",
      });
    } catch(err) {
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
    const preflightResultData = this.props.data.getKotsPreflightResult || this.props.data.getLatestKotsPreflightResult;
    const sequence = this.props.match.params.sequence ? parseInt(this.props.match.params.sequence, 10) : 0;

    const appSlug = preflightResultData.appSlug;
    fetch(`${window.env.API_ENDPOINT}/app/${appSlug}/sequence/${sequence}/preflight/ignore-rbac`, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
        "Authorization": Utilities.getToken(),
      },
      method: "POST",
    })
      .then(async (res) => {
        this.props.data.refetch();
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          errorMessage: err ? `Encountered an error while trying to ignore permissions: ${err.message}` : "Something went wrong, please try again."
        });
      });
  }

  rerunPreflights = () => {
    this.setState({ errorMessage: "" });
    const preflightResultData = this.props.data.getKotsPreflightResult || this.props.data.getLatestKotsPreflightResult;
    const sequence = this.props.match.params.sequence ? parseInt(this.props.match.params.sequence, 10) : 0;

    const appSlug = preflightResultData.appSlug;
    fetch(`${window.env.API_ENDPOINT}/app/${appSlug}/sequence/${sequence}/preflight/run`, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
        "Authorization": Utilities.getToken(),
      },
      method: "POST",
    })
      .then((res) => {
        if (res.status === 200) {
          this.props.data?.refetch();
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
    this.props.data?.startPolling(2000);

    const valueFromAPI = errors.map(error => {
      return error.error;
    }).join("\n");

    return (
      <PreflightResultErrors
        valueFromAPI={valueFromAPI}
        ignorePermissionErrors={this.ignorePermissionErrors}
        logo={this.props.logo}
        preflightResultData={this.props.data.getKotsPreflightResult || this.props.data.getLatestKotsPreflightResult}
      />
    );
  }
  
  render() {
    const { data } = this.props;
    const { showSkipModal, showWarningModal } = this.state;
    const isLoading = data.loading;

    const preflightResultData = isLoading
      ? null
      : data.getKotsPreflightResult || data.getLatestKotsPreflightResult;

    const stopPolling = !!preflightResultData?.result;
    let preflightJSON = {};
    if (stopPolling) {
      data.stopPolling();
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
              {this.state.errorMessage && this.state.errorMessage.length > 0 ?
                <div className="ErrorWrapper flex-auto flex alignItems--center">
                  <div className="icon redWarningIcon u-marginRight--10" />
                  <div>
                    <p className="title">Encountered an error</p>
                    <p className="error">{this.state.errorMessage}</p>
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
                <Link to={`/app/${preflightResultData?.appSlug}`}>
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
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Skipping preflight checks will not cancel them. They will continue to run in the background. Do you want to continue to the {preflightResultData?.appSlug} dashboard? </p>
            <div className="u-marginTop--10 flex justifyContent--flexEnd">
              <button type="button" className="btn secondary" onClick={this.hideSkipModal}>Close</button>
              <Link to={`/app/${preflightResultData?.appSlug}`}>
                <button type="button" className="btn blue primary u-marginLeft--10">Go to Dashboard</button>
              </Link>
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

export default compose(
  withApollo,
  withRouter,
  graphql(getKotsPreflightResult, {
    skip: props => {
      const { match } = props;
      return !match.params.downstreamSlug;
    },
    options: props => {
      const { match } = props;

      return {
        pollInterval: 2000,
        variables: {
          appSlug: match.params.slug,
          clusterSlug: match.params.downstreamSlug,
          sequence: match.params.sequence
        },
        fetchPolicy: "no-cache"
      };
    }
  }),
  graphql(getLatestKotsPreflightResult, {
    skip: props => {
      const { match } = props;

      return !!match.params.downstreamSlug;
    },
    options: () => {
      return {
        pollInterval: 2000,
        fetchPolicy: "no-cache"
      }
    }
  }),
)(PreflightResultPage);
