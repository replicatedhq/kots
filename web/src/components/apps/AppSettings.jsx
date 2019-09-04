import React, { Component } from "react";
import Helmet from "react-helmet";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import "../../scss/components/watches/WatchDetailPage.scss";
import { getAppRegistryDetails } from "@src/queries/AppsQueries";
import { updateRegistryDetails } from "@src/mutations/AppsMutations";
import Loader from "../shared/Loader";
import { Utilities } from "../../utilities/utilities";

class AppSettings extends Component {
  
  constructor(props) {
    super(props);

    this.state = {
      hostname: "",
      username: "",
      password: "",
      namespace: props.app.slug,
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
      // TODO: refetch registry settings here
    } catch (error) {
      console.log(error);
    }
  }

  testRegistryConnection = () => {
    console.log("implement test");
  }

  componentDidUpdate(lastProps) {
    const { getKotsAppRegistryQuery } = this.props;
    if (getKotsAppRegistryQuery.getAppRegistryDetails && getKotsAppRegistryQuery.getAppRegistryDetails !== lastProps.getKotsAppRegistryQuery.getAppRegistryDetails) {
      this.setState({
        hostname: getKotsAppRegistryQuery.getAppRegistryDetails.registryHostname,
        pingedEndpoint: getKotsAppRegistryQuery.getAppRegistryDetails.registryHostname,
        username: getKotsAppRegistryQuery.getAppRegistryDetails.registryUsername,
        password: getKotsAppRegistryQuery.getAppRegistryDetails.registryPassword,
        namespace: getKotsAppRegistryQuery.getAppRegistryDetails.namespace,
        lastSync: getKotsAppRegistryQuery.getAppRegistryDetails.lastSyncedAt
      })
    }
  }

  render() {
    const { app, getKotsAppRegistryQuery } = this.props;
    const { hostname, password, username, namespace, lastSync } = this.state;
    if (getKotsAppRegistryQuery.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    return (
      <div className="flex justifyContent--center">
        <Helmet>
          <title>{`${app.name} Airgap settings`}</title>
        </Helmet>
        <div className="AirgapSettings--wrapper u-textAlign--left u-paddingRight--20 u-paddingLeft--20">
          <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-marginTop--30 u-marginBottom--20 u-paddingBottom--5 u-lineHeight--normal">Registry settings</p>
          <form>
            <div className="flex u-marginBottom--20">
              <div className="flex1">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Hostname</p>
                <p className="u-lineHeight--normal u-fontSize--small u-color--dustyGray u-fontWeight--medium u-marginBottom--10">Ensure this domain supports the Docker V2 protocol.</p>
                <input type="text" className="Input" placeholder="artifactory.some-big-bank.com" value={hostname} autoComplete="" onChange={(e) => { this.setState({ hostname: e.target.value }) }}/>
              </div>
            </div>
            <div className="flex u-marginBottom--20">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Username</p>
                <input type="text" className="Input" placeholder="username" value={username} autoComplete="username" onChange={(e) => { this.setState({ username: e.target.value }) }}/>
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Password</p>
                <input type="password" className="Input" placeholder="password" autoComplete="current-password" value={password} onChange={(e) => { this.setState({ password: e.target.value }) }}/>
              </div>
            </div>
            <div className="test-connection-box u-marginBottom--20">
              <div className="flex">
                <div>
                  <button type="button" className="btn secondary" onClick={this.testRegistryConnection}>Test connection</button>
                </div>
                <div className="flex-column justifyContent--center">
                  <p className="u-marginLeft--10 u-fontSize--small u-fontWeight--medium u-color--tundora"><span className={`icon checkmark-icon u-marginRight--5 u-verticalAlign--neg3`} />Connected to {this.state.pingedEndpoint}</p>
                </div>
              </div>
              <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--10">Last connection test on {Utilities.dateFormat(lastSync, "MMMM D, YYYY")}</p>
            </div>
            <div className="flex u-marginBottom--5">
              <div className="flex1">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Namespace</p>
                <p className="u-lineHeight--normal u-fontSize--small u-color--dustyGray u-fontWeight--medium u-marginBottom--10">Changing the namespace will rewrite all of your airgap images and push them to your registry.</p>
                <input type="text" className="Input" placeholder="namespace" value={namespace} autoComplete="" onChange={(e) => { this.setState({ namespace: e.target.value }) }}/>
              </div>
            </div>
          </form>
          <div className="u-marginTop--20">
            <button className="btn primary" onClick={this.onSubmit}>Save changes</button>
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(getAppRegistryDetails, {
    name: "getKotsAppRegistryQuery",
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
)(AppSettings);
