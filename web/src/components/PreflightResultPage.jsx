import React, { Component } from "react";
import { graphql, compose } from "react-apollo";
import { withRouter } from "react-router-dom";

import { getKotsPreflightResult } from "@src/queries/AppsQueries";
import { getLatestKotsPreflight } from "../queries/AppsQueries";

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
            </div>
          </div>
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
          appSlug: match.slug,
          clusterSlug: match.clusterSlug,
          sequence: match.sequence
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
