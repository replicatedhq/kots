import * as React from "react";
import Select from "react-select";
import find from "lodash/find";
import classNames from "classnames";
import Loader from "../shared/Loader";
import { withRouter, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { listApps, getGitOpsRepo } from "@src/queries/AppsQueries";
import GitOpsFlowIllustration from "./GitOpsFlowIllustration";
import GitOpsRepoDetails from "./GitOpsRepoDetails";
import { createGitOpsRepo, updateGitOpsRepo, updateAppGitOps, resetGitOpsData } from "@src/mutations/AppsMutations";
import { getServiceSite, requiresHostname } from "../../utilities/utilities";

import "../../scss/components/gitops/GitOpsDeploymentManager.scss";

const STEPS = [
  {
    step: "setup",
    title: "Set up GitOps",
  },
  {
    step: "provider",
    title: "GitOps provider",
  },
  {
    step: "action",
    title: "GitOps action ",
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
  {
    value: "other",
    label: "Other",
  }
];

class GitOpsDeploymentManager extends React.Component {
  state = {
    step: "setup",
    hostname: "",
    services: SERVICES,
    selectedService: SERVICES[0],
    providerError: null,
    finishingSetup: false,
  }

  componentDidUpdate(lastProps) {
    const { getGitOpsRepoQuery } = this.props;
    if (getGitOpsRepoQuery?.getGitOpsRepo && getGitOpsRepoQuery.getGitOpsRepo !== lastProps.getGitOpsRepoQuery?.getGitOpsRepo) {
      const { enabled, provider, hostname } = getGitOpsRepoQuery.getGitOpsRepo;
      if (enabled) {
        const selectedService = find(SERVICES, service => service.value === provider);
        this.setState({
          selectedService: selectedService,
          hostname: hostname || ""
        });
      }
    }
  }

  isSingleApp = () => {
    const kotsApps = this.props.listAppsQuery.listApps?.kotsApps;
    return kotsApps?.length === 1;
  }

  providerChanged = () => {
    const { selectedService } = this.state;
    const getGitOpsRepo = this.props.getGitOpsRepoQuery?.getGitOpsRepo;
    return selectedService?.value !== getGitOpsRepo?.provider;
  }

  hostnameChanged = () => {
    const { hostname, selectedService } = this.state;
    const provider = selectedService?.value;
    const getGitOpsRepo = this.props.getGitOpsRepoQuery?.getGitOpsRepo;
    const savedHostname = getGitOpsRepo.hostname || "";
    return !this.providerChanged() && requiresHostname(provider) && hostname !== savedHostname;
  }

  getGitOpsInput = (provider, uri, branch, path, format, action, hostname) => {
    let gitOpsInput = new Object();
    gitOpsInput.provider = provider;
    gitOpsInput.uri = uri;
    gitOpsInput.branch = branch || "master";
    gitOpsInput.path = path;
    gitOpsInput.format = format;
    gitOpsInput.action = action;
    if (requiresHostname(provider)) {
      gitOpsInput.hostname = hostname;
    }

    return gitOpsInput;
  }

  finishSetup = async (repoDetails = {}) => {
    this.setState({ finishingSetup: true });

    const {
      ownerRepo = "",
      branch = "",
      path = "",
      action = "commit",
      format = "single"
    } = repoDetails;

    const {
      hostname,
      selectedService
    } = this.state;

    const provider = selectedService.value;
    const serviceSite = getServiceSite(provider);
    const repoUri = this.isSingleApp() ? `https://${serviceSite}/${ownerRepo}` : "";
    const gitOpsInput = this.getGitOpsInput(provider, repoUri, branch, path, format, action, hostname);

    try {
      const getGitOpsRepo = this.props.getGitOpsRepoQuery?.getGitOpsRepo;
      if (getGitOpsRepo?.enabled) {
        if (this.providerChanged()) {
          await this.props.resetGitOpsData();
          await this.props.createGitOpsRepo(gitOpsInput);
        } else {
          const uriToUpdate = this.isSingleApp() ? getGitOpsRepo?.uri : "";
          await this.props.updateGitOpsRepo(gitOpsInput, uriToUpdate);
        }
      } else {
        await this.props.createGitOpsRepo(gitOpsInput);
      }

      if (this.isSingleApp()) {
        const { listAppsQuery } = this.props;
        const kotsApps = listAppsQuery.listApps?.kotsApps;
        const app = kotsApps[0];
        const downstream = app.downstreams[0];
        const clusterId = downstream?.cluster?.id;

        await this.props.updateAppGitOps(app.id, clusterId, gitOpsInput);
        this.props.history.push(`/app/${app.slug}/gitops`);
      } else {
        this.setState({ step: "", finishingSetup: false });
        this.props.listAppsQuery.refetch();
        this.props.getGitOpsRepoQuery.refetch();
      }
    } catch (error) {
      console.log(error);
      this.setState({ finishingSetup: false });
    }
  }

  updateSettings = () => {
    if (this.isSingleApp()) {
      this.stepFrom("provider", "action");
    } else {
      this.finishSetup();
    }
  }

  enableAppGitOps = async app => {
    if (!app.downstreams?.length) {
      return;
    }

    const downstream = app.downstreams[0];
    const gitops = downstream?.gitops;
    if (gitops?.enabled) {
      return;
    }

    const getGitOpsRepo = this.props.getGitOpsRepoQuery?.getGitOpsRepo;
    if (!getGitOpsRepo) {
      return;
    }

    const { provider, hostname, uri } = getGitOpsRepo;
    const branch = "master";
    const path = "";
    const format = "single";
    const action = "commit";
    const gitOpsInput = this.getGitOpsInput(provider, uri, branch, path, format, action, hostname);

    try {
      const clusterId = downstream?.cluster?.id;
      await this.props.updateAppGitOps(app.id, clusterId, gitOpsInput);
      this.props.history.push(`/app/${app.slug}/gitops`);
    } catch (error) {
      console.log(error);
    }
  }

  validStep = (step) => {
    const {
      selectedService,
      hostname,
    } = this.state;

    this.setState({ providerError: null });
    if (step === "provider") {
      const provider = selectedService.value;
      if (requiresHostname(provider) && !hostname.length) {
        this.setState({
          providerError: {
            field: "hostname"
          }
        });
        return false;
      }
    }

    return true;
  }

  stepFrom = (from, to) => {
    if (this.validStep(from)) {
      this.setState({
        step: to
      });
    }
  }

  renderIcons = (service) => {
    if (service) {
      return <span className={`icon gitopsService--${service.value}`} />;
    } else {
      return;
    }
  }

  getLabel = (service, label) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span style={{ fontSize: 18, marginRight: "10px" }}>{this.renderIcons(service)}</span>
        <span style={{ fontSize: 14 }}>{label}</span>
      </div>
    );
  }

  handleServiceChange = (selectedService) => {
    this.setState({ selectedService });
  }

  renderGitOpsProviderSelector = (services, selectedService) => {
    return (
      <div className="flex flex1 flex-column u-marginRight--10">
        <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Which GitOps provider do you use?</p>
        <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">If your provider is not listed, select “Other”.</p>
        <div className="u-position--relative">
          <Select
            className="replicated-select-container"
            classNamePrefix="replicated-select"
            placeholder="Select a GitOps service"
            options={services}
            isSearchable={false}
            getOptionLabel={(service) => this.getLabel(service, service.label)}
            getOptionValue={(service) => service.label}
            value={selectedService}
            onChange={this.handleServiceChange}
            isOptionSelected={(option) => { option.value === selectedService }}
          />
        </div>
      </div>
    );
  }

  renderHostName = (provider, hostname, providerError) => {
    if (!requiresHostname(provider)) {
      return <div className="flex flex1" />;
    }
    return (
      <div className="flex flex1 flex-column u-marginLeft--10">
        <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Hostname</p>
        <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">Hostname of your GitOps server.</p>
        <input type="text" className={`Input ${providerError?.field === "hostname" && "has-error"}`} placeholder="hostname" value={hostname} onChange={(e) => this.setState({ hostname: e.target.value })} />
        {providerError?.field === "hostname" && <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">A hostname must be provided</p>}
      </div>
    );
  }

  renderActiveStep = (step) => {
    const {
      hostname,
      services,
      selectedService,
      otherService,
      providerError,
      finishingSetup,
    } = this.state;

    switch (step.step) {
      case "setup":
        return (
        <div key={`${step.step}-active`} className="GitOpsDeploy--step">
          <p className="step-title">Deploy using a GitOps workflow</p>
          <p className="step-sub">Connect a git version control system to this Admin Console. After setting this up, it will be<br/>possible to have all application updates (upstream updates, license updates, config changes)<br/>directly commited to any git repository and automatic deployments will be disabled.</p>
          <GitOpsFlowIllustration />
          <div>
            <button className="btn primary blue u-marginTop--10" type="button" onClick={() => this.stepFrom("setup", "provider")}>Get started</button>
          </div>
        </div>
      );
      case "provider":
        return (
          <div key={`${step.step}-active`} className="GitOpsDeploy--step u-textAlign--left">
            <p className="step-title">{step.title}</p>
            <p className="step-sub">Before the Admin Console can push changes to your Git repository, some information about your Git configuration is required.</p>
            <div className="flex-column u-textAlign--left u-marginBottom--30">
              <div className="flex flex1">
                {this.renderGitOpsProviderSelector(services, selectedService)}
                {this.renderHostName(selectedService?.value, hostname, providerError)}
              </div>
            </div>
            <div>
              <button
                className="btn primary blue"
                type="button"
                disabled={finishingSetup}
                onClick={this.updateSettings}
              >
                {finishingSetup
                  ? "Finishing setup"
                  : this.isSingleApp()
                    ? "Continue to deployment action"
                    : "Finish GitOps setup"
                }
              </button>
            </div>
          </div>
        );
      case "action":
        return (
          <GitOpsRepoDetails
            appName={this.props.appName}
            selectedService={selectedService}
            otherService={otherService}
            onFinishSetup={this.finishSetup}
          />
        );
      default:
        return <div key={`default-active`} className="GitOpsDeploy--step">default</div>;
    }
  }

  getGitOpsStatus = gitops => {
    if (gitops?.enabled && gitops?.isConnected) {
      return "Enabled, Working";
    }
    if (gitops?.enabled) {
      return "Enabled, Failing";
    }
    return "Not Enabled";
  }

  renderGitOpsStatusAction = (app, gitops) => {
    if (gitops?.enabled && gitops?.isConnected) {
      return null;
    }
    if (gitops?.enabled) {
      return <Link to={`/app/${app.slug}/troubleshoot`} className="gitops-action-link">Troubleshoot</Link>
    }

    return <span onClick={() => this.enableAppGitOps(app)} className="gitops-action-link">Enable</span>;
  }

  renderApps = () => {
    const { listAppsQuery } = this.props;
    const kotsApps = listAppsQuery.listApps?.kotsApps;
    return (
      <div>
        {kotsApps.map(app => {
          const downstream = app.downstreams?.length && app.downstreams[0];
          const gitops = downstream?.gitops;
          const gitopsEnabled = gitops?.enabled;
          const gitopsConnected = gitops?.isConnected;
          return (
            <div key={app.id} className="flex justifyContent--spaceBetween alignItems--center u-marginBottom--30">
              <div className="flex alignItems--center">
                <div style={{ backgroundImage: `url(${app.iconUri})` }} className="appIcon u-position--relative" />
                <p className="u-fontSize--large u-fontWeight--bold u-color--tundora u-marginLeft--10">{app.name}</p>
              </div>
              <div className="flex-column alignItems--flexEnd">
                <div className="flex alignItems--center u-marginBottom--5">
                  <div className={classNames("icon", {
                    "grayCircleMinus--icon": !gitopsEnabled && !gitopsConnected,
                    "error-small": gitopsEnabled && !gitopsConnected,
                    "checkmark-icon": gitopsEnabled && gitopsConnected
                    })}
                  />
                  <p className={classNames("u-fontSize--normal u-marginLeft--5", {
                    "u-color--dustyGray": !gitopsEnabled && !gitopsConnected,
                    "u-color--chestnut": gitopsEnabled && !gitopsConnected,
                    "u-color--chateauGreen": gitopsEnabled && gitopsConnected,
                  })}>
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
  }

  renderConfiguredGitOps = () => {
    const { services, selectedService, hostname, providerError, finishingSetup } = this.state;
    const dataChanged = this.providerChanged() || this.hostnameChanged();
    return (
      <div className="u-textAlign--center">
        <div className="ConfiguredGitOps--wrapper">
            <p className="u-fontSize--largest u-fontWeight--bold u-color--tundora u-lineHeight--normal u-marginBottom--30">Admin Console GitOps</p>
            <div className={`flex ${dataChanged ? "u-marginBottom--20" : "u-marginBottom--30"}`}>
              {this.renderGitOpsProviderSelector(services, selectedService)}
              {this.renderHostName(selectedService?.value, hostname, providerError)}
            </div>
            {dataChanged &&
              <button className="btn secondary u-marginBottom--30" disabled={finishingSetup} onClick={this.updateSettings}>
                {finishingSetup ? "Updating" : "Update"}
              </button>
            }
            <div className="separator" />
            {this.renderApps()}
        </div>
      </div>
    );
  }

  render() {
    const { listAppsQuery, getGitOpsRepoQuery } = this.props;
    if (listAppsQuery.loading || getGitOpsRepoQuery.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const gitopsRepo = getGitOpsRepoQuery.getGitOpsRepo;
    const activeStep = find(STEPS, { step: this.state.step });
    return (
      <div className="GitOpsDeploymentManager--wrapper flex-column flex1">
        {gitopsRepo.enabled && this.state.step !== "action" ?
          this.renderConfiguredGitOps()
          : activeStep &&
          this.renderActiveStep(activeStep)
        }
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(listApps, {
    name: "listAppsQuery",
    options: () => ({
      fetchPolicy: "no-cache"
    })
  }),
  graphql(getGitOpsRepo, {
    name: "getGitOpsRepoQuery",
    options: () => ({
      fetchPolicy: "no-cache"
    })
  }),
  graphql(createGitOpsRepo, {
    props: ({ mutate }) => ({
      createGitOpsRepo: (gitOpsInput) => mutate({ variables: { gitOpsInput } })
    })
  }),
  graphql(updateGitOpsRepo, {
    props: ({ mutate }) => ({
      updateGitOpsRepo: (gitOpsInput, uriToUpdate) => mutate({ variables: { gitOpsInput, uriToUpdate } })
    })
  }),
  graphql(resetGitOpsData, {
    props: ({ mutate }) => ({
      resetGitOpsData: () => mutate()
    })
  }),
  graphql(updateAppGitOps, {
    props: ({ mutate }) => ({
      updateAppGitOps: (appId, clusterId, gitOpsInput) => mutate({ variables: { appId, clusterId, gitOpsInput } })
    })
  }),
)(GitOpsDeploymentManager);
