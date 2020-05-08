import * as React from "react";
import { compose } from "react-apollo";
import { withRouter } from "react-router-dom";
import { Helmet } from "react-helmet";
import Dropzone from "react-dropzone";
import isEmpty from "lodash/isEmpty";
import Modal from "react-modal";
import { getFileContent, Utilities } from "../utilities/utilities";
import CodeSnippet from "./shared/CodeSnippet";
import LicenseUploadProgress from "./LicenseUploadProgress";

import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/Login.scss";

class UploadLicenseFile extends React.Component {
  state = {
    licenseFile: {},
    licenseFileContent: null,
    fileUploading: false,
    errorMessage: "",
    viewErrorMessage: false
  }

  clearFile = () => {
    this.setState({ licenseFile: {}, licenseFileContent: null, errorMessage: "", viewErrorMessage: false });
  }

  moveBar = (count) => {
    const elem = document.getElementById("myBar");
    const percent = count > 3 ? 96 : count * 30;
    if (elem) {
      elem.style.width = percent + "%";
    }
  }

  uploadLicenseFile = async () => {
    const { onUploadSuccess, history } = this.props;
    const { licenseFile, licenseFileContent } = this.state;

    let licenseText;
    if (licenseFile.name.substr(licenseFile.name.lastIndexOf('.')) === ".rli") {
      try {
        const base64String = btoa(String.fromCharCode.apply(null, new Uint8Array(licenseFileContent)));
        licenseText = await this.exchangeRliFileForLicense(base64String);
      } catch(err) {
        this.setState({
          fileUploading: false,
          errorMessage: err,
        });
        return;
      }
    } else {
      licenseText = (new TextDecoder("utf-8")).decode(licenseFileContent);
    }

    this.setState({
      fileUploading: true,
      errorMessage: "",
    });

    let data;
    fetch(`${window.env.API_ENDPOINT}/license`, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        licenseData: licenseText,
      }),
    })
    .then(async (result) => {
      data = await result.json();
    })
    .catch(err => {
      this.setState({
        fileUploading: false,
        errorMessage: err,
      });
      return;
    })

    let count = 0;
    const interval = setInterval(() => {
      if (this.state.errorMessage.length) {
        clearInterval(interval);
      }
      count++
      this.moveBar(count);
      if (count > 3) {
        if (data) {
          clearInterval(interval);
          // When successful, refetch all the user's apps with onUploadSuccess
          onUploadSuccess().then(() => {
            if (data.isAirgap) {
              if (data.needsRegistry) {
                history.replace(`/${data.slug}/airgap`);
              } else {
                history.replace(`/${data.slug}/airgap-bundle`);
              }
              return;
            }

            if (data.isConfigurable) {
              history.replace(`/${data.slug}/config`);
              return;
            }

            if (data.hasPreflight) {
              fetch(`${window.env.API_ENDPOINT}/app/${data.slug}/preflight/run`, {
                headers: {
                  "Content-Type": "application/json",
                  "Accept": "application/json",
                  "Authorization": Utilities.getToken(),
                },
                method: "POST",
              })
                .then(async (res) => {
                  history.replace("/preflight");
                })
                .catch((err) => {
                  // TODO: UI for this error
                  console.log(err);
                });
              return;
            }

            // No airgap, config or preflight? Go to the kotsApp detail view that was just uploaded
            if (data) {
              history.replace(`/app/${data.slug}`);
            }
          });
        }
      }
    }, 1000);
  }

  onDrop = async (files) => {
    const content = await getFileContent(files[0]);
    this.setState({
      licenseFile: files[0],
      licenseFileContent: content,
      errorMessage: ""
    });
  }

  exchangeRliFileForLicense = async (content) => {
    return new Promise((resolve, reject) => {
      const payload = {
        licenseData: content,
      };

      fetch(`${window.env.API_ENDPOINT}/license/platform`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Accept": "application/json",
          "Authorization": Utilities.getToken(),
        },
        body: JSON.stringify(payload),
      })
        .then(async (res) => {
          if (!res.ok) {
            reject(res.status === 401 ? "Invalid license. Please try again" : "There was an error uploading your license. Please try again");
            return;
          }
          resolve((await res.json()).licenseData);
        })
        .catch((err) => {
          console.log(err);
          reject("There was an error uploading your license. Please try again");
        });
    });
  }

  toggleViewErrorMessage = () => {
    this.setState({
      viewErrorMessage: !this.state.viewErrorMessage
    });
  }

  render() {
    const {
      appName,
      logo,
      fetchingMetadata,
      appsListLength,
    } = this.props;
    const { licenseFile, fileUploading, errorMessage, viewErrorMessage } = this.state;
    const hasFile = licenseFile && !isEmpty(licenseFile);

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
      <div className="UploadLicenseFile--wrapper container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <Helmet>
          <title>{`${applicationName ? `${applicationName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex-column alignItems--center">
              {logo
              ? <span className="icon brand-login-icon" style={{ backgroundImage: `url(${logoUri})` }} />
              : !fetchingMetadata ? <span className="icon kots-login-icon" />
              : <span style={{ width: "60px", height: "60px" }} />
              }
            </div>
            {!fileUploading ?
              <div className="flex-column">
                <p className="u-marginTop--10 u-paddingTop--5 u-fontSize--header u-color--tuna u-fontWeight--bold u-textAlign--center">Upload your license file</p>
                <div className="u-marginTop--30 flex">
                  <div className={`FileUpload-wrapper flex1 ${hasFile ? "has-file" : ""}`}>
                    <Dropzone
                      className="Dropzone-wrapper"
                      accept={["application/x-yaml", ".yaml", ".yml", ".rli"]}
                      onDropAccepted={this.onDrop}
                      multiple={false}
                    >
                      {hasFile ?
                        <div className="has-file-wrapper">
                          <p className="u-fontSize--normal u-fontWeight--medium">{licenseFile.name}</p>
                        </div>
                        :
                        <div className="u-textAlign--center">
                          <p className="u-fontSize--normal u-color--tundora u-fontWeight--medium u-lineHeight--normal">Drag your license here or <span className="u-color--astral u-fontWeight--medium u-textDecoration--underlineOnHover">choose a file to upload</span></p>
                          <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginTop--10">This will be a .yaml file{applicationName?.length > 0 ? ` ${applicationName} provided` : ""}. Please contact your account rep if you are unable to locate your license file.</p>
                        </div>
                      }
                    </Dropzone>
                  </div>
                  {hasFile &&
                    <div className="flex-auto flex-column u-marginLeft--10 justifyContent--center">
                      <button
                        type="button"
                        className="btn primary large flex-auto"
                        onClick={this.uploadLicenseFile}
                        disabled={fileUploading || !hasFile}
                      >
                        {fileUploading ? "Uploading" : "Upload license"}
                      </button>
                    </div>
                  }
                </div>
                {errorMessage && (
                  <div className="u-marginTop--10">
                    <span className="u-fontSize--small u-color--chestnut u-marginRight--5 u-fontWeight--bold">Unable to install license</span>
                    <span
                      className="u-fontSize--small replicated-link"
                      onClick={this.toggleViewErrorMessage}>
                      view more
                    </span>
                  </div>
                )}
                {hasFile &&
                  <div className="u-marginTop--10">
                    <span className="replicated-link u-fontSize--small" onClick={this.clearFile}>Select a different file</span>
                  </div>
                }
              </div>
              :
              <div><LicenseUploadProgress /></div>
            }
          </div>
        </div>

        <Modal
          isOpen={viewErrorMessage}
          onRequestClose={this.toggleViewErrorMessage}
          contentLabel="Online install error message"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body">
            <div className="ExpandedError--wrapper u-marginTop--10 u-marginBottom--10">
              <p className="u-fontSize--small u-fontWeight--bold u-color--tuna u-marginBottom--5">Error description</p>
              <p className="u-fontSize--small u-color--chestnut">{typeof errorMessage === "object" ? "An unknown error orrcured while trying to upload your license. Please try again." : errorMessage}</p>
              <p className="u-fontSize--small u-fontWeight--bold u-marginTop--15 u-color--tuna">Run this command to generate a support bundle</p>
              <CodeSnippet
                  language="bash"
                  canCopy={true}
                  onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
                >
                kubectl support-bundle https://kots.io
              </CodeSnippet>
            </div>
            <button type="button" className="btn primary u-marginTop--15" onClick={this.toggleViewErrorMessage}>Ok, got it!</button>
          </div>
        </Modal>
      </div>
    );
  }
}

export default compose(
  withRouter,
)(UploadLicenseFile);
