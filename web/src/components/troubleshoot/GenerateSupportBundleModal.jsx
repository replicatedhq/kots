import * as React from "react";
import trim from "trim";
import Clipboard from "clipboard";
import { graphql, compose, withApollo } from "react-apollo";
import { Link } from "react-router-dom";
import isEmpty from "lodash/isEmpty";

import { uploadSupportBundle, markSupportBundleUploaded } from "../../mutations/TroubleshootMutations";

import "../../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import Dropzone from "react-dropzone";

class GenerateSupportBundleModal extends React.Component {
  constructor() {
    super();
    this.state = {
      successState: false,
      fileUploading: false,
      bundleS3Url: "",
      newBundleId: "",
      supportBundle: {},
      currentView: "",
      showToast: false,
      copySuccess: false,
      copyMessage: "",
    };
  }

  toggleView(view) {
    this.setState({ currentView: view });
  }

  uploadToS3 = async () => {
    let response;
    this.setState({ fileUploading: true });
    try {
      response = await fetch(this.state.bundleS3Url, {
        method: "PUT",
        body: this.state.supportBundle,
        headers: {
          "Content-Type": "application/tar+gzip",
        },
      });
      await response;
      this.props.markSupportBundleUploaded(this.state.newBundleId)
        .then(async (response) => {
          this.setState({ fileUploading: false });
          if (this.props.submitCallback && typeof this.props.submitCallback === "function") {
            this.props.submitCallback(response.data.markSupportBundleUploaded);
          }
        })
        .catch((err) => {
          console.log(err);
          this.setState({ fileUploading: false });
        })
      return;
    } catch (err) {
      this.setState({ fileUploading: false });
      return;
    }
  }

  getBundleS3Url = (file) => {
    const { watch } = this.props
    this.props.uploadSupportBundle(watch.id, file.size)
      .then((response) => {
        this.setState({
          bundleS3Url: response.data.uploadSupportBundle.uploadUri,
          newBundleId: response.data.uploadSupportBundle.supportBundle.id
        });
      })
      .catch((err) => {
        console.log(err);
      });
  }

  onDrop = (files) => {
    this.getBundleS3Url(files[0]);
    this.setState({
      supportBundle: files[0]
    });
  }

  showCopyToast(message, didCopy) {
    this.setState({
      showToast: didCopy,
      copySuccess: didCopy,
      copyMessage: message
    });
    setTimeout(() => {
      this.setState({
        showToast: false,
        copySuccess: false,
        copyMessage: ""
      });
    }, 3000);
  }

  instantiateCopyAction() {
    let clipboard = new Clipboard(".copy-command");
    clipboard.on("success", () => {
      this.showCopyToast("Command has been copied to your clipboard", true);
    });
    clipboard.on("error", () => {
      this.showCopyToast("Unable to copy, select the text and use 'Command/Ctl + C'", false);
    });
  }

  componentDidMount() {
    this.instantiateCopyAction();
  }

  openReplicatedSupportBundleLink = () => {
    let page = window.open("https://help.replicated.com/docs/native/packaging-an-application/support-bundle/", "_blank");
    page.focus();
    return false;
  }

