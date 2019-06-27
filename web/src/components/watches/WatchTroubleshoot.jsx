import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { listSupportBundles } from "@src/queries/TroubleshootQueries";
import withTheme from "@src/components/context/withTheme";

class WatchTroubleshoot extends Component {

  render() {
    return (
      <div className="TroubleshootCode--wrapper flex-auto">

      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  withTheme,
  graphql(listSupportBundles, {
    name: "supportBundles",
    options: props => {
      const { owner, slug } = props.match.params;
      return {
        variables: {
          watchSlug: `${owner}/${slug}`
        }
      }
    }
  }),
)(WatchTroubleshoot);

