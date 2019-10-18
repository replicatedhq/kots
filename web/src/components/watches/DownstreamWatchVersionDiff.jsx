import * as React from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import sortBy from "lodash/sortBy";
import map from "lodash/map";
import groupBy from "lodash/groupBy";
import uniqBy from "lodash/uniqBy";

import { rootPath } from "../../utilities/utilities";
import Loader from "../shared/Loader";
import DiffEditor from "../shared/DiffEditor";

import { getKotsApplicationTree, getKotsFiles } from "../../queries/AppsQueries";


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
      variables: { slug: this.props.match.params.slug, sequence: this.props.match.params.firstSequence },
      fetchPolicy: "no-cache"
    })
      .then((res) => {
        this.setState({ firstApplicationTree: res.data.getKotsApplicationTree })
      }).catch();

    this.props.client.query({
      query: getKotsApplicationTree,
      name: "getKotsApplicationTree",
      variables: { slug: this.props.match.params.slug, sequence: this.props.match.params.secondSequence },
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
    const { params } = this.props.match;

    if (firstApplicationTree !== lastState.firstApplicationTree && firstApplicationTree.length > 0) {
      this.setFileTree(firstApplicationTree, true);
    }
    if (secondApplicationTree !== lastState.secondApplicationTree && secondApplicationTree.length > 0) {
      this.setFileTree(secondApplicationTree, false);
    }
    if (params.slug !== lastProps.match.params.slug) {
      this.fetchKotsApplicationTree();
    }
    if (firstSeqFiles !== lastState.firstSeqFiles && firstSeqFiles) {
      if (params.firstSequence) {
        this.allFilesForSequence(firstSeqFiles, params.firstSequence, true);
      }
    }
    if (secondSeqFiles !== lastState.secondSeqFiles && secondSeqFiles) {
      if (params.secondSequence) {
        this.allFilesForSequence(secondSeqFiles, params.secondSequence, false);
      }
    }
  }

  componentDidMount() {
    const { firstApplicationTree, secondApplicationTree, firstSeqFiles, secondSeqFiles } = this.state;
    const { params } = this.props.match;

    if (firstApplicationTree?.length > 0) {
      this.setFileTree(this.state.firstApplicationTree, true);
    }
    if (secondApplicationTree?.length > 0) {
      this.setFileTree(this.state.secondApplicationTree, false);
    }
    if (params.slug) {
      this.fetchKotsApplicationTree();
    }
    if (firstSeqFiles && params.firstSequence) {
      this.allFilesForSequence(firstSeqFiles, params.firstSequence, true);
    }
    if (secondSeqFiles && params.secondSequence) {
      this.allFilesForSequence(secondSeqFiles, params.secondSequence, false);
    }
  }

  fetchFiles = (path, sequence, first) => {
    const { params } = this.props.match;
    const slug = params.slug;
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

  allFilesForSequence = (files, sequence, first) => {
    files.map((file => {
      if (file.children) {
        file.children.map((chFile => {
          this.getAllFiles(chFile.path, sequence, first)
        }))
      }
      this.getAllFiles(file.path, sequence, first);
    }))
  }

  hasContentAlready = (path, first) => {
    const { firstSeqFileContents, secondSeqFileContents } = this.state;

    if (first) {
      let i;
      for (i = 0; i < firstSeqFileContents.length; i++) {
        if (firstSeqFileContents[i].key === path) { return true; }
      }
      return false;
    } else {
      let i;
      for (i = 0; i < secondSeqFileContents.length; i++) {
        if (secondSeqFileContents[i].key === path) { return true; }
      }
      return false;
    }
  }

  buildFileContent = (data, first) => {
    if (first) {
      const nextFiles = this.state.firstSeqFileContents;
      const key = Object.keys(data);
      let newObj = {};
      newObj.content = data[key];
      newObj.key = key[0];
      newObj.sequence = "first",
        nextFiles.push(newObj);
      this.setState({ secondSeqFileContents: nextFiles });
    } else {
      const nextFiles = this.state.secondSeqFileContents;
      const key = Object.keys(data);
      let newObj = {};
      newObj.content = data[key];
      newObj.key = key[0];
      newObj.sequence = "second",
        nextFiles.push(newObj);
      this.setState({ secondSeqFileContents: nextFiles });
    }
  }

  getAllFiles = (path, sequence, first) => {
    const newPath = rootPath(path);
    if (this.hasContentAlready(newPath, first)) { return; } // Don't go fetch it if we already have that content in our state
    this.fetchFiles(newPath, sequence, first)
  }


  render() {
    const { firstSeqFileContents, secondSeqFileContents, fileLoading } = this.state

    const files = [...firstSeqFileContents, ...secondSeqFileContents];
    const changedFiles = uniqBy(files, "content");
    const filesByKey = groupBy(changedFiles, "key");

    return (
      <div className="container u-overflow--auto u-position--relative u-paddingTop--30 u-paddingBottom--20 u-minHeight--full u-width--full">
        {fileLoading ?
          <div className="u-height--full u-width--full flex alignItems--center justifyContent--center">
            <Loader size="60" />
          </div>
          :
          map(filesByKey, (value, key) => {
            const first = value.find(val => val.sequence === "first");
            const second = value.find(val => val.sequence === "second");
            return (
              <div className="flex u-overflow--auto u-height--half" key={key}>
                <DiffEditor
                  original={first}
                  value={second}
                  specKey={key}
                />
              </div>
            )
          })
        }
      </div>
    );
  }
}

export default withRouter(compose(
  withApollo,
  withRouter
)(DownstreamWatchVersionDiff));
