import * as React from "react";
import Select from "react-select";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { listApps } from "@src/queries/AppsQueries";
import { setAppGitOps } from "@src/mutations/AppsMutations";
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
]
class GitOpsDeploymentManager extends React.Component {
  state = {
    step: "setup",
    visitedSteps: [],
    ownerRepo: "",
    branch: "",
    path: "",
    hostname: "",
    services: SERVICES,
    selectedService: SERVICES[0],
    otherService: "",
    providerError: null,
    actionPath: "commit",
    containType: ""
  }

  completeSetup = async () => {
    const { listAppsQuery } = this.props;
    const {
      selectedService,
      ownerRepo,
      branch,
      path,
      hostname,
      actionPath,
      otherService,
      containType
    } = this.state;

    const firstApp = listAppsQuery.listApps && listAppsQuery.listApps.kotsApps.length > 0 ? listAppsQuery.listApps.kotsApps[0] : null;
    if (!firstApp) {
      console.log("no app");
      return;
    }

    const clusterId = firstApp.downstreams[0]?.cluster?.id;
    const isGitlab = selectedService?.value === "gitlab" || selectedService?.value === "gitlab_enterprise";
    const isBitbucket = selectedService?.value === "bitbucket" || selectedService?.value === "bitbucket_server";
    const serviceUri = isGitlab ? "gitlab.com" : isBitbucket ? "bitbucket.org" : "github.com";

    let gitOpsInput = new Object();
    gitOpsInput.provider = selectedService.value;
    gitOpsInput.uri = `https://${serviceUri}/${ownerRepo}`;
    gitOpsInput.owner = ownerRepo;
    gitOpsInput.branch = branch || "master";
    gitOpsInput.path = path;
    gitOpsInput.format = containType;
    gitOpsInput.action = actionPath;
    if (selectedService.value === "gitlab_enterprise" || selectedService.value === "github_enterprise") {
      gitOpsInput.hostname = hostname;
    }
    if (selectedService.value === "other") {
      gitOpsInput.otherServiceName = otherService;
    }

    try {
      await this.props.setAppGitOps(firstApp.id, clusterId, gitOpsInput);
      this.props.history.push(`/app/${firstApp.slug}/gitops`);
    } catch (error) {
      console.log(error);
    }
  }

  validStep = (step) => {
    const {
      selectedService,
      otherService,
      ownerRepo,
      hostname,
    } = this.state;

    this.setState({ providerError: null });
    if (step === "provider") {
      if (selectedService.value === "other" && !otherService.length) {
        this.setState({
          providerError: {
            field: "other"
          }
        });
        return false;
      }
      if (selectedService.value !== "other") {
        if (!ownerRepo.length) {
          this.setState({
            providerError: {
              field: "ownerRepo"
            }
          });
          return false;
        }
        if (selectedService.value === "github_enterprise" && !hostname.length) {
          this.setState({
            providerError: {
              field: "hostname"
            }
          });
          return false;
        }
      }
    }

    return true;
  }

