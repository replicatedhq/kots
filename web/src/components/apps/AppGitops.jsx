import React, { Component } from "react";
import { graphql, compose, withApollo } from "react-apollo";
import Helmet from "react-helmet";
import Modal from "react-modal";
import Select from "react-select";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import url from "url";
import { testGitOpsConnection, updateAppGitOps } from "../../mutations/AppsMutations";

import "../../scss/components/gitops/GitOpsSettings.scss";

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
]

class AppGitops extends Component {
  constructor(props) {
    super(props);

    let gitops = null;
    let ownerRepo = "";
    let hostname = "";
    if (props.app?.downstreams && props.app.downstreams.length > 0) {
      gitops = props.app.downstreams[0].gitops;
      const parsed = url.parse(gitops?.uri);
      ownerRepo = parsed.path.slice(1);  // remove the "/"
      hostname = parsed.host;
    }

    this.state = {
      provider: gitops?.provider,
      ownerRepo,
      branch: gitops?.branch,
      path: gitops?.path,
      format: gitops?.format,
      hostname,
      testingConnection: false,
      displayDeployKeyModal: false,
      updatingIntegration: false
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

  handleTestConnection = async () => {
    this.setState({ testingConnection: true });
    const appId = this.props.app?.id;
    let clusterId;
    if (this.props.app?.downstreams && this.props.app.downstreams.length > 0) {
      clusterId = this.props.app.downstreams[0].cluster.id;
    }

    try {
      await this.props.testGitOpsConnection(appId, clusterId);
      this.setState({ testingConnection: false });
      this.props.refetch();
    } catch (err) {
      this.setState({ testingConnection: false });
      console.log(err);
    }
  }

  handleServiceChange = (selectedService) => {
    this.setState({
      provider: selectedService.value,
    });
  }

  handleUpdate = async () => {
    const {
      provider,
      ownerRepo,
      branch,
      path,
      actionPath,
      otherService,
      format,
      hostname
    } = this.state;

    const clusterId = this.props.app.downstreams[0]?.cluster?.id;
    const isGitlab = provider === "gitlab" || provider === "gitlab_enterprise";
    const isBitbucket = provider === "bitbucket" || provider === "bitbucket_server";
    const serviceUri = isGitlab ? "gitlab.com" : isBitbucket ? "bitbucket.org" : "github.com";

    let gitOpsInput = new Object();
    gitOpsInput.provider = provider;
    gitOpsInput.uri = `https://${serviceUri}/${ownerRepo}`;
    gitOpsInput.owner = ownerRepo;
    gitOpsInput.branch = branch || "master";
    gitOpsInput.path = path;
    gitOpsInput.format = format;
    gitOpsInput.action = actionPath;
    if (provider === "gitlab_enterprise" || provider === "github_enterprise") {
      gitOpsInput.hostname = hostname;
    }
    if (provider === "other") {
      gitOpsInput.otherServiceName = otherService;
    }

    try {
      this.setState({ updatingIntegration: true });
      await this.props.updateAppGitOps(this.props.app.id, clusterId, gitOpsInput);
      await this.props.refetch();
      this.setState({ updatingIntegration: false });
    } catch (error) {
      console.log(error);
      this.setState({ updatingIntegration: false });
    }
  }

  render() {
    const { app } = this.props;
    const appTitle = app.name;

    if (!app.downstreams || app.downstreams.length === 0) {
      return (
        <div />
      );
    }

    const gitops = app.downstreams[0].gitops;

    const otherService = "";
    const providerError = null;

    const {
      ownerRepo,
      provider,
      branch,
      path,
      hostname,
      testingConnection,
      displayDeployKeyModal,
    } = this.state;

    const selectedService = SERVICES.find((service) => {
      return service.value === provider;
    });

    const isGitlab = selectedService?.value === "gitlab" || selectedService?.value === "gitlab_enterprise";
    const isBitbucket = selectedService?.value === "bitbucket" || selectedService?.value === "bitbucket_server";

    const gitUri = gitops?.uri;
    const deployKey = gitops?.deployKey;

    let addKeyUri = `${gitUri}/settings/keys/new`;
    if (isGitlab) {
      addKeyUri = `${gitUri}/-/settings/repository`;
    } else if (isBitbucket) {
      const owner = ownerRepo.split("/").length && ownerRepo.split("/")[0];
      addKeyUri = `https://bitbucket.org/account/user/${owner}/ssh-keys/`;
    }

    if (this.props.app.downstreams.length !== 1) {
      return (
        <div>This feature is only available for applications that have exactly 1 downstream.</div>
      );
    }

    return (
      <div className="GitOpsSettings--wrapper container flex-column u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>{`${appTitle} GitOps`}</title>
        </Helmet>
        <div className="GitOpsSettings">
          <div className="u-marginTop--15">
            <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna">GitOps Settings</h2>
            <p className="u-fontSize--large u-fontWeight--medium u-color--tundora u-lineHeight--medium u-marginTop--30">
              <span className={`u-marginRight--5 icon ${gitops.isConnected ? "checkmark-icon" : "exclamationMark--icon"} u-verticalAlign--neg2`} />GitOps is enabled for {appTitle}{!gitops.isConnected && " but does not have permission to commit to your repository"}
            </p>
            {gitops.enabled && !gitops.isConnected &&
               <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginTop--10 u-marginBottom--20">
               You first need to add the deployment key at the bottom of this page to the GitHub repository you would like to have the commits made.
             </p>
            }
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginTop--10 u-marginBottom--20">
              When GitOps is enabled, all changes to the application (upstream updates, config changes, license updates) will be commited to a git repo instead of deployed directly from the Admin Console.
            </p>
            <div className="GitOpsDeploy--step">
              <div className="flex-column u-textAlign--left u-marginBottom--30">
                <div className={`flex flex1 ${selectedService?.value !== "other" && "u-marginBottom--20"}`}>
                  <div className="flex flex1 flex-column u-marginRight--10">
                    <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Which GitOps provider do you use?</p>
                    <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">If your provider is not listed, select “Other”.</p>
                    <div className="u-position--relative">
                      <Select
                        className="replicated-select-container"
                        classNamePrefix="replicated-select"
                        placeholder="Select a GitOps service"
                        options={SERVICES}
                        isSearchable={false}
                        getOptionLabel={(service) => this.getLabel(service, service.label)}
                        getOptionValue={(service) => service.label}
                        value={selectedService}
                        onChange={this.handleServiceChange}
                        isOptionSelected={(option) => { option.value === selectedService }}
                      />
                    </div>
                  </div>
                  {selectedService?.value === "other" ?
                    <div className="flex flex1 flex-column u-marginLeft--10">
                      <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">What GitOps service do you use?</p>
                      <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">Not all services are supported.</p>
                      <input type="text" className="Input" placeholder="What service would you like to use" value={otherService} onChange={(e) => this.setState({ otherService: e.target.value })} />
                      {providerError?.field === "other" && <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">A GitOps service name must be provided</p>}
                    </div>
                  :
                    <div className="flex flex1 flex-column u-marginLeft--10">
                      <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Owner &amp; Repository</p>
                      <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">Which repository will the commit be made?</p>
                      <input type="text" className="Input" placeholder="owner/repository" value={ownerRepo} onChange={(e) => this.setState({ ownerRepo: e.target.value })} />
                      {providerError?.field === "ownerRepo" && <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">A owner and repository must be provided</p>}
                    </div>
                  }
                </div>
                {selectedService?.value === "github_enterprise" || selectedService?.value === "gitlab_enterprise" ?
                  <div className="flex flex1 u-marginBottom--20">
                    <div className="flex flex1 flex-column u-marginRight--10">
                      <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Hostname</p>
                      <input type="text" className="Input" placeholder="hostname" value={hostname} onChange={(e) => this.setState({ hostname: e.target.value })} />
                      {providerError?.field === "hostname" && <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">A hostname must be provided</p>}
                    </div>
                    <div className="flex flex1 flex-column u-marginLeft--10" />
                  </div>
                : null}
                {selectedService?.value !== "other" &&
                  <div className="flex flex1">
                    <div className="flex flex1 flex-column u-marginRight--10">
                      <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Branch</p>
                      <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">If no branch is specified, master will be used.</p>
                      <input type="text" className={`Input`} placeholder="master" value={branch || ""} onChange={(e) => this.setState({ branch: e.target.value })} />
                    </div>
                    <div className="flex flex1 flex-column u-marginLeft--10">
                      <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Path</p>
                      <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">Where in your repo should deployment file live?</p>
                      <input type="text" className={"Input"} placeholder="/my-path" value={path || ""} onChange={(e) => this.setState({ path: e.target.value })} />
                    </div>
                  </div>
                }
              </div>
              <div className={`GitOpsSettingsConnected inactive ${gitops.isConnected ? "" : "u-display--none"}`}>
                <p className="u-fontSize--large u-color--tundora u-fontWeight--medium u-lineHeight--normal">
                  <span className="u-marginRight--5 icon checkmark-icon u-verticalAlign--neg2" />Connected and working!
                </p>
              </div>
              <div className={`GitOpsSettingsNotConnected inactive u-marginBottom--10 ${gitops.isConnected ? "u-display--none" : ""}`}>
                <div className="flex justifyContent--center alignItems--center u-marginBottom--20">
                  <p className="u-fontSize--large u-color--tundora u-fontWeight--medium u-lineHeight--normal">
                    <span className="u-marginRight--5 icon error-small u-verticalAlign--neg2" />Unable to connect to repo
                  </p>
                  <button className={`btn small secondary u-marginLeft--10 ${testingConnection && "is-disabled"}`} onClick={this.handleTestConnection} disabled={testingConnection}>{testingConnection ? "Testing connection" : "Test connection"}</button>
                </div>
                <p className="u-fontSize--large u-color--tundora u-fontWeight--medium u-lineHeight--normal u-marginBottom--20">
                  To complete the setup, please add your <span onClick={() => this.setState({ displayDeployKeyModal: true })} className="replicated-link">deploy key</span> to your repo. To add a deploy
                  key, <a className="replicated-link" href={addKeyUri} target="_blank" rel="noopener noreferrer">click here</a> and use the following key (check the box for write access).
                </p>
              </div>
              <div className="u-marginBottom--10">
                <button className="btn primary blue" type="button" onClick={this.handleUpdate} disabled={this.state.updatingIntegration}>{this.state.updatingIntegration ? "Updating" : "Update"}</button>
              </div>
            </div>
          </div>
        </div>
        {displayDeployKeyModal &&
          <Modal
            isOpen={displayDeployKeyModal}
            onRequestClose={() => this.setState({ displayDeployKeyModal: false })}
            shouldReturnFocusAfterClose={false}
            contentLabel="Deploy key modal"
            ariaHideApp={false}
            className="Modal LargeSize"
          >
            <div className="Modal-body">
              <p className="u-fontSize--large u-fontWeight--medium u-color--tundora u-lineHeight--normal u-marginBottom--10">
                When adding this key, be sure to check the box for write access.
              </p>
              <CodeSnippet
                canCopy={true}
                onCopyText={<span className="u-color--chateauGreen">Deploy key has been copied to your clipboard</span>}>
                {deployKey}
              </CodeSnippet>
              <div className="u-marginTop--10 u-textAlign--center">
                <button onClick={() => this.setState({ displayDeployKeyModal: false })} className="btn primary">Ok, got it!</button>
              </div>
            </div>
          </Modal>
        }
      </div>
    );
  }
}

export default compose(
  withApollo,
  graphql(testGitOpsConnection, {
    props: ({ mutate }) => ({
      testGitOpsConnection: (appId, clusterId) => mutate({ variables: { appId, clusterId } })
    })
  }),
  graphql(updateAppGitOps, {
    props: ({ mutate }) => ({
      updateAppGitOps: (appId, clusterId, gitOpsInput) => mutate({ variables: { appId, clusterId, gitOpsInput } })
    })
  }),
)(AppGitops);
