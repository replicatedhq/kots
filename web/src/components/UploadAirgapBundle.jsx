import * as React from "react";
import classNames from "classnames";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import Dropzone from "react-dropzone";
import isEmpty from "lodash/isEmpty";
import AirgapUploadProgress from "@src/components/AirgapUploadProgress";
import { resumeInstallOnline } from "../mutations/AppsMutations";
import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/Login.scss";
import AirgapRegistrySettings from "./shared/AirgapRegistrySettings";
import { Utilities } from "../utilities/utilities";
import Loader from "./shared/Loader";

class UploadAirgapBundle extends React.Component {
  state = {
    bundleFile: {},
    fileUploading: false,
    registryDetails: {},
    preparingOnlineInstall: false
  }

  clearFile = () => {
    this.setState({ bundleFile: {} });
  }

  uploadAirgapBundle = async () => {
    const { onUploadSuccess, match } = this.props;

    // Reset the airgap upload state
    const resetUrl = `${window.env.REST_ENDPOINT}/v1/kots/airgap/reset/${match.params.slug}`;
    await fetch(resetUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Authorization": Utilities.getToken(),
      }
    });

    this.setState({ fileUploading: true, errorMessage: "" });

    const formData = new FormData();
    formData.append("file", this.state.bundleFile);
    formData.append("registryHost", this.state.registryDetails.hostname);
    formData.append("namespace", this.state.registryDetails.namespace);
    formData.append("username", this.state.registryDetails.username);
    formData.append("password", this.state.registryDetails.password);
    const url = `${window.env.REST_ENDPOINT}/v1/kots/airgap`;
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
      const response = xhr.response;
      if (xhr.status === 200) {
        await onUploadSuccess();

        const jsonResponse = JSON.parse(response);

        if (jsonResponse.hasPreflight) {
          this.props.history.replace(`/preflight`);
        } else {
          this.props.history.replace(`/app/${jsonResponse.slug}`);
        }
      } else {
        throw new Error(`Error uploading airgap bundle: ${response}`);
      }
    }

    xhr.open("POST", url);
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
    });
  }

  handleOnlineInstall = async () => {
    const { slug } = this.props.match.params;
    this.setState({
      preparingOnlineInstall: true
    });
    try {
      const resp = await this.props.resumeInstallOnline(slug);
      const hasPreflight = resp?.data?.resumeInstallOnline?.hasPreflight;

      if (hasPreflight) {
        this.props.history.replace("/preflight");
      } else {
        await this.props.onUploadSuccess();
        this.props.history.replace(`/app/${slug}`);
      }

    } catch (error) {
      console.log(error);
      this.setState({
        preparingOnlineInstall: false
      });
    }
  }

  onProgressError = (errorMessage) => {
    // Push this setState call to the end of the call stack
    setTimeout(() => {
      const COMMON_ERRORS = {
        "HTTP 401": "Registry credentials are invalid",
        "invalid username/password": "Registry credentials are invalid",
        "no such host": "No such host"
      };

      Object.entries(COMMON_ERRORS).forEach( ([errorString, message]) => {
        if (errorMessage.includes(errorString)){
          errorMessage = message;
        }
      });

      this.setState({
        errorMessage,
        fileUploading: false,
        uploadSent: 0,
        uploadTotal: 0
      });
    }, 0);
  }

  render() {
    const {
      appName,
      logo,
      fetchingMetadata
    } = this.props;
    
    const {
      bundleFile,
      fileUploading,
      uploadSent,
      uploadTotal,
      errorMessage,
      registryDetails,
      preparingOnlineInstall
    } = this.state;

    const hasFile = bundleFile && !isEmpty(bundleFile);

    if (fileUploading) {
      return (
        <AirgapUploadProgress
          total={uploadTotal}
          sent={uploadSent}
          onProgressError={this.onProgressError}
        />
      );
    }

    return (
      <div className="UploadLicenseFile--wrapper container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <div className="flex-column flex1">
          <div className="LoginBox-wrapper u-flexTabletReflow flex-auto u-marginTop--20">
            <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
              <div className="flex-column alignItems--center">
                <div className="flex">
                  {logo
                  ? <span className="icon brand-login-icon u-marginRight--10" style={{ backgroundImage: `url(${logo})` }} />
                  : !fetchingMetadata ? <span className="icon ship-login-icon u-marginRight--10" />
                  : <span style={{ width: "60px", height: "60px" }} />
                  }
                  <span className="icon airgapBundleIcon" />
                </div>
                <p className="u-marginTop--10 u-paddingTop--5 u-fontSize--header u-color--tuna u-fontWeight--bold">Install in airgapped environment</p>
                <p className="u-marginTop--10 u-marginTop--5 u-fontSize--large u-textAlign--center u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">
                  To install on an airgapped network, you will need to provide access to a Docker registry. The images in {appName} will be retagged and pushed to the registry that you provide here.
                </p>
              </div>
              <div className="u-marginTop--30">
                <AirgapRegistrySettings
                  app={null}
                  hideCta={true}
                  hideTestConnection={true}
                  namespaceDescription="What namespace do you want the application images pushed to?"
                  gatherDetails={this.getRegistryDetails}
                  registryDetails={registryDetails}
                />
              </div>
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
                        <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginTop--10">This will be a .airgap file {appName} provided. Contact them if you are unable to locate a airgap bundle.</p>
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
                </div>
              )}
              {hasFile &&
                <div className="u-marginTop--10">
                  <span className="replicated-link u-fontSize--small" onClick={this.clearFile}>Select a different bundle</span>
                </div>
              }
            </div>
          </div>
          <div className="u-marginTop--20 u-marginBottom--20 u-textAlign--center">
            {preparingOnlineInstall
              ? <Loader size="40" />
              : <span className="u-fontSize--small u-color--dustyGray u-fontWeight--medium" onClick={this.handleOnlineInstall}>Optionally you can <span className="replicated-link">download {appName} from the Internet</span></span>
            }
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(resumeInstallOnline, {
    props:({ mutate }) => ({
      resumeInstallOnline: (slug) => mutate({ variables: { slug } })
    })
  })
)(UploadAirgapBundle);
