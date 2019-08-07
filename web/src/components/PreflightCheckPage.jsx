import React, { Component, Fragment } from "react";
import { withRouter } from "react-router-dom";
import { graphql, withApollo, compose } from "react-apollo";
import classNames from "classnames";

import { listPreflightResults, getWatch } from "@src/queries/WatchQueries";
import CodeSnippet from "./shared/CodeSnippet";

import "@src/scss/components/PreflightCheckPage.scss";
import { listClusters } from "../queries/ClusterQueries";

class PreflightChecksPage extends Component {
  state = {
    showPreflightResults: false
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

  createDownstreamCluster = async () => {
    const { history, getWatchQuery, match } = this.props;
    const watch = getWatchQuery?.getWatch;

    if (!watch) {
      // TODO: Throw an error. User is going too fast for graphql.
      return;
    }
    const searchParams = new URLSearchParams(location.search);
    const gitPath = searchParams.get("path");
    const upstreamUrl = `ship://ship-cloud/${match.params.owner}/${match.params.name}`;
    const queryResult = await this.props.client.query({
      query: listClusters
    });
    const cluster = queryResult.data.listClusters.find( c => c.slug === match.params.downstream );
    if (!cluster) {
      // TODO: Cant find cluster. Make a Graphql query to fetch an individual cluster instead of searching like this.
      return;
    }

    history.replace(`/watch/create/init?upstream=${upstreamUrl}&cluster_id=${cluster.id}${gitPath
      ? `&path=${gitPath}`
      : ""
    }`);
  }

  render() {
    const { showPreflightResults } = this.state;

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
                      {showPreflightResults.result && JSON.parse(showPreflightResults?.result)?.results.map( (row, idx) => {
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
                      <CodeSnippet
                        className="u-marginTop--10"
                        language="bash"
                        canCopy={true}
                        preText="If you already have the plugin installed, you can skip the first line"
                      >
                        kubectl krew install preflight
                        {`kubectl preflight ${window.env.REST_ENDPOINT}/v1${location.pathname}`}
                      </CodeSnippet>
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
            className="btn primary u-marginRight--30 u-marginBottom--15"
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
  graphql(getWatch, {
    name: "getWatchQuery",
    options: props => {
      const { match } = props;
      return {
        variables: {
          slug: `${match.params.owner}/${match.params.name}`
        }
      }
    }
  }),
  graphql(listPreflightResults, {
    name: "listPreflightResultsQuery",
    options: props => {
      const { match } = props;
      return {
        pollInterval: 2000,
        variables: {
          slug: `${match.params.owner}/${match.params.name}`
        }
      };

    }
   })
  )(PreflightChecksPage);
