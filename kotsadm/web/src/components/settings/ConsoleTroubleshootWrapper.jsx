import React, { Component } from "react";
import { withRouter, Switch, Route } from "react-router-dom";
import Toggle from "../shared/Toggle";
import SupportBundleList from "../troubleshoot/SupportBundleList";
import SupportBundleAnalysis from "../troubleshoot/SupportBundleAnalysis";
import GenerateSupportBundle from "../troubleshoot/GenerateSupportBundle";
import Redactors from "../redactors/Redactors";
import EditRedactor from "../redactors/EditRedactor";

class ConsoleTroubleshootWrapper extends Component {
  render() {
    const {
      app
    } = this.props;

    return (
      <div className="u-paddingLeft--20">
        <div className="flex justifyContent--center u-paddingBottom--20 u-paddingTop--30">
          <Toggle
            items={[
              {
                title: "Support bundles",
                onClick: () => this.props.history.push(`/settings/troubleshoot/support-bundle`),
                isActive: this.props.location.pathname.startsWith("/settings/troubleshoot/support-bundle")
              },
              {
                title: "Custom redactors",
                onClick: () => this.props.history.push(`/settings/troubleshoot/redactors`),
                isActive: this.props.location.pathname.startsWith("/settings/troubleshoot/redactors")
              }
            ]}
          />
        </div>
        <Switch>
          <Route exact path="/settings/troubleshoot/support-bundle" render={() => <SupportBundleList app={app} /> }/>
          <Route exact path="/settings/troubleshoot/support-bundle/generate" render={() => <GenerateSupportBundle app={app} /> } />
          <Route path="/settings/troubleshoot/support-bundle/analyze/:bundleSlug" render={() => <SupportBundleAnalysis app={app} /> } />
          <Route exact path="/settings/troubleshoot/redactors" render={(props) => <Redactors {...props} />} />
          <Route exact path="/settings/troubleshoot/redactors/new" render={(props) => <EditRedactor {...props} />} />
          <Route exact path="/settings/troubleshoot/redactors/:slug" render={(props) => <EditRedactor {...props} />} />
        </Switch>
      </div>
    )
  }
}

export default withRouter(ConsoleTroubleshootWrapper);