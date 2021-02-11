import React, { Component, Fragment } from "react";
import { withRouter, Switch, Route } from "react-router-dom";
import { Helmet } from "react-helmet";

import withTheme from "@src/components/context/withTheme";
import NotFound from "@src/components/static/NotFound";
import SubNavBar from "@src/components/shared/SubNavBar";
import Snapshots from "@src/components/snapshots/Snapshots";
import AppSnapshots from "@src/components/apps/AppSnapshots";
import SnapshotSettings from "@src/components/snapshots/SnapshotSettings";
import SnapshotDetails from "@src/components/snapshots/SnapshotDetails";

class SnapshotsWrapper extends Component {
  componentDidMount() {
    const { history } = this.props;

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


    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <Helmet>
          <title> Snapshots </title>
        </Helmet>
        <div className="flex-column flex1 u-width--full u-height--full u-overflow--auto">
          <Fragment>
            <SubNavBar
              className="flex"
              isSnapshots={true}
              activeTab={match.params.tab}
            />
            <Switch>
              <Route exact path="/snapshots/full" render={() =>
                <Snapshots
                  appName={this.props.appName}
                  isKurlEnabled={this.props.isKurlEnabled}
                  appsList={this.props.appsList}
                />
              } />
              <Route exact path="/snapshots/settings" render={(props) =>
                <SnapshotSettings
                  appName={this.props.appName}
                  isKurlEnabled={this.props.isKurlEnabled}
                  appsList={this.props.appsList}
                />}
              />
              <Route exact path="/snapshots/full/details/:id" render={(props) =>
                <SnapshotDetails
                  appName={this.props.appName}
                  isKurlEnabled={this.props.isKurlEnabled}
                  appsList={this.props.appsList}
                />}
              />
              <Route exact path="/snapshots/partial/:slug" render={() =>
                <AppSnapshots
                  appsList={this.props.appsList}
                  app={appsList[0]}
                  history={this.props.history}
                />
              } />
              <Route component={NotFound} />
            </Switch>
          </Fragment>
          {/* <Snapshots {...props} appName={this.state.selectedAppName} isKurlEnabled={this.state.isKurlEnabled} appsList={appsList} /> */}
        </div>
      </div>
    );
  }
}

export default withTheme(withRouter(SnapshotsWrapper));
