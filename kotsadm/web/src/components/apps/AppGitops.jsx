import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import Helmet from "react-helmet";
import url from "url";
import GitOpsRepoDetails from "../gitops/GitOpsRepoDetails";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import { testGitOpsConnection, disableAppGitops, updateAppGitOps, createGitOpsRepo } from "../../mutations/AppsMutations";
import { getServiceSite, getAddKeyUri, requiresHostname } from "../../utilities/utilities";
import Modal from "react-modal";

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
      showDisableGitopsModalPrompt: false,
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
      await this.props.testGitOpsConnection(appId, clusterId).then((res) => {
        if (res.data.testGitOpsConnection) {
          this.props.history.push("/gitops");
        } else {
          this.props.refetch();
        }
      })
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
      branch,
      path,
      otherService,
      action,
      format
    } = repoDetails;

    const { app } = this.props;
    const downstream = app.downstreams[0];
    const clusterId = downstream?.cluster?.id;

    const gitops = downstream?.gitops;
    const provider = gitops?.provider;
    const hostname = gitops?.hostname;
    const serviceSite = getServiceSite(provider, hostname);

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
      
      if (newUri !== oldUri || gitops?.branch !== branch) {
        await this.handleTestConnection();
      }

      this.setState({ showGitOpsSettings: false, ownerRepo });
    } catch(err) {
      console.log(err);
    }
  }

  promptToDisableGitOps = () => {
    this.setState({ showDisableGitopsModalPrompt: true });
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

  getProviderIconClassName = provider => {
    switch (provider) {
      case "github" || "github_enterprise":
        return "github-icon";
      case "gitlab":
        return "gitlab-icon";
      case "bitbucket":
        return "bitbucket-icon";
      default:
        return "github-icon";
    }
  }

  componentDidMount() {
    const gitops = this.props.app.downstreams[0].gitops;
    const gitopsEnabled = gitops?.enabled;
    if (!gitopsEnabled) {
      this.props.history.push(`/app/${this.props.app.slug}`);
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
      showDisableGitopsModalPrompt,
    } = this.state;

    const deployKey = gitops?.deployKey;
    const addKeyUri = getAddKeyUri(gitops?.uri, gitops?.provider, ownerRepo);
    const gitopsIsConnected = gitops?.enabled && gitops?.isConnected;

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
              stepTitle={`GitOps settings for ${appTitle}`}
              appName={appTitle}
              hostname={gitops?.hostname}
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
              ctaLoadingText="Updating settings"
              ctaText="Update settings"
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
              <span className={`icon ${this.getProviderIconClassName(gitops?.provider)} u-marginLeft--10`} />
            </div>

            {gitopsIsConnected ?
              <div className="u-marginLeft--auto u-marginRight--auto">
                <GitOpsRepoDetails
                  stepTitle={`GitOps settings for ${appTitle}`}
                  appName={appTitle}
                  hostname={gitops?.hostname}
                  ownerRepo={ownerRepo}
                  branch={gitops?.branch}
                  path={gitops?.path}
                  format={gitops?.format}
                  action={gitops?.action}
                  selectedService={selectedService}
                  onFinishSetup={this.finishGitOpsSetup}
                  otherService=""
                  ctaLoadingText="Updating settings"
                  ctaText="Update settings"
                />
                <div className="disable-gitops-wrapper">
                  <p className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-marginBottom--10">Disable GitOps for {appTitle}</p>
                  <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-marginBottom--20">Disabling GitOps will only affect this application. </p>
                  <button className="btn secondary red u-marginRight--10" disabled={disablingGitOps} onClick={this.promptToDisableGitOps}>{disablingGitOps ? "Disabling GitOps" : "Disable GitOps"}</button>
                </div>
              </div>
              :
              <div className="flex-column flex1">
                <div className="GitopsSettings-noRepoAccess">
                  <div className="u-textAlign--center">
                    <span className="success-checkmark-icon icon u-marginBottom--10" />
                  </div>
                  <p className="title">GitOps has been enabled. You're almost ready to deploy</p>
                  <p className="sub">In order for application updates to be pushed to your GitOps deployment pipeline we need to be able to access to the repository. To&nbsp;do this, copy the key below and add it to your repository settings page. If you need further assistance, you can <a href="https://kots.io/kotsadm/gitops/single-app-workflows/" target="_blank" rel="noopener noreferrer" className="replicated-link">check out our documentation</a>.</p>
                  <p className="sub u-marginTop--10">If you have already added this key to your repository and are seeing this message, check to make sure that the key has "Write access" for the repository and click "Try again".</p>
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
                    <button className="btn secondary blue u-marginRight--10" disabled={testingConnection} onClick={this.handleTestConnection}>{testingConnection ? "Testing connection" : "Try again"}</button>
                    <button className="btn primary blue" onClick={this.goToTroubleshootPage}>Troubleshoot</button>
                  </div>
                  <button className="btn secondary dustyGray" onClick={this.updateGitOpsSettings}>Update GitOps Settings</button>
                </div>
              </div>
            }
          </div>
        }
        <Modal
          isOpen={showDisableGitopsModalPrompt}
          onRequestClose={() => { this.setState({ showDisableGitopsModalPrompt: false }) }}
          contentLabel="Disable GitOps"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body">
            <div className="u-marginTop--10 u-marginBottom--10">
              <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-marginBottom--10">Are you sure you want to disable GitOps?</p>
              <p className="u-fontSize--large u-color--dustyGray">You can re-enable GitOps for this application by clicking "GitOps" in the Nav bar</p>
            </div>
            <div className="u-marginTop--30">
              <button type="button" className="btn secondary u-marginRight--10" onClick={() => { this.setState({ showDisableGitopsModalPrompt: false }) }}>Cancel</button>
              <button type="button" className="btn primary red" onClick={this.disableGitOps}>Disable GitOps</button>
            </div>
          </div>
        </Modal>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
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
