import * as React from "react";
import * as ReactDOM from "react-dom";
import ReplicatedErrorBoundary from "./components/shared/ErrorBoundary";
import Root from "./Root";

ReactDOM.render((
  <ReplicatedErrorBoundary>
    <Root />
  </ReplicatedErrorBoundary>
), document.getElementById("app"));
