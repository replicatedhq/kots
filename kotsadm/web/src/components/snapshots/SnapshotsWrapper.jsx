import React, { Component, Fragment } from "react";
import { withRouter, Switch, Route } from "react-router-dom";
import { Helmet } from "react-helmet";

import withTheme from "@src/components/context/withTheme";
import Loader from "@src/components/shared/Loader";
import NotFound from "@src/components/static/NotFound";
import SubNavBar from "@src/components/shared/SubNavBar";
import Snapshots from "@src/components/snapshots/Snapshots";
import AppSnapshots from "@src/components/apps/AppSnapshots";
import SnapshotSettings from "@src/components/snapshots/SnapshotSettings";
import SnapshotDetails from "@src/components/snapshots/SnapshotDetails";
import AppSnapshotRestore from "@src/components/apps/AppSnapshotRestore";

class SnapshotsWrapper extends Component {
  componentDidMount() {
    const { history } = this.props;

    if (history.location.pathname === "/snapshots") {
      history.replace(`/snapshots/full`);
      return;
    }
  }

  componentDidUpdate(_, lastState) {
    const { history } = this.props;
    // Used for a fresh reload
    if (history.location.pathname === "/snapshots") {
      history.replace(`/snapshots/full`);
      return;
    }
  }

  render() {
    const {
      match,
      appsList
    } = this.props;

    const snapshotsApps = appsList.filter(app => app.allowSnapshots);

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <Helmet>
          <title> Snapshots </title>
        </Helmet>
        <div className="flex-column flex1 u-width--full u-height--full u-overflow--auto">
          {!snapshotsApps[0]
            ? <div className="flex-column flex1 alignItems--center justifyContent--center">
              <Loader size="60" />
            </div> :
            (
              <Fragment>
                <SubNavBar
                  className="flex"
                  isSnapshots={true}
                  activeTab={match.params.tab}
                  app={snapshotsApps[0]}
                />
                <Switch>
                  <Route exact path="/snapshots/full" render={() =>
                    <Snapshots
                      isKurlEnabled={this.props.isKurlEnabled}
                      appsList={this.props.appsList}
                    />
                  } />
                  <Route exact path="/snapshots/settings" render={(props) =>
                    <SnapshotSettings
                      {...props}
                      isKurlEnabled={this.props.isKurlEnabled}
                      apps={snapshotsApps}
                      toggleSnapshotsRBACModal={this.props.toggleSnapshotsRBACModal}
                    />}
                  />
                  <Route exact path="/snapshots/full/details/:id" render={(props) =>
                    <SnapshotDetails
                      {...props}
                      isKurlEnabled={this.props.isKurlEnabled}
                      appsList={snapshotsApps}
                    />}
                  />
                  <Route exact path="/snapshots/partial/:slug" render={(props) =>
                    <AppSnapshots
                      {...props}
                      appsList={snapshotsApps}
                      app={snapshotsApps[0]}
                      appName={snapshotsApps.name}
                    />
                  } />
                  <Route exact path="/snapshots/partial/:slug/:id" render={(props) =>
                    <SnapshotDetails
                      {...props}
                      appsList={snapshotsApps}
                      app={snapshotsApps[0]}
                      appName={snapshotsApps.name} />
                  } />
                  <Route exact path="/snapshots/:slug/:id/restore" render={() =>
                    <AppSnapshotRestore appsList={snapshotsApps} app={snapshotsApps[0]} />
                  } />
                  <Route component={NotFound} />
                </Switch>
              </Fragment>
            )
          }
        </div>
      </div>
    );
  }
}

export default withTheme(withRouter(SnapshotsWrapper));
