import React, { Component } from "react";
import Helmet from "react-helmet";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import omit from "lodash/omit";

import { createShipOpsCluster } from "../../mutations/ClusterMutations";
import ShipClusterSuccess from "./ShipClusterSuccess";
import ConfigureGitHubCluster from "../shared/ConfigureGitHubCluster";
import "../../scss/components/clusters/CreateCluster.scss";

export class CreateCluster extends Component {
  constructor() {
    super();
    this.clusterNameInputRef = React.createRef();
  }
  state = {
    title: "",
    titleError: false,
    opsType: {
      value: "ship",
      label: "ShipOps",
    },
    saving: false,
    fetchError: false,
    createSuccess: false,
    clusterToken: "",
    createClusterErr: undefined
  };

  onClickCreate = () => {
    this.setState({ createClusterErr: undefined, saving: true });

    if (!this.state.title.length) {
      return this.setState({
        createClusterErr: {
          message: "In order to create a cluster, you must give it a name."
        },
        saving: false
      });
    }

    if (!this.state.createSuccess && this.state.opsType.value === "git") {
      return this.setState({ createSuccess: true, saving: false });
    }

    this.props.client.mutate({
      mutation: createShipOpsCluster,
      variables: {
        title: this.state.title,
      },
    })
      .then((res) => {
        this.setState({
          saving: false,
          createSuccess: true,
          clusterToken: res.data.createShipOpsCluster.shipOpsRef.token,
          clusterId: res.data.createShipOpsCluster.id
        })
      })
      .catch(() => {
        this.setState({
          saving: false,
          fetchError: true
        })
      });
  }

  onTitleChange = (title) => {
    this.setState({
      title,
    });
  }

  onOpsTypeChange = (e, selectedOption) => {
    if (e.target.classList.contains("js-preventDefault")) { return }
    this.setState({ opsType: omit(selectedOption, ["uiInfo"]) });
  }

  createOnEnterKey = (e) => {
    const { createSuccess, saving } = this.state;
    const enterKey = e.keyCode === 13;
    if (createSuccess || saving) { return; }
    if (enterKey) {
      e.preventDefault();
      e.stopPropagation();
      this.onClickCreate();
    }
  }

  handleCreationSuccessClick = () => {
    const redirect = localStorage.getItem("clusterRedirect");
    if (redirect) {
      localStorage.removeItem("clusterRedirect");
      if (redirect.includes("automaticDeploy")) {
        const watchSlug = redirect.slice(16);
        localStorage.setItem("clusterRedirect", `/watch/${watchSlug}/downstreams?add=1`);
      } else {
        this.props.history.push(redirect);
      }
    } else {
      this.props.history.push("/gitops");
    }
  }

  componentWillUnmount() {
    window.removeEventListener("keydown", this.createOnEnterKey);
  }

  async componentDidMount() {
    window.addEventListener("keydown", this.createOnEnterKey);
    const { search } = this.props.location;
    const queryParams = new URLSearchParams(search);
    const configStep = queryParams.get("configure");
    const clusterName = queryParams.get("name");

    if (configStep) {
      this.setState({
        ...this.state,
        title: clusterName || "",
        opsType: {
          value: "git",
          label: "GitOps"
        },
        createSuccess: true
      });
    }

    setTimeout(() => {
      this.clusterNameInputRef.current.focus();
    }, 100);
  }

