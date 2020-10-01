import get from "lodash/get";
import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import MonacoEditor from "react-monaco-editor"; 
import CodeSnippet from "./shared/CodeSnippet";
import ErrorModal from "./modals/ErrorModal";
import { Utilities } from "../utilities/utilities";
import "../scss/components/PreflightCheckPage.scss";

class PreflightResultErrors extends Component {
  state = {
    command: null,
    showErrorDetails: false,
    errorTitle: "",
    errorMsg: "",
    displayErrorModal: false,
  };

  async componentDidMount() {
    if (!this.props.preflightResultData) {
      return;
    }
    this.getPreflightCommand();
  }

  componentDidUpdate(prevProps) {
    if (!this.props.preflightResultData || (
      get(this.props, "preflightResultData.appSlug") === get(prevProps, "preflightResultData.appSlug") &&
      get(this.props, "preflightResultData.sequence") === get(prevProps, "preflightResultData.sequence")
    )) {
      return;
    }
    this.getPreflightCommand();
  }

  toggleShowErrorDetails = () => {
    this.setState({
      showErrorDetails: !this.state.showErrorDetails
    })
  }

  getPreflightCommand = async () => {
    const { match, preflightResultData } = this.props;
    const sequence = match.params.sequence ? parseInt(match.params.sequence, 10) : 0;
    try {
      const command = await this.fetchPreflightCommand(preflightResultData.appSlug, sequence);
      this.setState({
        command
      });
    } catch (err) {
      this.setState({
        errorTitle: `Failed to get preflight command`,
        errorMsg: err ? err.message : "Something went wrong, please try again.",
        displayErrorModal: true,
      });
    }
  }

  fetchPreflightCommand = async (slug, sequence) => {
    const res = await fetch(`${window.env.API_ENDPOINT}/app/${slug}/sequence/${sequence}/preflightcommand`, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        origin: window.location.origin,
      })
    });
    if (!res.ok) {
      throw new Error(`Unexpected status code: ${res.status}`);
    }
    const response = await res.json();
    return response.command;
  }

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  }

  render() {
    const { valueFromAPI, logo } = this.props;
    const {
      errorTitle,
      errorMsg,
      displayErrorModal,
      command,
    } = this.state;

    return (
      <div className="flex flex1 flex-column">
        <div className="flex flex1 u-height--full u-width--full u-marginTop--5 u-marginBottom--20">
          <div className="flex-column u-width--full u-overflow--hidden u-paddingTop--30 u-paddingBottom--5 alignItems--center justifyContent--center">
            <div className="PreChecksBox-wrapper flex-column u-padding--20">
              <div className="flex">
                {logo &&
                  <div className="flex-auto u-marginRight--10">
                    <div className="watch-icon" style={{ backgroundImage: `url(${logo})`, width: "36px", height: "36px" }}></div>
                  </div>
                }
                <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Unable to automatically run preflight checks</h2>
              </div>
              <p className="u-marginTop--10 u-marginBottom--10 u-fontSize--normal u-lineHeight--normal u-color--dustyGray u-fontWeight--normal">
                The Kubernetes RBAC policy that the Admin Console is running with does not have access to complete the Preflight Checks. It’s recommended that you run these manually before proceeding.
              </p>
              <p className="replicated-link u-fontSize--normal u-marginBottom--10" onClick={this.toggleShowErrorDetails}>{this.state.showErrorDetails ? "Hide details" : "Show details"}</p>
              {this.state.showErrorDetails &&
                <div className="flex-column flex flex1 monaco-editor-wrapper u-border--gray">
                  <MonacoEditor
                    language="bash"
                    value={valueFromAPI}
                    height="300"
                    width="100%"
                    options={{
                      readOnly: true,
                      contextmenu: false,
                      minimap: {
                        enabled: false
                      },
                      scrollBeyondLastLine: false,
                    }}
                  />
                </div>
              }
              <div className="u-marginTop--20">
                <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Run Preflight Checks Manually</h2>
                <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Run the commands below from your workstation to complete the Preflight Checks.</p>
                {command ?
                  <CodeSnippet
                    language="bash"
                    canCopy={true}
                    onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
                  >
                    {command}
                  </CodeSnippet>
                  : null
                }
              </div>
              <div className="u-marginTop--30 flex justifyContent--flexEnd">
                <span className="replicated-link u-marginLeft--20 u-fontSize--normal" onClick={this.props.ignorePermissionErrors}>
                  Proceed with limited Preflights
                </span>
              </div>
            </div>
          </div>
        </div>

        {errorMsg &&
          <ErrorModal
            errorModal={displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            err={errorTitle}
            errMsg={errorMsg}
          />}
      </div>
    );
  }
}

export default withRouter(PreflightResultErrors);
