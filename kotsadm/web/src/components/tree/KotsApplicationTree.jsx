import * as React from "react";
import { compose } from "react-apollo";
import { withRouter } from "react-router-dom";
import { getFileFormat, Utilities } from "../../utilities/utilities";
import sortBy from "lodash/sortBy";
import find from "lodash/find";
import keys from "lodash/keys";
import MonacoEditor from "react-monaco-editor";
import Modal from "react-modal";
import CodeSnippet from "../shared/CodeSnippet";

import Loader from "../shared/Loader";
import FileTree from "../shared/FileTree";

import "../../scss/components/troubleshoot/FileTree.scss";

class KotsApplicationTree extends React.Component {
  constructor() {
    super();
    this.state = {
      files: [],
      selectedFile: "/",
      line: null,
      activeMarkers: [],
      analysisError: false,
      displayInstructionsModal: false,
      applicationTree: {},
    };
  }

  setFileTree = () => {
    if (!this.state.fileTree) { return; }
    const parsedTree = JSON.parse(this.state.fileTree);
    let sortedTree = sortBy(parsedTree, (dir) => {
      dir.children ? dir.children.length : []
    });
    this.setState({ files: sortedTree });
  }

  fetchApplicationTree = () => {
    const url = `${window.env.API_ENDPOINT}/app/${this.props.match.params.slug}/sequence/${this.props.match.params.sequence}/contents`;
    fetch(url, {
      headers: {
        "Authorization": Utilities.getToken()
      },
      method: "GET",
    })
    .then(res => res.json())
    .then(async (files) => {
      this.setState({applicationTree: files});
    })
    .catch((err) => {
      throw err;
    });
  }

  compoenntDidUpdate(lastProps, lastState) {
    if (this.props.match.params.slug != lastProps.match.params.slug || this.props.match.params.sequence != lastProps.match.params.sequence) {
      this.fetchApplicationTree();
    }
  }

  componentDidMount() {
    this.fetchApplicationTree();
  }

  toggleInstructionsModal = () => {
    this.setState({
      displayInstructionsModal: !this.state.displayInstructionsModal,
    });
  }

  setSelectedFile = (path) => {
    this.setState({
      selectedFile: path,
    });
  }

  back = () => {
    this.props.history.goBack();
  }

  render() {
    const { fileLoadErr, fileLoadErrMessage, displayInstructionsModal } = this.state;

    const file = this.state.applicationTree.files ? this.state.applicationTree.files[this.state.selectedFile] : "";
    const contents = file ? new Buffer(file, "base64").toString() : "";

    return (
      <div className="flex-column flex1 ApplicationTree--wrapper container u-paddingTop--50 u-paddingBottom--30">
        <div className="edit-files-banner u-fontSize--small u-fontWeight--medium">Need to edit these files? <span onClick={this.toggleInstructionsModal} className="u-textDecoration--underline u-fontWeight--bold u-cursor--pointer">Click here</span> to learn how</div>
        <div className="flex flex1">
          <div className="flex1 dirtree-wrapper flex-column u-overflow-hidden u-background--biscay">
            <div className="u-overflow--auto dirtree">
              {!this.state.applicationTree.files ?
                <ul className="FileTree-wrapper">
                  <li>Loading file explorer</li>
                </ul>
                :
                <FileTree
                  files={Utilities.arrangeIntoTree(keys(this.state.applicationTree.files))}
                  isRoot={true}
                  keepOpenPaths={["overlays", "base"]}
                  topLevelPaths={Utilities.arrangeIntoTree(keys(this.state.applicationTree.files)).map(f => f.path)}
                  handleFileSelect={(path) => this.setSelectedFile(path)}
                  selectedFile={this.state.selectedFile}
                />
              }
            </div>
          </div>
          <div className="AceEditor flex1 flex-column file-contents-wrapper u-position--relative">
            {this.state.selectedFile === "" || this.state.selectedFile === "/" ?
              <div className="flex-column flex1 alignItems--center justifyContent--center">
                <p className="u-color--dustyGray u-fontSize--normal u-fontWeight--medium">Select a file from the file explorer to view it here.</p>
              </div>
              : fileLoadErr ?
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <p className="u-color--chestnut u-fontSize--normal u-fontWeight--medium">Oops, we ran into a problem getting that file, <span className="u-fontWeight--bold">{fileLoadErrMessage}</span></p>
                  <p className="u-marginTop--10 u-fontSize--small u-fontWeight--medium u-color--dustyGray">Don't worry, you can download a tar.gz of the resources and have access to all of the files</p>
                  <div className="u-marginTop--20">
                    <button className="btn secondary" onClick={this.handleDownload}>Download tar.gz</button>
                  </div>
                </div>
                :
                  <MonacoEditor
                    ref={(editor) => {
                      this.monacoEditor = editor;
                    }}
                    language={"yaml"}
                    value={contents}
                    height="100%"
                    width="100%"
                    options={{
                      readOnly: true,
                      contextmenu: false,
                      minimap: {
                        enabled: false
                      },
                      scrollBeyondLastLine: false,
                    }}
                  />
            }
          </div>
        </div>
        {displayInstructionsModal &&
          <Modal
            isOpen={displayInstructionsModal}
            onRequestClose={this.toggleInstructionsModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="Display edit instructions modal"
            ariaHideApp={false}
            className="DisplayInstructionsModal--wrapper Modal MediumSize"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Edit patches for your kots application</h2>
              <div className="flex flex1 u-marginTop--20">
                <div className="flex-auto">
                  <span className="instruction-modal-number">1</span>
                </div>
                <div className="flex1">
                  <p className="u-fontSize--large u-fontWeight--bold u-color--tuna u-marginBottom--5 u-lineHeight--normal">Download your application bundle.</p>
                  <CodeSnippet
                    language="bash"
                    canCopy={true}
                    onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
                  >
                    {`kubectl kots download --namespace ${this.props.appNameSpace} --slug ${this.props.match.params.slug}`}
                  </CodeSnippet>
                </div>
              </div>

              <div className="flex flex1 u-marginTop--30">
                <div className="flex-auto">
                  <span className="instruction-modal-number">2</span>
                </div>
                <div className="flex1">
                  <p className="u-fontSize--large u-fontWeight--bold u-color--tuna u-marginBottom--5 u-lineHeight--normal">Edit any of your files in your favorite IDE.</p>
                </div>
              </div>

              <div className="flex flex1 u-marginTop--30">
                <div className="flex-auto">
                  <span className="instruction-modal-number">3</span>
                </div>
                <div className="flex1">
                  <p className="u-fontSize--large u-fontWeight--bold u-color--tuna u-marginBottom--5 u-lineHeight--normal">Upload your edited application bundle.</p>
                  <CodeSnippet
                    language="bash"
                    canCopy={true}
                    onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
                  >
                    {`kubectl kots upload --namespace ${this.props.appNameSpace} --slug ${this.props.match.params.slug} ./${this.props.match.params.slug}`}
                  </CodeSnippet>
                </div>
              </div>
              <div className="u-marginTop--30 flex">
                <button onClick={this.toggleInstructionsModal} className="btn blue primary">Ok, got it!</button>
              </div>
            </div>
          </Modal>
        }
      </div>
    );
  }
}

export default withRouter(compose(
  withRouter,
)(KotsApplicationTree));
