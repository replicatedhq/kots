import React from "react";
import { withRouter } from "@src/utilities/react-router-utilities";

import Loader from "../../components/shared/Loader";
import DiffEditor from "../../components/shared/DiffEditor";

import "../../scss/components/watches/DownstreamWatchVersionDiff.scss";
import Icon from "../../components/Icon";
import { useNavigate } from "react-router-dom";

type Props = {
  firstSequence: number;
  hideBackButton?: boolean;
  onBackClick: (goBack: boolean) => void;
  secondSequence: number;
  slug: string;
  navigate: ReturnType<typeof useNavigate>;
};

type State = {
  loadingFileTrees: boolean;
  firstApplicationTree?: ApplicationTree;
  secondApplicationTree?: ApplicationTree;
  fileLoading: boolean;
  fileLoadErr: boolean;
  fileLoadErrMessage?: string;
  hasErrSettingDiff: boolean;
  errSettingDiff?: string;
  failedSequence?: number;
};

type ApplicationTree = {
  files: {
    [key: string]: string;
  };
};
class VersionDiff extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      loadingFileTrees: false,
      fileLoading: false,
      fileLoadErr: false,
      hasErrSettingDiff: false,
    };
  }

  fetchRenderedApplicationTree = (sequence: number, isFirst: boolean) => {
    this.setState({ loadingFileTrees: true });
    const url = `${process.env.API_ENDPOINT}/app/${this.props.slug}/sequence/${sequence}/renderedcontents`;
    return fetch(url, {
      method: "GET",
      credentials: "include",
    })
      .then((res) => res.json())
      .then(async (files) => {
        if (files.error) {
          return this.setState({
            hasErrSettingDiff: true,
            errSettingDiff: files.error,
            loadingFileTrees: false,
            failedSequence: sequence,
          });
        }
        if (isFirst) {
          this.setState({ firstApplicationTree: files });
        } else {
          this.setState({ secondApplicationTree: files });
        }
      })
      .catch((err) => {
        this.setState({ loadingFileTrees: false });
        throw err;
      });
  };

  componentDidUpdate(lastProps: Props) {
    const { slug, firstSequence, secondSequence } = this.props;

    if (slug !== lastProps.slug) {
      Promise.all([
        this.fetchRenderedApplicationTree(firstSequence, true),
        this.fetchRenderedApplicationTree(secondSequence, false),
      ]).then(() => this.setState({ loadingFileTrees: false }));
    }
  }

  componentDidMount() {
    const { firstSequence, secondSequence, navigate } = this.props;
    Promise.all([
      this.fetchRenderedApplicationTree(firstSequence, true),
      this.fetchRenderedApplicationTree(secondSequence, false),
    ]).then(() => this.setState({ loadingFileTrees: false }));

    const url = window.location.pathname;
    if (!url.includes("/diff") && !location.search.includes("?diff/")) {
      // what is this doing?
      // window.history.replaceState(
      //   "",
      //   "",
      //   `${url}/diff/${firstSequence}/${secondSequence}`
      // );
      navigate(`${url}/diff/${firstSequence}/${secondSequence}`, {
        replace: true,
      });
    }
  }

  componentWillUnmount() {
    const url = window.location.pathname;
    if (url.includes("/diff")) {
      const { firstSequence, secondSequence, navigate } = this.props;
      const diffPath = `/diff/${firstSequence}/${secondSequence}`;
      // window.history.replaceState(
      //   "",
      //   "",
      //   url.substring(0, url.indexOf(diffPath))
      // );
      navigate(url.substring(0, url.indexOf(diffPath)), { replace: true });
    }
  }

  goBack = () => {
    if (this.props.onBackClick) {
      this.props.onBackClick(true);
    }
  };

  render() {
    const {
      firstApplicationTree,
      secondApplicationTree,
      loadingFileTrees,
      hasErrSettingDiff,
      errSettingDiff,
      failedSequence,
    } = this.state;
    const { firstSequence, secondSequence } = this.props;

    if (loadingFileTrees) {
      return (
        <>
          <div className="flex u-marginBottom--15">
            {!this.props.hideBackButton && (
              <div
                className="u-fontWeight--bold u-marginRight--20 link"
                onClick={this.goBack}
              >
                <Icon
                  icon="prev-arrow"
                  size={10}
                  className="clickable u-marginRight--10"
                  style={{ verticalAlign: "0" }}
                />
                Back
              </div>
            )}
            <span className="u-fontWeight--bold u-textColor--primary">
              Diffing releases {firstSequence} and {secondSequence}
            </span>
          </div>
          <div className="u-height--full u-width--full flex alignItems--center justifyContent--center u-marginTop--15">
            {" "}
            <Loader size="60" />
          </div>
        </>
      );
    }

    if (hasErrSettingDiff) {
      return (
        <div className="u-height--full u-width--full flex-column alignItems--center justifyContent--center u-marginTop--15">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
            Unable to generate a file diff for the selected releases
          </p>
          <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
            The release with the sequence{" "}
            <span className="u-fontWeight--bold">{failedSequence}</span>{" "}
            contains invalid YAML or config values and is unable to generate a
            diff. The full error is below.
          </p>
          <div className="error-block-wrapper u-marginBottom--30 flex flex1">
            <span className="u-textColor--error">{errSettingDiff}</span>
          </div>
          <div className="flex u-marginBottom--10">
            <button className="btn primary" onClick={() => this.goBack()}>
              Back to all versions
            </button>
          </div>
        </div>
      );
    }

    const filesToInclude = [];
    for (const filename in firstApplicationTree?.files) {
      if (
        firstApplicationTree?.files[filename] ===
        secondApplicationTree?.files[filename]
      ) {
        continue;
      }

      filesToInclude.push(filename);
    }

    for (const filename in secondApplicationTree?.files) {
      if (
        firstApplicationTree?.files[filename] ===
        secondApplicationTree?.files[filename]
      ) {
        continue;
      }

      if (filesToInclude.indexOf(filename) === -1) {
        filesToInclude.push(filename);
      }
    }

    const diffEditors = [];
    for (const filename of filesToInclude) {
      const firstNumOfLines = firstApplicationTree?.files[filename]
        ? firstApplicationTree?.files[filename].split("\n").length
        : 0;
      const secondNumOfLines = secondApplicationTree?.files[filename]
        ? secondApplicationTree?.files[filename].split("\n").length
        : 0;
      const maxNumOfLines = Math.max(firstNumOfLines, secondNumOfLines) + 1;

      diffEditors.push(
        <div
          className="DiffEditor flex-column"
          key={filename}
          style={{ height: maxNumOfLines * 23 }}
        >
          <DiffEditor
            original={firstApplicationTree?.files[filename]}
            value={secondApplicationTree?.files[filename]}
            specKey={filename}
            options={{
              contextMenu: false,
              readOnly: true,
            }}
          />
        </div>
      );
    }

    const content =
      diffEditors.length > 0 ? (
        diffEditors
      ) : (
        <div className="flex flex-auto alignItems--center justifyContent--center">
          <div className="EmptyWrapper u-width--half u-textAlign--center">
            <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
              There isn’t anything to compare.
            </p>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--10">
              There are no changes in any of the files between these 2 versions.
            </p>
          </div>
        </div>
      );

    return (
      <div className="u-position--relative u-height--full u-width--full">
        <div className="flex u-marginBottom--15">
          {!this.props.hideBackButton && (
            <div
              className="u-fontWeight--bold u-marginRight--20 link"
              onClick={this.goBack}
            >
              <Icon
                icon="prev-arrow"
                size={10}
                className="clickable u-marginRight--10"
                style={{ verticalAlign: "0" }}
              />
              Back
            </div>
          )}
          <span className="u-fontWeight--bold u-textColor--primary">
            Diffing releases {firstSequence} and {secondSequence}
          </span>
        </div>
        {content}
      </div>
    );
  }
}

/* eslint-disable */
// @ts-ignore
export default withRouter(VersionDiff) as any;
/* eslint-enable */
