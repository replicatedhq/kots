import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import Loader from "../shared/Loader";
import { getAppRegistryDetails, getImageRewriteStatus } from "../../queries/AppsQueries";
import { validateRegistryInfo } from "@src/queries/UserQueries";
import { updateRegistryDetails } from "@src/mutations/AppsMutations";
import { Repeater } from "../../utilities/repeater";

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

      lastSync: null,
      testInProgress: false,
      testFailed: false,
      testMessage: "",

      updateChecker: new Repeater(),
      rewriteStatus: "",
      rewriteMessage: "",
    }
  }

  componentWillUnmount() {
    this.state.updateChecker.stop();
  }

  componentDidMount = () => {
    this.triggerStatusUpdates();
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

      this.state.updateChecker.start(this.updateStatus, 1000);
    } catch (error) {
      console.log(error);
    }
  }

  testRegistryConnection = () => {
    this.setState({
      testInProgress: true,
      testMessage: "",
    });

    const { slug } = this.props.app;
    this.props.client.query({
      query: validateRegistryInfo,
      variables: {
        slug: slug,
        endpoint: this.state.hostname,
        username: this.state.username,
        password: this.state.password,
        org: this.state.namespace,
      }
    }).then(result => {
      if (result.data.validateRegistryInfo) {
        this.setState({
          testInProgress: false,
          testMessage: result.data.validateRegistryInfo,
          testFailed: true,
        });
      } else {
        this.setState({
          testInProgress: false,
          testMessage: "Success!",
          testFailed: false,
          lastSync: new Date(),
        });
      }
    }).catch(err => {
      this.setState({
        testInProgress: false,
        testMessage: String(err),
        testFailed: true,
      });
    });
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

  triggerStatusUpdates = () => {
    this.props.client.query({
      query: getImageRewriteStatus,
      variables: {},
      fetchPolicy: "no-cache",
    }).then((res) => {
      this.setState({
        rewriteStatus: res.data.getImageRewriteStatus.status,
        rewriteMessage: res.data.getImageRewriteStatus.currentMessage,
      });
      if (res.data.getImageRewriteStatus.status !== "running") {
        return;
      }
      this.state.updateChecker.start(this.updateStatus, 1000);
    }).catch((err) => {
      console.log("failed to get rewrite status", err);
    });
  }

  updateStatus = () => {
    return new Promise((resolve, reject) => {
      this.props.client.query({
        query: getImageRewriteStatus,
        fetchPolicy: "no-cache",
      }).then((res) => {

        this.setState({
          rewriteStatus: res.data.getImageRewriteStatus.status,
          rewriteMessage: res.data.getImageRewriteStatus.currentMessage,
        });

        if (res.data.getImageRewriteStatus.status !== "running") {
          this.state.updateChecker.stop();
        }

        resolve();

      }).catch((err) => {
        console.log("failed to get rewrite status", err);
        reject();
      })
    });
  }

  render() {
    const { getKotsAppRegistryQuery, hideTestConnection, hideCta, namespaceDescription, showHostnameAsRequired } = this.props;
    const { hostname, password, username, namespace, lastSync, testInProgress, testFailed, testMessage } = this.state;
    const { rewriteMessage, rewriteStatus } = this.state;

    if (getKotsAppRegistryQuery?.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const namespaceSubtext = namespaceDescription || "Changing the namespace will rewrite all of your airgap images and push them to your registry."

    let testStatusText = "";
    if (testInProgress) {
      testStatusText = "Testing...";
    } else if (lastSync) {
      testStatusText = testMessage;
    } else {
      // TODO: this will always be displayed when page is refreshed
      testStatusText = "Connection has not been tested";
    }

    const disableSubmitButton = rewriteStatus === "running";
    const showProgress = rewriteStatus === "running";
    const showStatusError = rewriteStatus === "failed";

    return (
      <div>
        <form>
          <div className="flex u-marginBottom--20">
            <div className="flex1">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Hostname {showHostnameAsRequired && <span className="u-color--chestnut">(Required)</span>}</p>
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
              {testFailed ?
                <p className="u-fontSize--small u-fontWeight--medium u-color--chestnut u-marginTop--10">{testStatusText}</p>
              :
                <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--10">{testStatusText}</p>
              }
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
          <div>          
            { showProgress ?
              <div className="u-marginTop--20">
                <Loader size="30" />
                <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--10">{rewriteMessage}</p>
              </div>
            :
              null
            }
            { showStatusError ?
              <p className="u-fontSize--small u-fontWeight--medium u-color--chestnut u-marginTop--10">{rewriteMessage}</p>
            :
              null
            }
            <div className="u-marginTop--20">
              <button className="btn primary" disabled={disableSubmitButton} onClick={this.onSubmit}>Save changes</button>
            </div>
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
