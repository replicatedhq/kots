import React, { Component } from "react";
import Helmet from "react-helmet";

import {
  Utilities,
  getReadableLicenseType,
  isLicenseOutOfDate,
  getEntitlementSpecFromState,
  getWatchMetadata,
  getWatchLicenseFromState,
  getLicenseExpiryDate,
} from "@src/utilities/utilities";

import { graphql, compose, withApollo } from "react-apollo";
import { getWatchLicense, getLatestWatchLicense } from "@src/queries/WatchQueries";
import { syncWatchLicense } from "@src/mutations/WatchMutations";
import Loader from "../shared/Loader";

class WatchLicense extends Component {

  constructor(props) {
    super(props);

    this.state = {
      watchLicense: null,
      latestWatchLicense: null,
      syncing: false,
      syncingError: ""
    }
  }

  componentDidMount() {
    const { getWatchLicense } = this.props.getWatchLicense;
    if (getWatchLicense) {
      this.setState({ watchLicense: getWatchLicense });
    }
  }

  componentDidUpdate(lastProps) {
    // current license
    if (this.props.getWatchLicense?.error && !this.state.watchLicense) {
      // no current license found in db, construct from stateJSON
      const watchLicense = getWatchLicenseFromState(this.props.watch);
      this.setState({ watchLicense });
    } else if (this.props.getWatchLicense !== lastProps.getWatchLicense && this.props.getWatchLicense) {
      const { getWatchLicense } = this.props.getWatchLicense;
      if (getWatchLicense) {
        this.setState({ watchLicense: getWatchLicense });
      }
    }
    // latest license
    if (this.props.getLatestWatchLicense !== lastProps.getLatestWatchLicense && this.props.getLatestWatchLicense) {
      const { getLatestWatchLicense } = this.props.getLatestWatchLicense;
      if (getLatestWatchLicense) {
        this.setState({ latestWatchLicense: getLatestWatchLicense });
      }
    }
  }

  syncWatchLicense = () => {
    this.setState({ syncing: true, syncingError: "" });
    const { watch } = this.props;

    const appMeta = getWatchMetadata(watch.metadata);
    const licenseId = appMeta.license.id;
    const entitlementSpec = getEntitlementSpecFromState(watch.stateJSON);

    this.props.syncWatchLicense(watch.id, licenseId, entitlementSpec)
      .then(response => {
        this.setState({ watchLicense: response.data.syncWatchLicense });
      })
      .catch(err => {
        console.log(err);
        err.graphQLErrors.map(({ message }) => {
          this.setState({ syncingError: message });
        });
      })
      .finally(() => {
        this.setState({ syncing: false });
      });
  }

  render() {
    const { watchLicense, latestWatchLicense, syncing, syncingError } = this.state;

    if (!watchLicense || !latestWatchLicense) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const { watch } = this.props;

    const createdAt = Utilities.dateFormat(watchLicense.createdAt, "MMM D, YYYY");
    const licenseType = getReadableLicenseType(watchLicense.type);
    const assignedReleaseChannel = watchLicense.channel;
    const expiresAt = getLicenseExpiryDate(watchLicense);
    const isOutOfDate = isLicenseOutOfDate(watchLicense, latestWatchLicense);

    return (
      <div className="flex justifyContent--center">
        <Helmet>
          <title>{`${watch.watchName} License`}</title>
        </Helmet>
        <div className="LicenseDetails--wrapper u-textAlign--left u-paddingRight--20 u-paddingLeft--20">
          <div className="flex u-marginBottom--20 u-paddingBottom--5 u-marginTop--20 alignItems--center">
            <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginRight--10">License details</p>
            {isOutOfDate && <p className="u-fontWeight--bold u-color--orange">Outdated</p>}
          </div>
          <div className="u-color--tundora u-fontSize--normal u-fontWeight--medium">
            <div className="flex u-marginBottom--20">
              <p className="u-marginRight--10">Assigned release channel:</p>
              <p className="u-fontWeight--bold u-color--tuna">{assignedReleaseChannel}</p>
            </div>
            <div className="flex u-marginBottom--20">
              <p className="u-marginRight--10">Created:</p>
              <p className="u-fontWeight--bold u-color--tuna">{createdAt}</p>
            </div>
            <div className="flex u-marginBottom--20">
              <p className="u-marginRight--10">Expires:</p>
              <p className="u-fontWeight--bold u-color--tuna">{expiresAt}</p>
            </div>
            <div className="flex u-marginBottom--20">
              <p className="u-marginRight--10">License Type:</p>
              <p className="u-fontWeight--bold u-color--tuna">{licenseType}</p>
            </div>
            {watchLicense.entitlements?.map(entitlement => {
              return (
                <div key={entitlement.key} className="flex u-marginBottom--20">
                  <p className="u-marginRight--10">{entitlement.name}</p>
                  <p className="u-fontWeight--bold u-color--tuna">{entitlement.value}</p>
                </div>
              );
            })}
            <button className="btn secondary green u-marginBottom--10" disabled={syncing} onClick={() => this.syncWatchLicense(watch)}>{syncing ? "Syncing" : "Sync License"}</button>
            {syncingError && <p className="u-fontWeight--bold u-color--red u-fontSize--small u-position--absolute">{syncingError}</p>}
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  graphql(getWatchLicense, {
    name: "getWatchLicense",
    options: props => {
      const entitlementSpec = getEntitlementSpecFromState(props.watch.stateJSON);
      return {
        variables: {
          watchId: props.watch.id,
          entitlementSpec
        },
        fetchPolicy: "no-cache"
      };
    }
  }),
  graphql(getLatestWatchLicense, {
    name: "getLatestWatchLicense",
    options: props => {
      const appMeta = getWatchMetadata(props.watch.metadata);
      const licenseId = appMeta.license.id;
      const entitlementSpec = getEntitlementSpecFromState(props.watch.stateJSON);
      return {
        variables: {
          licenseId,
          entitlementSpec
        },
        fetchPolicy: "no-cache"
      };
    }
  }),
  graphql(syncWatchLicense, {
    props: ({ mutate }) => ({
      syncWatchLicense: (watchId, licenseId, entitlementSpec) => mutate({ variables: { watchId, licenseId, entitlementSpec } })
    })
  })
)(WatchLicense);
