import React, { Component } from "react";
import { graphql, compose } from "react-apollo";
import { withRouter } from "react-router-dom";
import moment from "moment";

import { getKotsPreflightResult } from "@src/queries/AppsQueries";
import { getLatestKotsPreflight } from "../queries/AppsQueries";
import Loader from "./shared/Loader";
import PreflightRenderer from "./PreflightRenderer";

class PreflightResultPage extends Component {
  constructor(props) {
    super(props);

    this.state = {
      preflightResults: false
    };

  }

  componentDidUpdate() {

  }

  render() {
    const { data } = this.props;
    const isLoading = data.loading;
    const preflightResultData = isLoading
      ? null
      : data.getKotsPreflightResult || data.getLatestKotsPreflight;
    const hasData = preflightResultData?.result;

    if (hasData) {
      data.stopPolling();
    }

    return (
      <div className="flex-column flex1">
        <div className="flex1 u-overflow--auto">
          <div className="PreflightChecks--wrapper u-paddingTop--30 u-overflow--hidden">
            <div className="u-minWidth--full u-minHeight--full">
              <p className="u-fontSize--header u-color--tuna u-fontWeight--bold">
                Preflight checks
              </p>
              <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--10">
                Preflight checks are designed to be run against a target cluster before installing an application. Preflights are simply a different set of collectors + analyzers. These checks are optional but are recommended to ensure that the application you install will work properly.
              </p>
              { (isLoading || !hasData ) && (
                <div className="flex-column justifyContent--center alignItems--center">
                  <Loader size="60" />
                </div>
              )}
              {
                hasData && (
                  <div className="flex-column">
                    <p className="u-fontSize--large u-color--dustyGray u-fontWeight--bold">Preflights last run at: {moment(preflightResultData.updatedAt).format("MMM D, YYYY h:mm A")}</p>
                    <PreflightRenderer
                      className="u-marginTop--30"
                      results={preflightResultData.result}
                    />
                  </div>
                )
              }
            </div>
          </div>
        </div>
        <div className="flex-auto flex justifyContent--flexEnd">
          <button
            type="button"
            className="btn primary u-marginRight--30 u-marginBottom--15"
          >
            Create Downstream Cluster
          </button>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  graphql(getKotsPreflightResult, {
    skip: props => {
      const { match } = props;
      return !!match.clusterSlug;
    },
    options: props => {
      const { match } = props;

      return {
        fetchPolicy: "no-cache",
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

      return !match.clusterSlug;
    },
    options: () => {
      return {
        fetchPolicy: "no-cace",
        pollInterval: 2000
      }
    }
  })
)(PreflightResultPage);
