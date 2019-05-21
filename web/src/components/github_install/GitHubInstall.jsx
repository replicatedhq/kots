import * as React from "react";
import { Switch, Route } from "react-router-dom";
import NotFound from "../static/NotFound";

import GitHubInstallCallback from "./GitHubInstallCallback";

export default class GitHubInstall extends React.Component {

  render() {
    return (
      <div className="flex-column flex1 Login-wrapper">
        <Switch>
          <Route exact path="/install/github/callback" component={GitHubInstallCallback} />
          <Route component={NotFound} />
        </Switch>
      </div>
    );
  }
}
