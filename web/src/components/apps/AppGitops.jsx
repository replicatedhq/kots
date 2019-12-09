import React, { Component } from "react";
import { graphql, compose, withApollo } from "react-apollo";
import Helmet from "react-helmet";
import url from "url";
import GitOpsRepoDetails from "../gitops/GitOpsRepoDetails";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import { testGitOpsConnection, disableAppGitops, updateAppGitOps, createGitOpsRepo } from "../../mutations/AppsMutations";
import { getServiceSite, getAddKeyUri, requiresHostname } from "../../utilities/utilities";

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

    let ownerRepo = "";
    if (props.app?.downstreams?.length) {
      const gitops = props.app.downstreams[0].gitops;
      if (gitops?.uri) {
        const parsed = url.parse(gitops?.uri);
        ownerRepo = parsed.path.slice(1);  // remove the "/"
      }
    }

    this.state = {
      ownerRepo,
      testingConnection: false,
      disablingGitOps: false,
      showGitOpsSettings: false
    };
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
    if (this.props.app?.downstreams?.length) {
      clusterId = this.props.app.downstreams[0].cluster.id;
    }

    try {
      await this.props.testGitOpsConnection(appId, clusterId);
      await this.props.refetch();
    } catch (err) {
      console.log(err);
    } finally {
      this.setState({ testingConnection: false, connectionTested: true });
    }
  }
  
  goToTroubleshootPage = () => {
    const { app, history } = this.props;
    history.push(`/app/${app.slug}/troubleshoot`);
  }

  updateGitOpsSettings = () => {
    this.setState({ showGitOpsSettings: true });
  }

  finishGitOpsSetup = async repoDetails => {
    const {
      ownerRepo,
      branch = "master",
      path = "",
      otherService = "",
      action = "commit",
      format = "single"
    } = repoDetails;

    const { app } = this.props;
    const downstream = app.downstreams[0];
    const clusterId = downstream?.cluster?.id;

    const gitops = downstream?.gitops;
    const provider = gitops?.provider;
    const hostname = gitops?.hostname;
    const serviceSite = getServiceSite(provider);

    const newUri = `https://${serviceSite}/${ownerRepo}`;
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
    if (provider === "other") {
      gitOpsInput.otherServiceName = otherService;
    }

    try {
      const oldUri = gitops?.uri;
      if (newUri !== oldUri) {
        await this.props.createGitOpsRepo(gitOpsInput);
      }
      await this.props.updateAppGitOps(app.id, clusterId, gitOpsInput);
      await this.props.refetch();
      await this.handleTestConnection();

      this.setState({ showGitOpsSettings: false, ownerRepo });
    } catch(err) {
      console.log(err);
    }
  }

  disableGitOps = async () => {
    this.setState({ disablingGitOps: true });
    const appId = this.props.app?.id;
    let clusterId;
    if (this.props.app?.downstreams?.length) {
      clusterId = this.props.app.downstreams[0].cluster.id;
    }

    try {
      await this.props.disableAppGitops(appId, clusterId);
      this.props.history.push(`/app/${this.props.app?.slug}`);
      this.props.refetch();
    } catch (err) {
      console.log(err);
    } finally {
      this.setState({ disablingGitOps: false });
    }
  }

  hideGitOpsSettings = () => {
    this.setState({ showGitOpsSettings: false });
  }

  render() {
    const { app } = this.props;
    const appTitle = app.name;

    if (!app.downstreams || app.downstreams.length === 0) {
      return (
        <div />
      );
    }

    if (this.props.app.downstreams.length !== 1) {
      return (
        <div>This feature is only available for applications that have exactly 1 downstream.</div>
      );
    }

    const gitops = app.downstreams[0].gitops;

    const {
      ownerRepo,
      testingConnection,
      disablingGitOps,
      showGitOpsSettings,
    } = this.state;

    const deployKey = gitops?.deployKey;
    const addKeyUri = getAddKeyUri(gitops?.uri, gitops?.provider, ownerRepo);
    const gitopsIsConnected = gitops.enabled && gitops.isConnected;

    const selectedService = SERVICES.find((service) => {
      return service.value === gitops?.provider;
    });

    return (
      <div className="GitOpsSettings--wrapper container flex-column u-overflow--auto u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>{`${appTitle} GitOps`}</title>
        </Helmet>

        {!ownerRepo || showGitOpsSettings ?
          <div className="u-marginTop--30">
            <GitOpsRepoDetails
              stepTitle={`Update GitOps for ${appTitle}`}
              appName={appTitle}
              ownerRepo={ownerRepo}
              branch={gitops?.branch}
              path={gitops?.path}
              format={gitops?.format}
              action={gitops?.action}
              selectedService={selectedService}
              onFinishSetup={this.finishGitOpsSetup}
              showCancelBtn={!!ownerRepo}
              onCancel={this.hideGitOpsSettings}
              otherService=""
            />
          </div>
          :
          <div className="GitOpsSettings">
            <div className={`flex u-marginTop--30 justifyContent--center alignItems--center ${gitopsIsConnected ? "u-marginBottom--30" : "u-marginBottom--20"}`}>
              {app.iconUri
                ? <div style={{ backgroundImage: `url(${app.iconUri})` }} className="appIcon u-position--relative" />
                : <span className="icon onlyAirgapBundleIcon" />
              }
              {gitopsIsConnected 
                ? <span className="icon connectionEstablished u-marginLeft--10" />
                : <span className="icon onlyNoConnectionIcon u-marginLeft--10" />
              }
              <span className="icon github-icon u-marginLeft--10" />
            </div>

            {gitopsIsConnected ?
              <div className="u-textAlign--center u-marginLeft--auto u-marginRight--auto">
                <p className="u-fontSize--largest u-fontWeight--bold u-color--tundora u-lineHeight--normal u-marginBottom--10">GitOps for {appTitle}</p>
                <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--30">
                  When an update is available for {appTitle}, the Admin Console will commit the fully<br/>rendered and deployable YAML to {gitops?.path ? `${gitops?.path}/rendered.yaml` : `the root of ${ownerRepo}/${gitops?.branch}`} in the {gitops?.branch} branch of<br/>the {ownerRepo} repo on {gitops?.provider}.
                </p>
                <div className="flex justifyContent--center">
                  <button className={`btn secondary u-marginRight--10 ${disablingGitOps ? "is-disabled" : "red"}`} onClick={this.disableGitOps}>{disablingGitOps ? "Disabling GitOps" : "Disable GitOps"}</button>
                  <button className="btn secondary lightBlue" onClick={this.updateGitOpsSettings}>Update GitOps Settings</button>
                </div>
              </div>
              :
              <div>
                <div className="GitopsSettings-noRepoAccess">
                  <p className="title">Unable to access the repository</p>
                  <p className="sub">Please check that the deploy key is added and has write access</p>
                </div>

                <div className="u-marginBottom--30">
                  <p className="u-fontSize--large u-fontWeight--bold u-color--tundora u-lineHeight--normal u-marginBottom--5">
                    Deployment key
                  </p>
                  <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-marginBottom--15">
                    Copy this deploy key to the
                    <a className="replicated-link" href={addKeyUri} target="_blank" rel="noopener noreferrer"> repo settings page.</a>
                  </p>
                  <CodeSnippet
                    canCopy={true}
                    copyText="Copy key"
                    onCopyText={<span className="u-color--chateauGreen">Deploy key has been copied to your clipboard</span>}>
                    {deployKey}
                  </CodeSnippet>
                </div>

                <div className="flex justifyContent--spaceBetween alignItems--center">
                  <div className="flex">
                    <button className={`btn secondary u-marginRight--10 ${testingConnection ? "is-disabled" : "lightBlue"}`} onClick={this.handleTestConnection}>{testingConnection ? "Testing connection" : "Try again"}</button>
                    <button className="btn primary blue" onClick={this.goToTroubleshootPage}>Troubleshoot</button>
                  </div>
                  <button className="btn secondary dustyGray" onClick={this.updateGitOpsSettings}>Update GitOps Settings</button>
                </div>
              </div>
            }
          </div>
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
  graphql(disableAppGitops, {
    props: ({ mutate }) => ({
      disableAppGitops: (appId, clusterId) => mutate({ variables: { appId, clusterId } })
    })
  }),
  graphql(createGitOpsRepo, {
    props: ({ mutate }) => ({
      createGitOpsRepo: (gitOpsInput) => mutate({ variables: { gitOpsInput } })
    })
  }),
  graphql(updateAppGitOps, {
    props: ({ mutate }) => ({
      updateAppGitOps: (appId, clusterId, gitOpsInput) => mutate({ variables: { appId, clusterId, gitOpsInput } })
    })
  }),
)(AppGitops);