  render() {
    const { saving, createSuccess, opsType } = this.state;
    const options = [
      {
        value: "git",
        label: "GitOps",
        uiInfo: {
          title: "Deploy with GitHub",
          description: [
            <p key="s-text" className="u-marginTop--10 u-color--dustyGray u-fontSize--small u-fontWeight--medium u-lineHeight--normal">Using Weave Flux or ArgoCD to deploy your Kubernetes applications?</p>,
            <div key="s-text-link" className="flex alignItems--center">
              <a href="" target="_blank" rel="noopener noreferrer" className="js-preventDefault unforkLink u-marginTop--10 u-fontSize--small u-fontWeight--medium u-color--chateauGreen">Learn more about GitOps <span className="arrow icon clickable u-arrow"></span></a>
            </div>
          ]
        }
      },
      {
        value: "ship",
        label: "ShipOps",
        uiInfo: {
          title: "Deploy with Ship",
          description: [
            <p key="s-text" className="u-marginTop--10 u-color--dustyGray u-fontSize--small u-fontWeight--medium u-lineHeight--normal">Don't have Flux or ArgoCD but want the same functionality? Try ShipOps.</p>,
            <div key="s-text-link" className="flex alignItems--center">
              <a href="" target="_blank" rel="noopener noreferrer" className="js-preventDefault unforkLink u-marginTop--10 u-fontSize--small u-fontWeight--medium u-color--chateauGreen">Learn more about Ship deployments <span className="arrow icon clickable u-arrow"></span></a>
            </div>
          ]
        }
      }
    ];

    return (
      <div className="Login-wrapper container flex-column flex1 u-overflow--auto">
        <Helmet>
          <title>Connect New Cluster</title>
        </Helmet>
        <div className="Form flex-column flex1 alignItems--center justifyContent--center">
          {createSuccess ?
            opsType.value === "ship" ?
              <ShipClusterSuccess
                clusterId={this.state.clusterId}
                token={this.state.clusterToken}
                handleCreationSuccessClick={this.handleCreationSuccessClick}
              />
              :
              <div className="CreateCluster--wrapper flex-auto">
                <div className="flex1 flex-column">
                  <ConfigureGitHubCluster
                    clusterTitle={this.state.title}
                    hideRootPath={true}
                    integrationToManage={null}
                    handleCreationSuccessClick={this.handleCreationSuccessClick}
                  />
                </div>
              </div>
            :
            <div className="CreateCluster--wrapper flex-auto">
              <div className="flex1 flex-column">
                <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">What's the name of your cluster?</p>
                <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Maybe this is Production, or Europe or something descriptive.</p>
                <div className="flex flex1">
                  <input ref={this.clusterNameInputRef} value={this.state.title} onChange={(e) => { this.onTitleChange(e.target.value) }} type="text" className="Input jumbo flex1" placeholder="Production" />
                </div>
                {this.state.createClusterErr && <p className="u-fontSize--small u-fontWeight--medium u-color--chestnut u-marginTop--10">{this.state.createClusterErr.message}</p>}
              </div>

              <div className="flex-column flex1 u-marginTop--30">
                <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">How do you want to deploy applications to your cluster?</p>
                <div className="flex flex1 u-marginTop--normal cluster-deployment-type">
                  {options.map((option, i) => {
                    const isChecked = this.state.opsType.value === option.value;
                    return (
                      <div key={option.value} className={`BoxedCheckbox-wrapper flex1 ${i === 0 ? "u-marginRight--5" : "u-marginLeft--5"}`}>
                        <div className={`BoxedCheckbox flex1 flex ${isChecked ? "is-active" : ""}`}>
                          <input
                            type="radio"
                            className="u-cursor--pointer"
                            id={option.label}
                            checked={isChecked}
                            defaultValue={option.value}
                            onChange={(e) => { this.onOpsTypeChange(e, option) }}
                          />
                          <label htmlFor={option.label} className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                            <div className="flex-column u-paddingLeft--20">
                              <div className="u-fontWeight--medium u-color--tuna u-fontSize--normal u-marginBottom--small u-lineHeight--normal">
                                <span className={`icon clusterType ${option.value} u-marginRight--5`}></span>
                                <span>{option.uiInfo.title}</span>
                              </div>
                              {option.uiInfo.description}
                              <span className="u-fontSize--small u-color--dustyGray u-fontWeight--normal u-lineHeight--normal"></span>
                            </div>
                          </label>
                        </div>
                      </div>
                    )})
                  }
                </div>
              </div>
              <div className="flex-auto u-marginTop--30 u-textAlign--center">
                <button className="btn primary large" disabled={saving} onClick={this.onClickCreate}>{saving ? "Creating cluster" : "Create deployment cluster"}</button>
              </div>
            </div>
          }
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
)(CreateCluster);
