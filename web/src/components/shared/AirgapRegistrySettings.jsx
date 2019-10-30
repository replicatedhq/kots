import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import Loader from "../shared/Loader";
import { Utilities } from "../../utilities/utilities";
import { getAppRegistryDetails } from "@src/queries/AppsQueries";
import { updateRegistryDetails } from "@src/mutations/AppsMutations";

import "../../scss/components/watches/WatchDetailPage.scss";

class AirgapRegistrySettings extends Component {

  constructor(props) {
    super(props);

    const {
      hostname =  "",
      username = "",
      password = "",
      namespace = props.app ? props.app.slug : ""
    } = props?.registryDetails || {};

    this.state = {
      hostname,
      username,
      password,
      namespace,
      lastSync: null
    }
  }

  onSubmit = async () => {
    const {
      hostname,
      username,
      password,
      namespace,
    } = this.state;
    const { slug } = this.props.match.params;
    const appSlug = slug;
    try {
      await this.props.updateRegistryDetails({ appSlug, hostname, username, password, namespace });
      await this.props.getKotsAppRegistryQuery.refetch();

    } catch (error) {
      console.log(error);
    }
  }

  testRegistryConnection = () => {
    // TODO: set last sync and connection uri
    console.log("implement test");
  }

  handleFormChange = (field, val) => {
    let nextState = {};
    nextState[field] = val;
    this.setState(nextState, () => {
      if (this.props.gatherDetails) {
        const { hostname, username, password, namespace } = this.state;
        this.props.gatherDetails({ hostname, username, password, namespace });
      }
    });
  }

  componentDidUpdate(lastProps) {
    const { getKotsAppRegistryQuery, app } = this.props;
    if (getKotsAppRegistryQuery?.getAppRegistryDetails && getKotsAppRegistryQuery?.getAppRegistryDetails !== lastProps.getKotsAppRegistryQuery?.getAppRegistryDetails) {
      this.setState({
        hostname: getKotsAppRegistryQuery.getAppRegistryDetails.registryHostname,
        username: getKotsAppRegistryQuery.getAppRegistryDetails.registryUsername,
        password: getKotsAppRegistryQuery.getAppRegistryDetails.registryPassword,
        namespace: getKotsAppRegistryQuery.getAppRegistryDetails.namespace || app.slug,
      })
    }
  }

  render() {
    const { getKotsAppRegistryQuery, hideTestConnection, hideCta, namespaceDescription } = this.props;
    const { hostname, password, username, namespace, lastSync } = this.state;
    if (getKotsAppRegistryQuery?.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const namespaceSubtext = namespaceDescription || "Changing the namespace will rewrite all of your airgap images and push them to your registry."

    return (
      <div>
        <form>
          <div className="flex u-marginBottom--20">
            <div className="flex1">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Hostname <span className="u-color--chestnut">(Required)</span></p>
              <p className="u-lineHeight--normal u-fontSize--small u-color--dustyGray u-fontWeight--medium u-marginBottom--10">Ensure this domain supports the Docker V2 protocol.</p>
              <input type="text" className="Input" placeholder="artifactory.some-big-bank.com" value={hostname || ""} autoComplete="" onChange={(e) => { this.handleFormChange("hostname", e.target.value) }}/>
            </div>
          </div>
          <div className="flex u-marginBottom--20">
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Username</p>
              <input type="text" className="Input" placeholder="username" value={username || ""} autoComplete="username" onChange={(e) => { this.handleFormChange("username", e.target.value) }}/>
            </div>
            <div className="flex1 u-paddingLeft--5">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Password</p>
              <input type="password" className="Input" placeholder="password" autoComplete="current-password" value={password || ""} onChange={(e) => { this.handleFormChange("password", e.target.value) }}/>
            </div>
          </div>
          {hideTestConnection ? null :
            <div className="test-connection-box u-marginBottom--20">
              <div className="flex">
                <div>
                  <button type="button" className="btn secondary" onClick={this.testRegistryConnection}>Test connection</button>
                </div>
                {this.state.pingedEndpoint &&
                  <div className="flex-column justifyContent--center">
                    <p className="u-marginLeft--10 u-fontSize--small u-fontWeight--medium u-color--tundora"><span className={`icon checkmark-icon u-marginRight--5 u-verticalAlign--neg3`} />Connected to {this.state.pingedEndpoint}</p>
                  </div>
                }
              </div>
              <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--10">{lastSync ? `Last connection test on ${Utilities.dateFormat(lastSync, "MMMM D, YYYY")}`: "Connection has not been tested"}</p>
            </div>
          }
          <div className="flex u-marginBottom--5">
            <div className="flex1">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Namespace</p>
              <p className="u-lineHeight--normal u-fontSize--small u-color--dustyGray u-fontWeight--medium u-marginBottom--10">{namespaceSubtext}</p>
              <input type="text" className="Input" placeholder="namespace" value={namespace || ""} autoComplete="" onChange={(e) => { this.handleFormChange("namespace", e.target.value) }}/>
            </div>
          </div>
        </form>
        {hideCta ? null :
          <div className="u-marginTop--20">
            <button className="btn primary" onClick={this.onSubmit}>Save changes</button>
          </div>
        }
      </div>
    )
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(getAppRegistryDetails, {
    name: "getKotsAppRegistryQuery",
    skip: props => {
      if (!props.app) {
        return true;
      }
      return false;
    },
    options: props => {
      const { slug } = props.match.params;
      return {
        fetchPolicy: "no-cache",
        variables: {
          slug: slug
        }
      }
    }
  }),
  graphql(updateRegistryDetails, {
    props: ({ mutate }) => ({
      updateRegistryDetails: (registryDetails) => mutate({ variables: { registryDetails } })
    })
  }),
)(AirgapRegistrySettings);