  stepFrom = (from, to) => {
    if (this.validStep(from)) {
      const visited = this.state.visitedSteps;
      if (!visited.includes(from)) {
        visited.push(from);
      }
      this.setState({
        step: to,
        visitedSteps: visited
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

  onActionTypeChange = (e) => {
    if (e.target.classList.contains("js-preventDefault")) { return }
    this.setState({ actionPath: e.target.value });
  }

  onFileContainChange = (e) => {
    if (e.target.classList.contains("js-preventDefault")) { return }
    this.setState({ containType: e.target.value });
  }

  handleServiceChange = (selectedService) => {
    this.setState({ selectedService });
  }

  getActiveStep = (step) => {
    const {
      ownerRepo,
      branch,
      path,
      hostname,
      services,
      selectedService,
      otherService,
      providerError
    } = this.state;

    const isGitlab = selectedService?.value === "gitlab" || selectedService?.value === "gitlab_enterprise";
    const isBitbucket = selectedService?.value === "bitbucket" || selectedService?.value === "bitbucket_server";
    const serviceSite = isGitlab ? "gitlab.com" : isBitbucket ? "bitbucket.org" : "github.com";

    switch (step.step) {
      case "setup":
        return (
        <div key={`${step.step}-active`} className="GitOpsDeploy--step">
          <div className="icon gitopsLogo u-marginBottom--20" />
          <p className="step-title">Deploy using a GitOps workflow</p>
          <p className="step-sub">You are managing application updates using this Admin Console. If you’d prefer, updates can be pushed to a git repository, allowing for a more customizable process to be run on every update.</p>
          <p className="step-sub">Switching to a GitOps workflow can be completed without any downtime of the application.</p>
          <div>
            <button className="btn primary blue" type="button" onClick={() => this.stepFrom("setup", "provider")}>Set up GitOps</button>
          </div>
        </div>
      );
      case "provider":
        return (
          <div key={`${step.step}-active`} className="GitOpsDeploy--step">
            <p className="step-title">{step.title}</p>
            <p className="step-sub">Before the Admin Console can push changes to your Git repository, some information about your Git configuration is required.</p>
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
                {selectedService?.value === "other" ?
                  <div className="flex flex1 flex-column u-marginLeft--10">
                    <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">What GitOps service do you use?</p>
                    <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">Not all services are supported.</p>
                    <input type="text" className={`Input ${providerError?.field === "other" && "has-error"}`} placeholder="What service would you like to use" value={otherService} onChange={(e) => this.setState({ otherService: e.target.value })} />
                    {providerError?.field === "other" && <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">A GitOps service name must be provided</p>}
                  </div>
                :
                  <div className="flex flex1 flex-column u-marginLeft--10">
                    <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Owner &amp; Repository</p>
                    <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">Which repository will the commit be made?</p>
                    <input type="text" className={`Input ${providerError?.field === "ownerRepo" && "has-error"}`} placeholder="owner/repository" value={ownerRepo} onChange={(e) => this.setState({ ownerRepo: e.target.value })} />
                    {providerError?.field === "ownerRepo" && <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">A owner and repository must be provided</p>}
                  </div>
                }
              </div>
              {selectedService?.value === "github_enterprise" || selectedService?.value === "gitlab_enterprise" ?
                <div className="flex flex1 u-marginBottom--20">
                  <div className="flex flex1 flex-column u-marginRight--10">
                    <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Hostname</p>
                    <input type="text" className={`Input ${providerError?.field === "hostname" && "has-error"}`} placeholder="hostname" value={hostname} onChange={(e) => this.setState({ hostname: e.target.value })} />
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
            <div>
              <button className="btn primary blue" type="button" onClick={() => this.stepFrom("provider", "action")}>Continue to deployment action</button>
            </div>
          </div>
        );
      case "action":
        return (
          <div key={`${step.step}-active`} className="GitOpsDeploy--step">
            <div className="StepContent--widthRestrict">
              <p className="step-title">{step.title}</p>
              <p className="step-sub">When an update is available{this.props.appName ? ` to ${this.props.appName} ` : ""}, how should the updates YAML be delivered to&nbsp;{selectedService.label === "Other" ? otherService : serviceSite}?</p>
              <div className="flex flex1 u-marginTop--normal gitops-checkboxes justifyContent--center u-marginBottom--30">
                <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left u-marginRight--10">
                  <div className={`BoxedCheckbox flex-auto flex ${this.state.actionPath === "commit" ? "is-active" : ""}`}>
                    <input
                      type="radio"
                      className="u-cursor--pointer hidden-input"
                      id="commitOption"
                      checked={this.state.actionPath === "commit"}
                      defaultValue="commit"
                      onChange={(e) => { this.onActionTypeChange(e) }}
                    />
                    <label htmlFor="commitOption" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                      <div className="flex-auto">
                        <span className="icon clickable commitOptionIcon u-marginRight--10" />
                      </div>
                      <div className="flex1">
                        <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Create a commit</p>
                        <p className="u-color--dustyGray u-fontSize--small u-fontWeight--medium u-marginTop--5">Automatic commits to repo</p>
                      </div>
                    </label>
                  </div>
                </div>
                <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left u-marginLeft--10">
                  <div className={`BoxedCheckbox flex1 flex ${this.state.actionPath === "pullRequest" ? "is-active" : ""} is-disabled`}>
                    <input
                      type="radio"
                      className="u-cursor--pointer hidden-input"
                      id="pullRequestOption"
                      checked={this.state.actionPath === "pullRequest"}
                      defaultValue="pullRequest"
                      onChange={(e) => { this.onActionTypeChange(e) }}
                      disabled={true}
                    />
                    <label htmlFor="pullRequestOption" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                      <div className="flex-auto">
                      <span className="icon pullRequestOptionIcon u-marginRight--10" />
                      </div>
                      <div className="flex1">
                        <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Open a {isBitbucket ? "Merge" : "Pull"} Request</p>
                        <p className="u-color--dustyGray u-fontSize--small u-fontWeight--medium u-marginTop--5">Coming soon!</p>
                      </div>
                    </label>
                  </div>
                </div>
              </div>

              <div className="u-marginBottom--10 u-textAlign--left">
                <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">What content will it contain?</p>
                <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">Your commit can include a single rendered yaml file or it’s full output.</p>
              </div>

              <div className="flex flex1 u-marginTop--normal gitops-checkboxes justifyContent--center u-marginBottom--30">
                <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left u-marginRight--10">
                  <div className={`BoxedCheckbox flex1 flex ${this.state.containType === "single" ? "is-active" : ""}`}>
                    <input
                      type="radio"
                      className="u-cursor--pointer hidden-input"
                      id="singleOption"
                      checked={this.state.containType === "commit"}
                      defaultValue="single"
                      onChange={(e) => { this.onFileContainChange(e) }}
                    />
                    <label htmlFor="singleOption" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                      <div className="flex-auto">
                        <span className="icon clickable singleOptionIcon u-marginRight--10" />
                      </div>
                      <div className="flex1">
                        <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Rendered YAML</p>
                        <p className="u-color--dustyGray u-fontSize--small u-fontWeight--medium u-marginTop--5">Apply using <span className="inline-code no-bg">kubectl apply -f</span></p>
                      </div>
                    </label>
                  </div>
                </div>
                <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left u-marginLeft--10">
                  <div className={`BoxedCheckbox flex1 flex ${this.state.containType === "fullFiles" ? "is-active" : ""} is-disabled`}>
                    <input
                      type="radio"
                      className="u-cursor--pointer hidden-input"
                      id="fullFilesOption"
                      checked={this.state.containType === "fullFiles"}
                      defaultValue="fullFiles"
                      onChange={(e) => { this.onFileContainChange(e) }}
                      disabled={true}
                    />
                    <label htmlFor="fullFilesOption" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                      <div className="flex-auto">
                      <span className="icon clickable fullFilesOptionIcon u-marginRight--10" />
                      </div>
                      <div className="flex1">
                        <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Full Kustomizable Output</p>
                        <p className="u-color--dustyGray u-fontSize--small u-fontWeight--medium u-marginTop--5">Coming soon</p>
                      </div>
                    </label>
                  </div>
                </div>
              </div>

              <div>
                <button className="btn primary blue" type="button" onClick={this.completeSetup}>Complete GitOps setup</button>
              </div>
            </div>
          </div>
        );
      default:
        return <div key={`default-active`} className="GitOpsDeploy--step">default</div>;
    }
  }

  render() {
    const { visitedSteps } = this.state;
    return (
      <div className="GitOpsDeploymentManager--wrapper flex-column flex1">
        {STEPS.map(s => {
          const activeStep = s.step === this.state.step;
          const hasBeenVisited = visitedSteps.includes(s.step);
          if (activeStep) {
            return this.getActiveStep(s);
          } else {
            return (
              <div key={`${s.step}-inactive`} className={`GitOpsDeploy--step inactive ${hasBeenVisited && "u-cursor--pointer"}`} onClick={hasBeenVisited ? () => { this.stepFrom("", s.step) }: undefined }>
                <p className={`u-fontSize--large u-color--${hasBeenVisited ? "tundora" : "dustyGray"} u-fontWeight--medium u-lineHeight--normal`}>
                  {hasBeenVisited && <span className="u-marginRight--5 icon checkmark-icon u-verticalAlign--neg2" />}{s.title}
                </p>
              </div>
            )
          }
        })}
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(listApps, {
    name: "listAppsQuery",
  }),
  graphql(setAppGitOps, {
    props: ({ mutate }) => ({
      setAppGitOps: (appId, clusterId, gitOpsInput) => mutate({ variables: { appId, clusterId, gitOpsInput } })
    })
  }),
)(GitOpsDeploymentManager);
