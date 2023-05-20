import React, { Component } from "react";
import { withRouter } from "@src/utilities/react-router-utilities";
import get from "lodash/get";

import Loader from "../shared/Loader";
import ErrorModal from "../modals/ErrorModal";
import Modal from "react-modal";
import "../../scss/components/watches/WatchDetailPage.scss";
import { Repeater } from "../../utilities/repeater";
import Icon from "../Icon";
import { App, KotsParams } from "@types";
import InputField from "./forms/InputField";

type Props = {
  app: App;
  registryDetails: RegistryDetails;
  updateCallback: () => void;
  gatherDetails: (fields: GatherDetails) => void;
  hideCta: boolean;
  hideTestConnection: boolean;
  namespaceDescription: string;
  showHostnameAsRequired: boolean;
};

interface RegistryDetails {
  hostname: string;
  username: string;
  password: string;
  namespace: string;
}

interface GatherDetails extends RegistryDetails {
  isReadOnly: boolean;
}

type State = {
  loading: boolean;
  hostname: string;
  username: string;
  password: string;
  namespace: string;
  lastSync: Object | null;
  testInProgress: boolean;
  testFailed: boolean;
  testMessage: string | unknown;
  updateChecker: Repeater;
  rewriteStatus: string;
  rewriteMessage: string;
  fetchRegistryErrMsg: string;
  displayErrorModal: boolean;
  isReadOnly: boolean;
  originalRegistry: RegistryDetails | null;
  pingedEndpoint: string;
  showStopUsingWarning: boolean;
  isFirstPasswordChange: boolean;
};

