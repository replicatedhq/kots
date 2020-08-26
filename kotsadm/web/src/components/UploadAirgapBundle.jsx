import * as React from "react";
import classNames from "classnames";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { Helmet } from "react-helmet";
import Dropzone from "react-dropzone";
import isEmpty from "lodash/isEmpty";
import Modal from "react-modal";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import AirgapUploadProgress from "@src/components/AirgapUploadProgress";
import LicenseUploadProgress from "./LicenseUploadProgress";
import AirgapRegistrySettings from "./shared/AirgapRegistrySettings";
import ErrorModal from "./modals/ErrorModal";
import { Utilities } from "../utilities/utilities";

import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/Login.scss";

const COMMON_ERRORS = {
  "HTTP 401": "Registry credentials are invalid",
  "invalid username/password": "Registry credentials are invalid",
  "no such host": "No such host"
};

class UploadAirgapBundle extends React.Component {
  state = {
    bundleFile: {},
    fileUploading: false,
    registryDetails: {},
    preparingOnlineInstall: false,
    supportBundleCommand: undefined,
    showSupportBundleCommand: false,
    onlineInstallErrorMessage: "",
    viewOnlineInstallErrorMessage: false,
    errorTitle: "",
    errorMsg: "",
    displayErrorModal: false,
  }

  emptyRequiredFields = "Please enter a value for \"Hostname\" and \"Namespace\" fields"
  emptyHostnameErrMessage = "Please enter a value for \"Hostname\" field"
  emptyNamespaceField = "Please enter a value for \"Namespace\" field"

  clearFile = () => {
    this.setState({ bundleFile: {} });
  }

  toggleShowRun = () => {
    this.setState({ showSupportBundleCommand: true });
  }

