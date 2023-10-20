import { Component } from "react";
import Clipboard from "clipboard";
import isEmpty from "lodash/isEmpty";
import Dropzone from "react-dropzone";
import randomstring from "randomstring";

import "../../scss/components/troubleshoot/UploadSupportBundleModal.scss";

class UploadSupportBundleModal extends Component {
  constructor() {
    super();
    this.state = {
      fileUploading: false,
      supportBundle: {},
      uploadBundleErrMsg: "",
    };
  }

  uploadAndAnalyze = async () => {
    try {
      const { watch } = this.props;
      const bundleId = randomstring.generate({ capitalization: "lowercase" });
      const uploadBundleUrl = `${process.env.API_ENDPOINT}/troubleshoot/${watch.id}/${bundleId}`;

      this.setState({ fileUploading: true, uploadBundleErrMsg: "" });

      const response = await fetch(uploadBundleUrl, {
        method: "PUT",
        body: this.state.supportBundle,
        headers: {
          "Content-Type": "application/tar+gzip",
        },
      });

      if (!response.ok) {
        this.setState({
          fileUploading: false,
          uploadBundleErrMsg: `Unable to upload the bundle: Status ${response.status}`,
        });
        return;
      }
      const analyzedBundle = await response.json();
      this.setState({ fileUploading: false, uploadBundleErrMsg: "" });
      if (this.props.onBundleUploaded) {
        this.props.onBundleUploaded(analyzedBundle.id);
      }
    } catch (err) {
      this.setState({
        fileUploading: false,
        uploadBundleErrMsg: err
          ? `Unable to upload the bundle: ${err.message}`
          : "Something went wrong, please try again.",
      });
    }
  };

  onDrop = (files) => {
    this.setState({ supportBundle: files[0] });
  };

  showCopyToast(message, didCopy) {
    this.setState({
      showToast: didCopy,
      copySuccess: didCopy,
      copyMessage: message,
    });
    setTimeout(() => {
      this.setState({
        showToast: false,
        copySuccess: false,
        copyMessage: "",
      });
    }, 3000);
  }

  instantiateCopyAction() {
    let clipboard = new Clipboard(".copy-command");
    clipboard.on("success", () => {
      this.showCopyToast("Command has been copied to your clipboard", true);
    });
    clipboard.on("error", () => {
      this.showCopyToast(
        "Unable to copy, select the text and use 'Command/Ctl + C'",
        false
      );
    });
  }

  componentDidMount() {
    this.instantiateCopyAction();
  }

  render() {
    const { supportBundle, fileUploading } = this.state;
    const hasFile = supportBundle && !isEmpty(supportBundle);

    return (
      <div className="console">
        <div id="UploadSupportBundleModal">
          <div>
            <p className="u-fontSize--largest u-fontWeight--bold u-lineHeight--default u-textColor--primary u-marginBottom--small">
              Upload a support bundle
            </p>
            <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy u-marginTop--10">
              Upload a support bundle from your environment to visually analyze
              the server and receive insights about the server, the network and
              your application.
            </p>
            {this.state.uploadBundleErrMsg && (
              <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginTop--10 u-marginBottom--10">
                {this.state.uploadBundleErrMsg}
              </p>
            )}
            <div className="u-marginTop--20">
              <div>
                <div
                  className={`FileUpload-wrapper ${hasFile ? "has-file" : ""}`}
                >
                  <Dropzone
                    className="Dropzone-wrapper"
                    accept="application/gzip, .gz"
                    onDropAccepted={this.onDrop}
                    multiple={false}
                  >
                    {hasFile ? (
                      <p className="u-fontSize--normal u-fontWeight--medium">
                        {supportBundle.name}
                      </p>
                    ) : (
                      <div className="u-textAlign--center">
                        <span className="icon u-TarFileIcon u-marginBottom--20"></span>
                        <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal">
                          Drag your bundle here or{" "}
                          <span className="link u-textDecoration--underlineOnHover">
                            choose a file to upload
                          </span>
                        </p>
                      </div>
                    )}
                  </Dropzone>
                </div>
                <div className="u-marginTop--normal">
                  <div className="FormButton-wrapper flex justifyContent--center">
                    <button
                      type="button"
                      className="btn secondary flex-auto"
                      onClick={this.uploadAndAnalyze}
                      disabled={fileUploading || !hasFile}
                    >
                      {fileUploading ? "Uploading" : "Upload support bundle"}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default UploadSupportBundleModal;
