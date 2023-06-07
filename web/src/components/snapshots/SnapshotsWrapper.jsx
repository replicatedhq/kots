import React, { Component, Fragment } from "react";
import { Outlet } from "react-router-dom";
import { withRouter } from "@src/utilities/react-router-utilities";
import { KotsPageTitle } from "@components/Head";

import withTheme from "@src/components/context/withTheme";
import Loader from "@src/components/shared/Loader";
import NotFound from "@src/components/static/NotFound";
import SubNavBar from "@src/components/shared/SubNavBar";
import Snapshots from "@src/components/snapshots/Snapshots";
import AppSnapshots from "@src/components/snapshots/AppSnapshots";
import SnapshotSettings from "@src/components/snapshots/SnapshotSettings";
import SnapshotRestore from "@src/components/snapshots/SnapshotRestore";
import SnapshotDetails from "@src/components/snapshots/SnapshotDetails";
import AppSnapshotRestore from "@src/components/snapshots/AppSnapshotRestore";

class SnapshotsWrapper extends Component {
  render() {
    const { appsList } = this.props;

    const snapshotsApps = appsList.filter((app) => app.allowSnapshots);
    const selectedApp =
      snapshotsApps.find((app) => app.slug === this.props.params?.slug) ||
      snapshotsApps[0];
    const tab = this.props.params["*"].split("/")[0];
    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <KotsPageTitle pageName="Version History" />
        <div className="flex-column flex1 u-width--full u-height--full u-overflow--auto">
          {!snapshotsApps[0] ? (
            <div className="flex-column flex1 alignItems--center justifyContent--center">
              <Loader size="60" />
            </div>
          ) : (
            <Fragment>
              <SubNavBar
                className="flex"
                isSnapshots={true}
                activeTab={
                  tab ? (tab === "details" ? "snapshots" : tab) : "snapshots"
                }
                app={selectedApp}
              />
              <Outlet context={{ app: selectedApp }} />
            </Fragment>
          )}
        </div>
      </div>
    );
  }
}

export default withTheme(withRouter(SnapshotsWrapper));