  render() {
    const { currentView } = this.state;
    const hasFile = this.state.supportBundle && !isEmpty(this.state.supportBundle);
    const bundleGenerateCommand = `docker pull replicated/support-bundle:alpha && \
    docker run -it --rm \
      --name support-bundle \
      --volume $PWD:/out \
      --volume /var/run/docker.sock:/var/run/docker.sock \
      --net host --pid host --workdir /out  \
      -e HTTP_PROXY -e HTTPS_PROXY -e NO_PROXY \
      replicated/support-bundle:alpha \
      generate \
      --endpoint http://172.17.0.1:30065/graphql \
      --watch-id ${this.props.watch.id}`;
    return (
      <div className="console">
        <div id="UploadSupportBundleModal">
          {this.state.successState ?
            <div className="UploadSuccess-wrapper u-textAlign--center">
              <div className="analysis-illustration-wrapper u-marginBottom--20">
                <div className="icon u-analyzingBundleIcon u-position--relative">
                  <div className="icon u-analyzingBundleMagnifyingGlassIcon magifying-glass-animate"></div>
                </div>
              </div>
              <p className="u-fontSize--largest u-fontWeight--bold u-lineHeight--default u-color--tuna u-marginBottom--normal">Your bundle has been uploaded!</p>
              <p className="u-fontSize--normal u-fontWeight--normal u-lineHeight--normal u-color--dustyGray">We've begun and analysis of your support bundle. Check out the analysis page for a breakdown of this support bundle</p>
              <div className="button-wrapper">
                <Link to={`/troubleshoot/analyze/${this.state.newBundleId}`} className="btn primary">View bundle analysis</Link>
              </div>
            </div>
            :
            <div>
              <p className="u-fontSize--largest u-fontWeight--bold u-lineHeight--default u-color--tuna u-marginBottom--small">
                {currentView === "command" 
                  ? "Generate a support bundle"
                  : "Analyze a support bundle"
                }
              </p>
              <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">
                {currentView === "command" 
                  ? "If your customer can’t generate a support bundle from the Replicated Admin Console, this command will generate and optionally upload one from the command line."
                  : "Upload a support bundle from any customer environment to visually analyze the server and receive insights about the server, the network and your application."
                }
              </p>
              <div className="u-marginTop--20">
                {currentView === "command" ?
                  <div>
                    <div className="bundle-command-wrapper">
                      <pre className="language-bash docker-command">
                        <code className="console-code">
                          {/* // TODO: add correct endpoint based on env */}
                          {bundleGenerateCommand}
                        </code>
                      </pre>
                      <textarea value={trim(bundleGenerateCommand)} className="hidden-input" id="docker-command" readOnly={true}></textarea>
                      <div className="u-marginTop--small u-marginBottom--normal">
                        {this.state.showToast ?
                          <span className={`u-color--tuna u-fontSize--small u-fontWeight--medium ${this.state.copySuccess ? "u-color--vidaLoca" : "u-color--chestnut"}`}>{this.state.copyMessage}</span>
                          :
                          <span className="flex-auto u-color--astral u-fontSize--small u-fontWeight--medium u-textDecoration--underlineOnHover copy-command" data-clipboard-target="#docker-command">
                            Copy command
                          </span>
                        }
                      </div>
                    </div>
                  </div>
                  :
                  <div>
                    <div className={`FileUpload-wrapper ${hasFile ? "has-file" : ""}`}>
                      <Dropzone
                        className="Dropzone-wrapper"
                        accept="application/gzip, .gz"
                        onDropAccepted={this.onDrop}
                        multiple={false}
                      >
                        {hasFile ?
                          <p className="u-fontSize--normal u-fontWeight--medium">{this.state.supportBundle.name}</p>
                          :
                          <div className="u-textAlign--center">
                            <span className="icon u-TarFileIcon u-marginBottom--20"></span>
                            <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal">Drag your bundle here or <span className="u-color--astral u-fontWeight--medium u-textDecoration--underlineOnHover">choose a file to upload</span></p>
                            <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginTop--small">This file can be any
                              <span className="u-fontWeight--medium u-color--astral u-textDecoration--underlineOnHover" onClick={this.openReplicatedSupportBundleLink}> Replicated Support Bundle </span>
                            </p>
                          </div>
                        }
                      </Dropzone>
                    </div>
                    <div className="u-marginTop--normal">
                      <div className="FormButton-wrapper flex justifyContent--center">
                        <button
                          type="button"
                          className="btn secondary flex-auto"
                          onClick={this.uploadToS3}
                          disabled={this.state.fileUploading || !hasFile}
                        >
                          {this.state.fileUploading ? "Uploading" : "Upload support bundle"}
                        </button>
                      </div>
                    </div>
                  </div>
                }
                <p
                  className="u-textAlign--center u-marginTop--normal u-fontSize--small u-fontWeight--medium u-color--astral u-textDecoration--underlineOnHover"
                  onClick={() => this.toggleView(currentView === "command" ? "upload" : "command")}>
                    {currentView === "command" 
                    ? "I already generated a support bundle"
                    : "I need to generate a support bundle"
                    }   
                </p>
              </div>
            </div>
          }
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  graphql(uploadSupportBundle, {
    props: ({ mutate }) => ({
      uploadSupportBundle: (watchId, size) => mutate({ variables: { watchId, size } })
    })
  }),
  graphql(markSupportBundleUploaded, {
    props: ({ mutate }) => ({
      markSupportBundleUploaded: (id) => mutate({ variables: { id } })
    })
  })
)(GenerateSupportBundleModal);
