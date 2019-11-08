import React from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import sortBy from "lodash/sortBy";
import map from "lodash/map";
import groupBy from "lodash/groupBy";
import filter from "lodash/filter";
import flatMap from "lodash/flatMap";
import size from "lodash/size";

import { rootPath } from "../../utilities/utilities";
import Loader from "../shared/Loader";
import DiffEditor from "../shared/DiffEditor";

import { getKotsApplicationTree, getKotsFiles } from "../../queries/AppsQueries";

import "../../scss/components/watches/DownstreamWatchVersionDiff.scss";


class DownstreamWatchVersionDiff extends React.Component {
  constructor() {
    super();
    this.state = {
      firstApplicationTree: [],
      secondApplicationTree: [],
      firstSeqFiles: [],
      secondSeqFiles: [],
      firstSeqFileContents: [],
      secondSeqFileContents: [],
      fileLoading: false,
      fileLoadErr: false,
      fileLoadErrMessage: "",
    };
  }

  fetchKotsApplicationTree = () => {
    this.props.client.query({
      query: getKotsApplicationTree,
      name: "getKotsApplicationTree",
      variables: { slug: this.props.slug, sequence: this.props.firstSequence },
      fetchPolicy: "no-cache"
    })
      .then((res) => {
        this.setState({ firstApplicationTree: res.data.getKotsApplicationTree })
      }).catch();

    this.props.client.query({
      query: getKotsApplicationTree,
      name: "getKotsApplicationTree",
      variables: { slug: this.props.slug, sequence: this.props.secondSequence },
      fetchPolicy: "no-cache"
    })
      .then((res) => {
        this.setState({ secondApplicationTree: res.data.getKotsApplicationTree })
      }).catch();
  }

  setFileTree = (tree, first) => {
    if (!tree || tree.length <= 0) { return; }

    const parsedTree = JSON.parse(tree);

    let sortedTree = sortBy(parsedTree, (dir) => {
      dir.children ? dir.children.length : []
    });

    if (first) {
      this.setState({ firstSeqFiles: sortedTree });
    } else {
      this.setState({ secondSeqFiles: sortedTree });
    }
  }

  componentDidUpdate(lastProps, lastState) {
    const { firstApplicationTree, secondApplicationTree, firstSeqFiles, secondSeqFiles } = this.state;
    const { slug, firstSequence, secondSequence } = this.props;

    if (firstApplicationTree !== lastState.firstApplicationTree && firstApplicationTree.length > 0) {
      this.setFileTree(firstApplicationTree, true);
    }
    if (secondApplicationTree !== lastState.secondApplicationTree && secondApplicationTree.length > 0) {
      this.setFileTree(secondApplicationTree, false);
    }
    if (slug !== lastProps.slug) {
      this.fetchKotsApplicationTree();
    }
    if (firstSeqFiles !== lastState.firstSeqFiles && firstSeqFiles) {
      this.allFilesForSequence(firstSeqFiles, firstSequence, true);
    }
    if (secondSeqFiles !== lastState.secondSeqFiles && secondSeqFiles) {
      this.allFilesForSequence(secondSeqFiles, secondSequence, false);
    }
  }

  componentDidMount() {
    const { firstApplicationTree, secondApplicationTree, firstSeqFiles, secondSeqFiles } = this.state;
    const { slug, firstSequence, secondSequence } = this.props;

    if (firstApplicationTree?.length > 0) {
      this.setFileTree(this.state.firstApplicationTree, true);
    }
    if (secondApplicationTree?.length > 0) {
      this.setFileTree(this.state.secondApplicationTree, false);
    }
    if (slug) {
      this.fetchKotsApplicationTree();
    }
    if (firstSeqFiles) {
      this.allFilesForSequence(firstSeqFiles, firstSequence, true);
    }
    if (secondSeqFiles) {
      this.allFilesForSequence(secondSeqFiles, secondSequence, false);
    }

    const url = window.location.pathname;
    if (!url.includes("/diff")) {
      window.history.replaceState("", "", `${url}/diff/${firstSequence}/${secondSequence}`);
    }
  }

  componentWillUnmount() {
    const url = window.location.pathname;
    if (url.includes("/diff")) {
      const { firstSequence, secondSequence } = this.props;
      const diffPath = `/diff/${firstSequence}/${secondSequence}`;
      window.history.replaceState("", "", url.substring(0, url.indexOf(diffPath)));
    }
  }

