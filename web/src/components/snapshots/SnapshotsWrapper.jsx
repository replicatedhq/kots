import React, { Component, Fragment } from "react";
import { Switch, Route, Redirect } from "react-router-dom";
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
    const { match, appsList } = this.props;

    const selectedAppSlug = match?.params?.slug || "";

    console.log("selectedAppSlug", selectedAppSlug);

    const snapshotsApps = appsList.filter(
      // locate snapshottable app by slug
      (app) =>
        app.allowSnapshots && (selectedAppSlug === "" || app.slug === selectedAppSlug)
    );

    console.log("appsList", appsList);
    console.log("snapshotsApps", snapshotsApps);

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
                  match.params.tab
                    ? match.params.tab === "details"
                      ? "snapshots"
                      : match.params.tab
                    : "snapshots"
                }
                app={snapshotsApps[0]}
              />
              <Switch>
                <Route
                  exact
                  path="/snapshots"
                  render={() => (
                    <Snapshots
                      isKurlEnabled={this.props.isKurlEnabled}
                      appsList={snapshotsApps}
                    />
                  )}
                />
                <Route
                  exact
                  path="/snapshots/settings"
                  render={(props) => (
                    <SnapshotSettings
                      {...props}
                      isKurlEnabled={this.props.isKurlEnabled}
                      apps={snapshotsApps}
                    />
                  )}
                />
                <Route
                  exact
                  path="/snapshots/details/:id"
                  render={(props) => (
                    <SnapshotDetails
                      {...props}
                      isKurlEnabled={this.props.isKurlEnabled}
                      appsList={snapshotsApps}
                    />
                  )}
                />
                <Route
                  exact
                  path="/snapshots/:slug/:id/restore"
                  render={() => <SnapshotRestore />}
                />
                <Route
                  exact
                  path="/snapshots/partial/:slug"
                  render={(props) => (
                    <AppSnapshots
                      {...props}
                      appsList={snapshotsApps}
                      app={snapshotsApps[0]}
                      appName={snapshotsApps.name}
                    />
                  )}
                />
                <Route
                  exact
                  path="/snapshots/partial/:slug/:id"
                  render={(props) => (
                    <SnapshotDetails
                      {...props}
                      appsList={snapshotsApps}
                      app={snapshotsApps[0]}
                      appName={snapshotsApps.name}
                    />
                  )}
                />
                <Route
                  exact
                  path="/snapshots/partial/:slug/:id/restore"
                  render={() => (
                    <AppSnapshotRestore
                      appsList={snapshotsApps}
                      app={snapshotsApps[0]}
                    />
                  )}
                />
                <Route component={NotFound} />
              </Switch>
            </Fragment>
          )}
        </div>
      </div>
    );
  }
}

export default withTheme(withRouter(SnapshotsWrapper));
