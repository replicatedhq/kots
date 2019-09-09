import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import Dropzone from "react-dropzone";
import isEmpty from "lodash/isEmpty";
import { getAirgapPutUrl, markAirgapBundleUploaded } from "../mutations/AppsMutations";

import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/Login.scss";
import AirgapRegistrySettings from "./shared/AirgapRegistrySettings";

class UploadAirgapBundle extends React.Component {
  state = {
    bundleFile: {},
    fileUploading: false,
    filePutUrl: "",
    registryDetails: {}
  }

  clearFile = () => {
    this.setState({ bundleFile: {} });
  }

  uploadAirgapBundle = async () => {
    // const { onUploadSuccess } = this.props;
    this.setState({ fileUploading: true });
    try {
      let response = await fetch(this.state.filePutUrl, {
        method: "PUT",
        body: this.state.bundleFile,
        headers: {
          "Content-Type": "application/tar+gzip",
        },
      });
      await response;


      this.props.markAirgapBundleUploaded(
        this.state.bundleFile.name,
        this.state.registryDetails.hostname,
        this.state.registryDetails.namespace,
        this.state.registryDetails.username,
        this.state.registryDetails.password)
        .then(async () => {
          this.setState({ fileUploading: false });
          // this.props.history.replace(`/app/${response[0].slug}`);


          // if (this.props.submitCallback && typeof this.props.submitCallback === "function") {
          //   this.props.submitCallback(response.data.markSupportBundleUploaded.id);
          // }
        })
        .catch(() => {
          this.setState({ fileUploading: false });
        })

      // onUploadSuccess().then((res) => {
      //   this.props.history.replace(`/app/${res[0].slug}`);
      // });
    } catch (err) {
      this.setState({ fileUploading: false });
      console.log(err);
    }
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
    const file = files[0];
    this.props.getAirgapPutUrl(file.name)
      .then((response) => {
        this.setState({
          bundleFile: file,
          filePutUrl: response.data.getAirgapPutUrl,
        });
      })
      .catch((err) => {
        console.log(err);
      });
  }

  render() {
    const {
      appName,
      logo,
    } = this.props;
    const { bundleFile, fileUploading } = this.state;
    const hasFile = bundleFile && !isEmpty(bundleFile);

    return (
      <div className="UploadLicenseFile--wrapper container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex-column alignItems--center">
              <div className="flex">
                {logo ?
                  <span className="icon brand-login-icon u-marginRight--10" style={{ backgroundImage: `url(${logo})` }} />
                :
                  <span className="icon ship-login-icon u-marginRight--10" />
                }
                <span className="icon airgapBundleIcon" />
              </div>
              <p className="u-marginTop--10 u-paddingTop--5 u-fontSize--header u-color--tuna u-fontWeight--bold">Install in airgapped environment</p>
            </div>
            <div className="u-marginTop--30">
              <AirgapRegistrySettings
                app={null}
                hideCta={true}
                hideTestConnection={true}
                gatherDetails={this.getRegistryDetails}
              />
            </div>
            <div className="u-marginTop--20 flex">
              <div className={`FileUpload-wrapper flex1 ${hasFile ? "has-file" : ""}`}>
                <Dropzone
                  className="Dropzone-wrapper"
                  accept="application/gzip, .gz"
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
                      <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginTop--10">This will be a .tar.gz file {appName} provided. Contact them if you are unable to locate a airgap bundle.</p>
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
          <button className="btn secondary green large">Download {appName} from the internet</button>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(getAirgapPutUrl, {
    props: ({ mutate }) => ({
      getAirgapPutUrl: (filename) => mutate({ variables: { filename } })
    })
  }),
  graphql(markAirgapBundleUploaded, {
    props: ({ mutate }) => ({
      markAirgapBundleUploaded: (filename, registryHost, registryNamespace, username, password) => mutate({ variables: { filename, registryHost, registryNamespace, username, password } })
    })
  })
)(UploadAirgapBundle);
