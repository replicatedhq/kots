import * as React from "react";
import classNames from "classnames";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { Helmet } from "react-helmet";
import Dropzone from "react-dropzone";
import isEmpty from "lodash/isEmpty";
import Modal from "react-modal";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import AirgapUploadProgress from "@src/components/AirgapUploadProgress";
import { resumeInstallOnline } from "../mutations/AppsMutations";
import LicenseUploadProgress from "./LicenseUploadProgress";
import AirgapRegistrySettings from "./shared/AirgapRegistrySettings";
import { Utilities } from "../utilities/utilities";
import Loader from "./shared/Loader";
import { validateRegistryInfo } from "../queries/UserQueries";
import { getSupportBundleCommand } from "../queries/TroubleshootQueries";
import { getKotsApp } from "../queries/AppsQueries";

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
    viewOnlineInstallErrorMessage: false
  }

  emptyHostnameErrMessage = "Please enter a value for \"Hostname\" field"

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
      if (isEmpty(this.state.registryDetails.hostname)) {
        this.setState({
          fileUploading: false,
          uploadSent: 0,
          uploadTotal: 0,
          errorMessage: this.emptyHostnameErrMessage,
        });
        return;
      }
      try {
        const validated = await this.props.client.query({
          query: validateRegistryInfo,
          variables: {
            slug: slug,
            endpoint: this.state.registryDetails.hostname,
            username: this.state.registryDetails.username,
            password: this.state.registryDetails.password,
            org: this.state.registryDetails.namespace,
          }
        });
        if (validated.data.validateRegistryInfo) {
          this.setState({
            fileUploading: false,
            uploadSent: 0,
            uploadTotal: 0,
            errorMessage: validated.data.validateRegistryInfo,
          });
          return;
        }
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
    }

    const formData = new FormData();
    formData.append("file", this.state.bundleFile);

    if (showRegistry) {
      formData.append("registryHost", this.state.registryDetails.hostname);
      formData.append("namespace", this.state.registryDetails.namespace);
      formData.append("username", this.state.registryDetails.username);
      formData.append("password", this.state.registryDetails.password);
    }

    const url = `${window.env.API_ENDPOINT}/kots/airgap`;
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

  handleOnlineInstall = async () => {
    const { slug } = this.props.match.params;
    this.setState({
      preparingOnlineInstall: true,
      onlineInstallErrorMessage: ""
    });
    try {
      const resp = await this.props.resumeInstallOnline(slug);
      const app = resp?.data?.resumeInstallOnline;
      const hasPreflight = app?.hasPreflight;
      const isConfigurable = app?.isConfigurable;

      await this.props.onUploadSuccess();
      
      if (isConfigurable) {
        this.props.history.replace(`/${slug}/config`);
      } else if (hasPreflight) {
        this.props.history.replace("/preflight");
      } else {
        this.props.history.replace(`/app/${slug}`);
      }

    } catch (err) {
      console.log(err);
      err.graphQLErrors.map(({ msg }) => {
        this.setState({
          preparingOnlineInstall: false,
          onlineInstallErrorMessage: msg
        });
      });
    }
  }

  onProgressError = async (errorMessage) => {
    // Push this setState call to the end of the call stack
    const supportBundleCommand = await this.props.client.query({
      query: getSupportBundleCommand
    });

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
        supportBundleCommand: supportBundleCommand.data.getSupportBundleCommand
      });
    }, 0);
  }

  onProgressSuccess = async () => {
    const { onUploadSuccess, match } = this.props;

    await onUploadSuccess();

    const app = await this.getKotsApp(match.params.slug);

    if (app.isConfigurable) {
      this.props.history.replace(`/${app.slug}/config`);
    } else if (app.hasPreflight) {
      this.props.history.replace(`/preflight`);
    } else {
      this.props.history.replace(`/app/${app.slug}`);
    }
  }

  getKotsApp = (slug) => {
    return new Promise((resolve, reject) => {
      this.props.client.query({
        query: getKotsApp,
        variables: {
          slug: slug,
        },
        fetchPolicy: "no-cache"
      }).then(response => {
        resolve(response.data.getKotsApp);
      }).catch((error) => {
        reject(error);
      });
    });
  }

  toggleViewOnlineInstallErrorMessage = () => {
    this.setState({
      viewOnlineInstallErrorMessage: !this.state.viewOnlineInstallErrorMessage
    });
  }

  render() {
    const {
      appName,
      logo,
      fetchingMetadata,
      showRegistry,
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
      viewOnlineInstallErrorMessage
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

    let supportBundleCommand = this.state.supportBundleCommand;
    if (supportBundleCommand) {
      supportBundleCommand = supportBundleCommand.replace("API_ADDRESS", window.location.origin);
    }

    return (
      <div className="UploadLicenseFile--wrapper container flex-column u-overflow--auto u-marginTop--auto u-marginBottom--auto alignItems--center">
        <Helmet>
          <title>{`${appName ? `${appName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto u-marginTop--20 u-marginBottom--5">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex-column alignItems--center">
              <div className="flex">
                {logo
                  ? <span className="icon brand-login-icon u-marginRight--10" style={{ backgroundImage: `url(${logo})` }} />
                  : !fetchingMetadata ? <span className="icon kots-login-icon u-marginRight--10" />
                    : <span style={{ width: "60px", height: "60px" }} />
                }
                <span className="icon airgapBundleIcon" />
              </div>
            </div>
            {preparingOnlineInstall ?
              <div className="flex-column alignItems--center u-marginTop--30">
                <Loader size="40" />
                <LicenseUploadProgress />
              </div>
            :
            <div>
              <p className="u-marginTop--10 u-paddingTop--5 u-fontSize--header u-color--tuna u-fontWeight--bold">Install in airgapped environment</p>
              <p className="u-marginTop--10 u-marginTop--5 u-fontSize--large u-textAlign--center u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">
                {showRegistry ?
                  `To install on an airgapped network, you will need to provide access to a Docker registry. The images ${appName ? `in ${appName}` : ""} will be retagged and pushed to the registry that you provide here.`
                  :
                  `To install on an airgapped network, the images ${appName ? `in ${appName}` : ""} will be uploaded from the bundle you provide to the cluster.`
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
                  showHostnameAsRequired={errorMessage === this.emptyHostnameErrMessage}
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
                        <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginTop--10">This will be a .airgap file {appName} provided. Please contact your account rep if you are unable to locate your .airgap file.</p>
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
                        {supportBundleCommand.split("\n")}
                      </CodeSnippet>
                    </div>
                    :
                    <div>
                      <div className="u-marginTop--10">
                        <a href="#" className="replicated-link" onClick={this.toggleShowRun}>Click here</a> to get a command to generate a support bundle.
                      </div>
                    </div>
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
        <div className={classNames("u-marginTop--10 u-textAlign--center", { "u-marginBottom--20": !onlineInstallErrorMessage }, {"u-display--none": preparingOnlineInstall})}>
          <span className="u-fontSize--small u-color--dustyGray u-fontWeight--medium" onClick={this.handleOnlineInstall}>Optionally you can <span className="replicated-link">download {appName} from the Internet</span></span>
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
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(resumeInstallOnline, {
    props: ({ mutate }) => ({
      resumeInstallOnline: (slug) => mutate({ variables: { slug } })
    })
  })
)(UploadAirgapBundle);
