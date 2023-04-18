import React, { Component } from "react";
import { Switch, Route } from "react-router-dom";
import NotFound from "../static/NotFound";
import SupportBundleList from "../troubleshoot/SupportBundleList";
import SupportBundleAnalysis from "../troubleshoot/SupportBundleAnalysis";
import GenerateSupportBundle from "../troubleshoot/GenerateSupportBundle";
import Redactors from "../redactors/Redactors";
import EditRedactor from "../redactors/EditRedactor";

// Types
import { App, SupportBundleProgress } from "@types";

type Props = {
  app: App | null;
  appName: string;
};
type State = {
  newBundleSlug: string;
  isGeneratingBundle: false;
  generateBundleErrMsg: string;
  loading: boolean;
  bundleAnalysisProgress?: SupportBundleProgress;
  getSupportBundleErrMsg: string;
  displayErrorModal: boolean;
  bundle: object;
  loadingBundleId: string;
  loadingBundle: boolean;
};
class TroubleshootContainer extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      newBundleSlug: "",
      isGeneratingBundle: false,
      generateBundleErrMsg: "",
      loading: false,
      getSupportBundleErrMsg: "",
      displayErrorModal: false,
      bundle: {},
      loadingBundleId: "",
      loadingBundle: false,
    };
  }

  updateBundleSlug = (value: string) => {
    this.setState({ newBundleSlug: value });
  };

  updateState = (value: State) => {
    this.setState(value);
  };

  pollForBundleAnalysisProgress = async () => {
    this.setState({ loadingBundle: true });
    const { newBundleSlug } = this.state;
    if (!newBundleSlug) {
      // component may start polling before bundle slug is set
      // this is to prevent an api call if the slug is not set
      return;
    }
    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/${newBundleSlug}`,
      {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        method: "GET",
      }
    )
      .then(async (res) => {
        if (!res.ok) {
          this.setState({
            loading: false,
            getSupportBundleErrMsg: `Unexpected status code: ${res.status}`,
            displayErrorModal: true,
          });
          return;
        }
        const bundle = await res.json();
        this.setState({
          bundleAnalysisProgress: bundle.progress,
          bundle,
          loadingBundleId: bundle.id,
          loadingBundle: true,
        });

        if (bundle.status !== "running") {
          this.setState({ loadingBundleId: "", loadingBundle: false });
        }
      })

      .catch((err) => {
        this.setState({
          loading: false,
          getSupportBundleErrMsg: err
            ? err.message
            : "Something went wrong, please try again.",
          displayErrorModal: true,
          loadingBundle: false,
        });
      });
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
                pollForBundleAnalysisProgress={
                  this.pollForBundleAnalysisProgress
                }
                bundle={this.state.bundle}
                bundleProgress={this.state.bundleAnalysisProgress}
                loadingBundleId={this.state.loadingBundleId}
                loadingBundle={this.state.loadingBundle}
                updateState={this.updateState}
                displayErrorModal={this.state.displayErrorModal}
                loading={this.state.loading}
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
                bundle={this.state.bundle}
              />
            )}
          />
          <Route
            path="/app/:slug/troubleshoot/analyze/:bundleSlug"
            render={() => (
              <SupportBundleAnalysis
                watch={app}
                pollForBundleAnalysisProgress={
                  this.pollForBundleAnalysisProgress
                }
                bundle={this.state.bundle}
                bundleProgress={this.state.bundleAnalysisProgress}
                updateState={this.updateState}
                displayErrorModal={this.state.displayErrorModal}
                getSupportBundleErrMsg={this.state.getSupportBundleErrMsg}
                loading={this.state.loading}
              />
            )}
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
