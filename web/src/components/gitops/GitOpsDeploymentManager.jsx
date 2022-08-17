import * as React from "react";
import Select from "react-select";
import find from "lodash/find";
import isEmpty from "lodash/isEmpty";
import classNames from "classnames";
import Loader from "../shared/Loader";
import ErrorModal from "../modals/ErrorModal";
import { withRouter, Link } from "react-router-dom";
import GitOpsRepoDetails from "./GitOpsRepoDetails";
import {
  getGitOpsUri,
  requiresHostname,
  Utilities,
} from "../../utilities/utilities";

import "../../scss/components/gitops/GitOpsDeploymentManager.scss";
import SetupProvider from "./SetupProvider";
import { Flex, Paragraph } from "../../styles/common";
import AppGitops from "../apps/AppGitops";

const STEPS = [
  {
    step: "provider",
    title: "GitOps Configuration",
  },
  {
    step: "action",
    title: "GitOps Configuration ",
  },
];

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

const BITBUCKET_SERVER_DEFAULT_HTTP_PORT = "7990";
const BITBUCKET_SERVER_DEFAULT_SSH_PORT = "7999";

class GitOpsDeploymentManager extends React.Component {
  state = {
    step: "provider",
    hostname: "",
    httpPort: "",
    sshPort: "",
    services: SERVICES,
    selectedService: SERVICES[0],
    providerError: null,
    finishingSetup: false,
    appsList: [],
    gitops: {},
    errorMsg: "",
    errorTitle: "",
    displayErrorModal: false,
    selectedApp: {},
    owner: "",
    repo: "",
    branch: "",
    path: "",
    gitopsConnected: false,
    gitopsEnabled: false,
  };

  componentDidMount() {
    this.getAppsList();
    this.getGitops();
  }

  componentDidUpdate(prevProps, prevState) {
    if (this.state.appsList !== prevState.appsList) {
      if (isEmpty(this.state.selectedApp)) {
        const updateSelectedApp = this.state.appsList.map((app) => {
          return {
            ...this.state.appsList[0],
            label: this.state.appsList[0].name,
            value: this.state.appsList[0].name,
          };
        });
        this.setState({ selectedApp: updateSelectedApp[0] });
      } else {
        const updateSelectedApp = this.state.appsList.map((app) => {
          return { ...app, label: app.name, value: app.name };
        });

        const newApp = updateSelectedApp.find((app) => {
          return app.id === this.state.selectedApp?.id;
        });
        this.setState({ selectedApp: newApp });
      }
    }
  }

