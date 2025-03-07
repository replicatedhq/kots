import { Component } from "react";
import Loader from "../shared/Loader";
import isEmpty from "lodash/isEmpty";
import filter from "lodash/filter";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import { sortAnalyzers, parseIconUri } from "../../utilities/utilities";
import { withRouter } from "@src/utilities/react-router-utilities";

export class AnalyzerInsights extends Component {
  constructor(props) {
    super(props);
    this.state = {
      insights: [],
      analyzing: false,
      filterTiles: false,
    };
  }

  componentDidUpdate(lastProps) {
    if (
      this.props.outletContext.insights !== lastProps.outletContext.insights &&
      this.props.outletContext.insights
    ) {
      const hasProblems = this.props.outletContext.insights.some(
        (i) => i.severity === "warn" || i.severity === "error"
      );
      this.handleFilterTiles(hasProblems);
    }
    if (this.props.outletContext.insights) {
      clearInterval(this.interval);
    }
  }

  componentDidMount() {
    if (this.props.outletContext.insights) {
      const hasProblems = this.props.outletContext.insights.some(
        (i) => i.severity === "warn" || i.severity === "error"
      );
      this.handleFilterTiles(hasProblems);
    }

    this.checkBundleStatus();
  }

  checkBundleStatus = () => {
    const { status, refetchSupportBundle, insights } = this.props.outletContext;
    if (status === "uploaded" || status === "analyzing") {
      // Check if the bundle is ready only if the user is on the page
      if (!insights) {
        this.interval = setInterval(refetchSupportBundle, 2000);
      } else {
        clearInterval(this.interval);
      }
    }
  };

  componentWillUnmount() {
    clearInterval(this.interval);
  }

  handleFilterTiles = (checked) => {
    let insights = sortAnalyzers(this.props.outletContext.insights);
    if (checked) {
      insights = filter(
        insights,
        (i) => i.severity === "error" || i.severity === "warn"
      );
    }
    this.setState({
      filterTiles: checked,
      insights,
    });
  };

  render() {
    const { insights, status } = this.props.outletContext;
    const { filterTiles } = this.state;
    const filteredInsights = this.state.insights;

    let noInsightsNode;
    if (isEmpty(insights)) {
      if (status === "uploaded" || status === "analyzing") {
        noInsightsNode = (
          <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-textColor--bodyCopy">
            <Loader size="40" />
            <p className="u-textColor--primary u-fontSize--normal u-fontWeight--bold">
              We are still analyzing this Support Bundle
            </p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">
              This can tak up to a minute, you can refresh the page to see if
              your analysis is ready.
            </p>
          </div>
        );
      } else {
        noInsightsNode = (
          <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-textColor--bodyCopy">
            <p className="u-textColor--primary u-fontSize--normal u-fontWeight--bold">
              We were unable to surface any insights for this Support Bundle
            </p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">
              It's possible that the file that was uploaded was not a Replicated
              Support Bundle,
              <br />
              or that collection of OS or Docker stats was not enabled in your
              spec.
            </p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">
              We're adding new bundle analyzers all the time, so check back
              soon.
            </p>
          </div>
        );
      }
    }

    return (
      <div
        className="flex flex1 u-width--full"
        data-testid="support-bundle-analysis-bundle-insights"
      >
        {isEmpty(insights) ? (
          noInsightsNode
        ) : (
          <div className="flex-column u-width--full card-bg">
            <div className="u-marginBottom--20 u-paddingLeft--10 u-paddingRight--10">
              <label
                htmlFor="filterTiles"
                className="flex alignItems--center u-fontWeight--medium u-textColor--primary u-fontSize--normal u-lineHeight--normal u-userSelect--none"
              >
                <input
                  type="checkbox"
                  className="filter-tiles-checkbox"
                  id="filterTiles"
                  checked={filterTiles}
                  onChange={(e) => {
                    this.handleFilterTiles(e.target.checked);
                  }}
                  data-testid="support-bundle-analysis-bundle-insights-filter-tiles-checkbox"
                />
                Only show errors and warnings
              </label>
              <span className="u-fontSize--small u-fontWeight--medium u-marginLeft--20">
                By default we show you everything that was analyzed but you can
                choose to see only errors and warnings.
              </span>
            </div>

            {isEmpty(filteredInsights) ? (
              <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-textColor--bodyCopy">
                <p className="u-textColor--primary u-fontSize--normal u-fontWeight--bold">
                  There were no errors or warnings found.
                </p>
                <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">
                  Turn off "Only show errors and warnings" to see informational
                  analyzers that we were able to surface.
                </p>
              </div>
            ) : (
              <div className="flex-column flex1 u-overflow--auto">
                <div className="flex flex-auto flexWrap--wrap">
                  {filteredInsights &&
                    filteredInsights.map((tile, i) => {
                      let iconObj;
                      if (tile.icon) {
                        iconObj = parseIconUri(tile.icon);
                      }
                      return (
                        <div
                          key={i}
                          className="insight-tile-wrapper flex-column"
                        >
                          <div
                            className={`insight-tile flex-auto u-textAlign--center flex-verticalCenter flex-column ${tile.severity}`}
                          >
                            <div className="flex justifyContent--center u-marginBottom--10">
                              {tile.icon ? (
                                <span
                                  className="tile-icon"
                                  style={{
                                    backgroundImage: `url(${iconObj.uri})`,
                                    width: `${iconObj.dimensions?.w}px`,
                                    height: `${iconObj.dimensions?.h}px`,
                                  }}
                                ></span>
                              ) : tile.icon_key ? (
                                <span
                                  className={`icon analysis-${tile.icon_key} tile-icon`}
                                ></span>
                              ) : (
                                <span
                                  className={`icon analysis tile-icon`}
                                ></span>
                              )}
                            </div>
                            <p
                              className={
                                tile.severity === "debug"
                                  ? "u-textColor--bodyCopy u-fontSize--normal u-fontWeight--bold"
                                  : "u-textColor--primary u-fontSize--normal u-fontWeight--bold"
                              }
                            >
                              {tile.primary}
                            </p>
                            <MarkdownRenderer
                              id={`markdown-wrapper-${i}`}
                              className={
                                tile.severity === "debug"
                                  ? "u-textColor--bodyCopy u-fontSize--smaller u-fontWeight--medium u-marginTop--5"
                                  : "u-textColor--accent u-fontSize--smaller u-fontWeight--medium u-marginTop--5"
                              }
                            >
                              {tile.detail}
                            </MarkdownRenderer>
                            {tile?.involvedObject?.kind === "Pod" && (
                              <div>
                                <span
                                  className="link u-fontSize--small u-marginTop--5"
                                  onClick={() =>
                                    this.props.outletContext.openPodDetailsModal(
                                      tile?.involvedObject
                                    )
                                  }
                                >
                                  See details
                                </span>
                              </div>
                            )}
                          </div>
                        </div>
                      );
                    })}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    );
  }
}

export default withRouter(AnalyzerInsights);
