import * as React from "react";
import Loader from "../shared/Loader";
import isEmpty from "lodash/isEmpty";
import filter from "lodash/filter";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import { sortAnalyzers, parseIconUri, Utilities } from "../../utilities/utilities";

export class AnalyzerInsights extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      insights: [],
      analyzing: false,
      filterTiles: "0",
    };
  }

  componentDidUpdate(lastProps) {
    let isError, isWarn;
    if (this.props.insights !== lastProps.insights && this.props.insights) {
      isError = this.props.insights.some(i => i.severity === "error");
      isWarn = this.props.insights.some(i => i.severity === "warn");
      this.setState({
        insights: sortAnalyzers(this.props.insights),
      });
      if (isWarn || isError) {
        const insights = filter(this.props.insights, (i) => { return i.severity !== "debug" && i.severity !== "info" });
        this.setState({ filterTiles: "1", insights: insights })
      }
    }

    if (this.props.insights) {
      clearInterval(this.interval);
    }
  }

  componentDidMount() {
    this.testApi();
    let isError, isWarn;
    if (this.props.insights) {
      isError = this.props.insights.some(i => i.severity === "error");
      isWarn = this.props.insights.some(i => i.severity === "warn");
      this.setState({
        insights: sortAnalyzers(this.props.insights),
      });
      if (isError || isWarn) {
        const insights = filter(this.props.insights, (i) => { return i.severity !== "debug" && i.severity !== "info" });
        this.setState({ filterTiles: "1", insights: insights })
      }
    }

    this.checkBundleStatus();
  }

  testApi = async () => {
    this.setState({ sendingBundle: true, sendingBundleErrMsg: "", downloadBundleErrMsg: "" });
      fetch(`${window.env.API_ENDPOINT}/troubleshoot/app/qakots/supportbundle/2041y5f3xzi5ewauoaiqogyccme/pod?podNamespace=default&podName=sqs-7449b544fc-mw4dx`, {
        method: "GET",
        headers: {
          "Authorization": Utilities.getToken(),
        }
      })
        .then(async (result) => {
          console.log(result)
        })
        .catch(err => {
          console.log(err);
        })
  }

  checkBundleStatus = () => {
    const { status, refetchSupportBundle, insights } = this.props;
    if (status === "uploaded" || status === "analyzing") {
      // Check if the bundle is ready only if the user is on the page
      if (!insights) {
        this.interval = setInterval(refetchSupportBundle, 2000);
      } else {
        clearInterval(this.interval);
      }
    }
  }

  componentWillUnmount() {
    clearInterval(this.interval);
  }

  handleFilterTiles = (field, e) => {
    let nextState = {};
    const val = e.target.checked ? "1" : "0";
    nextState[field] = val;
    let insights = sortAnalyzers(this.props.insights);
    if (val === "1") {
      insights = filter(insights, (i) => { return i.severity !== "debug" && i.severity !== "info" });
    }
    this.setState({
      ...nextState,
      insights
    });
  }

  render() {
    const { insights, status } = this.props;
    const { filterTiles, analyzing } = this.state;
    const filteredInsights = this.state.insights;

    let noInsightsNode;
    if (isEmpty(insights)) {
      if (status === "uploaded" || status === "analyzing") {
        noInsightsNode = (
          <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-textColor--bodyCopy">
            <Loader size="40" />
            <p className="u-textColor--primary u-fontSize--normal u-fontWeight--bold">We are still analyzing this Support Bundle</p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">This can tak up to a minute, you can refresh the page to see if your analysis is ready.</p>
          </div>
        )
      } else {
        noInsightsNode = (
          <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-textColor--bodyCopy">
            <p className="u-textColor--primary u-fontSize--normal u-fontWeight--bold">We were unable to surface any insights for this Support Bundle</p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">It's possible that the file that was uploaded was not a Replicated Support Bundle,<br />or that collection of OS or Docker stats was not enabled in your spec.</p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">We're adding new bundle analyzers all the time, so check back soon.</p>
          </div>
        )
      }
    }

    return (
      <div className="flex flex1">
        {isEmpty(insights)
          ? noInsightsNode :
          <div className="flex-column flex1">
            <div className="flex-auto">
              <div className="u-position--relative flex u-marginBottom--20 u-paddingLeft--10 u-paddingRight--10">
                <input
                  type="checkbox"
                  className="filter-tiles-checkbox"
                  id="filterTiles"
                  checked={filterTiles === "1"}
                  value={filterTiles}
                  onChange={(e) => { this.handleFilterTiles("filterTiles", e) }}
                />
                <label htmlFor="filterTiles" className="flex1 u-width--full u-position--relative u-marginLeft--5 u-cursor--pointer">
                  <div className="flex-column">
                    <span className="u-fontWeight--medium u-textColor--primary u-fontSize--normal u-marginBottom--5 u-lineHeight--normal u-userSelect--none">Only show errors and warnings</span>
                    <span className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--normal u-lineHeight--normal u-userSelect--none">By default we show you everything that was analyzed but you can choose to see only errors and warnings.</span>
                  </div>
                </label>
              </div>
            </div>
            {isEmpty(filteredInsights) ?
              <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-textColor--bodyCopy">
                <p className="u-textColor--primary u-fontSize--normal u-fontWeight--bold">There were no errors or warnings found.</p>
                <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">Turn off "Only show errors and warnings" to see informational analyzers that we were able to surface.</p>
              </div>
              :
              <div className="flex-column flex1 u-overflow--auto">
                  <div className="flex flex-auto flexWrap--wrap">
                  {filteredInsights && filteredInsights.map((tile, i) => {
                    let iconObj;
                    if (tile.icon) {
                      iconObj = parseIconUri(tile.icon);
                    }
                    return (
                      <div key={i} className="insight-tile-wrapper flex-column">
                        <div className={`insight-tile flex-auto u-textAlign--center flex-verticalCenter flex-column ${tile.severity}`}>
                          <div className="flex justifyContent--center u-marginBottom--10">
                            {tile.icon ?
                              <span className="tile-icon" style={{ backgroundImage: `url(${iconObj.uri})`, width: `${iconObj.dimensions?.w}px`, height: `${iconObj.dimensions?.h}px` }}></span>
                              : tile.icon_key ?
                                <span className={`icon analysis-${tile.icon_key} tile-icon`}></span>
                                :
                                <span className={`icon analysis tile-icon`}></span>
                            }
                          </div>
                          <p className={tile.severity === "debug" ? "u-textColor--bodyCopy u-fontSize--normal u-fontWeight--bold" : "u-textColor--primary u-fontSize--normal u-fontWeight--bold"}>{tile.primary}</p>
                          <MarkdownRenderer id={`markdown-wrapper-${i}`} className={tile.severity === "debug" ? "u-textColor--bodyCopy u-fontSize--smaller u-fontWeight--medium u-marginTop--5" : "u-textColor--accent u-fontSize--smaller u-fontWeight--medium u-marginTop--5"}>
                            {tile.detail}
                          </MarkdownRenderer>
                          {tile?.involvedObject?.kind === "Pod" && <div><span className="replicated-link u-fontSize--small u-marginTop--5" onClick={() => this.props.openPodDetailsModal(tile?.involvedObject)}>See details</span></div>}
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            }
          </div>
        }
      </div>
    );
  }
}

export default AnalyzerInsights;