  getAppsList = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/apps`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        console.log(
          "failed to get apps list, unexpected status code",
          res.status
        );
        return;
      }
      const response = await res.json();
      const apps = response.apps;

      this.setState({
        appsList: apps,
      });
      const updateSelectedApp = apps.find((app) => {
        return app.id === this.state.selectedApp?.id;
      });
      this.getInitialOwnerRepo(updateSelectedApp);

      return apps;
    } catch (err) {
      console.log(err);
      throw err;
    }
  };

  getGitops = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/gitops/get`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        console.log(
          "failed to get gitops settings, unexpected status code",
          res.status
        );
        return;
      }

      const freshGitops = await res.json();

      if (freshGitops?.enabled) {
        this.getInitialOwnerRepo(this.state.selectedApp);
        const selectedService = find(
          SERVICES,
          (service) => service.value === freshGitops.provider
        );
        this.setState({
          selectedService: selectedService
            ? selectedService
            : this.state.selectedService,
          hostname: freshGitops.hostname || "",
          httpPort: freshGitops.httpPort || "",
          sshPort: freshGitops.sshPort || "",
          gitops: freshGitops,
        });
      } else {
        this.setState({
          gitops: freshGitops,
        });
      }
    } catch (err) {
      console.log(err);
      throw err;
    }
  };

  getInitialOwnerRepo = (app) => {
    if (!app?.downstream) {
      this.setState({
        owner: "",
        repo: "",
        branch: "",
        path: "",
        gitopsEnabled: false,
        gitopsConnected: false,
      });
      return "";
    }

    const gitops = app.downstream.gitops;
    if (!gitops?.uri) {
      this.setState({
        owner: "",
        repo: "",
        branch: "",
        path: "",
        gitopsEnabled: gitops.enabled,
        gitopsConnected: gitops.isConnected,
      });
      return "";
    }

    const parsed = new URL(gitops?.uri);
    if (gitops?.provider === "bitbucket_server") {
      const project =
        parsed.pathname.split("/").length > 2 && parsed.pathname.split("/")[2];
      const repo =
        parsed.pathname.split("/").length > 4 && parsed.pathname.split("/")[4];
      if (project && repo) {
        this.setState({ owner: project, repo: repo });
      }
    } else {
      let path = parsed.pathname.slice(1); // remove the "/"
      const project = path.split("/")[0];
      const repo = path.split("/")[1];
      this.setState({
        owner: project,
        repo: repo,
        branch: gitops.branch,
        path: gitops.path,
        gitopsEnabled: gitops.enabled,
        gitopsConnected: gitops.isConnected,
      });
    }
  };

  isSingleApp = () => {
    return this.state.appsList?.length === 1;
  };

  providerChanged = () => {
    const { selectedService } = this.state;
    return selectedService?.value !== this.state.gitops?.provider;
  };

  hostnameChanged = () => {
    const { hostname, selectedService } = this.state;
    const provider = selectedService?.value;
    const savedHostname = this.state.gitops?.hostname || "";
    return (
      !this.providerChanged() &&
      requiresHostname(provider) &&
      hostname !== savedHostname
    );
  };

  httpPortChanged = () => {
    const { httpPort, selectedService } = this.state;
    const provider = selectedService?.value;
    const savedHttpPort = this.state.gitops?.httpPort || "";
    const isBitbucketServer = provider === "bitbucket_server";
    return (
      !this.providerChanged() && isBitbucketServer && httpPort !== savedHttpPort
    );
  };

  sshPortChanged = () => {
    const { sshPort, selectedService } = this.state;
    const provider = selectedService?.value;
    const savedSshPort = this.state.gitops?.sshPort || "";
    const isBitbucketServer = provider === "bitbucket_server";
    return (
      !this.providerChanged() && isBitbucketServer && sshPort !== savedSshPort
    );
  };

  getGitOpsInput = (
    provider,
    uri,
    branch,
    path,
    format,
    action,
    hostname,
    httpPort,
    sshPort
  ) => {
    let gitOpsInput = new Object();
    gitOpsInput.provider = provider;
    gitOpsInput.uri = uri;
    gitOpsInput.branch = branch || "";
    gitOpsInput.path = path;
    gitOpsInput.format = format;
    gitOpsInput.action = action;

    if (requiresHostname(provider)) {
      gitOpsInput.hostname = hostname;
    }

    const isBitbucketServer = provider === "bitbucket_server";
    if (isBitbucketServer) {
      gitOpsInput.httpPort = httpPort;
      gitOpsInput.sshPort = sshPort;
    }

    return gitOpsInput;
  };

  finishSetup = async (repoDetails = {}) => {
    this.setState({
      finishingSetup: true,
      errorTitle: "",
      errorMsg: "",
      displayErrorModal: false,
    });

    const {
      ownerRepo = "",
      branch = "",
      path = "",
      action = "commit",
      format = "single",
    } = repoDetails;

    const { hostname, selectedService } = this.state;

    const httpPort = this.state.httpPort || BITBUCKET_SERVER_DEFAULT_HTTP_PORT;
    const sshPort = this.state.sshPort || BITBUCKET_SERVER_DEFAULT_SSH_PORT;

    const provider = selectedService.value;
    const repoUri = getGitOpsUri(provider, ownerRepo, hostname, httpPort);
    const gitOpsInput = this.getGitOpsInput(
      provider,
      repoUri,
      branch,
      path,
      format,
      action,
      hostname,
      httpPort,
      sshPort
    );

    try {
      if (this.state.gitops?.enabled && this.providerChanged()) {
        await this.resetGitOps();
      }
      await this.createGitOpsRepo(gitOpsInput);

      const currentApp = find(this.state.appsList, {
        id: this.state.selectedApp.id,
      });

      const downstream = currentApp?.downstream;
      const clusterId = downstream?.cluster?.id;

      await this.updateAppGitOps(currentApp.id, clusterId, gitOpsInput);
      await this.getAppsList();
      await this.getGitops();

      return true;
    } catch (err) {
      console.log(err);
      this.setState({
        errorTitle: "Failed to finish gitops setup",
        errorMsg: err ? err.message : "Something went wrong, please try again.",
        displayErrorModal: true,
      });
      return false;
    } finally {
      this.setState({ finishingSetup: false });
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

  resetGitOps = async () => {
    const res = await fetch(`${process.env.API_ENDPOINT}/gitops/reset`, {
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
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

  updateSettings = () => {
    //  if (this.isSingleApp()) {
    this.stepFrom("provider", "action");
    // } else {
    //   this.finishSetup();
    // }
  };

  enableAppGitOps = async (app) => {
    if (!app.downstream) {
      return;
    }

    const downstream = app?.downstream;
    const gitops = downstream?.gitops;
    if (gitops?.enabled) {
      return;
    }

    if (isEmpty(this.state.gitops)) {
      return;
    }

    const { provider, hostname, httpPort, sshPort, uri } = this.state.gitops;
    const branch = "";
    const path = "";
    const format = "single";
    const action = "commit";
    const gitOpsInput = this.getGitOpsInput(
      provider,
      uri,
      branch,
      path,
      format,
      action,
      hostname,
      httpPort,
      sshPort
    );

    try {
      const clusterId = downstream?.cluster?.id;

      await this.updateAppGitOps(app.id, clusterId, gitOpsInput);
      this.props.history.push(`/app/${app.slug}/gitops`);
    } catch (err) {
      console.log(err);
      this.setState({
        errorTitle: "Failed to enable app gitops",
        errorMsg: err ? err.message : "Something went wrong, please try again.",
        displayErrorModal: true,
      });
    }
  };

  validStep = (step) => {
    const { selectedService, hostname } = this.state;

    this.setState({ providerError: null });
    if (step === "provider") {
      const provider = selectedService.value;
      if (requiresHostname(provider) && !hostname.length) {
        this.setState({
          providerError: {
            field: "hostname",
          },
        });
        return false;
      }
    }

    return true;
  };

  stepFrom = (from, to) => {
    if (this.validStep(from)) {
      this.setState({
        step: to,
      });
    }
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

  handleServiceChange = (selectedService) => {
    this.setState({ selectedService });
  };

  renderGitOpsProviderSelector = ({
    provider,
    hostname,
    httpPort,
    sshPort,
    services,
    selectedService,
    providerError,
  }) => {
    const isBitbucketServer = provider === "bitbucket_server";

    return (
      <Flex direction="column">
        <Flex width="100%">
          {/* left column */}
          <Flex direction="column" flex="1" mr="20">
            <div style={{ width: "100%" }}>
              <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                Git provider
              </p>
              <div className="u-position--relative  u-marginTop--5">
                <Select
                  className="replicated-select-container"
                  classNamePrefix="replicated-select"
                  placeholder="Select a GitOps service"
                  options={services}
                  isSearchable={false}
                  getOptionLabel={(service) =>
                    this.getLabel(service, service.label)
                  }
                  getOptionValue={(service) => service.label}
                  value={selectedService}
                  onChange={this.handleServiceChange}
                  isOptionSelected={(option) => {
                    option.value === selectedService;
                  }}
                />
              </div>
            </div>

            {isBitbucketServer && (
              <Flex flex="1" mt="30" width="100%">
                {this.renderHttpPort(provider, httpPort)}
              </Flex>
            )}
          </Flex>
          <Flex direction="column" flex="1" width="100%">
            {/* right column */}
            {this.renderHostName(
              provider,
              hostname,
              providerError,
              httpPort,
              sshPort
            )}
            {isBitbucketServer && (
              <Flex flex="1" mt="30" width="100%">
                {this.renderSshPort(provider, sshPort)}
              </Flex>
            )}
          </Flex>
        </Flex>
        <GitOpsRepoDetails
          owner={this.state.owner}
          repo={this.state.repo}
          branch={this.state.branch}
          path={this.state.path}
          appName={this.props.appName}
          hostname={hostname}
          selectedService={selectedService}
          onFinishSetup={this.finishSetup}
          ctaLoadingText="Finishing setup"
          ctaText="Finish setup"
          updateSettings={this.updateSettings}
          gitopsEnabled={this.state.gitopsEnabled}
          gitopsConnected={this.state.gitopsConnected}
        />
      </Flex>
    );
  };
  renderHttpPort = (provider, httpPort) => {
    const isBitbucketServer = provider === "bitbucket_server";
    if (isBitbucketServer) {
      return (
        <Flex flex="1" direction="column" width="100%">
          <Paragraph size="16" weight="bold" className="u-lineHeight--normal">
            HTTP Port <span>(Required)</span>
          </Paragraph>
          <input
            type="text"
            className="Input"
            placeholder={BITBUCKET_SERVER_DEFAULT_HTTP_PORT}
            value={httpPort}
            onChange={(e) => this.setState({ httpPort: e.target.value })}
          />
        </Flex>
      );
    }
  };

  renderSshPort = (provider, sshPort) => {
    const isBitbucketServer = provider === "bitbucket_server";
    if (!isBitbucketServer) {
      return <div className="flex flex1" />;
    }
    return (
      <div className="flex flex1 flex-column">
        <Paragraph size="16" weight="bold" className="u-lineHeight--normal">
          SSH Port <span>(Required)</span>
        </Paragraph>
        <input
          type="text"
          className="Input"
          placeholder={BITBUCKET_SERVER_DEFAULT_SSH_PORT}
          value={sshPort}
          onChange={(e) => this.setState({ sshPort: e.target.value })}
        />
      </div>
    );
  };

  renderHostName = (provider, hostname, providerError) => {
    if (requiresHostname(provider)) {
      return (
        <Flex direction="column" className="flex1" width="100%">
          <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
            Hostname
            <span> (Required)</span>
          </p>
          <input
            type="text"
            className={`Input ${
              providerError?.field === "hostname" && "has-error"
            } u-marginTop--5`}
            placeholder="hostname"
            value={hostname}
            onChange={(e) => this.setState({ hostname: e.target.value })}
          />
          {providerError?.field === "hostname" && (
            <p className="u-fontSize--small u-marginTop--5 u-textColor--error u-fontWeight--medium u-lineHeight--normal">
              A hostname must be provided
            </p>
          )}
        </Flex>
      );
    }
  };

  updateHttpPort = (httpPort) => {
    this.setState({ httpPort });
  };

  updateSSHPort = (sshPort) => {
    this.setState({ sshPort });
  };

  handleAppChange = (app) => {
    const currentApp = find(this.state.appsList, { id: app.id });
    this.getInitialOwnerRepo(currentApp);
    this.setState({ selectedApp: app, currentApp });
  };

  renderActiveStep = (step) => {
    const {
      hostname,
      httpPort,
      sshPort,
      services,
      selectedService,
      providerError,
      finishingSetup,
      selectedApp,
      owner,
      repo,
      branch,
      path,
    } = this.state;

    const provider = selectedService?.value;
    switch (step.step) {
      case "provider":
        return (
          <SetupProvider
            app={this.props.app}
            step={step}
            appsList={this.state.appsList}
            state={this.state}
            selectedApp={this.state.selectedApp}
            provider={provider}
            updateSettings={this.updateSettings}
            isSingleApp={this.isSingleApp}
            updateHttpPort={this.updateHttpPort}
            renderGitOpsProviderSelector={this.renderGitOpsProviderSelector}
            renderHostName={this.renderHostName}
            handleAppChange={this.handleAppChange}
            getAppsList={this.getAppsList}
            getGitops={this.getGitops}
          />
        );
      case "action":
        return (
          <AppGitops
            app={selectedApp}
            appsList={this.state.appsList}
            selectedApp={selectedApp}
            handleAppChange={this.handleAppChange}
            stepFrom={this.stepFrom}
            getAppsList={this.getAppsList}
            getGitops={this.getGitops}
          />
        );
      default:
        return (
          <div key={`default-active`} className="GitOpsDeploy--step">
            default
          </div>
        );
    }
  };

  getGitOpsStatus = (gitops) => {
    if (gitops?.enabled && gitops?.isConnected) {
      return "Enabled, Working";
    }
    if (gitops?.enabled) {
      return "Enabled, Failing";
    }
    return "Not Enabled";
  };

  renderGitOpsStatusAction = (app, gitops) => {
    if (gitops?.enabled && gitops?.isConnected) {
      return null;
    }
    if (gitops?.enabled) {
      return (
        <Link
          to={`/app/${app.slug}/troubleshoot`}
          className="gitops-action-link"
        >
          Troubleshoot
        </Link>
      );
    }

    return (
      <span
        onClick={() => this.enableAppGitOps(app)}
        className="gitops-action-link"
      >
        Enable
      </span>
    );
  };

  renderApps = () => {
    return (
      <div>
        {this.state.appsList.map((app) => {
          const downstream = app?.downstream;
          const gitops = downstream?.gitops;
          const gitopsEnabled = gitops?.enabled;
          const gitopsConnected = gitops?.isConnected;
          return (
            <div
              key={app.id}
              className="flex justifyContent--spaceBetween alignItems--center u-marginBottom--30"
            >
              <div className="flex alignItems--center">
                <div
                  style={{ backgroundImage: `url(${app.iconUri})` }}
                  className="appIcon u-position--relative"
                />
                <div className="u-marginLeft--10">
                  <p className="u-fontSize--large u-fontWeight--bold u-textColor--secondary u-marginBottom--5">
                    {app.name}
                  </p>
                  {gitopsEnabled && (
                    <Link
                      to={`/app/${app.slug}/gitops`}
                      className="gitops-action-link"
                    >
                      Manage GitOps settings
                    </Link>
                  )}
                </div>
              </div>
              <div className="flex-column alignItems--flexEnd">
                <div className="flex alignItems--center u-marginBottom--5">
                  <div
                    className={classNames("icon", {
                      "grayCircleMinus--icon":
                        !gitopsEnabled && !gitopsConnected,
                      "error-small": gitopsEnabled && !gitopsConnected,
                      "checkmark-icon": gitopsEnabled && gitopsConnected,
                    })}
                  />
                  <p
                    className={classNames(
                      "u-fontSize--normal u-marginLeft--5",
                      {
                        "u-textColor--bodyCopy":
                          !gitopsEnabled && !gitopsConnected,
                        "u-textColor--error": gitopsEnabled && !gitopsConnected,
                        "u-textColor--success":
                          gitopsEnabled && gitopsConnected,
                      }
                    )}
                  >
                    {this.getGitOpsStatus(gitops)}
                  </p>
                </div>
                {this.renderGitOpsStatusAction(app, gitops)}
              </div>
            </div>
          );
        })}
      </div>
    );
  };

  dataChanged = () => {
    return (
      this.providerChanged() ||
      this.hostnameChanged() ||
      this.httpPortChanged() ||
      this.sshPortChanged()
    );
  };

  renderConfiguredGitOps = () => {
    const {
      services,
      selectedService,
      hostname,
      httpPort,
      sshPort,
      providerError,
      finishingSetup,
    } = this.state;
    const provider = selectedService?.value;
    const isBitbucketServer = provider === "bitbucket_server";
    const dataChanged = this.dataChanged();
    return (
      <div className="u-textAlign--center">
        <div className="ConfiguredGitOps--wrapper">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--accent u-lineHeight--normal u-marginBottom--30">
            Admin Console GitOps
          </p>
          <div className="flex u-marginBottom--30">
            {this.renderGitOpsProviderSelector(services, selectedService)}
            {requiresHostname(selectedService?.value) &&
              this.renderHostName(
                selectedService?.value,
                hostname,
                providerError,
                httpPort,
                sshPort
              )}
          </div>
          {isBitbucketServer && (
            <div className="flex u-marginBottom--30">
              {this.renderHttpPort(selectedService?.value, httpPort)}
              {this.renderSshPort(selectedService?.value, sshPort)}
            </div>
          )}
          {dataChanged && (
            <button
              className="btn secondary u-marginBottom--30"
              disabled={finishingSetup}
              onClick={this.updateSettings}
            >
              {finishingSetup ? "Updating" : "Update"}
            </button>
          )}
          <div className="separator" />
          {this.renderApps()}
        </div>
      </div>
    );
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  render() {
    const { appsList, errorMsg, errorTitle, displayErrorModal } = this.state;

    if (!appsList.length) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const activeStep = find(STEPS, { step: this.state.step });
    return (
      <div className="GitOpsDeploymentManager--wrapper flex-column flex1">
        {this.renderActiveStep(activeStep)}
        {errorMsg && (
          <ErrorModal
            errorModal={displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            err={errorTitle}
            errMsg={errorMsg}
          />
        )}
      </div>
    );
  }
}

export default withRouter(GitOpsDeploymentManager);
