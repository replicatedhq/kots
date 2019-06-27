import * as React from "react";
import autoBind from "react-autobind";
import trim from "trim";
import Clipboard from "clipboard";
import { graphql, compose, withApollo } from "react-apollo";
import { Link } from "react-router-dom";
import isEmpty from "lodash/isEmpty";

import { getSupportBundles, getGenerateSupportBundleCommand } from "../../queries/SupportBundleQueries";
import { uploadSupportBundle, markSupportBundleUploaded } from "../../mutations/SupportBundleMutations";
import { getAccount } from "../../queries/AccountQuery";
import { getAppChannels } from "../../queries/ShipQueries";

import "../../scss/components/modals/UploadSupportBundleModal.scss";
import Dropzone from "react-dropzone";
import Select from "react-select";


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
      bundleGenerateCommand: "",
      showToast: false,
      copySuccess: false,
      copyMessage: "",
      pageSize: 1000,
      page: 0,
      selectedApp: "",
      shipChannels: [],
      selectedChannel: ""
    };
    autoBind(this);
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
          await this.props.getSupportBundles.refetch();
          await this.props.accountQuery.refetch();
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
    let appId;
    if (this.props.currentApp) {
      appId = this.props.currentApp.id || this.props.currentApp.Id;
    } else {
      appId = null;
    }
    this.props.uploadSupportBundle(appId, file.size)
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

  componentDidUpdate(lastProps, lastState) {
    if (this.props.selectedApp !== lastProps.selectedApp && this.props.selectedApp) {
      if (this.props.apps) {
        this.setState({ selectedApp: this.props.apps[this.props.selectedApp]})
      }
    }

    if (this.state.selectedApp !== lastState.selectedApp) {
      if (this.state.selectedApp) {
        if (this.state.selectedApp.Id) {
          this.setState({ selectedChannel: "" })
          this.props.getChannelList(this.state.selectedApp.Id);
        } else {
          this.props.client.query({
            query: getAppChannels,
            name: "getAppChannels",
            variables: { appId: this.state.selectedApp.id }
          })
            .then((res) => {
              this.setState({ selectedChannel: "" })
              this.setState({ shipChannels: res.data.getAppChannels });
            }).catch();
        }
      }
    }

    if (this.state.selectedChannel !== lastState.selectedChannel) {
      this.props.client.query({
        query: getGenerateSupportBundleCommand,
        variables: { channelId: this.state.selectedChannel.Id || this.state.selectedChannel.id }
      })
        .then((res) => {
          this.setState({ bundleGenerateCommand: res.data.getGenerateSupportBundleCommand });
        })
    }
  }

  componentDidMount() {
    if (this.props.selectedApp && this.props.apps) {
      this.setState({ selectedApp: this.props.apps[this.props.selectedApp]})
    }


    this.instantiateCopyAction();
  }

  openReplicatedSupportBundleLink = () => {
    let page = window.open("https://help.replicated.com/docs/native/packaging-an-application/support-bundle/", "_blank");
    page.focus();
    return false;
  }

  onAppChange = (selectedApp) => {
    this.setState({ selectedApp });
  }

  onChannelChange = (selectedChannel) => {
    this.setState({ selectedChannel });
  }


  render() {
    const hasFile = this.state.supportBundle && !isEmpty(this.state.supportBundle);

    const apps = Object.keys(this.props.apps).map((key) => this.props.apps[key]);
    const sortedApps = apps.sort((a, b) => {
      const aName = a.Name || a.name;
      const bName = b.name || b.name;
      return aName.localeCompare(bName);
    });
    const selectValue = this.state.selectedApp;

    const channels = this.state.selectedApp && this.state.selectedApp.Id && Object.values(this.props.channels) || this.state.shipChannels;
    const selectChannelValue = this.state.selectedChannel;


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
              {this.state.currentView === "command" ?
                <div>
                  <p className="u-fontSize--largest u-fontWeight--bold u-lineHeight--default u-color--tuna u-marginBottom--small">Generate a support bundle</p>
                  <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">If your customer can’t generate a support bundle from the Replicated Admin Console, this command will generate and optionally upload one from the command line.</p>
                </div>
                :
                <div>
                  <p className="u-fontSize--largest u-fontWeight--bold u-lineHeight--default u-color--tuna u-marginBottom--small">Analyze a support bundle</p>
                  <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">Upload a support bundle from any customer environment to visually analyze the server and receive insights about the server, the network and your application.</p>
                </div>

              }
              <div className="u-marginTop--20">
                {this.state.currentView === "command" ?
                  <div>
                    <div className="u-marginTop--20">
                      <div className="u-position--relative u-marginBottom--normal flex flex-row">
                        <div className="flex flex1 flex-column u-marginRight--30">
                          <p className="Form-label">Select an application <span className="required-label">*</span> </p>
                          {sortedApps &&
                            <div className="UploadBundleSelect--wrapper">
                              <div className="UploadBundleSelect-menu">
                                <Select
                                  options={sortedApps}
                                  getOptionLabel={(sortedApp) => sortedApp.Name || sortedApp.name}
                                  value={selectValue}
                                  onChange={this.onAppChange}
                                  isOptionSelected={() => false}
                                />
                              </div>
                            </div>}
                        </div>
                        <div className="flex flex1 flex-column">
                          <p className="Form-label">Select a channel <span className="required-label">*</span> </p>
                          {channels &&
                            <div className="UploadBundleSelect--wrapper">
                              <div className="UploadBundleSelect-menu">
                                <Select
                                  options={channels}
                                  getOptionLabel={(channel) => channel.Name || channel.name}
                                  value={selectChannelValue}
                                  onChange={this.onChannelChange}
                                  isOptionSelected={() => false}
                                />
                              </div>
                            </div>}
                        </div>
                      </div>
                    </div>
                    {this.state.bundleGenerateCommand ?
                      <div className="bundle-command-wrapper">
                        <pre className="language-bash docker-command">
                          <code className="console-code">{this.state.bundleGenerateCommand}</code>
                        </pre>
                        <textarea value={trim(this.state.bundleGenerateCommand)} className="hidden-input" id="docker-command" readOnly={true}></textarea>
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
                      :
                      <div className="InitialCommand--wrapper">
                        <div className="InitialCommand-text justifyContent--center">
                            To get your command to generate a support bundle, select the Application and the Channel that contains the release that you want to troubleshoot.
                        </div>
                      </div>
                    }
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
                          className="Button primary button flex-auto"
                          onClick={this.uploadToS3}
                          disabled={this.state.fileUploading || !hasFile}
                        >
                          {this.state.fileUploading ? "Uploading" : "Upload support bundle"}
                        </button>
                      </div>
                    </div>
                  </div>
                }
                {this.state.currentView === "command" ?
                  <p className="u-textAlign--center u-marginTop--normal u-fontSize--small u-fontWeight--medium u-color--astral u-textDecoration--underlineOnHover" onClick={() => this.toggleView("upload")}>I have a support bundle from my customer</p>
                  :
                  <p className="u-textAlign--center u-marginTop--normal u-fontSize--small u-fontWeight--medium u-color--astral u-textDecoration--underlineOnHover" onClick={() => this.toggleView("command")}>My customer can’t create a support bundle</p>
                }
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
  graphql(getSupportBundles, { name: "getSupportBundles" }),
  graphql(getAccount, { name: "accountQuery" }),
  graphql(uploadSupportBundle, {
    props: ({ mutate }) => ({
      uploadSupportBundle: (appId, size) => mutate({ variables: { appId, size } })
    })
  }),
  graphql(markSupportBundleUploaded, {
    props: ({ mutate }) => ({
      markSupportBundleUploaded: (id) => mutate({ variables: { id } })
    })
  })
)(GenerateSupportBundleModal);
