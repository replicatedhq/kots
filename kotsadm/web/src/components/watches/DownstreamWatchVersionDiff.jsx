import React from "react";
import { withRouter } from "react-router-dom";

import { Utilities } from "../../utilities/utilities";
import Loader from "../shared/Loader";
import DiffEditor from "../shared/DiffEditor";

import "../../scss/components/watches/DownstreamWatchVersionDiff.scss";


class DownstreamWatchVersionDiff extends React.Component {
  constructor() {
    super();
    this.state = {
      loadingFileTrees: false,
      firstApplicationTree: {},
      secondApplicationTree: {},
      fileLoading: false,
      fileLoadErr: false,
      fileLoadErrMessage: "",
      hasErrSettingDiff: false,
      errSettingDiff: "",
      failedSequence: undefined
    };
  }

  fetchRenderedApplicationTree = (sequence, isFirst) => {
    this.setState({ loadingFileTrees: true });
    const url = `${window.env.API_ENDPOINT}/app/${this.props.slug}/sequence/${sequence}/renderedcontents`;
    fetch(url, {
      headers: {
        "Authorization": Utilities.getToken()
      },
      method: "GET",
    })
    .then(res => res.json())
    .then(async (files) => {
      if (files.error) {
        return this.setState({
          hasErrSettingDiff: true,
          errSettingDiff: files.error,
          loadingFileTrees: false,
          failedSequence: sequence
        })
      }
      if (isFirst) {
        this.setState({ firstApplicationTree: files });
      } else {
        this.setState({ secondApplicationTree: files });
      }
      if (this.state.firstApplicationTree.files && this.state.secondApplicationTree.files) {
        this.setState({ loadingFileTrees: false });  
      }
    })
    .catch((err) => {
      this.setState({ loadingFileTrees: false });
      throw err;
    });
  }

  componentDidUpdate(lastProps, lastState) {
    const { slug, firstSequence, secondSequence } = this.props;

    if (slug !== lastProps.slug) {
      this.fetchRenderedApplicationTree(firstSequence, true);
      this.fetchRenderedApplicationTree(secondSequence, false);
    }
  }

  componentDidMount() {
    const { firstSequence, secondSequence } = this.props;
    this.fetchRenderedApplicationTree(firstSequence, true);
    this.fetchRenderedApplicationTree(secondSequence, false);

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

  goBack = () => {
    if (this.props.onBackClick) {
      this.props.onBackClick(true);
    }
  }

  render() {
    const { firstApplicationTree, secondApplicationTree, loadingFileTrees, hasErrSettingDiff, errSettingDiff, failedSequence } = this.state;
    const { firstSequence, secondSequence } = this.props;

    if (loadingFileTrees) {
      return (
        <div className="u-height--full u-width--full flex alignItems--center justifyContent--center u-marginTop--15">
          <Loader size="60" />
        </div>
      );
    }

    if (hasErrSettingDiff) {
      return (
        <div className="u-height--full u-width--full flex-column alignItems--center justifyContent--center u-marginTop--15">
          <p className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-lineHeight--normal u-marginBottom--10">Unable to generate a file diff for the selected releases</p>
          <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">The release with the sequence <span className="u-fontWeight--bold">{failedSequence}</span> contains invalid YAML or config values and is unable to generate a diff. The full error is below.</p>
          <div className="error-block-wrapper u-marginBottom--30 flex flex1">
            <span className="u-color--chestnut">{errSettingDiff}</span>
          </div>
          <div className="flex u-marginBottom--10">
            <button className="btn primary" onClick={() => this.goBack()}>Back to all versions</button>
          </div>
        </div>
      )
    }

    const filesToInclude = [];
    for (const filename in firstApplicationTree.files) {
      if (firstApplicationTree.files[filename] === secondApplicationTree.files[filename]) {
        continue;
      }

      filesToInclude.push(filename);
    }

    for (const filename in secondApplicationTree.files) {
      if (firstApplicationTree.files[filename] === secondApplicationTree.files[filename]) {
        continue;
      }

      if (filesToInclude.indexOf(filename) === -1) {
        filesToInclude.push(filename);
      }
    }

    const diffEditors = [];
    for (const filename of filesToInclude) {
      const firstNumOfLines = firstApplicationTree.files[filename] ? firstApplicationTree.files[filename].split("\n").length : 0;
      const secondNumOfLines = secondApplicationTree.files[filename] ? secondApplicationTree.files[filename].split("\n").length : 0;
      const maxNumOfLines = Math.max(firstNumOfLines, secondNumOfLines) + 1;

      diffEditors.push(<div className="DiffEditor flex-column" key={filename} style={{ height: maxNumOfLines * 23 }}>
        <DiffEditor
          original={firstApplicationTree.files[filename]}
          value={secondApplicationTree.files[filename]}
          key={filename}
          specKey={filename}
          options={{
            contextMenu: false,
            readOnly: true,
          }}
        />
      </div>);
    }

    const content = diffEditors.length > 0 ? diffEditors :
      (<div className="flex flex-auto alignItems--center justifyContent--center">
        <div className="EmptyWrapper u-width--half u-textAlign--center">
          <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">There isn’t anything to compare.</p>
          <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--10">There are no changes in any of the files between these 2 versions.</p>
        </div>
      </div>);

    return (
      <div className="u-position--relative u-height--full u-width--full">
        <div className="flex u-marginBottom--15">
          <div className="u-fontWeight--bold u-color--astral u-cursor--pointer" onClick={this.goBack}>
            <span className="icon clickable backArrow-icon u-marginRight--10" style={{ verticalAlign: "0" }} />
            Back
          </div>
          <span className="u-fontWeight--bold u-marginLeft--20 u-color--tuna">Diffing releases {firstSequence} and {secondSequence}</span>
        </div>
        {content}
      </div>
    );
  }
}

export default withRouter(DownstreamWatchVersionDiff);