class AirgapRegistrySettings extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    const {
      hostname = "",
      username = "",
      password = "",
      namespace = props.app ? props.app.slug : "",
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
      displayErrorModal: false,
      //newly added
      isReadOnly: false,
      originalRegistry: null,
      pingedEndpoint: "",
      showStopUsingWarning: false,
      isFirstPasswordChange: true,
    };
  }

  componentWillUnmount() {
    this.state.updateChecker.stop();
  }

  componentDidMount = () => {
    this.fetchRegistryInfo();
    this.triggerStatusUpdates();
  };

  onSaveRegistrySettings = async (stopUsingRegistry: boolean) => {
    let { hostname, username, password, namespace, isReadOnly } = this.state;
    const { slug } = this.props.params;

    if (stopUsingRegistry) {
      hostname = "";
      username = "";
      password = "";
      namespace = "";
      isReadOnly = false;
    }

    fetch(`${process.env.API_ENDPOINT}/app/${slug}/registry`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify({
        hostname,
        username,
        password,
        namespace,
        isReadOnly,
      }),
    })
      .then(async (res) => {
        const registryDetails = await res.json();
        if (registryDetails.error) {
          this.setState({
            rewriteStatus: "failed",
            rewriteMessage: registryDetails.error,
          });
        } else {
          this.setState({
            originalRegistry: {
              hostname: hostname,
              username: username,
              password: password,
              namespace: namespace,
            },
            hostname: hostname,
            username: username,
            password: password,
            namespace: namespace,
            isReadOnly: isReadOnly,
            isFirstPasswordChange: false,
          });
          this.state.updateChecker.start(this.updateStatus, 1000);
        }
      })
      .catch((err) => {
        this.setState({
          rewriteStatus: "failed",
          rewriteMessage: err,
        });
      });
  };

  testRegistryConnection = async () => {
    this.setState({
      testInProgress: true,
      testMessage: "",
    });

    const { slug } = this.props.outletContext.app;

    let res;
    try {
      res = await fetch(
        `${process.env.API_ENDPOINT}/app/${slug}/registry/validate`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({
            hostname: this.state.hostname,
            namespace: this.state.namespace,
            username: this.state.username,
            password: this.state.password,
            isReadOnly: this.state.isReadOnly,
          }),
        }
      );
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
  };

  handleFormChange = (field: string, val: string | boolean) => {
    let nextState: { [key: string]: string | boolean } = {};
    nextState[field] = val;

    if (
      this.props.outletContext.app?.isAirgap &&
      field === "isReadOnly" &&
      !val
    ) {
      // Pushing images in airgap mode is not yet supported, so registry name cannot be changed.
      nextState.hostname = this.state.originalRegistry!.hostname;
      nextState.namespace = this.state.originalRegistry!.namespace;
    }

    //TODO: understand this better
    // Argument of type '{ [key: string]: string | boolean; }' is not assignable to parameter of type 'State | Pick<State, keyof State>
    // | ((prevState: Readonly<State>, props: Readonly<Props>) => State | Pick<...> | null) | null'.
    // @ts-ignore
    this.setState(nextState, () => {
      if (this.props.outletContext.gatherDetails) {
        const { hostname, username, password, namespace, isReadOnly } =
          this.state;
        this.props.outletContext.gatherDetails({
          hostname,
          username,
          password,
          namespace,
          isReadOnly,
        });
      }
    });
  };

  componentDidUpdate(lastProps: { app: { slug: string } }) {
    const { app } = this.props.outletContext;

    if (app?.slug !== lastProps.outletContext.app?.slug) {
      this.fetchRegistryInfo();
    }
  }

  fetchRegistryInfo = () => {
    if (this.state.loading) {
      return;
    }

    this.setState({
      loading: true,
      fetchRegistryErrMsg: "",
      displayErrorModal: false,
    });

    let url = `${process.env.API_ENDPOINT}/registry`;
    if (this.props.outletContext.app) {
      url = `${process.env.API_ENDPOINT}/app/${this.props.outletContext.app.slug}/registry`;
    }

    fetch(url, {
      credentials: "include",
      method: "GET",
    })
      .then((res) => res.json())
      .then((result) => {
        if (result.success) {
          this.setState({
            originalRegistry: result,
            hostname: result.hostname,
            username: result.username,
            password: result.password,
            namespace: result.namespace,
            isReadOnly: result.isReadOnly,
            loading: false,
            fetchRegistryErrMsg: "",
            displayErrorModal: false,
          });

          if (this.props.outletContext.gatherDetails) {
            const { hostname, username, password, namespace, isReadOnly } =
              result;
            this.props.outletContext.gatherDetails({
              hostname,
              username,
              password,
              namespace,
              isReadOnly,
            });
          }
        } else {
          this.setState({
            loading: false,
            fetchRegistryErrMsg:
              "Unable to get registry info, please try again.",
            displayErrorModal: true,
          });
        }
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          loading: false,
          fetchRegistryErrMsg: err
            ? `Unable to get registry info: ${err.message}`
            : "Something went wrong, please try again.",
          displayErrorModal: true,
        });
      });
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  triggerStatusUpdates = () => {
    let url = `${process.env.API_ENDPOINT}/imagerewritestatus`;
    if (this.props.outletContext.app) {
      url = `${process.env.API_ENDPOINT}/app/${this.props.outletContext.app.slug}/imagerewritestatus`;
    }
    fetch(url, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
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
  };

  updateStatus = () => {
    let url = `${process.env.API_ENDPOINT}/imagerewritestatus`;
    if (this.props.outletContext.app) {
      url = `${process.env.API_ENDPOINT}/app/${this.props.outletContext.app.slug}/imagerewritestatus`;
    }
    return new Promise<void>((resolve, reject) => {
      fetch(url, {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
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

            if (this.props.outletContext.updateCallback) {
              this.props.outletContext.updateCallback();
            }
          }

          resolve();
        })
        .catch((err) => {
          console.log("failed to get rewrite status", err);
          reject();
        });
    });
  };

  render() {
    const {
      app,
      hideTestConnection,
      hideCta,
      namespaceDescription,
      showHostnameAsRequired,
    } = this.props.outletContext;
    const {
      hostname,
      password,
      username,
      namespace,
      isReadOnly,
      lastSync,
      testInProgress,
      testFailed,
      testMessage,
      originalRegistry,
    } = this.state;
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

    const namespaceSubtext =
      namespaceDescription ||
      "Changing the namespace will rewrite all of your airgap images and push them to your registry.";
    const imagePushSubtext = `Selecting this option will disable writing images to the associated registry.
    Images will still be read from this registry when the application is deployed.
    This option should only be selected in environments where an external process is fully responsible for pushing needed images into the associated repository.`;

    // Pushing images in airgap mode is not supported yet
    const disableRegistryFields = app?.isAirgap && !isReadOnly;

    let testStatusText = "";
    if (testInProgress) {
      testStatusText = "Testing...";
    } else if (lastSync) {
      testStatusText = testMessage as string;
    } else {
      // TODO: this will always be displayed when page is refreshed
      testStatusText = "Connection has not been tested";
    }

    const disableSubmitButton = rewriteStatus === "running";
    const disableStopUsingButton =
      rewriteStatus === "running" || originalRegistry?.hostname === "";
    const showProgress = rewriteStatus === "running";
    const showStatusError = rewriteStatus === "failed";

    return (
      <div className="card-item u-padding--15">
        <form>
          <div className="flex u-marginBottom--20">
            <div className="flex1">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">
                Hostname{" "}
                {showHostnameAsRequired && (
                  <span className="u-textColor--error">(Required)</span>
                )}
              </p>
              <p className="u-lineHeight--normal u-fontSize--small help-text-color u-fontWeight--medium u-marginBottom--10">
                Ensure this domain supports the Docker V2 protocol.
              </p>
              <input
                type="text"
                className={`Input ${disableRegistryFields && "is-disabled"}`}
                disabled={disableRegistryFields}
                placeholder="artifactory.some-big-bank.com"
                value={hostname || ""}
                autoComplete=""
                onChange={(e) => {
                  this.handleFormChange("hostname", e.target.value);
                }}
              />
            </div>
          </div>
          <div className="flex u-marginBottom--20">
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Username
              </p>
              <input
                type="text"
                className="Input"
                placeholder="username"
                value={username || ""}
                autoComplete="username"
                onChange={(e) => {
                  this.handleFormChange("username", e.target.value);
                }}
              />
            </div>
            <div className="flex1 u-paddingLeft--5">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Password
              </p>

              <InputField
                autoFocus
                type={"password"}
                className="tw-gap-0"
                placeholder="password"
                value={password || ""}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  this.handleFormChange("password", e.target.value);
                }}
                id={"airgap-registry-password"}
                isFirstChange={this.state.isFirstPasswordChange}
                label={undefined}
                onFocus={undefined}
                onBlur={undefined}
                helperText={undefined}
              />
            </div>
          </div>
          {hideTestConnection ? null : (
            <div className="test-connection-box u-marginBottom--20">
              <div className="flex">
                <div>
                  <button
                    type="button"
                    disabled={this.state.hostname === ""}
                    className="btn secondary"
                    onClick={this.testRegistryConnection}
                  >
                    Test connection
                  </button>
                </div>
                {this.state.pingedEndpoint && (
                  <div className="flex-column justifyContent--center">
                    <p className="u-marginLeft--10 u-fontSize--small u-fontWeight--medium u-textColor--secondary">
                      <Icon
                        icon="check-circle-filled"
                        size={16}
                        className="success-color u-marginRight--5 u-verticalAlign--neg3"
                        color={""}
                        style={{}}
                        disableFill={false}
                        removeInlineStyle={false}
                      />
                      Connected to {this.state.pingedEndpoint}
                    </p>
                  </div>
                )}
              </div>
              {testFailed && !testInProgress ? (
                <p className="u-fontSize--small u-fontWeight--medium u-textColor--error u-marginTop--10 u-lineHeight--normal">
                  {testStatusText}
                </p>
              ) : (
                <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--10 u-lineHeight--normal">
                  {testStatusText}
                </p>
              )}
            </div>
          )}
          <div className="flex u-marginBottom--30">
            <div className="flex1">
              <div className="flex flex1 alignItems--center u-marginBottom--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">
                  Registry Namespace
                </p>
              </div>
              <p className="u-lineHeight--normal u-fontSize--small help-text-color u-fontWeight--medium u-marginBottom--10">
                {namespaceSubtext}
              </p>
              <input
                type="text"
                className={`Input ${disableRegistryFields && "is-disabled"}`}
                placeholder="namespace"
                disabled={disableRegistryFields}
                value={namespace || ""}
                autoComplete=""
                onChange={(e) => {
                  this.handleFormChange("namespace", e.target.value);
                }}
              />
            </div>
          </div>
          <div className="flex u-marginBottom--5">
            <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left">
              <div
                className={`flex-auto flex ${isReadOnly ? "is-active" : ""}`}
              >
                <input
                  type="checkbox"
                  className="u-cursor--pointer"
                  id="ingressEnabled"
                  checked={isReadOnly}
                  onChange={(e) => {
                    this.handleFormChange("isReadOnly", e.target.checked);
                  }}
                />
                <label
                  htmlFor="ingressEnabled"
                  className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                  style={{ marginTop: "2px" }}
                >
                  <div className="flex flex-column u-marginLeft--5 justifyContent--center">
                    <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-marginBottom--5">
                      Disable Pushing Images to Registry
                    </p>
                    <p className="u-lineHeight--normal u-fontSize--small help-text-color u-fontWeight--medium">
                      {imagePushSubtext}
                    </p>
                  </div>
                </label>
              </div>
            </div>
          </div>
        </form>
        {hideCta ? null : (
          <div className="u-paddingTop--10">
            {showProgress ? (
              <div className="u-marginTop--20">
                <Loader size="30" />
                <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--10">
                  {statusText}
                </p>
              </div>
            ) : null}
            {showStatusError ? (
              <p className="u-fontSize--small u-fontWeight--medium u-textColor--error u-marginTop--10">
                {statusText}
              </p>
            ) : null}
            <div className="u-marginTop--20">
              <button
                className="btn primary blue u-marginRight--10"
                disabled={disableSubmitButton}
                onClick={() => {
                  this.onSaveRegistrySettings(false);
                }}
              >
                Save changes
              </button>
              {!this.props.outletContext.app?.isAirgap && (
                <button
                  className="btn secondary blue"
                  disabled={disableStopUsingButton}
                  onClick={() => {
                    this.setState({ showStopUsingWarning: true });
                  }}
                >
                  Stop using registry
                </button>
              )}
            </div>
          </div>
        )}
        {this.state.fetchRegistryErrMsg && (
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={this.state.fetchRegistryErrMsg}
            tryAgain={this.fetchRegistryInfo}
            err="Failed to get registry info"
            loading={this.state.loading}
          />
        )}

        <Modal
          isOpen={this.state.showStopUsingWarning}
          onRequestClose={() => {
            this.setState({ showStopUsingWarning: false });
          }}
          shouldReturnFocusAfterClose={false}
          ariaHideApp={false}
          className="Modal MediumSize"
        >
          <div className="Modal-body">
            <p className="u-fontSize--large u-textColor--primary u-lineHeight--medium u-marginBottom--20">
              This will create a version of {this.props.outletContext.app?.name}{" "}
              without registry settings. Do you want to proceed?
            </p>
            <div className="flex justifyContent--flexEnd">
              <button
                type="button"
                className="btn blue primary u-marginRight--10"
                onClick={() => {
                  this.setState({ showStopUsingWarning: false });
                  this.onSaveRegistrySettings(true);
                }}
              >
                OK
              </button>
              <button
                type="button"
                className="btn blue secondary u-marginRight--10"
                onClick={() => {
                  this.setState({ showStopUsingWarning: false });
                }}
              >
                Cancel
              </button>
            </div>
          </div>
        </Modal>
      </div>
    );
  }
}

/* eslint-disable */
// @ts-ignore
export default withRouter(AirgapRegistrySettings) as any;
/* eslint-enable */
