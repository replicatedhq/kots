import * as React from "react";
import AceEditor from "react-ace";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { getFileFormat, rootPath } from "../../utilities/utilities";
import sortBy from "lodash/sortBy";
import find from "lodash/find";
import has from "lodash/has";

import Loader from "../shared/Loader";
import FileTree from "../shared/FileTree";
import { supportBundleFiles } from "../../queries/TroubleshootQueries";

import "../../scss/components/troubleshoot/FileTree.scss";

import "brace/mode/json";
import "brace/mode/text";
import "brace/mode/yaml";
import "brace/theme/chrome";


class AnalyzerFileTree extends React.Component {
  constructor(props) {
    super();
    this.state = {
      files: [],
      selectedFile: "/" + props.location.pathname.split("/").slice(8, props.location.pathname.length).join("/"),
      fileContents: [],
      fileLoading: false,
      fileLoadErr: false,
      fileLoadErrMessage: "",
      line: null,
      activeMarkers: [],
      analysisError: false
    };
  }

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
    const { watchSlug } = this.props;
    const newPath = rootPath(path);
    this.props.history.replace(`/app/${watchSlug}/troubleshoot/analyze/${this.props.match.params.bundleSlug}/contents${newPath}`);
    this.setState({ selectedFile: newPath, activeMarkers: [] });
    if (this.hasContentAlready(newPath)) { return; } // Don't go fetch it if we already have that content in our state
    this.fetchFiles(this.state.bundleId, newPath)
  }

  fetchFiles = (bundleId, path) => {
    this.setState({ fileLoading: true, fileLoadErr: false });
    this.props.client.query({
      query: supportBundleFiles,
      variables: {
        bundleId: bundleId,
        fileNames: [path]
      }
    })
      .then((res) => {
        this.buildFileContent(JSON.parse(res.data.supportBundleFiles));
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
    sortedTree.reverse(); // If something has a directory, render those first, all top level files should be at that bottom
    this.setState({ files: sortedTree });
  }

  componentDidUpdate(lastProps, lastState) {
    const { bundle } = this.props;
    if (this.state.fileTree !== lastState.fileTree && this.state.fileTree) {
      this.setFileTree();
    }
    if (this.props.isFullscreen !== lastProps.isFullscreen) {
      if (this.refAceEditor) {
        this.refAceEditor.editor.resize(); // ace editor needs to resize itself so that content does not get cut off
      }
    }
    if (bundle !== lastProps.bundle && bundle) {
      this.setState({
        bundleId: bundle.id,
        fileTree: bundle.treeIndex
      });
      if (this.props.location) {
        if (this.props.location.pathname) {
          this.fetchFiles(bundle.id, "/" + this.props.location.pathname.split("/").slice(8, this.props.location.pathname.length).join("/"))
        }
        if (this.props.location.hash) {
          let newMarker = [];
          newMarker.push({
            startRow: parseInt(this.props.location.hash.substring(2)) - 1,
            endRow: parseInt(this.props.location.hash.substring(2)),
            className: "active-highlight",
            type: "background"
          })
          this.setState({ activeMarkers: newMarker })
        }
      }

      if (this.props.location !== lastProps.location && this.props.location) {
        this.setState({ selectedFile: "/" + this.props.location.pathname.split("/").slice(8, this.props.location.pathname.length).join("/") })
        this.fetchFiles(bundle.id, "/" + this.props.location.pathname.split("/").slice(8, this.props.location.pathname.length).join("/"))
        if (this.props.location.hash) {
          let newMarker = [];
          newMarker.push({
            startRow: parseInt(this.props.location.hash.substring(2)) - 1,
            endRow: parseInt(this.props.location.hash.substring(2)),
            className: "active-highlight",
            type: "background"
          })
          this.setState({ activeMarkers: newMarker })
        }
      }
    }
  }

  componentDidMount() {
    const { bundle } = this.props;
    if (this.state.fileTree) {
      this.setFileTree();
    }
    if (bundle) {
      this.setState({
        bundleId: bundle.id,
        fileTree: bundle.treeIndex
      })
      if (this.props.location) {
        this.setState({ selectedFile: "/" + this.props.location.pathname.split("/").slice(8, this.props.location.pathname.length).join("/") })
        this.fetchFiles(bundle.id, "/" + this.props.location.pathname.split("/").slice(8, this.props.location.pathname.length).join("/"))
        if (this.props.location.hash) {
          let newMarker = [];
          newMarker.push({
            startRow: parseInt(this.props.location.hash.substring(2)) - 1,
            endRow: parseInt(this.props.location.hash.substring(2)),
            className: "active-highlight",
            type: "background"
          })
          this.setState({ activeMarkers: newMarker })
        }
      }
    }
  }

  onSelectionChange = (selection) => {
    if (selection.anchor.column === 0) {
      this.setState({ line: selection.anchor.row });
      let newMarker = [];
      newMarker.push({
        startRow: this.state.line - 1,
        endRow: this.state.line,
        className: "active-highlight",
        type: "background"
      })
      this.setState({ activeMarkers: newMarker });
      this.props.history.replace(`${this.props.location.pathname}#L${this.state.line}`);
    }
  }

  handleDownload = () => {
    const { downloadBundle} = this.props;
    if (downloadBundle && typeof downloadBundle == "function") {
      downloadBundle();
    }
  }

  reAnalyzeBundle = () => {
    this.setState({ isReanalyzing: true });
    this.props.reAnalyzeBundle((response, analysisError) => {
      this.setState({ isReanalyzing: false, analysisError });
    });
  }

  render() {
    const { files, fileContents, selectedFile, fileLoadErr, fileLoadErrMessage, fileLoading, analysisError } = this.state;
    const fileToView = find(fileContents, ["key", selectedFile]);
    const format = getFileFormat(selectedFile);
    const isOld = files && has(files[0], "size");

    const analysisErrorExists = analysisError && analysisError.graphQLErrors && analysisError.graphQLErrors.length;

    return (
      <div className="flex-column flex1 AnalyzerFileTree--wrapper">
        {!files || !files.length || isOld ?
          <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center">
            <p className="u-color--tuna u-fontSize--normal u-fontWeight--bold">We were unable to detect files from this Support Bundle</p>
            <p className="u-fontSize--small u-color--dustyGray u-fontWeight--medium u-marginTop--10">It's possible that this feature didn't exists when you uploaded the bundle, try re-analyzing it to have files detected.</p>
            <div className="u-marginTop--20">
              <button className="btn secondary" onClick={() => this.reAnalyzeBundle()} disabled={this.state.isReanalyzing}>{this.state.isReanalyzing ? "Re-analyzing" : "Re-analyze bundle"}</button>
            </div>
            {analysisErrorExists && <span style={{ maxWidth: 420 }} className="u-fontSize--small u-lineHeight--normal u-fontWeight--bold u-color--error u-marginTop--20 u-textAlign--center">{analysisError.graphQLErrors[0].message}</span>}
          </div>
          :
          <div className="flex flex1">
            <div className={`flex1 dirtree-wrapper flex-column u-overflow-hidden u-background--biscay ${this.props.isFullscreen ? "fs-mode" : ""}`}>
              <div className="u-overflow--auto dirtree">
                <FileTree
                  files={files}
                  isRoot={true}
                  handleFileSelect={(path) => this.setSelectedFile(path)}
                  selectedFile={this.state.selectedFile}
                />
              </div>
            </div>
            <div className="AceEditor flex1 flex-column file-contents-wrapper u-position--relative">
              <div className="fullscreen-icon-wrapper" onClick={() => this.props.toggleFullscreen()}>
                <span className={`icon u-fullscreen${this.props.isFullscreen ? "Close" : "Open"}Icon clickable`}></span>
              </div>
              {selectedFile === "" || selectedFile === "/" ?
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <p className="u-color--dustyGray u-fontSize--normal u-fontWeight--medium">Select a file from the directory tree to view it here.</p>
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
                    : fileToView.content === "" ?
                      <div className="flex-column flex1 alignItems--center justifyContent--center">
                        <p className="u-color--tundora u-fontSize--normal u-fontWeight--medium">This file was collected and analyzed but it contains no data.</p>
                      </div>
                    :
                    <AceEditor
                      ref={(input) => this.refAceEditor = input}
                      mode={format}
                      theme="chrome"
                      className="flex1 flex"
                      readOnly={true}
                      value={fileToView.content}
                      height="100%"
                      width="100%"
                      markers={this.state.activeMarkers}
                      editorProps={{
                        $blockScrolling: Infinity,
                        useSoftTabs: true,
                        tabSize: 2,
                      }}
                      onLoad={(editor) => editor.gotoLine(this.props.location.hash !== "" && parseInt(this.props.location.hash.substring(2)))}
                      onSelectionChange={this.onSelectionChange}
                      setOptions={{
                        scrollPastEnd: false,
                        showGutter: true,
                      }}
                    />
              }
            </div>
          </div>
        }
      </div>
    );
  }
}

export default withRouter(compose(
  withApollo
)(AnalyzerFileTree));