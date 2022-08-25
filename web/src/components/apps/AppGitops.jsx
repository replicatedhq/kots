import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import Helmet from "react-helmet";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import {
  getGitOpsUri,
  getAddKeyUri,
  requiresHostname,
  Utilities,
} from "../../utilities/utilities";
import Select from "react-select";
import not_enabled from "../../images/not_enabled.svg";
import warning from "../../images/warning.svg";
import enabled from "../../images/enabled.svg";

import "../../scss/components/gitops/GitOpsDeploymentManager.scss";
import "../../scss/components/gitops/GitOpsSettings.scss";
import "../../scss/components/gitops/GitopsPrism.scss";

import styled from "styled-components";

import ConnectionModal from "../gitops/modals/ConnectionModal";

import Loader from "../shared/Loader";
import DisableModal from "../gitops/modals/DisableModal";
import { Flex } from "../../styles/common";

const IconWrapper = styled.div`
  height: 30px;
  width: 30px;
  border-radius: 50%;
  background-position: center;
  background-size: contain;
  background-repeat: no-repeat;
  box-shadow: inset 0 0 3px rgba(0, 0, 0, 0.3);
  background-color: #ffffff;
  z-index: 1;
`;

const SERVICES = [
  {
    value: "github",
    label: "GitHub",
  },
  {
    value: "github_enterprise",
    label: "GitHub Enterprise",
  },
  {
    value: "gitlab",
    label: "GitLab",
  },
  {
    value: "gitlab_enterprise",
    label: "GitLab Enterprise",
  },
  {
    value: "bitbucket",
    label: "Bitbucket",
  },
  {
    value: "bitbucket_server",
    label: "Bitbucket Server",
  },
  // {
  //   value: "other",
  //   label: "Other",
  // }
];

class AppGitops extends Component {
  constructor(props) {
    super(props);

    this.state = {
      ownerRepo: this.getInitialOwnerRepo(props),
      testingConnection: false,
      disablingGitOps: false,
      showDisableGitopsModalPrompt: false,
      showGitOpsSettings: false,
      errorMsg: "",
      showConnectionModal: false,
      modalType: "",
    };
  }

  getInitialOwnerRepo = (props) => {
    if (!props.app?.downstream) {
      return "";
    }

    const gitops = props.app.downstream.gitops;
    if (!gitops?.uri) {
      return "";
    }

    let ownerRepo = "";
    const parsed = new URL(gitops?.uri);
    if (gitops?.provider === "bitbucket_server") {
      const project =
        parsed.pathname.split("/").length > 2 && parsed.pathname.split("/")[2];
      const repo =
        parsed.pathname.split("/").length > 4 && parsed.pathname.split("/")[4];
      if (project && repo) {
        ownerRepo = `${project}/${repo}`;
      }
    } else {
      ownerRepo = parsed.pathname.slice(1); // remove the "/"
    }

    return ownerRepo;
  };

  renderIcons = (service) => {
    if (service) {
      return <span className={`icon gitopsService--${service.value}`} />;
    } else {
      return;
    }
  };

