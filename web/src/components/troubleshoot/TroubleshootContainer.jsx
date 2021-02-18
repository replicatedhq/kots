import React, { Component } from "react";
import { withRouter, Switch, Route } from "react-router-dom";
import NotFound from "../static/NotFound";
import SupportBundleList from "../troubleshoot/SupportBundleList";
import SupportBundleAnalysis from "../troubleshoot/SupportBundleAnalysis";
import GenerateSupportBundle from "../troubleshoot/GenerateSupportBundle";
import Redactors from "../redactors/Redactors";
import EditRedactor from "../redactors/EditRedactor";

class TroubleshootContainer extends Component {
 
  render() {
    const { app, appName } = this.props;

    return (
      <div className="flex-column flex1">
        <Switch>
          <Route exact path="/app/:slug/troubleshoot" render={() =>
            <SupportBundleList watch={app} />
          } />
          <Route exact path="/app/:slug/troubleshoot/generate" render={() =>
            <GenerateSupportBundle watch={app} />
          } />
          <Route path="/app/:slug/troubleshoot/analyze/:bundleSlug" render={() =>
            <SupportBundleAnalysis watch={app} />
          } />
          <Route exact path="/app/:slug/troubleshoot/redactors" render={(props) =>
            <Redactors {...props} appSlug={app.slug} appName={appName} />}
          />
          <Route exact path="/app/:slug/troubleshoot/redactors/new" render={(props) =>
            <EditRedactor {...props} appSlug={app.slug} appName={appName} isNew={true} />}
          />
          <Route exact path="/app/:slug/troubleshoot/redactors/:redactorSlug" render={(props) =>
            <EditRedactor {...props} appSlug={app.slug} appName={appName} />}
          />
          <Route component={NotFound} />
        </Switch>
      </div>
    );
  }
}

export default withRouter(TroubleshootContainer);
