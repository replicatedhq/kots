import * as React from "react";
import { Route, Routes } from "react-router-dom";
import NotFound from "../static/NotFound";

import GitHubAuthBegin from "./GitHubAuthBegin";
import GitHubAuthCallback from "./GitHubAuthCallback";

export default class GitHubAuth extends React.Component {
  render() {
    return (
      <div className="flex-column flex1 Login-wrapper">
        <Routes>
          <Route exact path="/auth/github" component={GitHubAuthBegin} />
          <Route
            exact
            path="/auth/github/callback"
            component={GitHubAuthCallback}
          />
          <Route component={NotFound} />
        </Routes>
      </div>
    );
  }
}
