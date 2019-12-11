import * as React from "react";
import { compose, withApollo, graphql } from "react-apollo";
import { withRouter } from "react-router-dom";
import { getFileFormat, rootPath } from "../../utilities/utilities";
import sortBy from "lodash/sortBy";
import find from "lodash/find";
import MonacoEditor from "react-monaco-editor";
import Modal from "react-modal";
import CodeSnippet from "../shared/CodeSnippet";

import Loader from "../shared/Loader";
import FileTree from "../shared/FileTree";
import { getKotsApplicationTree, getKotsFiles, } from "../../queries/AppsQueries";

import "../../scss/components/troubleshoot/FileTree.scss";

class KotsApplicationTree extends React.Component {

  state = {
    files: [],
    selectedFile: "/",
    fileContents: [],
    fileLoading: false,
    fileLoadErr: false,
    fileLoadErrMessage: "",
    line: null,
    activeMarkers: [],
    analysisError: false,
    displayInstructionsModal: false,
  };

  hasContentAlready = (path) => {
    const { fileContents } = this.state;
    let i;
    for (i = 0; i < fileContents.length; i++) {
      if (fileContents[i].key === path) { return true; }
    }
    return false;
  }

  buildFileContent = (data) => {
    const nextFiles = this.state.fileContents;
    const key = Object.keys(data);
    let newObj = {};
    newObj.content = data[key];
    newObj.key = key[0];
    nextFiles.push(newObj);
    this.setState({ fileContents: nextFiles });
  }

  async setSelectedFile(path) {
    const newPath = rootPath(path);
    this.setState({ selectedFile: newPath });
    if (this.hasContentAlready(newPath)) { return; } // Don't go fetch it if we already have that content in our state
    this.fetchFiles(newPath)
  }

  fetchFiles = (path) => {
    const { params } = this.props.match;
    const slug = params.slug;
    const sequence = parseInt(params.sequence);
    this.setState({ fileLoading: true, fileLoadErr: false });
    this.props.client.query({
      query: getKotsFiles,
      variables: {
        slug: slug,
        sequence,
        fileNames: [path]
      }
    })
      .then((res) => {
        this.buildFileContent(JSON.parse(res.data.getKotsFiles));
        this.setState({ fileLoading: false });
      })
      .catch((err) => {
        err.graphQLErrors.map(({ msg }) => {
          this.setState({
            fileLoading: false,
            fileLoadErr: true,
            fileLoadErrMessage: msg,
          });
        });
      })
  }

  setFileTree = () => {
    if (!this.state.fileTree) { return; }
    const parsedTree = JSON.parse(this.state.fileTree);
    let sortedTree = sortBy(parsedTree, (dir) => {
      dir.children ? dir.children.length : []
    });
    this.setState({ files: sortedTree });
  }

  componentDidUpdate(lastProps, lastState) {
    const { getKotsApplicationTree } = this.props;
    if (this.state.fileTree !== lastState.fileTree && this.state.fileTree) {
      this.setFileTree();
    }
    if (getKotsApplicationTree?.getKotsApplicationTree !== lastProps.getKotsApplicationTree?.getKotsApplicationTree) {
      this.setState({
        fileTree: getKotsApplicationTree.getKotsApplicationTree
      });
    }
  }

  componentDidMount() {
    const { getKotsApplicationTree } = this.props;
    if (this.state.fileTree) {
      this.setFileTree();
    }
    if (getKotsApplicationTree?.getKotsApplicationTree) {
      this.setState({
        fileTree: getKotsApplicationTree.getKotsApplicationTree
      })
    }
  }

  toggleInstructionsModal = () => {
    this.setState({ displayInstructionsModal: !this.state.displayInstructionsModal });
  }

  back = () => {
    this.props.history.goBack();
  }

  render() {
    const { files, fileContents, selectedFile, fileLoadErr, fileLoadErrMessage, fileLoading, displayInstructionsModal } = this.state;
    const fileToView = find(fileContents, ["key", selectedFile]);
    const format = getFileFormat(selectedFile);

    return (
      <div className="flex-column flex1 ApplicationTree--wrapper container u-paddingTop--50 u-paddingBottom--30">
        <div className="edit-files-banner u-fontSize--small u-fontWeight--medium">Need to edit these files? <span onClick={this.toggleInstructionsModal} className="u-textDecoration--underline u-fontWeight--bold u-cursor--pointer">Click here</span> to learn how</div>
        <div className="flex flex1">
          <div className="flex1 dirtree-wrapper flex-column u-overflow-hidden u-background--biscay">
            <div className="u-overflow--auto dirtree">
              {this.props.getKotsApplicationTree.loading ?
                <ul className="FileTree-wrapper">
                  <li>Loading file explorer</li>
                </ul>
                :
                <FileTree
                  files={files}
                  isRoot={true}
                  keepOpenPaths={["overlays", "base"]}
                  topLevelPaths={files?.map(f => f.path)}
                  handleFileSelect={(path) => this.setSelectedFile(path)}
                  selectedFile={this.state.selectedFile}
                />
              }
            </div>
          </div>
          <div className="AceEditor flex1 flex-column file-contents-wrapper u-position--relative">
            {selectedFile === "" || selectedFile === "/" ?
              <div className="flex-column flex1 alignItems--center justifyContent--center">
                <p className="u-color--dustyGray u-fontSize--normal u-fontWeight--medium">Select a file from the file explorer to view it here.</p>
              </div>
              : fileLoadErr ?
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <p className="u-color--chestnut u-fontSize--normal u-fontWeight--medium">Oops, we ran into a probelm getting that file, <span className="u-fontWeight--bold">{fileLoadErrMessage}</span></p>
                  <p className="u-marginTop--10 u-fontSize--small u-fontWeight--medium u-color--dustyGray">Don't worry, you can download a tar.gz of the resources and have access to all of the files</p>
                  <div className="u-marginTop--20">
                    <button className="btn secondary" onClick={this.handleDownload}>Download tar.gz</button>
                  </div>
                </div>
                : fileLoading || !fileToView ?
                  <div className="flex-column flex1 alignItems--center justifyContent--center">
                    <Loader size="50" color="#44bb66" />
                  </div>
                  :
                  <MonacoEditor
                    ref={(editor) => { this.monacoEditor = editor }}
                    language={format}
                    value={fileToView.content}
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
                    {`kubectl kots download ${this.props.match.params.slug} --namespace ${this.props.appNameSpace} --dest ~/${this.props.match.params.slug}`}
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
                  {`kubectl kots upload --slug ${this.props.match.params.slug} ~/${this.props.match.params.slug}`}
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
  withApollo,
  withRouter,
  graphql(getKotsApplicationTree, {
    name: "getKotsApplicationTree",
    options: props => {
      const { params } = props.match;
      const slug = params.slug;
      const sequence = parseInt(params.sequence);
      return {
        variables: {
          slug,
          sequence,
        },
        fetchPolicy: "no-cache"
      };
    }
  }),
)(KotsApplicationTree));
