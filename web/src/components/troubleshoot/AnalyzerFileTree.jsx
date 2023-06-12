import * as React from "react";
import AceEditor from "react-ace";
import { withRouter } from "@src/utilities/react-router-utilities";
import { getFileFormat, rootPath, Utilities } from "../../utilities/utilities";
import sortBy from "lodash/sortBy";
import find from "lodash/find";
import has from "lodash/has";
import queryString from "query-string";
import ReactTooltip from "react-tooltip";
import Loader from "../shared/Loader";
import FileTree from "../shared/FileTree";

import "../../scss/components/troubleshoot/FileTree.scss";

import "brace/mode/json";
import "brace/mode/text";
import "brace/mode/yaml";
import "brace/theme/chrome";
import Icon from "../Icon";

class AnalyzerFileTree extends React.Component {
  constructor(props) {
    super();
    this.state = {
      files: [],
      selectedFile:
        "/" +
        props.location.pathname
          .split("/")
          .slice(8, props.location.pathname.length)
          .join("/"),
      fileContents: [],
      fileLoading: false,
      fileLoadErr: false,
      fileLoadErrMessage: "",
      activeMarkers: [],
      analysisError: false,
      currentViewIndex: 0,
    };
  }

  hasContentAlready = (path) => {
    const { fileContents } = this.state;
    let i;
    for (i = 0; i < fileContents.length; i++) {
      if (fileContents[i].key === path) {
        return true;
      }
    }
    return false;
  };

  buildFileContent = (data) => {
    const nextFiles = this.state.fileContents;
    const keys = Object.keys(data);

    if (!keys.length) {
      return;
    }

    let newObj = {};
    newObj.content = new Buffer(data[keys[0]], "base64").toString();
    newObj.key = keys[0];
    nextFiles.push(newObj);

    this.setState({ fileContents: nextFiles });
  };

  async setSelectedFile(path) {
    console.log(path, "path");
    console.log("hello");
    const { navigate, outletContext, params } = this.props;
    const newPath = rootPath(path);
    navigate(
      `/app/${outletContext.watchSlug}/troubleshoot/analyze/${params.bundleSlug}/contents${newPath}`,
      { replace: true }
    );
    this.setState({ selectedFile: newPath, activeMarkers: [] });
    if (this.hasContentAlready(newPath)) {
      return;
    } // Don't go fetch it if we already have that content in our state
    this.fetchFiles(this.state.bundleId, newPath);
  }

