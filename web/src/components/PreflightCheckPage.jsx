import React, { Component, Fragment } from "react";
import { withRouter } from "react-router-dom";
import { graphql, withApollo, compose } from "react-apollo";
import classNames from "classnames";

import {listPreflightResults } from "@src/queries/WatchQueries";
import CodeSnippet from "./shared/CodeSnippet";
import TabView, { Tab } from "./shared/TabView";

import "@src/scss/components/PreflightChecksPage.scss";

class PreflightChecksPage extends Component {
  constructor(props) {
    super(props);

    this.state = {
      showPreflightInstructions: false
    };
  }

  togglePreflightInstructions = () => {
    const { showPreflightInstructions } = this.state;
    this.setState({
      showPreflightInstructions: !showPreflightInstructions
    });
  }

  onTabChange = (/* name */) => {

  }

  componentDidUpdate (/* lastProps */) {
    const { listPreflightResultsQuery } = this.props;
    const { initialResultCount, hasInitialResults, showPreflightResults } = this.state;
    const hasResults = !listPreflightResultsQuery.loading;

    if (showPreflightResults) {
      return;
    }

    if (hasResults) {
      if (!hasInitialResults) {
        this.setState({
          hasInitialResults: true,
          initialResultCount: listPreflightResultsQuery.listPreflightResults.length
        });
      } else if (initialResultCount < listPreflightResultsQuery.listPreflightResults.length) {
        listPreflightResultsQuery.stopPolling();
        this.setState({
          showPreflightResults: listPreflightResultsQuery.listPreflightResults[0]
        });
      }
    }
  }

  createDownstreamCluster = () => {
    // const { history } = this.props;
    // const upstreamUrl = "ship://ship-cloud/${}"
    // history.replace(`/watch/create/init?upstream=${upstreamUrl}&cluster_id=${clusterId}&path=${githubPath}`);
  }

  render() {
    const { showPreflightInstructions, showPreflightResults } = this.state;

    return (
      <div className="flex-column flex1">
        <div className = "flex1 u-overflow--auto">
          <div className = "PreflightChecks--wrapper u-paddingTop--30 u-overflow--hidden">
            <div className = "u-minWidth--full u-minHeight--full">
              <p className = "u-fontSize--header u-color--tuna u-fontWeight--bold">
                Preflight checks
              </p>
              <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--10">
                Preflight checks are designed to be run against a target cluster before installing an application. Preflights are simply a different set of collectors + analyzers. These checks are optional but are recommended to ensure that the application you install will work properly.
              </p>
              {
                showPreflightResults
                  ? (
                    <div className="u-marginTop--30">
                      <p className="u-fontSize--jumbo u-color--tuna u-fontWeight--bold u-marginBottom--15">
                        Results from your preflight checks
                      </p>
                      {JSON.parse(showPreflightResults.result).results.map( (row, idx) => {
                        console.log(row);
                        let icon;

                        if (row.isWarn) {
                          icon = "exclamationMark--icon";
                        } else if (row.isFail) {
                          icon = "error-small";
                        } else {
                          icon = "checkmark-icon";
                        }
                        return (
                          <div key={idx} className="flex justifyContent--space-between preflight-check-row u-paddingTop--10 u-paddingBottom--10">
                            <div className={classNames("flex-auto icon", icon, "u-marginRight--10")}/>
                            <div className="flex1">
                              <p className="u-color--tuna u-fontSize--larger u-fontWeight--bold">{row.title}</p>
                              <p className="u-marginTop--5 u-fontSize--normal u-fontWeight--medium">{row.message}</p>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )
                  : (
                    <Fragment>
                      <p className = "u-fontSize--jumbo u-color--tuna u-fontWeight--bold u-marginTop--30">
                        Run this command from your workstation
                      </p>
                      <p className="u-marginTop--10 u-marginBottom-10">
                        You will be able to see the results in your terminal window as well as in this UI.
                      </p>
                      <CodeSnippet className="u-marginTop--10" language="bash" canCopy={true}>
                        {`kubectl preflight ${window.env.REST_ENDPOINT}/v1${location.pathname}`}
                      </CodeSnippet>
                      <div className="section-border flex justifyContent--center u-position--relative u-marginTop--20">
                        <p
                          className="preflight-button flex-auto u-fontSize--small u-color--astral u-fontWeight--medium u-cursor--pointer"
                          onClick={this.togglePreflightInstructions}
                        >
                          {showPreflightInstructions
                            ? "I already have the preflight tool"
                            : "I need to install the preflight tool"
                          }
                        </p>
                      </div>
                      {showPreflightInstructions && (
                        <Fragment>
                          <p className="u-fontSize--jumbo u-color--tuna u-fontWeight--bold u-marginTop--20">
                            Install the preflight tool
                          </p>
                          <CodeSnippet
                            className="u-marginTop--10"
                            preText={(
                              <span className="u-fontSize--small u-fontWeight--bold">
                                The best way to install the preflight tool is using Kubernetes package manager
                                <a target="_blank" rel="noopener noreferrer" href="https://github.com/kubernetes-sigs/krew/"> krew</a>.
                              </span>
                            )}
                            language="bash"
                            canCopy={true}
                          >
                            kubectl krew install preflight
                          </CodeSnippet>
                          <TabView
                            className="u-marginTop--10 hidden"
                            onTabChange={this.onTabChange}
                          >
                            <Tab name="mac" displayText="MacOS">
                              <CodeSnippet className="u-marginTop--10" language="bash" canCopy={false}>
                                {"brew tap replicatedhq/troubleshoot\nbrew install replicatedhq/preflight"}
                              </CodeSnippet>
                            </Tab>
                            <Tab name="win" displayText="Windows">
                              <CodeSnippet className="u-marginTop--10" language="bash" canCopy={false}>
                                choco install replicatedhq/preflight
                              </CodeSnippet>
                            </Tab>
                            <Tab name="linux" displayText="Linux">
                              <CodeSnippet className="u-marginTop--10" language="bash" canCopy={false}>
                                sudo apt-get install replicatedhq/troubleshoot
                              </CodeSnippet>
                            </Tab>
                          </TabView>
                        </Fragment>
                      )}
                    </Fragment>
                  )
              }
        </div>
          </div>
        </div>
        )
        <div className="flex-auto flex justifyContent--flexEnd">
          <button
            type="button"
            className="btn primary red u-marginRight--30 u-marginBottom--15"
            onClick={this.createDownstreamCluster}
          >
              { showPreflightResults ? "Create Downstream Cluster" : "Skip this step" }
            </button>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(listPreflightResults, {
    name: "listPreflightResultsQuery",
    options: props => {
      const { match } = props;
      return {
        pollInterval: 10000,
        variables: {
          slug: `${match.params.owner}/${match.params.name}`
        }
      };

    }
   })
  )(PreflightChecksPage);
