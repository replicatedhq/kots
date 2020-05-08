import * as React from "react";
import { withRouter, Switch, Route, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import dayjs from "dayjs";
import Modal from "react-modal";

import Loader from "../shared/Loader";
import AnalyzerInsights from "./AnalyzerInsights";
import AnalyzerFileTree from "./AnalyzerFileTree";
import { getSupportBundle } from "../../queries/TroubleshootQueries";
import { updateSupportBundle } from "../../mutations/TroubleshootMutations";
import { Utilities } from "../../utilities/utilities";
import "../../scss/components/troubleshoot/SupportBundleAnalysis.scss";

export class SupportBundleAnalysis extends React.Component {
  constructor(props) {
    super();
    this.state = {
      activeTab: props.location.pathname.indexOf("/contents") !== -1 ? "fileTree" : "bundleAnalysis",
      filterTiles: "0"
    };
  }

  reAnalyzeBundle = async (callback) => {
    try {
      const bundleId = this.props.getSupportBundle.getSupportBundle.id;
      const bundleUrl = `${window.env.API_ENDPOINT}/troubleshoot/analyzebundle/${bundleId}`;

      const response = await fetch(bundleUrl, {
        method: "POST"
      });
      const analyzedBundle = await response.json();
      if (callback && typeof callback === "function") {
        callback(analyzedBundle, analyzedBundle.status === "analysis_error");
      }
      this.props.getSupportBundle.refetch();
    } catch (error) {
      if (callback && typeof callback === "function") {
        callback(undefined, error);
      }
    }
  }

  downloadBundle = async (bundle) => {
    const bundleId = bundle.id;
    const hiddenIFrameID = "hiddenDownloader";
    let iframe = document.getElementById(hiddenIFrameID);
    const url = `${window.env.API_ENDPOINT}/troubleshoot/supportbundle/${bundleId}/download?token=${Utilities.getToken()}`;
    if (iframe === null) {
      iframe = document.createElement("iframe");
      iframe.id = hiddenIFrameID;
      iframe.style.display = "none";
      document.body.appendChild(iframe);
    }
    iframe.src = url;
  }

  toggleAnalysisAction = (active) => {
    this.setState({
      activeTab: active,
    });
  }


  render() {
    const { watch, getSupportBundle } = this.props;
    const bundle = getSupportBundle?.getSupportBundle;

    if (getSupportBundle.loading) {
      return (
        <div className="flex-column flex1 justifyContent--center alignItems--center">
          <Loader size="60" />
        </div>
      )
    }

    const insightsUrl = `/app/:slug/troubleshoot/analyze/:bundleSlug`;
    const fileTreeUrl = `/app/:slug/troubleshoot/analyze/:bundleSlug/contents/*`;

    return (
      <div className="container u-marginTop--20 u-paddingBottom--30 flex1 flex-column">
        <div className="flex1 flex-column">
          {bundle &&
            <div className="flex1 flex-column">
              <div className="u-position--relative flex-auto u-marginBottom--20 flex justifyContent--spaceBetween">
                <div className="flex flex1 u-marginTop--10 u-marginBottom--10">
                  <div className="flex-column flex1">
                    <div className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginBottom--20">
                      <Link to={`/app/${this.props.watch.slug}/troubleshoot`} className="replicated-link u-marginRight--5">Support bundles</Link> > <span className="u-marginLeft--5">{dayjs(bundle.createdAt).format("MMMM D, YYYY")}</span>
                    </div>
                    <div className="flex flex1 justifyContent--spaceBetween">
                      <div className="flex flex-column">
                        <h2 className="u-fontSize--header2 u-fontWeight--bold u-color--tuna flex alignContent--center alignItems--center">Support bundle analysis</h2>
                      </div>
                    </div>
                    <div className="upload-date-container flex u-marginTop--5 alignItems--center">
                      <div className="flex alignSelf--center">
                        <p className="flex u-fontSize--normal u-color--dustyGray u-fontWeight--medium">Collected on <span className="u-fontWeight--bold u-marginLeft--5">{dayjs(bundle.createdAt).format("MMMM D, YYYY @ h:mm a")}</span></p>
                      </div>
                    </div>
                  </div>
                  <div className="flex flex-auto alignItems--center justifyContent--flexEnd">
                    <button className="btn primary lightBlue" onClick={() => this.downloadBundle(bundle)}> Download bundle </button>
                  </div>
                </div>
              </div>
              {bundle.kotsLicenseType === "community" &&
                <div className="flex">
                  <div className="CommunityLicenseBundle--wrapper flex flex1 alignItems--center">
                    <div className="flex flex-auto">
                      <span className="icon communityIcon"></span>
                    </div>
                    <div className="flex1 flex-column u-marginLeft--10">
                      <p className="u-color--emperor u-fontSize--large u-fontWeight--bold u-lineHeight--medium u-marginBottom--5"> This bundle was uploaded by a customer under a Community license type. </p>
                      <p className="u-color--silverChalice u-fontSize--normal u-lineHeight--medium"> Customers with Community licenses are using the free, Community-Supported version of Nomad Enterprise. </p>
                    </div>
                    <div className="flex justifyContent--flexEnd">
                      <a href="https://kots.io/vendor/entitlements/community-licenses/" target="_blank" rel="noopener noreferrer" className="btn secondary lightBlue"> Learn more about Community Licenses </a>
                    </div>
                  </div>
                </div>}
              <div className="flex-column flex1">
                <div className="customer-actions-wrapper flex1 flex-column">
                  <div className="flex action-tab-bar">
                    <Link to={`/app/${watch.slug}/troubleshoot/analyze/${bundle.slug}`} className={`${this.state.activeTab === "bundleAnalysis" ? "is-active" : ""} tab-item blue`} onClick={() => this.toggleAnalysisAction("bundleAnalysis")}>Analysis overview</Link>
                    <Link to={`/app/${watch.slug}/troubleshoot/analyze/${bundle.slug}/contents/`} className={`${this.state.activeTab === "fileTree" ? "is-active" : ""} tab-item blue`} onClick={() => this.toggleAnalysisAction("fileTree")}>File inspector</Link>
                  </div>
                  <div className="flex-column flex1 action-content blue">
                    <Switch>
                      <Route exact path={insightsUrl} render={() =>
                        <AnalyzerInsights
                          status={bundle.status}
                          refetchSupportBundle={this.props.getSupportBundle.refetch}
                          insights={bundle.analysis?.insights}
                          reAnalyzeBundle={this.reAnalyzeBundle}
                        />
                      } />
                      <Route exact path={fileTreeUrl} render={() =>
                        <AnalyzerFileTree
                          watchSlug={watch.slug}
                          bundle={bundle}
                          downloadBundle={() => this.downloadBundle(bundle)}
                        />
                      } />
                    </Switch>
                  </div>
                </div>
              </div>
            </div>
          }
        </div>
      </div>
    );
  }
}

export default withRouter(compose(
  withApollo,
  graphql(getSupportBundle, {
    name: "getSupportBundle",
    options: ({ match }) => ({
      variables: {
        watchSlug: match.params.bundleSlug
      },
      fetchPolicy: "no-cache"
    }),
  })
)(SupportBundleAnalysis));