  fetchFiles = (bundleId, path) => {
    if (path === "/") {
      return;
    }

    this.setState({
      fileLoading: true,
      fileLoadErr: false,
    });

    fetch(
      `${
        process.env.API_ENDPOINT
      }/troubleshoot/supportbundle/${bundleId}/files?filename=${encodeURIComponent(
        path
      )}`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      }
    )
      .then(async (result) => {
        const data = await result.json();
        this.buildFileContent(data.files);
        this.setState({ fileLoading: false });
      })
      .catch((err) => {
        this.setState({
          fileLoading: false,
          fileLoadErr: true,
          fileLoadErrMessage: err,
        });
      });
  };

  setFileTree = () => {
    if (!this.state.fileTree) {
      return;
    }
    const parsedTree = JSON.parse(this.state.fileTree);
    let sortedTree = sortBy(parsedTree, (dir) => {
      return dir.children ? dir.children.length : [];
    });
    this.setState({ files: sortedTree });
  };

  setRedactorMarkersFromHash = () => {
    const { hash, search } = this.props.location;
    let redactorFileName = "";
    if (search) {
      redactorFileName = queryString.parse(search);
    }
    let newMarkers = [];
    const lines = hash.substring(1).split(",");
    lines.forEach((line) => {
      newMarkers.push({
        startRow: parseInt(line) - 1,
        endRow: parseInt(line),
        className: "active-highlight",
        type: "background",
      });
    });
    this.setState(
      {
        activeMarkers: newMarkers,
        redactionMarkersSet: true,
        redactorFileName: redactorFileName.file,
      },
      () => {
        // Clear hash from URL to prevent highlighting again on a refresh
        const splitLocation = this.props.location.pathname.split("#");
        this.props.navigate(splitLocation[0], { replace: true });
      }
    );
  };

  scrollToRedactions = (index) => {
    this.setState({ currentViewIndex: index });
    const editor = this.aceEditor.editor;
    editor.scrollToLine(this.state.activeMarkers[index].endRow, true, true);
    editor.gotoLine(this.state.activeMarkers[index].endRow, 1, true);
  };

  componentDidUpdate(lastProps, lastState) {
    const { bundle } = this.props.outletContext;
    if (this.state.fileTree !== lastState.fileTree && this.state.fileTree) {
      this.setFileTree();
    }
    if (bundle !== lastProps.outletContext.bundle && bundle) {
      this.setState({
        bundleId: bundle.id,
        fileTree: bundle.treeIndex,
      });

      if (this.props.location !== lastProps.location && this.props.location) {
        this.setState({
          selectedFile:
            "/" +
            this.props.location.pathname
              .split("/")
              .slice(7, this.props.location.pathname.length)
              .join("/"),
        });
        this.fetchFiles(
          bundle.id,
          "/" +
            this.props.location.pathname
              .split("/")
              .slice(7, this.props.location.pathname.length)
              .join("/")
        );
        if (this.props.location.hash) {
          this.setRedactorMarkersFromHash();
        }
      }
    }
    if (this.aceEditor && this.state.redactionMarkersSet) {
      this.setState({ redactionMarkersSet: false });
      this.aceEditor.editor.resize(true);
      this.scrollToRedactions(0);
    }
  }

  componentDidMount() {
    const { location } = this.props;
    const { bundle } = this.props.outletContext;
    if (this.state.fileTree) {
      this.setFileTree();
    }

    if (bundle) {
      this.setState({
        bundleId: bundle.id,
        fileTree: bundle.treeIndex,
      });
      if (location) {
        this.setState({
          selectedFile:
            "/" +
            location.pathname
              .split("/")
              .slice(7, location.pathname.length)
              .join("/"),
        });
        this.fetchFiles(
          bundle.id,
          "/" +
            location.pathname
              .split("/")
              .slice(7, location.pathname.length)
              .join("/")
        );
        if (location.hash) {
          this.setRedactorMarkersFromHash();
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
        type: "background",
      });
      this.setState({ activeMarkers: newMarker });
      this.props.navigate(`${this.props.location.pathname}#L${row}`, {
        replace: true,
      });
    }
  };

  handleDownload = () => {
    const { downloadBundle } = this.props.outletContext;
    if (downloadBundle && typeof downloadBundle == "function") {
      downloadBundle();
    }
  };

  render() {
    const {
      files,
      fileContents,
      selectedFile,
      fileLoadErr,
      fileLoadErrMessage,
      fileLoading,
    } = this.state;
    const fileToView = find(fileContents, ["key", selectedFile]);
    const format = getFileFormat(selectedFile);
    const isOld = files && has(files[0], "size");
    const isFirstRedaction = this.state.currentViewIndex === 0;
    const isLastRedaction =
      this.state.currentViewIndex + 1 === this.state.activeMarkers.length;

    return (
      <div className="flex-column flex1 AnalyzerFileTree--wrapper">
        {!files || !files.length || isOld ? (
          <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center">
            <p className="u-textColor--primary u-fontSize--normal u-fontWeight--bold">
              We were unable to detect files from this Support Bundle
            </p>
            <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginTop--10">
              It's possible that this feature didn't exists when you uploaded
              the bundle, try re-analyzing it to have files detected.
            </p>
          </div>
        ) : (
          <div className="flex flex1">
            <div className="flex1 dirtree-wrapper flex-column u-overflow-hidden">
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
              {this.state.activeMarkers.length > 0 ? (
                <div className="redactor-pager flex alignItems--center">
                  <div
                    className={`arrow-wrapper prev ${
                      isFirstRedaction ? "" : "can-scroll"
                    }`}
                    onClick={
                      isFirstRedaction
                        ? undefined
                        : () =>
                            this.scrollToRedactions(
                              this.state.currentViewIndex - 1
                            )
                    }
                  >
                    <Icon
                      icon="left-arrow-pointer"
                      size={14}
                      className={isFirstRedaction ? "gray-color" : "clickable"}
                    />
                  </div>
                  <div className="flex alignItems--center">
                    <span>
                      Redaction {this.state.currentViewIndex + 1} of{" "}
                      {this.state.activeMarkers.length}
                    </span>
                    <Icon
                      icon="info-circle-outline"
                      size={16}
                      className="gray-color u-marginLeft--10"
                      data-tip
                      data-for="current-redator-filename"
                    />
                  </div>
                  <div
                    className={`arrow-wrapper next ${
                      isLastRedaction ? "" : "can-scroll"
                    }`}
                    onClick={
                      isLastRedaction
                        ? undefined
                        : () =>
                            this.scrollToRedactions(
                              this.state.currentViewIndex + 1
                            )
                    }
                  >
                    <Icon
                      icon="right-arrow-pointer"
                      size={13}
                      className={isLastRedaction ? "gray-color" : "clickable"}
                    />
                  </div>
                </div>
              ) : null}
              {selectedFile === "" || selectedFile === "/" ? (
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <p className="u-textColor--bodyCopy u-fontSize--normal u-fontWeight--medium">
                    Select a file from the directory tree to view it here.
                  </p>
                </div>
              ) : fileLoadErr ? (
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium">
                    Oops, we ran into a probelm getting that file,{" "}
                    <span className="u-fontWeight--bold">
                      {fileLoadErrMessage}
                    </span>
                  </p>
                  <p className="u-marginTop--10 u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">
                    Don't worry, you can download the bundle and have access to
                    all of the files
                  </p>
                  <div className="u-marginTop--20">
                    <button
                      className="btn secondary"
                      onClick={this.handleDownload}
                    >
                      Download bundle
                    </button>
                  </div>
                </div>
              ) : fileLoading || !fileToView ? (
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <Loader size="50" />
                </div>
              ) : fileToView.content === "" ? (
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <p className="u-textColor--secondary u-fontSize--normal u-fontWeight--medium">
                    This file was collected and analyzed but it contains no
                    data.
                  </p>
                </div>
              ) : (
                <AceEditor
                  ref={(el) => (this.aceEditor = el)}
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
                  onLoad={(editor) =>
                    editor.gotoLine(
                      this.props.location.hash !== "" &&
                        parseInt(this.props.location.hash.substring(2))
                    )
                  }
                  onSelectionChange={this.onSelectionChange}
                  setOptions={{
                    scrollPastEnd: false,
                    showGutter: true,
                  }}
                />
              )}
              <ReactTooltip
                id="current-redator-filename"
                type="light"
                effect="solid"
                borderColor="#C4C4C4"
                textColor="#4A4A4A"
                border={true}
                className="u-textColor--secondary"
              >
                Viewing redactions from{" "}
                <span className="u-fontWeight--bold">
                  {this.state.redactorFileName}
                </span>
              </ReactTooltip>
            </div>
          </div>
        )}
      </div>
    );
  }
}

export default withRouter(AnalyzerFileTree);
