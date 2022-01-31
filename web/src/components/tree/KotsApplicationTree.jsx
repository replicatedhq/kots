import * as React from "react";
import { withRouter } from "react-router-dom";
import { Utilities } from "../../utilities/utilities";
import Helmet from "react-helmet";
import isEmpty from "lodash/isEmpty";
import keys from "lodash/keys";
import MonacoEditor from "react-monaco-editor";
import Modal from "react-modal";
import CodeSnippet from "../shared/CodeSnippet";

import FileTree from "../shared/FileTree";

import "../../scss/components/troubleshoot/FileTree.scss";

class KotsApplicationTree extends React.Component {
  constructor() {
    super();
    this.state = {
      files: {},
      selectedFile: "/",
      displayInstructionsModal: false,
      applicationTree: [],
    };
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
    .then(async (res) => {
      const files = res?.files || {};
      const paths = keys(files);
      const applicationTree = Utilities.arrangeIntoApplicationTree(paths);
      if (this.props.history.location.search) {
        this.setState({ selectedFile: `/skippedFiles/${this.props.history.location.search.slice(1)}`})
      }
      this.setState({
        files,
        applicationTree,
      });
    })
    .catch((err) => {
      throw err;
    });
  }

  componentDidUpdate(lastProps, lastState) {
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
      selectedFile: path
    });
  }

  back = () => {
    this.props.history.goBack();
  }

  render() {
    const { displayInstructionsModal, files, applicationTree, selectedFile } = this.state;

    const contents = files[selectedFile] ? new Buffer(files[selectedFile], "base64").toString() : "";

    return (
      <div className="flex-column flex1 ApplicationTree--wrapper container u-paddingTop--50 u-paddingBottom--30">
        <Helmet>
          <title>{`${this.props.app?.name} Files`}</title>
        </Helmet>

        <div className="edit-files-banner u-fontSize--small u-fontWeight--medium">Need to edit these files? <span onClick={this.toggleInstructionsModal} className="replicated-link">Click here</span> to learn how</div>
        <div className="flex flex1">
          <div className="flex1 dirtree-wrapper flex-column u-overflow-hidden">
            <div className="u-overflow--auto dirtree">
              <FileTree
                files={applicationTree}
                isRoot={true}
                handleFileSelect={this.setSelectedFile}
                selectedFile={this.state.selectedFile}
              />
              {isEmpty(applicationTree) &&
                <ul className="FileTree-wrapper">
                  <li>Loading file explorer</li>
                </ul>
              }
            </div>
          </div>
          <div className="AceEditor flex1 flex-column file-contents-wrapper u-position--relative">
            {this.state.selectedFile === "" || this.state.selectedFile === "/" ?
              <div className="flex-column flex1 alignItems--center justifyContent--center">
                <p className="u-textColor--bodyCopy u-fontSize--normal u-fontWeight--medium">Select a file from the file explorer to view it here.</p>
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
              <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal">Edit patches for your kots application</h2>
              <div className="flex flex1 u-marginTop--20">
                <div className="flex-auto">
                  <span className="instruction-modal-number">1</span>
                </div>
                <div className="flex1">
                  <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-marginBottom--5 u-lineHeight--normal">Download your application bundle.</p>
                  <CodeSnippet
                    language="bash"
                    canCopy={true}
                    onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
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
                  <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-marginBottom--5 u-lineHeight--normal">Edit any of your files in your favorite IDE.</p>
                </div>
              </div>

              <div className="flex flex1 u-marginTop--30">
                <div className="flex-auto">
                  <span className="instruction-modal-number">3</span>
                </div>
                <div className="flex1">
                  <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-marginBottom--5 u-lineHeight--normal">Upload your edited application bundle.</p>
                  <CodeSnippet
                    language="bash"
                    canCopy={true}
                    onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
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

export default withRouter(KotsApplicationTree);
