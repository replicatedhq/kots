import * as React from "react";
import * as ReactDOM from "react-dom";
import bugsnag from "@bugsnag/js"
import bugsnagReact from "@bugsnag/plugin-react"
import ReplicatedErrorBoundary from "./components/shared/ErrorBoundary";
import Root from "./Root";

if (window.env.BUGSNAG_API_KEY && window.env.ENVIRONMENT !== "development") {
  const bugsnagClient = bugsnag({
    apiKey: window.env.BUGSNAG_API_KEY,
    releaseStage: window.env.ENVIRONMENT,
    appVersion: window.env.SHIP_CLUSTER_BUILD_VERSION
  });
  bugsnagClient.use(bugsnagReact, React);

  const ErrorBoundary = bugsnagClient.getPlugin("react");
  ReactDOM.render((
    <ErrorBoundary FallbackComponent={ReplicatedErrorBoundary}>
      <Root />
    </ErrorBoundary>
  ), document.getElementById("app"));
} else {
  ReactDOM.render((
    <ReplicatedErrorBoundary>
      <Root />
    </ReplicatedErrorBoundary>
  ), document.getElementById("app"));
}
