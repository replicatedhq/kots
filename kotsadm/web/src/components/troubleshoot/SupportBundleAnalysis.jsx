import * as React from "react";
import { withRouter, Switch, Route, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import dayjs from "dayjs";

import Loader from "../shared/Loader";
import AnalyzerInsights from "./AnalyzerInsights";
import AnalyzerFileTree from "./AnalyzerFileTree";
import AnalyzerRedactorReport from "./AnalyzerRedactorReport";
import { getSupportBundle } from "../../queries/TroubleshootQueries";
import { Utilities } from "../../utilities/utilities";
import "../../scss/components/troubleshoot/SupportBundleAnalysis.scss";
import download from "downloadjs";

export class SupportBundleAnalysis extends React.Component {
  constructor(props) {
    super();
    this.state = {
      activeTab: props.location.pathname.indexOf("/contents") !== -1 ? "fileTree" : location.pathname.indexOf("/redactor") !== -1  ? "redactorReport" : "bundleAnalysis",
      filterTiles: "0"
    };
  }

  downloadBundle = async (bundle) => {
    fetch(`${window.env.API_ENDPOINT}/troubleshoot/supportbundle/${bundle.id}/download`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
      }
    })
      .then(async (result) => {
        if (result.ok) {
          const blob = await result.blob();
          download(blob, "supportbundle.tar.gz", "application/gzip")
        }
      })
      .catch(err => {
        console.log(err);
      })
  }

  toggleAnalysisAction = (active) => {
    this.setState({
      activeTab: active,
    });
  }

  componentDidUpdate = (lastProps) => {
    const { location } = this.props;
    if (location !== lastProps.location) {
      this.setState({
        activeTab: location.pathname.indexOf("/contents") !== -1 ? "fileTree" : location.pathname.indexOf("/redactor") !== -1  ? "redactorReport" : "bundleAnalysis"
      });
    }
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
    const redactorUrl = `/app/:slug/troubleshoot/analyze/:bundleSlug/redactor/report`;

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
                <div className="SupportBundleTabs--wrapper flex1 flex-column">
                  <div className="tab-items flex">
                    <Link to={`/app/${watch.slug}/troubleshoot/analyze/${bundle.slug}`} className={`${this.state.activeTab === "bundleAnalysis" ? "is-active" : ""} tab-item blue`} onClick={() => this.toggleAnalysisAction("bundleAnalysis")}>Analysis overview</Link>
                    <Link to={`/app/${watch.slug}/troubleshoot/analyze/${bundle.slug}/contents/`} className={`${this.state.activeTab === "fileTree" ? "is-active" : ""} tab-item blue`} onClick={() => this.toggleAnalysisAction("fileTree")}>File inspector</Link>
                    <Link to={`/app/${watch.slug}/troubleshoot/analyze/${bundle.slug}/redactor/report`} className={`${this.state.activeTab === "redactorReport" ? "is-active" : ""} tab-item blue`} onClick={() => this.toggleAnalysisAction("redactorReport")}>Redactor report</Link>
                  </div>
                  <div className="flex-column flex1 action-content">
                    <Switch>
                      <Route exact path={insightsUrl} render={() =>
                        <AnalyzerInsights
                          status={bundle.status}
                          refetchSupportBundle={this.props.getSupportBundle.refetch}
                          insights={bundle.analysis?.insights}
                        />
                      } />
                      <Route exact path={fileTreeUrl} render={() =>
                        <AnalyzerFileTree
                          watchSlug={watch.slug}
                          bundle={bundle}
                          downloadBundle={() => this.downloadBundle(bundle)}
                        />
                      } />
                      <Route exact path={redactorUrl} render={() =>
                        <AnalyzerRedactorReport
                          watchSlug={watch.slug}
                          bundle={bundle}
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
