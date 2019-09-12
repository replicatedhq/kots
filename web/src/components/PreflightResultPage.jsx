import React, { Component } from "react";
import { Link } from "react-router-dom";
import { graphql, compose } from "react-apollo";
import { withRouter } from "react-router-dom";

import { getKotsPreflightResult, getLatestKotsPreflight } from "@src/queries/AppsQueries";
import { deployKotsVersion } from "@src/mutations/AppsMutations";
import Loader from "./shared/Loader";
import PreflightRenderer from "./PreflightRenderer";

class PreflightResultPage extends Component {

  deployKotsDownstream = () => {
    const { makeCurrentVersion, match, data, history } = this.props;
    const gqlData = data.getKotsPreflightResult || data.getLatestKotsPreflightResult;
    const upstreamSlug = match.params.slug;

    const sequence = parseInt(match.params.sequence, 10);
    makeCurrentVersion(upstreamSlug, sequence, gqlData.clusterSlug).then( () => {
      history.push(`/app/${match.params.slug}/downstreams/${match.params.downstreamSlug}/version-history`);
    });
  }

  render() {
    const { data, match } = this.props;
    const isLoading = data.loading;

    // No cluster slug is present if coming from the license upload view
    const isLicenseFlow = !match.params.clusterSlug;
    const preflightResultData = isLoading
      ? null
      : data.getKotsPreflightResult || data.getLatestKotsPreflightResult;
    const hasData = preflightResultData?.result;

    if (hasData) {
      data.stopPolling();
    }

    return (
      <div className="flex-column flex1">
        <div className="flex1 u-overflow--auto">
          <div className="PreflightChecks--wrapper flex u-paddingTop--30 u-overflow--hidden">
            <div className="u-minWidth--full u-minHeight--full">
              <p className="u-fontSize--header u-color--tuna u-fontWeight--bold">
                Preflight checks
              </p>
              <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--10">
                Preflight checks are designed to be run against a target cluster before installing an application. Preflights are simply a different set of collectors + analyzers. These checks are optional but are recommended to ensure that the application you install will work properly.
              </p>
              { (isLoading || !hasData ) && (
                <div className="flex-column justifyContent--center alignItems--center u-minHeight--full u-minWidth--full">
                  <Loader size="60" />
                </div>
              )}
              {
                hasData && (
                  <div className="flex-column">
                    <PreflightRenderer
                      className="u-marginTop--20"
                      onDeployClick={this.deployKotsDownstream}
                      results={preflightResultData.result}
                    />
                  </div>
                )
              }
            </div>
          </div>
        </div>
        { hasData && !isLicenseFlow && (
          <div className="flex-auto flex justifyContent--flexEnd">
            <button
              type="button"
              className="btn primary u-marginRight--30 u-marginBottom--15"
              onClick={this.deployKotsDownstream}
            >
              Create Downstream Cluster
          </button>
          </div>
        )}
        {
          hasData && isLicenseFlow && (
            <div className="flex-auto flex justifyContent--flexEnd">
              <Link to={`/app/${preflightResultData.appSlug}`}>
              <button
                type="button"
                className="btn primary u-marginRight--30 u-marginBottom--15"
              >
                Continue
              </button>
              </Link>
            </div>
          )
        }
      </div>
    );
  }
}

export default compose(
  withRouter,
  graphql(getKotsPreflightResult, {
    skip: props => {
      const { match } = props;
      return !match.params.downstreamSlug;
    },
    options: props => {
      const { match } = props;

      return {
        pollInterval: 2000,
        variables: {
          appSlug: match.params.slug,
          clusterSlug: match.params.downstreamSlug,
          sequence: match.params.sequence
        }
      };
    }
  }),
  graphql(getLatestKotsPreflight, {
    skip: props => {
      const { match } = props;

      return !!match.params.downstreamSlug;
    },
    options: () => {
      return {
        pollInterval: 2000
      }
    }
  }),
  graphql(deployKotsVersion, {
    props: ({ mutate }) => ({
      deployKotsVersion: (upstreamSlug, sequence, clusterSlug) => mutate({ variables: { upstreamSlug, sequence, clusterSlug } })
    })
  }),
)(PreflightResultPage);
