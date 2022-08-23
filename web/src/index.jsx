import * as React from "react";
import * as ReactDOM from "react-dom";
import { ThemeProvider } from "styled-components";
import ReplicatedErrorBoundary from "./components/shared/ErrorBoundary";
import Root from "./Root";

const theme = {
  colors: {
    primary: "red",
  },
};

ReactDOM.render(
  <ReplicatedErrorBoundary>
    <ThemeProvider theme={theme}>
      <Root />
    </ThemeProvider>
  </ReplicatedErrorBoundary>,
  document.getElementById("app")
);
