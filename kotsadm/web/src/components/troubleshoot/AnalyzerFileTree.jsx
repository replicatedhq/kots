import * as React from "react";
import AceEditor from "react-ace";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { getFileFormat, rootPath, Utilities } from "../../utilities/utilities";
import sortBy from "lodash/sortBy";
import find from "lodash/find";
import has from "lodash/has";

import Loader from "../shared/Loader";
import FileTree from "../shared/FileTree";

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
      activeMarkers: [],
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
    newObj.content = new Buffer(data[key], "base64").toString();
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
    if (path === "/") {
      return;
    }

    this.setState({
      fileLoading: true,
      fileLoadErr: false,
    });

    fetch(`${window.env.API_ENDPOINT}/troubleshoot/supportbundle/${bundleId}/files?filename=${encodeURIComponent(path)}`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
    })
    .then(async (result) => {
      const data = await result.json();
      this.buildFileContent(data.files);
      this.setState({ fileLoading: false });
    })
    .catch(err => {
      this.setState({
        fileLoading: false,
        fileLoadErr: true,
        fileLoadErrMessage: err,
      })
    });
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

  setMarkersFromHash = () => {
    const { hash } = this.props.location;
    let newMarkers = [];
    const lines = hash.substring(1).split(",");
    lines.forEach(line => {
      newMarkers.push({
        startRow: parseInt(line) - 1,
        endRow: parseInt(line),
        className: "active-highlight",
        type: "background"
      })
    });
    this.setState({ activeMarkers: newMarkers }, () => {
      // Clear hash from URL to prevent highlighting again on a refresh
      const splitLocation = this.props.location.pathname.split("#");
      this.props.history.replace(splitLocation[0]);
    })
  }

  componentDidUpdate(lastProps, lastState) {
    const { bundle } = this.props;
    if (this.state.fileTree !== lastState.fileTree && this.state.fileTree) {
      this.setFileTree();
    }
    if (bundle !== lastProps.bundle && bundle) {
      this.setState({
        bundleId: bundle.id,
        fileTree: bundle.treeIndex
      });

      if (this.props.location !== lastProps.location && this.props.location) {
        this.setState({ selectedFile: "/" + this.props.location.pathname.split("/").slice(7, this.props.location.pathname.length).join("/") })
        this.fetchFiles(bundle.id, "/" + this.props.location.pathname.split("/").slice(7, this.props.location.pathname.length).join("/"))
        if (this.props.location.hash) {
          this.setMarkersFromHash();
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
        this.setState({ selectedFile: "/" + this.props.location.pathname.split("/").slice(7, this.props.location.pathname.length).join("/") })
        this.fetchFiles(bundle.id, "/" + this.props.location.pathname.split("/").slice(7, this.props.location.pathname.length).join("/"))
        if (this.props.location.hash) {
          this.setMarkersFromHash();
        }
      }
    }
  }

  onSelectionChange = () => {
    const column = this.refAceEditor?.editor?.selection?.anchor.column;
    const row = this.refAceEditor?.editor?.selection?.anchor.row;
    if (column === 0) {
      let newMarker = [];
      newMarker.push({
        startRow: row - 1,
        endRow: row,
        className: "active-highlight",
        type: "background"
      })
      this.setState({ activeMarkers: newMarker });
      this.props.history.replace(`${this.props.location.pathname}#L${row}`);
    }
  }

  handleDownload = () => {
    const { downloadBundle } = this.props;
    if (downloadBundle && typeof downloadBundle == "function") {
      downloadBundle();
    }
  }

  render() {
    const { files, fileContents, selectedFile, fileLoadErr, fileLoadErrMessage, fileLoading } = this.state;
    const fileToView = find(fileContents, ["key", selectedFile]);
    const format = getFileFormat(selectedFile);
    const isOld = files && has(files[0], "size");

    return (
      <div className="flex-column flex1 AnalyzerFileTree--wrapper">
        {!files || !files.length || isOld ?
          <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center">
            <p className="u-color--tuna u-fontSize--normal u-fontWeight--bold">We were unable to detect files from this Support Bundle</p>
            <p className="u-fontSize--small u-color--dustyGray u-fontWeight--medium u-marginTop--10">It's possible that this feature didn't exists when you uploaded the bundle, try re-analyzing it to have files detected.</p>
          </div>
          :
          <div className="flex flex1">
            <div className="flex1 dirtree-wrapper flex-column u-overflow-hidden u-background--biscay">
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
                      <Loader size="50" />
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