  fetchFiles = (paths, sequence, first) => {
    const { slug } = this.props;
    this.setState({ fileLoading: true, fileLoadErr: false });
    this.props.client.query({
      query: getKotsFiles,
      variables: {
        slug: slug,
        sequence,
        fileNames: paths
      }
    })
      .then((res) => {
        this.buildFileContent(JSON.parse(res.data.getKotsFiles), first);
        this.setState({ fileLoading: false });
      })
      .catch((err) => {
        err.graphQLErrors.map(({ message }) => {
          this.setState({
            fileLoading: false,
            fileLoadErr: true,
            fileLoadErrMessage: message,
          });
        });
      })
  }

  getPaths = (files) => {
    let paths = [];
    files.map(file => {
      if (file.children.length) {
        const subPaths = this.getPaths(file.children);
        paths = paths.concat(subPaths);
      } else {
        paths.push(file.path);
      }
    });
    return paths;
  }

  allFilesForSequence = (files, sequence, first) => {
    const paths = this.getPaths(files);
    this.getFilesForPathAndSequence(paths, sequence, first);
  }

  buildFileContent = (data, first) => {
    if (first) {
      const nextFiles = this.state.firstSeqFileContents;
      map(data, (value, key) => {
        let newObj = {};
        newObj.content = value;
        newObj.key = key;
        newObj.sequence = "first";
        nextFiles.push(newObj);
      })
      this.setState({ firstSeqFileContents: nextFiles });
    } else {
      const nextFiles = this.state.secondSeqFileContents;
      map(data, (value, key) => {
        let newObj = {};
        newObj.content = value;
        newObj.key = key;
        newObj.sequence = "second";
        nextFiles.push(newObj);
      })
      this.setState({ secondSeqFileContents: nextFiles });
    }
  }

  getFilesForPathAndSequence = (paths, sequence, first) => {
    const newPaths = paths.map((path) => rootPath(path));
    this.fetchFiles(newPaths, sequence, first);
  }

  goBack = () => {
    if (this.props.onBackClick) {
      this.props.onBackClick();
    }
  }

  render() {
    const { firstSeqFileContents, secondSeqFileContents, fileLoading } = this.state;
    const { firstSequence, secondSequence } = this.props;

    if (fileLoading || size(firstSeqFileContents) === 0 || size(secondSeqFileContents) === 0) {
      return (
        <div className="u-height--full u-width--full flex alignItems--center justifyContent--center u-marginTop--15">
          <Loader size="60" />
        </div>
      );
    }

    const files = [...firstSeqFileContents, ...secondSeqFileContents];
    const groupedFilesByContent = groupBy(files, "content");
    const changedFiles = filter(groupedFilesByContent, g => g.length === 1);
    const filesByKey = groupBy(flatMap(changedFiles), "key");

    return (
      <div className="u-position--relative u-height--full u-width--full">
        <div className="flex u-marginBottom--15">
          <div className="u-fontWeight--bold u-color--astral u-cursor--pointer" onClick={this.goBack}>
            <span className="icon clickable backArrow-icon u-marginRight--10" style={{ verticalAlign: "0" }} />
            Back
          </div>
          <span className="u-fontWeight--bold u-marginLeft--20 u-color--tuna">Diffing releases {firstSequence} and {secondSequence}</span>
        </div>
        {size(filesByKey) > 0 ?
          map(filesByKey, (value, key) => {
            const first = value.find(val => val.sequence === "first");
            const second = value.find(val => val.sequence === "second");
            const maxNumOfLines = Math.max(first.content.split("\n").length, second.content.split("\n").length);
            return (
              <div className="flex-column" key={key} style={{ height: maxNumOfLines * 23 }}>
                <DiffEditor
                  original={first}
                  value={second}
                  specKey={key}
                  options={{
                    contextMenu: false,
                    readOnly: true
                  }}
                />
              </div>
            );
          })
          :
          <div className="flex flex-auto alignItems--center justifyContent--center">
            <div className="EmptyWrapper u-width--half u-textAlign--center">
              <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">There isn’t anything to compare.</p>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--10">There are no changes in any of the files between these 2 versions.</p>
            </div>
          </div>
        }
      </div>
    );
  }
}

export default withRouter(compose(
  withApollo,
  withRouter
)(DownstreamWatchVersionDiff));
