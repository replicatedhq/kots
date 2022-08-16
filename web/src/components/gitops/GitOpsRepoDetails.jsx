import * as React from "react";
import PropTypes from "prop-types";
import { getGitOpsServiceSite } from "../../utilities/utilities";
import Loader from "../shared/Loader";

import "../../scss/components/gitops/GitOpsDeploymentManager.scss";
import { Flex, Paragraph } from "../../styles/common";

class GitOpsRepoDetails extends React.Component {
  static propTypes = {
    appName: PropTypes.string.isRequired,
    selectedService: PropTypes.object.isRequired,
    hostname: PropTypes.string,
    owner: PropTypes.string,
    repo: PropTypes.string,
    branch: PropTypes.string,
    path: PropTypes.string,
    action: PropTypes.string,
    format: PropTypes.string,
    stepTitle: PropTypes.string,
  };

  static defaultProps = {
    hostname: "",
    owner: "",
    repo: "",
    branch: "",
    path: "",
    action: "commit",
    format: "single",
  };

  constructor(props) {
    super(props);

    const {
      appName,
      selectedService,
      hostname,
      owner,
      repo,
      branch,
      path,
      action = "commit",
      format = "single",
    } = this.props;

    this.state = {
      appName,
      selectedService,
      providerError: null,
      hostname,
      owner,
      repo,
      branch,
      path,
      action,
      format,
      finishingSetup: false,
      showFinishedConfirm: false,
    };
  }
  componentDidUpdate(prevProps, prevState) {
    if (
      prevProps.owner !== this.props.owner ||
      prevProps.repo !== this.props.repo ||
      prevProps.branch !== this.props.branch ||
      prevProps.path !== this.props.path
    ) {
      this.setState({
        owner: this.props.owner,
        repo: this.props.repo,
        branch: this.props.branch,
        path: this.props.path,
      });
    }
  }

  componentDidMount() {
    this._mounted = true;
  }

  componentWillUnmount() {
    this._mounted = false;
  }

  onActionTypeChange = (e) => {
    if (e.target.classList.contains("js-preventDefault")) {
      return;
    }
    this.setState({ action: e.target.value });
  };

  onFileContainChange = (e) => {
    if (e.target.classList.contains("js-preventDefault")) {
      return;
    }
    this.setState({ format: e.target.value });
  };

  isValid = () => {
    const { owner, selectedService } = this.state;
    const provider = selectedService?.value;
    if (provider !== "other" && !owner.length) {
      this.setState({
        providerError: {
          field: "owner",
        },
      });
      return false;
    }
    return true;
  };

  onFinishSetup = async () => {
    if (!this.isValid() || !this.props.onFinishSetup) {
      return;
    }

    this.setState({ finishingSetup: true });

    const ownerRepo = this.state.owner + "/" + this.state.repo;

    const repoDetails = {
      ownerRepo: ownerRepo,
      branch: this.state.branch,
      path: this.state.path,
      action: this.state.action,
      format: this.state.format,
    };

    const success = await this.props.onFinishSetup(repoDetails);
    if (this._mounted) {
      if (success) {
        this.setState({ finishingSetup: false, showFinishedConfirm: true });
        this.props.updateSettings();
        setTimeout(() => {
          this.setState({ showFinishedConfirm: false });
        }, 3000);
      } else {
        this.setState({ finishingSetup: false });
      }
    }
  };

  allowUpdate = () => {
    const { owner, repo, branch, path, action, format, selectedService } =
      this.state;
    const provider = selectedService?.value;
    if (provider === "other") {
      return true;
    }
    const isAllowed =
      owner !== this.props.owner ||
      repo !== this.props.repo ||
      branch !== this.props.branch ||
      path !== this.props.path ||
      action !== this.props.action ||
      format !== this.props.format;
    return isAllowed;
  };

  render() {
    const {
      appName,
      selectedService,
      providerError,
      hostname,
      owner,
      repo,
      branch,
      path,
      action,
      format,
      finishingSetup,
      showFinishedConfirm,
    } = this.state;

    const { gitopsConnected, gitopsEnabled } = this.props;
    const provider = selectedService?.value;
    const serviceSite = getGitOpsServiceSite(provider, hostname);
    const isBitbucketServer = provider === "bitbucket_server";

    return (
      <>
        <Flex
          key={`action-active`}
          // className="GitOpsDeploy--step u-textAlign--left"
          width="100%"
          direction="column"
        >
          <Flex flex="1" mb="30" mt="20" width="100%">
            {provider !== "other" && (
              <div className="flex flex1 flex-column u-marginRight--20">
                <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                  {isBitbucketServer ? "Project" : "Owner"}
                  <span> (Required)</span>
                </p>
                <input
                  type="text"
                  className={`Input ${
                    providerError?.field === "owner" && "has-error"
                  }`}
                  placeholder={isBitbucketServer ? "project" : "owner"}
                  value={owner}
                  onChange={(e) => this.setState({ owner: e.target.value })}
                  autoFocus
                />
                {providerError?.field === "owner" && (
                  <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">
                    {isBitbucketServer
                      ? "A project must be provided"
                      : "An owner must be provided"}
                  </p>
                )}
              </div>
            )}

            {provider !== "other" && (
              <Flex flex="1" direction="column">
                <Paragraph
                  size="16"
                  weight="bold"
                  className="u-lineHeight--normal"
                >
                  Repository <span>(Required)</span>
                </Paragraph>
                <input
                  type="text"
                  className={`Input ${
                    providerError?.field === "repo" && "has-error"
                  }`}
                  placeholder={"Repository"}
                  value={repo}
                  onChange={(e) => this.setState({ repo: e.target.value })}
                  autoFocus
                />
                {providerError?.field === "owner" && (
                  <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">
                    A repository must be provided
                  </p>
                )}
              </Flex>
            )}
          </Flex>
          <Flex width="100%">
            {provider !== "other" && (
              <div className="flex flex1 flex-column u-marginRight--20">
                <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                  Branch
                </p>
                <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">
                  Leave blank to use the default branch.
                </p>
                <input
                  type="text"
                  className={`Input`}
                  placeholder="main"
                  value={branch}
                  onChange={(e) => this.setState({ branch: e.target.value })}
                />
              </div>
            )}
            {provider !== "other" && (
              <div className="flex flex1 flex-column">
                <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                  Path
                </p>
                <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">
                  Path in repository to cmmmit deployment file
                </p>
                <input
                  type="text"
                  className={"Input"}
                  placeholder="/path/to-deployment"
                  value={path}
                  onChange={(e) => this.setState({ path: e.target.value })}
                />
              </div>
            )}
          </Flex>
        </Flex>
        <div
          className="flex justifyContent--flexEnd u-marginTop--30"
          style={{ width: "100%" }}
        >
          {finishingSetup ? (
            <Loader className="u-marginLeft--5" size="30" />
          ) : gitopsConnected && gitopsEnabled ? (
            <button
              className="btn primary blue"
              type="button"
              disabled={finishingSetup || !this.allowUpdate()}
              onClick={this.onFinishSetup}
            >
              Save Configuration
            </button>
          ) : (
            <button
              className="btn primary blue"
              type="button"
              disabled={finishingSetup || !this.allowUpdate()}
              onClick={this.onFinishSetup}
            >
              Generate SSH key
            </button>
          )}
        </div>
      </>
    );
  }
}

export default GitOpsRepoDetails;
