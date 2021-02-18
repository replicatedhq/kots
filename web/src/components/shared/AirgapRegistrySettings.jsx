import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import get from "lodash/get";

import Loader from "../shared/Loader";
import ErrorModal from "../modals/ErrorModal";
import "../../scss/components/watches/WatchDetailPage.scss";
import { Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";

class AirgapRegistrySettings extends Component {

  constructor(props) {
    super(props);

    const {
      hostname = "",
      username = "",
      password = "",
      namespace = props.app ? props.app.slug : ""
    } = props?.registryDetails || {};

    this.state = {
      loading: false,

      hostname,
      username,
      password,
      namespace,

      lastSync: null,
      testInProgress: false,
      testFailed: false,
      testMessage: "",

      updateChecker: new Repeater(),
      rewriteStatus: "",
      rewriteMessage: "",
      fetchRegistryErrMsg: "",
      displayErrorModal: false
    }
  }

  componentWillUnmount() {
    this.state.updateChecker.stop();
  }

  componentDidMount = () => {
    this.fetchRegistryInfo();
    this.triggerStatusUpdates();
  }

  onSubmit = async () => {
    const {
      hostname,
      username,
      password,
      namespace,
    } = this.state;
    const { slug } = this.props.match.params;

    fetch(`${window.env.API_ENDPOINT}/app/${slug}/registry`, {
      method: "PUT",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        hostname,
        username,
        password,
        namespace,
      })
    })
      .then(async (res) => {
        const registryDetails = await res.json();
        if (registryDetails.error) {
          this.setState({
            rewriteStatus: "failed",
            rewriteMessage: registryDetails.error,
          });
        } else {
          this.state.updateChecker.start(this.updateStatus, 1000);
        }
      })
      .catch((err) => {
        this.setState({
          rewriteStatus: "failed",
          rewriteMessage: err,
        });
      });
  }

  testRegistryConnection = async () => {
    this.setState({
      testInProgress: true,
      testMessage: "",
    });

    const { slug } = this.props.app;

    let res;
    try {
      res = await fetch(`${window.env.API_ENDPOINT}/app/${slug}/registry/validate`, {
        method: "POST",
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          hostname: this.state.hostname,
          namespace: this.state.namespace,
          username: this.state.username,
          password: this.state.password,
        }),
      });
    } catch (err) {
      this.setState({
        testInProgress: false,
        testMessage: err,
        testFailed: true,
        lastSync: new Date(),
      });
      return;
    }

    const response = await res.json();
    if (response.success) {
      this.setState({
        testInProgress: false,
        testMessage: "Success!",
        testFailed: false,
        lastSync: new Date(),
      });
    } else {
      this.setState({
        testInProgress: false,
        testMessage: response.error,
        testFailed: true,
        lastSync: new Date(),
      });
    }
  }

  handleFormChange = (field, val) => {
    let nextState = {};
    nextState[field] = val;
    this.setState(nextState, () => {
      if (this.props.gatherDetails) {
        const { hostname, username, password, namespace } = this.state;
        this.props.gatherDetails({ hostname, username, password, namespace });
      }
    });
  }

  componentDidUpdate(lastProps) {
    const { app } = this.props;

    if (app?.slug !== lastProps.app?.slug) {
      this.fetchRegistryInfo();
    }
  }

  fetchRegistryInfo = () => {
    if (this.state.loading) {
      return;
    }

    this.setState({ loading: true, fetchRegistryErrMsg: "", displayErrorModal: false });

    let url = `${window.env.API_ENDPOINT}/registry`;
    if (this.props.app) {
      url = `${window.env.API_ENDPOINT}/app/${this.props.app.slug}/registry`;
    }

    fetch(url, {
      headers: {
        "Authorization": Utilities.getToken()
      },
      method: "GET",
    })
      .then(res => res.json())
      .then(result => {
        if (result.success) {
          this.setState({
            hostname: result.hostname,
            username: result.username,
            password: result.password,
            namespace: result.namespace,
            loading: false,
            fetchRegistryErrMsg: "",
            displayErrorModal: false
          });

          if (this.props.gatherDetails) {
            const { hostname, username, password, namespace } = result;
            this.props.gatherDetails({ hostname, username, password, namespace });
          }

        } else {
          this.setState({ loading: false, fetchRegistryErrMsg: "Unable to get registry info, please try again.", displayErrorModal: true });
        }
      })
      .catch(err => {
        console.log(err);
        this.setState({ loading: false, fetchRegistryErrMsg: err ? `Unable to get registry info: ${err.message}` : "Something went wrong, please try again.", displayErrorModal: true });
      })
  }

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  }

  triggerStatusUpdates = () => {
    let url = `${window.env.API_ENDPOINT}/imagerewritestatus`;
    if (this.props.app) {
      url = `${window.env.API_ENDPOINT}/app/${this.props.app.slug}/imagerewritestatus`;
    }
    fetch(url, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "GET",
    })
      .then(async (response) => {
        const res = await response.json();
        this.setState({
          rewriteStatus: res.status,
          rewriteMessage: res.currentMessage,
        });
        if (res.status !== "running") {
          return;
        }
        this.state.updateChecker.start(this.updateStatus, 1000);
      })
      .catch((err) => {
        console.log("failed to get rewrite status", err);
      });
  }

  updateStatus = () => {
    let url = `${window.env.API_ENDPOINT}/imagerewritestatus`;
    if (this.props.app) {
      url = `${window.env.API_ENDPOINT}/app/${this.props.app.slug}/imagerewritestatus`;
    }
    return new Promise((resolve, reject) => {
      fetch(url, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      })
        .then(async (response) => {
          const res = await response.json();

          this.setState({
            rewriteStatus: res.status,
            rewriteMessage: res.currentMessage,
          });

          if (res.status !== "running") {
            this.state.updateChecker.stop();

            if (this.props.updateCallback) {
              this.props.updateCallback();
            }
          }

          resolve();
        })
        .catch((err) => {
          console.log("failed to get rewrite status", err);
          reject();
        });
    });
  }

  render() {
    const { app, hideTestConnection, hideCta, namespaceDescription, showHostnameAsRequired } = this.props;
    const { hostname, password, username, namespace, lastSync, testInProgress, testFailed, testMessage } = this.state;
    const { rewriteMessage, rewriteStatus } = this.state;

    let statusText = rewriteMessage;
    try {
      const jsonMessage = JSON.parse(statusText);
      const type = get(jsonMessage, "type");
      if (type === "progressReport") {
        statusText = jsonMessage.compatibilityMessage;
        // TODO: handle image upload progress here
      }
    } catch {
      // empty
    }

    if (this.state.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const namespaceSubtext = namespaceDescription || "Changing the namespace will rewrite all of your airgap images and push them to your registry."

    let testStatusText = "";
    if (testInProgress) {
      testStatusText = "Testing...";
    } else if (lastSync) {
      testStatusText = testMessage;
    } else {
      // TODO: this will always be displayed when page is refreshed
      testStatusText = "Connection has not been tested";
    }

    const disableSubmitButton = rewriteStatus === "running";
    const showProgress = rewriteStatus === "running";
    const showStatusError = rewriteStatus === "failed";

    return (
      <div>
        <form>
          <div className="flex u-marginBottom--20">
            <div className="flex1">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Hostname {showHostnameAsRequired && <span className="u-color--chestnut">(Required)</span>}</p>
              <p className="u-lineHeight--normal u-fontSize--small u-color--dustyGray u-fontWeight--medium u-marginBottom--10">Ensure this domain supports the Docker V2 protocol.</p>
              <input type="text" className={`Input ${app?.isAirgap && "is-disabled"}`} disabled={app?.isAirgap} placeholder="artifactory.some-big-bank.com" value={hostname || ""} autoComplete="" onChange={(e) => { this.handleFormChange("hostname", e.target.value) }} />
            </div>
          </div>
          <div className="flex u-marginBottom--20">
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Username</p>
              <input type="text" className="Input" placeholder="username" value={username || ""} autoComplete="username" onChange={(e) => { this.handleFormChange("username", e.target.value) }} />
            </div>
            <div className="flex1 u-paddingLeft--5">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Password</p>
              <input type="password" className="Input" placeholder="password" autoComplete="current-password" value={password || ""} onChange={(e) => { this.handleFormChange("password", e.target.value) }} />
            </div>
          </div>
          {hideTestConnection ? null :
            <div className="test-connection-box u-marginBottom--20">
              <div className="flex">
                <div>
                  <button type="button" className="btn secondary" onClick={this.testRegistryConnection}>Test connection</button>
                </div>
                {this.state.pingedEndpoint &&
                  <div className="flex-column justifyContent--center">
                    <p className="u-marginLeft--10 u-fontSize--small u-fontWeight--medium u-color--tundora"><span className={`icon checkmark-icon u-marginRight--5 u-verticalAlign--neg3`} />Connected to {this.state.pingedEndpoint}</p>
                  </div>
                }
              </div>
              {testFailed && !testInProgress ?
                <p className="u-fontSize--small u-fontWeight--medium u-color--chestnut u-marginTop--10 u-lineHeight--normal">{testStatusText}</p>
                :
                <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--10 u-lineHeight--normal">{testStatusText}</p>
              }
            </div>
          }
          <div className="flex u-marginBottom--5">
            <div className="flex1">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Registry Namespace</p>
              <p className="u-lineHeight--normal u-fontSize--small u-color--dustyGray u-fontWeight--medium u-marginBottom--10">{namespaceSubtext}</p>
              <input type="text" className={`Input ${app?.isAirgap && "is-disabled"}`} placeholder="namespace" disabled={app?.isAirgap} value={namespace || ""} autoComplete="" onChange={(e) => { this.handleFormChange("namespace", e.target.value) }} />
            </div>
          </div>
        </form>
        {hideCta ? null :
          <div>
            {showProgress ?
              <div className="u-marginTop--20">
                <Loader size="30" />
                <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--10">{statusText}</p>
              </div>
              :
              null
            }
            {showStatusError ?
              <p className="u-fontSize--small u-fontWeight--medium u-color--chestnut u-marginTop--10">{statusText}</p>
              :
              null
            }
            <div className="u-marginTop--20">
              <button className="btn primary blue" disabled={disableSubmitButton} onClick={this.onSubmit}>Save changes</button>
            </div>
          </div>
        }
        {this.state.fetchRegistryErrMsg &&
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={this.state.fetchRegistryErrMsg}
            tryAgain={this.fetchRegistryInfo}
            err="Failed to get registry info"
            loading={this.state.loading}
          />}
      </div>
    )
  }
}

export default withRouter(AirgapRegistrySettings);
