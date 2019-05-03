import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { trackScmLead } from "../mutations/AuthMutations";
import "../scss/components/Login.scss";
import Select from "react-select";

class TrackSCMLeads extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      deploymentType: "",
      scmType: "",
      otherProvider: "",
      email: "",
      notifyLoading: false,
      hasBeenNotified: false,
      hasError: false
    }
  }

  success = () => {
    this.setState({
      hasBeenNotified: true,
      notifyLoading: false
    });
    setTimeout(() => {
      this.setState({
        hasBeenNotified: false,
        hasError: false,
        deploymentType: "",
        scmType: "",
        otherProvider: "",
        email: ""
      });
    }, 8000);
  }

  ensureValues = () => {
    this.setState({ hasError: false });
    const { deploymentType, email, scmType} = this.state;
    if (deploymentType === "" || email === "" || scmType === "") {
      this.setState({ hasError: true });
      return false;
    }
    return true;
  }

  handleNotify = async () => {
    const { deploymentType, email, scmType, otherProvider } = this.state;
    const provider = deploymentType === "1" ? "on-prem" : "saas";
    let otherVal = scmType.value;
    if (scmType.value === "other") {
      otherVal = otherProvider || "other";
    }
    if (this.ensureValues()) {
      this.setState({ notifyLoading: true });
      await this.props.trackScmLead(provider, email, otherVal)
        .then(() => this.success() )
        .catch(() => this.setState({ notifyLoading: false }) );
    }
  }

  handleScmChange = (selectedOption) => {
    this.setState({ scmType: selectedOption });
  }

  render() {
    const {
      notifyLoading,
      deploymentType,
      email,
      scmType,
      otherProvider,
      hasBeenNotified
    } = this.state;

    const options = [
      { value: "gh-enterprise", label: "GitHub Enterprise" },
      { value: "gl-hosted", label: "GitLab Hosted" },
      { value: "gl-on-prem", label: "GitLab On-prem" },
      { value: "bb-hosted", label: "Bitbucket Hosted" },
      { value: "bb-on-prem", label: "Bitbucket On-prem" },
      { value: "other", label: "Other" }
    ];
    
    return (
      <div className="flex1 flex-column scm-form-wrapper">
        {hasBeenNotified ?
          <div className="success-wrapper flex1 flex-column justifyContent--center alignItems--center">
            <div className="icon success-checkmark-icon u-marginBottom--10 u-marginTop--10"></div>
            <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Thanks! We'll notify you when we support {scmType.value === "other" ? otherProvider ? otherProvider : "more SCM providers" : scmType.label}</p>
          </div>
          :
          <div className="flex1 flex-column justifyContent--center">
            <p className="u-lineHeight--normal u-fontSize--large u-color--doveGray u-fontWeight--medium u-marginBottom--30">Interested in accessing Replicated Ship with another SCM provider or deploying a private instance on-prem? Sign up to get notified.</p>
            <div className="u-flexTabletReflow u-marginBottom--20">
              <div className="component-wrapper flex1">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Deployment type</p>
                <div className="flex">
                  <div className="flex flex-auto u-marginRight--30">
                    <input
                      type="radio"
                      name="deploymentType"
                      id="onPrem"
                      checked={deploymentType === "1"}
                      value={1}
                      onChange={(e) => { this.setState({ deploymentType: e.target.value }) }}
                    />
                    <label htmlFor="onPrem" className="flex1 u-width--full u-position--relative u-marginLeft--5 u-cursor--pointer">
                      <div className="flex-auto flex-column u-width--full">
                        <span className="u-fontWeight--medium u-color--tuna u-fontSize--normal u-lineHeight--normal">On-prem</span>
                      </div>
                    </label>
                  </div>
                  <div className="flex flex-auto">
                    <input
                      type="radio"
                      name="deploymentType"
                      id="saas"
                      checked={deploymentType === "0"}
                      value={0}
                      onChange={(e) => { this.setState({ deploymentType: e.target.value }) }}
                    />
                    <label htmlFor="saas" className="flex1 u-width--full u-position--relative u-marginLeft--5 u-cursor--pointer">
                      <div className="flex-auto flex-column u-width--full">
                        <span className="u-fontWeight--medium u-color--tuna u-fontSize--normal u-lineHeight--normal">SaaS</span>
                      </div>
                    </label>
                  </div>
                </div>
              </div>
              <div className="component-wrapper flex1">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Email address</p>
                <input type="text" className="Input" placeholder="you@example.com" value={email} onChange={(e) => { this.setState({ email: e.target.value }) }}/>
              </div>
            </div>
            <div className="u-flexTabletReflow">
              <div className="component-wrapper flex1">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">SCM provider</p>
                <Select
                  className="replicated-select-container"
                  classNamePrefix="replicated-select"
                  options={options}
                  getOptionLabel={(scmType) => scmType.label}
                  placeholder="Select an SCM provider"
                  value={scmType}
                  onChange={this.handleScmChange}
                  isOptionSelected={(option) => {option.value === scmType.value}}
                />
              </div>
              <div className="component-wrapper flex1">
                {scmType.value === "other" &&
              <div>
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Which SCM Provider do you use?</p>
                <input type="text" className="Input" value={otherProvider} onChange={(e) => { this.setState({ otherProvider: e.target.value }) }}/>
              </div>
                }
              </div>
            </div>
            <div className="u-marginTop--20 flex">
              <button onClick={this.handleNotify} className="btn primary" disabled={notifyLoading}>Get notified</button>
              {this.state.hasError &&
              <div className="flex-column justifyContent--center">
                <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-marginLeft--10 u-lineHeight--normal">Please fill out all of the fields</p>
              </div>
              }
            </div>
          </div>
        }
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(trackScmLead, {
    props: ({ mutate }) => ({
      trackScmLead: (deploymentPreference, emailAddress, scmProvider) => mutate({ variables: { deploymentPreference, emailAddress, scmProvider }})
    })
  })
)(TrackSCMLeads);
