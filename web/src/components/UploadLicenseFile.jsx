import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { Helmet } from "react-helmet";
import Dropzone from "react-dropzone";
import isEmpty from "lodash/isEmpty";
import Modal from "react-modal";
import { uploadKotsLicense } from "../mutations/AppsMutations";
import { getFileContent } from "../utilities/utilities";
import CodeSnippet from "./shared/CodeSnippet";

import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/Login.scss";

class UploadLicenseFile extends React.Component {
  state = {
    licenseFile: {},
    licenseValue: "",
    fileUploading: false,
    errorMessage: "",
    viewErrorMessage: false
  }

  clearFile = () => {
    this.setState({ licenseFile: {}, licenseValue: "", errorMessage: "", viewErrorMessage: false });
  }

  uploadLicenseFile = async () => {
    const { onUploadSuccess, history } = this.props;
    const { licenseValue } = this.state;

    this.setState({ fileUploading: true, errorMessage: "" });
    try {
      const resp = await this.props.uploadKotsLicense(licenseValue);
      const data = resp.data.uploadKotsLicense;

      // When successful, refetch all the user's apps with onUploadSuccess
      onUploadSuccess().then(() => {
        if (data?.isAirgap) {
          if (data?.needsRegistry) {
            history.replace(`/${data.slug}/airgap`);
          } else {
            history.replace(`/${data.slug}/airgap-bundle`);
          }
          return;
        }

        if (data?.isConfigurable) {
          history.replace(`/${data.slug}/config`);
          return;
        }

        if (data?.hasPreflight) {
          history.replace("/preflight");
          return;
        }

        // No airgap, config or preflight? Go to the kotsApp detail view that was just uploaded
        if (data) {
          history.replace(`/app/${data.slug}`);
        }
      });
    } catch (err) {
      console.log(err);
      err.graphQLErrors.map(({ msg }) => {
        this.setState({ fileUploading: false, errorMessage: msg });
      });
    }
  }

  onDrop = async (files) => {
    const content = await getFileContent(files[0]);
    this.setState({
      licenseFile: files[0],
      licenseValue: content,
      errorMessage: ""
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
    } = this.props;
    const { licenseFile, fileUploading, errorMessage, viewErrorMessage } = this.state;
    const hasFile = licenseFile && !isEmpty(licenseFile);

    return (
      <div className="UploadLicenseFile--wrapper container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <Helmet>
          <title>{`${appName ? `${appName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex-column alignItems--center">
              {logo
              ? <span className="icon brand-login-icon" style={{ backgroundImage: `url(${logo})` }} />
              : !fetchingMetadata ? <span className="icon kots-login-icon" />
              : <span style={{ width: "60px", height: "60px" }} />
              }
              <p className="u-marginTop--10 u-paddingTop--5 u-fontSize--header u-color--tuna u-fontWeight--bold">Upload your license file</p>
            </div>
            <div className="u-marginTop--30 flex">
              <div className={`FileUpload-wrapper flex1 ${hasFile ? "has-file" : ""}`}>
                <Dropzone
                  className="Dropzone-wrapper"
                  accept={["application/x-yaml", ".yaml", ".yml"]}
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
                      <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginTop--10">This will be a .yaml file {appName} provided. Please contact your account rep if you are unable to locate your license file.</p>
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
                  className="u-fontSize--small u-color--astral u-cursor--pointer u-fontWeight--bold u-textDecoration--underline"
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
              <p className="u-fontSize--small u-color--chestnut">{errorMessage}</p>
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
  withApollo,
  graphql(uploadKotsLicense, {
    props: ({ mutate }) => ({
      uploadKotsLicense: (value) => mutate({ variables: { value } })
    })
  }),
)(UploadLicenseFile);