  getLabel = (service, label) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span style={{ fontSize: 18, marginRight: "10px" }}>
          {this.renderIcons(service)}
        </span>
        <span style={{ fontSize: 14 }}>{label}</span>
      </div>
    );
  };

  handleTestConnection = async () => {
    this.setState({ testingConnection: true, errorMsg: "" });

    const appId = this.props.app?.id;
    let clusterId;
    if (this.props.app?.downstream) {
      clusterId = this.props.app.downstream.cluster.id;
    }

    try {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/gitops/app/${appId}/cluster/${clusterId}/initconnection`,
        {
          method: "POST",
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
        }
      );
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }

        if (res.status === 400) {
          const response = await res.json();
          if (response?.error) {
            this.setState({ showConnectionModal: true, modalType: "fail" });
            console.log(response?.error);
          }
          throw new Error(`authentication failed`);
        }
        throw new Error(`unexpected status code: ${res.status}`);
      }

      this.setState({ showConnectionModal: true, modalType: "success" });
    } catch (err) {
      console.log(err);
      this.setState({
        errorMsg: `Failed to test connection: ${
          err ? err.message : "Something went wrong, please try again."
        }`,
      });
    } finally {
      this.setState({ testingConnection: false, connectionTested: true });
    }
  };

  updateGitOpsSettings = () => {
    this.setState({ showGitOpsSettings: true });
  };

  finishGitOpsSetup = async (repoDetails) => {
    const { ownerRepo, branch, path, otherService, action, format } =
      repoDetails;

    const { app } = this.props;
    const downstream = app?.downstream;
    const clusterId = downstream?.cluster?.id;

    const gitops = downstream?.gitops;
    const provider = gitops?.provider;
    const hostname = gitops?.hostname;
    const httpPort = gitops?.httpPort;
    const sshPort = gitops?.sshPort;
    const newUri = getGitOpsUri(provider, ownerRepo, hostname, httpPort);

    const gitOpsInput = {
      provider,
      uri: newUri,
      branch: branch,
      path,
      format,
      action,
    };

    if (requiresHostname(provider)) {
      gitOpsInput.hostname = hostname;
    }
    if (provider === "bitbucket_server") {
      gitOpsInput.httpPort = httpPort;
      gitOpsInput.sshPort = sshPort;
    }
    if (provider === "other") {
      gitOpsInput.otherServiceName = otherService;
    }

    this.setState({ errorMsg: "" });

    try {
      const oldUri = gitops?.uri;
      if (newUri !== oldUri) {
        await this.createGitOpsRepo(gitOpsInput);
      }
      await this.updateAppGitOps(app.id, clusterId, gitOpsInput);

      if (newUri !== oldUri || gitops?.branch !== branch) {
        await this.handleTestConnection();
      }

      this.setState({ showGitOpsSettings: false, ownerRepo });
      return true;
    } catch (err) {
      console.log(err);
      this.setState({
        errorMsg: err ? err.message : "Something went wrong, please try again.",
      });
      return false;
    }
  };

  createGitOpsRepo = async (gitOpsInput) => {
    const res = await fetch(`${process.env.API_ENDPOINT}/gitops/create`, {
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        gitOpsInput: gitOpsInput,
      }),
      method: "POST",
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
  };

  updateAppGitOps = async (appId, clusterId, gitOpsInput) => {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/gitops/app/${appId}/cluster/${clusterId}/update`,
      {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          gitOpsInput: gitOpsInput,
        }),
        method: "PUT",
      }
    );
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
  };

  promptToDisableGitOps = () => {
    this.setState({ showDisableGitopsModalPrompt: true });
  };

  disableGitOps = async () => {
    this.setState({ disablingGitOps: true });

    const appId = this.props.app?.id;
    let clusterId;
    if (this.props.app?.downstream) {
      clusterId = this.props.app.downstream.cluster.id;
    }

    try {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/gitops/app/${appId}/cluster/${clusterId}/disable`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "POST",
        }
      );
      if (!res.ok && res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      if (res.ok && res.status === 204) {
        this.props.getAppsList();
        this.props.getGitops();
        this.props.refetch();
      }
    } catch (err) {
      console.log(err);
    } finally {
      this.setState({
        disablingGitOps: false,
        showDisableGitopsModalPrompt: false,
      });
    }
  };

  hideGitOpsSettings = () => {
    this.setState({ showGitOpsSettings: false });
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  componentDidMount() {
    const gitops = this.props.app.downstream.gitops;
    const gitopsEnabled = gitops?.enabled;
    if (!gitopsEnabled) {
      this.props.history.push(`/app/${this.props.app.slug}`);
    }
  }

  render() {
    const { app, isSingleApp } = this.props;
    const appTitle = app?.name;

    if (!app.downstream) {
      return <div />;
    }

    const gitops = app.downstream.gitops;
    const gitopsEnabled = gitops?.enabled;
    const gitopsConnected = gitops.isConnected;

    const { ownerRepo, testingConnection, disablingGitOps, errorMsg } =
      this.state;

    const deployKey = gitops?.deployKey;
    const addKeyUri = getAddKeyUri(gitops, ownerRepo);

    const selectedService = SERVICES.find((service) => {
      return service.value === gitops?.provider;
    });

    const renderIcons = (app) => {
      if (app?.iconUri) {
        return (
          <IconWrapper
            style={{ backgroundImage: `url(${app?.iconUri})` }}
          ></IconWrapper>
        );
      }
    };
    const getLabel = (app) => {
      const downstream = app?.downstream;
      const gitops = downstream?.gitops;
      const gitopsEnabled = gitops?.enabled;
      const gitopsConnected = gitops?.isConnected;
      return (
        <div style={{ alignItems: "center", display: "flex" }}>
          <span style={{ fontSize: 18, marginRight: "10px" }}>
            {renderIcons(app)}
          </span>
          <div className="flex flex-column">
            <div className={isSingleApp && "u-marginBottom--5"}>
              {isSingleApp ? (
                <span
                  style={{
                    fontSize: "16",
                    fontWeight: "bold",
                    color: "#323232",
                  }}
                >
                  {app.label}
                </span>
              ) : (
                <span style={{ fontSize: 14 }}>{app.label}</span>
              )}
            </div>
            <div style={{ fontSize: "14px" }}>
              {!gitopsEnabled && !gitopsConnected ? (
                <div className="flex" style={{ gap: "5px", color: "gray" }}>
                  <img src={not_enabled} alt="not_enabled" />
                  <p>Not Enabled</p>
                </div>
              ) : gitopsEnabled && !gitopsConnected ? (
                <div className="flex" style={{ gap: "5px", color: "orange" }}>
                  <img src={warning} alt="warning" />
                  <p>Repository access needed</p>
                </div>
              ) : (
                gitopsEnabled &&
                gitopsConnected && (
                  <div className="flex" style={{ gap: "5px", color: "green" }}>
                    <img src={enabled} alt="enabled" />
                    <p>Enabled</p>
                  </div>
                )
              )}
            </div>
          </div>
        </div>
      );
    };

    const apps = this.props?.appsList?.map((app) => ({
      ...app,
      value: app.name,
      label: app.name,
    }));

    return (
      <div className="GitOpsDeploy--step u-textAlign--left">
        <Helmet>
          <title>{`${appTitle} GitOps`}</title>
        </Helmet>
        <div className="flex-column flex1">
          <div className="GitopsSettings-noRepoAccess u-textAlign--left">
            <p className="step-title">GitOps Configuration</p>
            <p className="step-sub">
              Connect a git version control system so all application updates
              are committed to a git <br />
              repository. When GitOps is enabled, you cannot deploy updates
              directly from the <br />
              admin console.
            </p>
          </div>
          <div className="flex alignItems--center u-marginBottom--30">
            {isSingleApp && app ? (
              <div className="u-marginRight--5">{getLabel(app)}</div>
            ) : (
              <div className="flex flex1 flex-column u-marginRight--10">
                <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                  Select an application to configure
                </p>
                <div className="u-position--relative u-marginTop--5 u-marginBottom--10">
                  <Select
                    className="replicated-select-container select-large"
                    classNamePrefix="replicated-select"
                    placeholder="Select an application"
                    options={apps}
                    isSearchable={false}
                    getOptionLabel={(app) => getLabel(app)}
                    value={this.props.selectedApp}
                    onChange={this.props.handleAppChange}
                    isOptionSelected={(option) => {
                      option.value === this.props.selectedApp;
                    }}
                  />
                </div>
              </div>
            )}
            <div className="flex flex1 flex-column u-fontSize--small u-marginTop--20">
              {gitopsEnabled && gitopsConnected && (
                <a
                  style={{ color: "blue", cursor: "pointer" }}
                  disabled={disablingGitOps}
                  onClick={this.promptToDisableGitOps}
                >
                  {disablingGitOps
                    ? "Disabling GitOps"
                    : "Disable GitOps for this app"}
                </a>
              )}
            </div>
          </div>

          <div
            style={{
              marginBottom: "30px",
            }}
          >
            <Flex mb="15" align="center">
              <span
                className="icon small-warning-icon"
                style={{ width: "35px" }}
              />
              <p
                className="u-fontSize--large u-fontWeight--bold u-lineHeight--normal"
                style={{ color: "#DB9016" }}
              >
                Access to your repository is needed to push application updates
              </p>
            </Flex>
            <p
              className="u-fontSize--normal u-fontWeight--normal u-marginBottom--15"
              style={{ color: "#585858" }}
            >
              Add this SSH key on your
              <a
                className="replicated-link"
                href={addKeyUri}
                target="_blank"
                rel="noopener noreferrer"
              >
                {this.props.selectedApp.downstream.gitops.provider ===
                "bitbucket_server"
                  ? " account settings page, "
                  : " repository settings page, "}
              </a>
              and grant it write access.
            </p>
            <CodeSnippet
              canCopy={true}
              copyText="Copy key"
              onCopyText={<span className="u-textColor--success">Copied</span>}
            >
              {deployKey}
            </CodeSnippet>
          </div>

          <div className="flex justifyContent--spaceBetween alignItems--center">
            <div className="flex">
              <button
                className="btn secondary blue"
                onClick={() => this.props.stepFrom("action", "provider")}
              >
                Back to configuration
              </button>
            </div>
            {testingConnection ? (
              <Loader size="30" />
            ) : (
              <button
                className="btn primary blue"
                disabled={testingConnection}
                onClick={this.handleTestConnection}
              >
                Test connection to repository
              </button>
            )}
          </div>
        </div>

        <DisableModal
          isOpen={this.state.showDisableGitopsModalPrompt}
          setOpen={(e) => this.setState({ showDisableGitopsModalPrompt: e })}
          disableGitOps={this.disableGitOps}
          provider={selectedService}
        />

        <ConnectionModal
          isOpen={this.state.showConnectionModal}
          modalType={this.state.modalType}
          setOpen={(e) => this.setState({ showConnectionModal: e })}
          handleTestConnection={this.handleTestConnection}
          isTestingConnection={this.state.testingConnection}
          stepFrom={this.props.stepFrom}
          appSlug={this.props.app.slug}
          getAppsList={this.props.getAppsList}
          getGitops={this.props.getGitops}
        />
      </div>
    );
  }
}

export default withRouter(AppGitops);
