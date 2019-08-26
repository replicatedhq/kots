import * as React from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import Dropzone from "react-dropzone";
import isEmpty from "lodash/isEmpty";

import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/Login.scss";

class UploadLicenseFile extends React.Component {
  state = {
    licenseFile: {},
    fileUploading: false
  }

  clearFile = () => {
    this.setState({ licenseFile: {} });
  }

  uploadToS3 = async () => {
    const { licenseFile } = this.state;
    this.setState({ fileUploading: true });
    try {
      await this.props.uploadLicense(licenseFile);
    } catch (err) {
      this.setState({ fileUploading: false });
      console.log(err);
    }
  }

  onDrop = (files) => {
    this.setState({
      licenseFile: files[0]
    });
  }

  render() {
    const { licenseFile, fileUploading } = this.state;
    const hasFile = licenseFile && !isEmpty(licenseFile);

    return (
      <div className="UploadLicenseFile--wrapper container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex">
              <span className="icon ship-login-icon"></span>
              <p className="login-text u-color--tuna u-fontWeight--bold">Upload your license file</p>
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
                      <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginTop--10">This will be a .yaml file your vendor provided. Contact them if you are unable to locate a license file.</p>
                    </div>
                  }
                </Dropzone>
              </div>
              {hasFile && 
                <div className="flex-auto flex-column u-marginLeft--10 justifyContent--center">
                  <button
                    type="button"
                    className="btn primary large flex-auto"
                    onClick={this.uploadToS3}
                    disabled={fileUploading || !hasFile}
                  >
                    {fileUploading ? "Uploading" : "Upload license"}
                  </button>
                </div>
              }
            </div>
            {hasFile &&
              <div className="u-marginTop--10">
                <p className="replicated-link u-fontSize--small" onClick={this.clearFile}>Select a different file</p>
              </div>
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
)(UploadLicenseFile);
