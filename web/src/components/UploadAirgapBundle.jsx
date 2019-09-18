import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import Dropzone from "react-dropzone";
import isEmpty from "lodash/isEmpty";
import AirgapUploadProgress from "@src/components/AirgapUploadProgress";

import { setAirgapToInstalled } from "../mutations/AppsMutations";
import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/Login.scss";
import AirgapRegistrySettings from "./shared/AirgapRegistrySettings";

class UploadAirgapBundle extends React.Component {
  state = {
    bundleFile: {},
    fileUploading: false,
    registryDetails: {}
  }

  clearFile = () => {
    this.setState({ bundleFile: {} });
  }

  uploadAirgapBundle = async () => {
    const { onUploadSuccess } = this.props;
    this.setState({ fileUploading: true });
      const formData = new FormData();
      formData.append("file", this.state.bundleFile);
      formData.append("registryHost", this.state.registryDetails.hostname);
      formData.append("namespace", this.state.registryDetails.namespace);
      formData.append("username", this.state.registryDetails.username);
      formData.append("password", this.state.registryDetails.password);
      const url = `${window.env.REST_ENDPOINT}/v1/kots/airgap`;
      fetch(url, {
        method: "POST",
        body: formData
      })
      .then(function (result) {
        return result.json();
      })
      .then(onUploadSuccess)
      .then( () => {
        this.props.history.replace("/apps");
      })
      .catch(function (err) {
        this.setState({ fileUploading: false });
        console.log(err);
      }.bind(this));
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
    try {
      await this.props.setAirgapToInstalled(slug);
      this.props.onUploadSuccess();
      this.props.history.push(`/app/${slug}`);
    } catch (error) {
      console.log(error);
    }
  }

  render() {
    const {
      appName,
      logo,
      fetchingMetadata,
    } = this.props;
    const { bundleFile, fileUploading } = this.state;
    const hasFile = bundleFile && !isEmpty(bundleFile);

    if (fileUploading) {
      return <AirgapUploadProgress />;
    }

    return (
      <div className="UploadLicenseFile--wrapper container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
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
              />
            </div>
            <div className="u-marginTop--20 flex">
              <div className={`FileUpload-wrapper flex1 ${hasFile ? "has-file" : ""}`}>
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
            {hasFile &&
              <div className="u-marginTop--10">
                <span className="replicated-link u-fontSize--small" onClick={this.clearFile}>Select a different bundle</span>
              </div>
            }
          </div>
        </div>
        <div className="u-marginTop--20">
          <span className="u-fontSize--small u-color--dustyGray u-fontWeight--medium" onClick={this.handleOnlineInstall}>Optionally you can <span className="replicated-link">download {appName} from the Internet</span></span>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(setAirgapToInstalled, {
    props:({ mutate }) => ({
      setAirgapToInstalled: (slug) => mutate({ variables: { slug } })
    })
  })
)(UploadAirgapBundle);
