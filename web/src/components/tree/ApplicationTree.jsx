import * as React from "react";
import { compose, withApollo, graphql } from "react-apollo";
import { withRouter } from "react-router-dom";
import { getFileFormat, rootPath } from "../../utilities/utilities";
import sortBy from "lodash/sortBy";
import find from "lodash/find";
import MonacoEditor from "react-monaco-editor";

import Loader from "../shared/Loader";
import FileTree from "../shared/FileTree";
import { getApplicationTree, getFiles, getParentWatch } from "../../queries/WatchQueries";

import "../../scss/components/troubleshoot/FileTree.scss";

class ApplicationTree extends React.Component {
  
  state = {
    files: [],
    selectedFile: "/",
    fileContents: [],
    fileLoading: false,
    fileLoadErr: false,
    fileLoadErrMessage: "",
    line: null,
    activeMarkers: [],
    analysisError: false
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
    const slug = `${params.owner}/${params.slug}`;
    const sequence = parseInt(params.sequence);
    this.setState({ fileLoading: true, fileLoadErr: false });
    this.props.client.query({
      query: getFiles,
      variables: {
        slug: slug,
        sequence,
        fileNames: [path]
      }
    })
      .then((res) => {
        this.buildFileContent(JSON.parse(res.data.getFiles));
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
    const { getApplicationTree } = this.props;
    if (this.state.fileTree !== lastState.fileTree && this.state.fileTree) {
      this.setFileTree();
    }
    if (getApplicationTree?.getApplicationTree !== lastProps.getApplicationTree?.getApplicationTree) {
      this.setState({
        fileTree: getApplicationTree.getApplicationTree
      });
    }
  }

  componentDidMount() {
    const { getApplicationTree } = this.props;
    if (this.state.fileTree) {
      this.setFileTree();
    }
    if (getApplicationTree?.getApplicationTree) {
      this.setState({
        fileTree: getApplicationTree.getApplicationTree
      })
    }
  }

  back = () => {
    this.props.history.goBack();
  }

  render() {
    const { files, fileContents, selectedFile, fileLoadErr, fileLoadErrMessage, fileLoading } = this.state;
    const { getParentWatch, match } = this.props;
    const fileToView = find(fileContents, ["key", selectedFile]);
    const format = getFileFormat(selectedFile);
    const parentWatch = getParentWatch?.getParentWatch;
    let breadcrumbs = <span>&nbsp;</span>;
    if (parentWatch) {
      const selectedDownstream = find(parentWatch.watches, ["slug", `${match.params.owner}/${match.params.slug}`]);
      const versions = selectedDownstream?.pendingVersions.concat(selectedDownstream?.pastVersions);
      // TODO: This logic is bleh and i'm still not 100% convinced it's correct
      let versionTitle = "";
      if (versions.length) {
        const selectedVersion = find(versions, ["sequence", parseInt(match.params.sequence)]);
        if (!selectedVersion) {
          if (selectedDownstream.currentVersion?.sequence === 0) {
            versionTitle = selectedDownstream.currentVersion.title;
          } else {
            versionTitle = "";
          }
        } else {
          versionTitle = selectedVersion?.title;
        };
      } else if (selectedDownstream.currentVersion?.sequence === 0) {
        versionTitle = selectedDownstream.currentVersion.title;
      }
      breadcrumbs = `${parentWatch.watchName} > ${selectedDownstream?.cluster.title} > ${versionTitle} (${match.params.sequence}) > files`;
    }

    return (
      <div className="flex-column flex1 ApplicationTree--wrapper container u-paddingTop--20 u-paddingBottom--30">
        <p className="u-marginBottom--20 u-fontSize--small u-color--tundora u-fontWeight--bold u-lineHeight--normal">
          <span onClick={this.back} className="replicated-link u-marginRight--5"> &lt; Back</span> {breadcrumbs}
        </p>
        <div className="flex flex1">
          <div className="flex1 dirtree-wrapper flex-column u-overflow-hidden u-background--biscay">
            <div className="u-overflow--auto dirtree">
              {this.props.getApplicationTree.loading ?
                <ul className="FileTree-wrapper">
                  <li>Loading file explorer</li>
                </ul>
                :
                <FileTree
                  files={files}
                  isRoot={true}
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
                  <p className="u-marginTop--10 u-fontSize--small u-fontWeight--medium u-color--dustyGray">Don't worry, you can download the bundle and have access to all of the files</p>
                  <div className="u-marginTop--20">
                    <button className="btn secondary" onClick={this.handleDownload}>Download bundle</button>
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
                      minimap: {
                        enabled: false
                      },
                      scrollBeyondLastLine: false,
                    }}
                  />
            }
          </div>
        </div>
      </div>
    );
  }
}

export default withRouter(compose(
  withApollo,
  withRouter,
  graphql(getApplicationTree, {
    name: "getApplicationTree",
    options: props => {
      const { params } = props.match;
      const slug = `${params.owner}/${params.slug}`;
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
  graphql(getParentWatch, {
    name: "getParentWatch",
    options: props => {
      const { params } = props.match;
      const slug = `${params.owner}/${params.slug}`;
      return {
        variables: { slug },
      };
    }
  }),
)(ApplicationTree));