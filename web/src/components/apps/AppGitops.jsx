import React, { Component } from "react";
import Helmet from "react-helmet";
import Select from "react-select";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import url from "url";

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

export default class AppGitops extends Component {
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
    this.setState({
      provider: selectedService.value,
    });
  }

  render() {
    const { app } = this.props;
    const appTitle = app.name;

    if (length(app.downstreams) === 0) {
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
    } = this.state;

    const selectedService = SERVICES.find((service) => {
      return service.value === provider;
    });

    const gitUri = gitops?.uri;
    const deployKey = gitops?.deployKey;

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
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-marginTop--40">
              <span className="u-marginRight--5 icon checkmark-icon u-verticalAlign--neg2" />GitOps is enabled for {appTitle}
            </p>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-marginTop--20 u-marginBottom--20">
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
                {selectedService?.value === "github_enterprise" &&
                  <div className="flex flex1 u-marginBottom--20">
                    <div className="flex flex1 flex-column u-marginRight--10">
                      <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Hostname</p>
                      <input type="text" className="Input" placeholder="hostname" value={hostname} onChange={(e) => this.setState({ hostname: e.target.value })} />
                      {providerError?.field === "hostname" && <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">A hostname must be provided</p>}
                    </div>
                    <div className="flex flex1 flex-column u-marginLeft--10" />
                  </div>
                }
                {selectedService?.value !== "other" &&
                  <div className="flex flex1">
                    <div className="flex flex1 flex-column u-marginRight--10">
                      <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Branch</p>
                      <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">If no branch is specified, master will be used.</p>
                      <input type="text" className={`Input`} placeholder="master" value={branch} onChange={(e) => this.setState({ branch: e.target.value })} />
                    </div>
                    <div className="flex flex1 flex-column u-marginLeft--10">
                      <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Path</p>
                      <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">Where in your repo should deployment file live?</p>
                      <input type="text" className={"Input"} placeholder="/my-path" value={path} onChange={(e) => this.setState({ path: e.target.value })} />
                    </div>
                  </div>
                }
              </div>
              <div className={`${gitops.isConnected ? "u-display--none" : "u-marginBottom--10"}`}>
                <button className="btn primary blue" type="button" onClick={() => this.stepFrom("provider", "action")}>Update</button>
              </div>
              <div className={`GitOpsSettingsConnected inactive u-cursor--pointer ${gitops.isConnected ? "" : "u-display--none"}`}>
                <p className="u-fontSize--large u-color--tundora u-fontWeight--medium u-lineHeight--normal">
                  <span className="u-marginRight--5 icon checkmark-icon u-verticalAlign--neg2" />Connected and working!
                </p>
              </div>
              <div className={`GitOpsSettingsNotConnected inactive u-cursor--pointer u-marginBottom--10 ${gitops.isConnected ? "u-display--none" : ""}`}>
                <p className="u-fontSize--large u-color--tundora u-fontWeight--medium u-lineHeight--normal">
                  <span className="u-marginRight--5 icon error-small u-verticalAlign--neg2" />Unable to connect to repo
                </p>
                <button>Try Again</button>
                <p className="u-fontSize--large u-color--tundora u-fontWeight--medium u-lineHeight--normal">
                  To complete the setup, please add the following <a href="#">deploy key</a> to your repo. To add a deploy
                  key, <a href={`${gitUri}/settings/keys/new`} target="_blank" rel="noopener noreferrer">click here</a> and use the following key (check the box for write access):
                </p>
                <CodeSnippet
                  canCopy={true}
                  onCopyText={<span className="u-color--chateauGreen">Deploy key has been copied to your clipboard</span>}>
                  {deployKey}
                </CodeSnippet>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