  uploadAirgapBundle = async () => {
    const { match, showRegistry } = this.props;

    // Reset the airgap upload state
    const resetUrl = `${window.env.API_ENDPOINT}/kots/airgap/reset/${match.params.slug}`;
    try {
      await fetch(resetUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": Utilities.getToken(),
        }
      });
    } catch (error) {
      console.error(error);
      this.setState({
        fileUploading: false,
        uploadSent: 0,
        uploadTotal: 0,
        errorMessage: "An error occurred while uploading your airgap bundle. Please try again"
      });
      return;
    }

    this.setState({
      fileUploading: true,
      errorMessage: "",
      showSupportBundleCommand: false,
      onlineInstallErrorMessage: "",
    });

    if (showRegistry) {
      const { slug } = this.props.match.params;
      if (isEmpty(this.state.registryDetails.hostname) && isEmpty(this.state.registryDetails.namespace)) {
        this.setState({
          fileUploading: false,
          uploadSent: 0,
          uploadTotal: 0,
          errorMessage: this.emptyRequiredFields,
        });
        return;
      }
      if (isEmpty(this.state.registryDetails.hostname)) {
        this.setState({
          fileUploading: false,
          uploadSent: 0,
          uploadTotal: 0,
          errorMessage: this.emptyHostnameErrMessage,
        });
        return;
      }
      if (isEmpty(this.state.registryDetails.namespace)) {
        this.setState({
          fileUploading: false,
          uploadSent: 0,
          uploadTotal: 0,
          errorMessage: this.emptyNamespaceField,
        });
        return;
      }

      let res;
      try {
        res = await fetch(`${window.env.API_ENDPOINT}/app/${slug}/registry/validate`, {
          method: "POST",
          headers: {
            "Authorization": Utilities.getToken(),
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            hostname: this.state.registryDetails.hostname,
            namespace: this.state.registryDetails.namespace,
            username: this.state.registryDetails.username,
            password: this.state.registryDetails.password,          
          }),
        });
      } catch(err) {
        this.setState({
          fileUploading: false,
          uploadSent: 0,
          uploadTotal: 0,
          errorMessage: err,
        });
        return;
      }

      const response = await res.json();
      if (!response.success) {
        let msg = "An error occurred while uploading your airgap bundle. Please try again";
        if (response.error) {
          msg = response.error;
        }
        this.setState({
          fileUploading: false,
          uploadSent: 0,
          uploadTotal: 0,
          errorMessage: msg,
        });
        return;
      }
    }

    const formData = new FormData();
    formData.append("file", this.state.bundleFile);

    if (showRegistry) {
      formData.append("registryHost", this.state.registryDetails.hostname);
      formData.append("namespace", this.state.registryDetails.namespace);
      formData.append("username", this.state.registryDetails.username);
      formData.append("password", this.state.registryDetails.password);
    }

    const url = `${window.env.API_ENDPOINT}/app/airgap`;
    const xhr = new XMLHttpRequest();

    xhr.upload.onprogress = event => {
      const total = event.total;
      const sent = event.loaded;

      this.setState({
        uploadSent: sent,
        uploadTotal: total
      });
    }

    xhr.upload.onerror = () => {
      this.setState({
        fileUploading: false,
        uploadSent: 0,
        uploadTotal: 0,
        errorMessage: "An error occurred while uploading your airgap bundle. Please try again"
      });
    }

    xhr.onloadend = async () => {
      // airgap upload progress will alert us of success
      const response = xhr.response;
      if (xhr.status !== 202) {
        throw new Error(`Error uploading airgap bundle: ${response}`);
      }
    }

    xhr.open("POST", url);
    xhr.setRequestHeader("Authorization", Utilities.getToken());
    xhr.send(formData);
  }

  getRegistryDetails = (fields) => {
    this.setState({
      ...this.state,
      registryDetails: {
        hostname: fields.hostname,
        username: fields.username,
        password: fields.password,
        namespace: fields.namespace
      }
    });
  }

  onDrop = async (files) => {
    this.setState({
      bundleFile: files[0],
      onlineInstallErrorMessage: ""
    });
  }

  moveBar = (count) => {
    const elem = document.getElementById("myBar");
    const percent = count > 3 ? 96 : count * 30;
    if (elem) {
      elem.style.width = percent + "%";
    }
  }

  handleOnlineInstall = async () => {
    const { slug } = this.props.match.params;

    this.setState({
      preparingOnlineInstall: true,
      onlineInstallErrorMessage: ""
    });

    let resumeResult;
    fetch(`${window.env.API_ENDPOINT}/license/resume`, {
      method: "PUT",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        slug,
      }),
    })
    .then(async (result) => {
      resumeResult = await result.json();
    })
    .catch(err => {
      this.setState({
        // TODO: use fewer flags
        fileUploading: false,
        errorMessage: err,
        preparingOnlineInstall: false,
        onlineInstallErrorMessage: err,
      });
      return;
    })

    let count = 0;
    const interval = setInterval(() => {
      if (this.state.onlineInstallErrorMessage.length) {
        clearInterval(interval);
      }
      count++
      this.moveBar(count);
      if (count > 3) {
        if (!resumeResult) {
          return
        }

        clearInterval(interval);

        if (resumeResult.error) {
          this.setState({
            // TODO: use fewer flags
            fileUploading: false,
            errorMessage: resumeResult.error,
            preparingOnlineInstall: false,
            onlineInstallErrorMessage: resumeResult.error,
          });
          return;    
        }

        this.props.onUploadSuccess().then(() => {
          // When successful, refetch all the user's apps with onUploadSuccess
          const hasPreflight = resumeResult.hasPreflight;
          const isConfigurable = resumeResult.isConfigurable;
          if (isConfigurable) {
            this.props.history.replace(`/${slug}/config`);
          } else if (hasPreflight) {
            this.props.history.replace("/preflight");
          } else {
            this.props.history.replace(`/app/${slug}`);
          }
        });
      }
    }, 1000);
  }

  getSupportBundleCommand = async (slug) => {
    const res = await fetch(`${window.env.API_ENDPOINT}/troubleshoot/app/${slug}/supportbundlecommand`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
      }
    });
    if (!res.ok) {
      throw new Error(`Unexpected status code: ${res.status}`);
    }
    const response = await res.json();
    return response.command;
  }

  onProgressError = async (errorMessage) => {
    // Push this setState call to the end of the call stack
    const { slug } = this.props.match.params;

    let supportBundleCommand = "";
    try {
      supportBundleCommand = await this.getSupportBundleCommand(slug);
    } catch (err) {
      this.setState({
        errorTitle: `Failed to get support bundle command`,
        errorMsg: err ? err.message : "Something went wrong, please try again.",
        displayErrorModal: true,
      });
    }

    setTimeout(() => {
      Object.entries(COMMON_ERRORS).forEach(([errorString, message]) => {
        if (errorMessage.includes(errorString)) {
          errorMessage = message;
        }
      });

      this.setState({
        errorMessage,
        fileUploading: false,
        uploadSent: 0,
        uploadTotal: 0,
        supportBundleCommand: supportBundleCommand,
      });
    }, 0);
  }

  onProgressSuccess = async () => {
    const { onUploadSuccess, match } = this.props;

    await onUploadSuccess();

    const app = await this.getApp(match.params.slug);

    if (app?.isConfigurable) {
      this.props.history.replace(`/${app.slug}/config`);
    } else if (app?.hasPreflight) {
      this.props.history.replace(`/preflight`);
    } else {
      this.props.history.replace(`/app/${app.slug}`);
    }
  }

  getApp = async (slug) => {
    try {
      const res = await fetch(`${window.env.API_ENDPOINT}/app/${slug}`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const app = await res.json();
        return app;
      }
    } catch(err) {
      console.log(err);
    }
    return null;
  }

  toggleViewOnlineInstallErrorMessage = () => {
    this.setState({
      viewOnlineInstallErrorMessage: !this.state.viewOnlineInstallErrorMessage
    });
  }

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  }

  render() {
    const {
      appName,
      logo,
      fetchingMetadata,
      showRegistry,
      appsListLength
    } = this.props;

    const {
      bundleFile,
      fileUploading,
      uploadSent,
      uploadTotal,
      errorMessage,
      registryDetails,
      preparingOnlineInstall,
      onlineInstallErrorMessage,
      viewOnlineInstallErrorMessage,
      errorTitle,
      errorMsg,
    } = this.state;

    const hasFile = bundleFile && !isEmpty(bundleFile);

    if (fileUploading) {
      return (
        <AirgapUploadProgress
          total={uploadTotal}
          sent={uploadSent}
          onProgressError={this.onProgressError}
          onProgressSuccess={this.onProgressSuccess}
        />
      );
    }

    let supportBundleCommand;
    if (this.state.supportBundleCommand) {
      supportBundleCommand = this.state.supportBundleCommand.map((part) => {
        return part.replace("API_ADDRESS", window.location.origin);
      });
    }

    let logoUri;
    let applicationName;
    if (appsListLength && appsListLength > 1) {
      logoUri = "https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png";
      applicationName = "";
    } else {
      logoUri = logo;
      applicationName = appName;
    }

    return (
      <div className="UploadLicenseFile--wrapper container flex-column u-overflow--auto u-marginTop--auto u-marginBottom--auto alignItems--center">
        <Helmet>
          <title>{`${applicationName ? `${applicationName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto u-marginTop--20 u-marginBottom--5">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex-column alignItems--center">
              <div className="flex">
                {logo
                  ? <span className="icon brand-login-icon u-marginRight--10" style={{ backgroundImage: `url(${logoUri})` }} />
                  : !fetchingMetadata ? <span className="icon kots-login-icon u-marginRight--10" />
                    : <span style={{ width: "60px", height: "60px" }} />
                }
                <span className="icon airgapBundleIcon" />
              </div>
            </div>
            {preparingOnlineInstall ?
              <div className="flex-column alignItems--center u-marginTop--30">
                <LicenseUploadProgress hideProgressBar={true} />
              </div>
              :
              <div>
                <p className="u-marginTop--10 u-paddingTop--5 u-fontSize--header u-color--tuna u-fontWeight--bold">Install in airgapped environment</p>
                <p className="u-marginTop--10 u-marginTop--5 u-fontSize--large u-textAlign--center u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">
                  {showRegistry ?
                    `To install on an airgapped network, you will need to provide access to a Docker registry. The images ${applicationName?.length > 0 ? `in ${applicationName}` : ""} will be retagged and pushed to the registry that you provide here.`
                    :
                    `To install on an airgapped network, the images ${applicationName?.length > 0 ? `in ${applicationName}` : ""} will be uploaded from the bundle you provide to the cluster.`
                  }
                </p>
                {showRegistry &&
                  <div className="u-marginTop--30">
                    <AirgapRegistrySettings
                      app={null}
                      hideCta={true}
                      hideTestConnection={true}
                      namespaceDescription="What namespace do you want the application images pushed to?"
                      gatherDetails={this.getRegistryDetails}
                      registryDetails={registryDetails}
                      showRequiredFields={errorMessage === this.emptyRequiredFields}
                      showHostnameAsRequired={errorMessage === this.emptyHostnameErrMessage}
                      showNamespaceAsRequired={errorMessage === this.emptyNamespaceField}
                    />
                  </div>
                }
                <div className="u-marginTop--20 flex">
                  <div className={classNames("FileUpload-wrapper", "flex1", {
                    "has-file": hasFile,
                    "has-error": errorMessage
                  })}>
                    <Dropzone
                      className="Dropzone-wrapper"
                      accept=".airgap"
                      onDropAccepted={this.onDrop}
                      multiple={false}
                    >
                      {hasFile ?
                        <div className="has-file-wrapper">
                          <p className="u-fontSize--normal u-fontWeight--medium">{bundleFile.name}</p>
                        </div>
                        :
                        <div className="u-textAlign--center">
                          <p className="u-fontSize--normal u-color--tundora u-fontWeight--medium u-lineHeight--normal">Drag your airgap bundle here or <span className="u-color--astral u-fontWeight--medium u-textDecoration--underlineOnHover">choose a bundle to upload</span></p>
                          <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginTop--10">This will be a .airgap file{applicationName?.length > 0 ? ` ${applicationName} provided` : ""}. Please contact your account rep if you are unable to locate your .airgap file.</p>
                        </div>
                      }
                    </Dropzone>
                  </div>
                  {hasFile &&
                    <div className="flex-auto flex-column u-marginLeft--10 justifyContent--center">
                      <button
                        type="button"
                        className="btn primary large flex-auto"
                        onClick={this.uploadAirgapBundle}
                        disabled={fileUploading || !hasFile}
                      >
                        {fileUploading ? "Uploading" : "Upload airgap bundle"}
                      </button>
                    </div>
                  }
                </div>
                {errorMessage && (
                  <div className="u-marginTop--10">
                    <span className="u-color--chestnut">{errorMessage}</span>
                    {this.state.showSupportBundleCommand ?
                      <div className="u-marginTop--10">
                        <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Run this command in your cluster</h2>
                        <CodeSnippet
                          language="bash"
                          canCopy={true}
                          onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
                        >
                          {supportBundleCommand}
                        </CodeSnippet>
                      </div>
                      : (supportBundleCommand ?
                          <div>
                            <div className="u-marginTop--10">
                              <a href="#" className="replicated-link" onClick={this.toggleShowRun}>Click here</a> to get a command to generate a support bundle.
                            </div>
                          </div>
                          : null 
                      )
                    }
                  </div>
                )}
                {hasFile &&
                  <div className="u-marginTop--10">
                    <span className="replicated-link u-fontSize--small" onClick={this.clearFile}>Select a different bundle</span>
                  </div>
                }
              </div>
            }

          </div>
        </div>
        <div className={classNames("u-marginTop--10 u-textAlign--center", { "u-marginBottom--20": !onlineInstallErrorMessage }, { "u-display--none": preparingOnlineInstall })}>
          <span className="u-fontSize--small u-color--dustyGray u-fontWeight--medium" onClick={this.handleOnlineInstall}>Optionally you can <span className="replicated-link">download {applicationName?.length > 0 ? applicationName : "this application"} from the Internet</span></span>
        </div>
        {onlineInstallErrorMessage && (
          <div className="u-marginTop--10 u-marginBottom--20">
            <span className="u-fontSize--small u-color--chestnut u-marginRight--5 u-fontWeight--bold">Unable to install license</span>
            <span
              className="u-fontSize--small replicated-link"
              onClick={this.toggleViewOnlineInstallErrorMessage}>
              view more
            </span>
          </div>
        )}

        <Modal
          isOpen={viewOnlineInstallErrorMessage}
          onRequestClose={this.toggleViewOnlineInstallErrorMessage}
          contentLabel="Online install error message"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body">
            <div className="ExpandedError--wrapper u-marginTop--10 u-marginBottom--10">
              <p className="u-fontSize--small u-fontWeight--bold u-color--tuna u-marginBottom--5">Error description</p>
              <p className="u-fontSize--small u-color--chestnut">{onlineInstallErrorMessage}</p>
              <p className="u-fontSize--small u-fontWeight--bold u-marginTop--15 u-color--tuna">Run this command to generate a support bundle</p>
              <CodeSnippet
                language="bash"
                canCopy={true}
                onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
              >
                kubectl support-bundle https://kots.io
              </CodeSnippet>
            </div>
            <button type="button" className="btn primary u-marginTop--15" onClick={this.toggleViewOnlineInstallErrorMessage}>Ok, got it!</button>
          </div>
        </Modal>

        {errorMsg &&
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            err={errorTitle}
            errMsg={errorMsg}
          />}
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
)(UploadAirgapBundle);
