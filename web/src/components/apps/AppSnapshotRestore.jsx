import React, { Component } from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import Helmet from "react-helmet";
import Loader from "../shared/Loader";
import { restoreDetail } from "../../queries/SnapshotQueries";

class AppSnapshotRestore extends Component {
  state = {
  };

  componentDidMount() {
    this.props.restoreDetail.startPolling(2000);
  }

  render() {
    const { restoreDetail } = this.props;

    if (restoreDetail?.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>{`${this.props.app.name} Snapshots Restore`}</title>
        </Helmet>
        <div className="flex1 flex-column justifyContent--center alignItems--center">
          <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginBottom--10"> Application restore in progress </p>
          <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal"> After all volumes have been restored you will need to log back in to the admin console. </p>
        </div>
        <div>
          {restoreDetail?.volumes?.map((volume, i) => (
            <div className="flex flex1" key={`${volume.name}-${i}`}>
              <p className="flex1 u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold">Restoring volume: {volume.name}</p>
            </div>
          ))}
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(restoreDetail, {
    name: "restoreDetail",
    options: ({ app }) => {
      const appId = app.id
      return {
        variables: { appId },
        fetchPolicy: "no-cache"
      }
    }
  }),
)(AppSnapshotRestore);
