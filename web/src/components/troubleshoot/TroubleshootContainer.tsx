import React, { Component } from "react";
import { Switch, Route } from "react-router-dom";
import NotFound from "../static/NotFound";
import SupportBundleList from "../troubleshoot/SupportBundleList";
import SupportBundleAnalysis from "../troubleshoot/SupportBundleAnalysis";
import GenerateSupportBundle from "../troubleshoot/GenerateSupportBundle";
import Redactors from "../redactors/Redactors";
import EditRedactor from "../redactors/EditRedactor";

// Types
import { App } from "@types";

type Props = {
  app: App | null;
  appName: string;
};
type State = {
  newBundleSlug: string;
};
class TroubleshootContainer extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      newBundleSlug: "",
    };
  }

  updateBundleSlug = (value: string) => {
    this.setState({ newBundleSlug: value });
  };

  render() {
    const { app, appName } = this.props;

    return (
      <div className="flex-column flex1">
        <Switch>
          <Route
            exact
            path="/app/:slug/troubleshoot"
            render={() => (
              <SupportBundleList
                watch={app}
                newBundleSlug={this.state.newBundleSlug}
                updateBundleSlug={this.updateBundleSlug}
              />
            )}
          />
          <Route
            exact
            path="/app/:slug/troubleshoot/generate"
            render={() => (
              <GenerateSupportBundle
                watch={app}
                newBundleSlug={this.state.newBundleSlug}
                updateBundleSlug={this.updateBundleSlug}
              />
            )}
          />
          <Route
            path="/app/:slug/troubleshoot/analyze/:bundleSlug"
            render={() => <SupportBundleAnalysis watch={app} />}
          />
          <Route
            exact
            path="/app/:slug/troubleshoot/redactors"
            render={(props) => (
              <Redactors
                {...props}
                appSlug={app?.slug || ""}
                appName={appName}
              />
            )}
          />
          <Route
            exact
            path="/app/:slug/troubleshoot/redactors/new"
            render={() => <EditRedactor />}
          />
          <Route
            exact
            path="/app/:slug/troubleshoot/redactors/:redactorSlug"
            render={() => <EditRedactor />}
          />
          <Route component={NotFound} />
        </Switch>
      </div>
    );
  }
}

export default TroubleshootContainer;
