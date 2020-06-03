import React from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";

import { Utilities } from "../../utilities/utilities";
import Loader from "../shared/Loader";
import DiffEditor from "../shared/DiffEditor";

import "../../scss/components/watches/DownstreamWatchVersionDiff.scss";


class DownstreamWatchVersionDiff extends React.Component {
  constructor() {
    super();
    this.state = {
      firstApplicationTree: {},
      secondApplicationTree: {},
      fileLoading: false,
      fileLoadErr: false,
      fileLoadErrMessage: "",
    };
  }

  fetchRenderedApplicationTree = (sequence, isFirst) => {
    const url = `${window.env.API_ENDPOINT}/app/${this.props.slug}/sequence/${sequence}/renderedcontents`;
    fetch(url, {
      headers: {
        "Authorization": Utilities.getToken()
      },
      method: "GET",
    })
    .then(res => res.json())
    .then(async (files) => {
      if (isFirst) {
        this.setState({firstApplicationTree: files});
      } else {
        this.setState({secondApplicationTree: files});
      }
    })
    .catch((err) => {
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
      this.props.onBackClick();
    }
  }

  render() {
    const { firstApplicationTree, secondApplicationTree } = this.state;
    const { firstSequence, secondSequence } = this.props;

    if (!firstApplicationTree.files || !secondApplicationTree.files) {
      return (
        <div className="u-height--full u-width--full flex alignItems--center justifyContent--center u-marginTop--15">
          <Loader size="60" />
        </div>
      );
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

export default withRouter(compose(
  withRouter
)(DownstreamWatchVersionDiff));
