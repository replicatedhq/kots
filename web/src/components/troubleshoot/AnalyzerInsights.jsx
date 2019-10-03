import * as React from "react";
import Loader from "../shared/Loader";
import isEmpty from "lodash/isEmpty";
import filter from "lodash/filter";
import sortBy from "lodash/sortBy";
import { sortAnalyzers } from "../../utilities/utilities";

export class AnalyzerInsights extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      insights: [],
      analyzing: false,
      filterTiles: "0",
      hasAnalysisError: false
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
    let insights;
    if (val === "1") {
      insights = filter(this.props.insights, (i) => { return i.severity !== "debug" && i.severity !== "info" });
    } else {
      insights = sortBy(this.props.insights, (item) => {
        if (item.severity === "error") {
          return 1
        }
        if (item.severity === "warn") {
          return 2
        }
        if (item.severity === "info") {
          return 3
        }
        if (item.severity === "debug") {
          return 4
        }
      })
    }
    this.setState({
      ...nextState,
      insights
    });
  }

  reAnalyzeBundle = () => {
    this.setState({ analyzing: true });
    this.props.reAnalyzeBundle((_, hasAnalysisError) => {
      this.setState({ analyzing: false, hasAnalysisError });
    });
  }

  renderAnalysisError = () => {
    return <span style={{ maxWidth: 420 }} className="u-fontSize--small u-fontWeight--bold u-color--red u-marginTop--20 u-textAlign--center">An error occured during analysis</span>;
  }

  render() {
    const { insights, status } = this.props;
    const { filterTiles, analyzing, hasAnalysisError } = this.state;
    const filteredInsights = this.state.insights;

    let noInsightsNode;
    if (isEmpty(insights)) {
      if (status === "uploaded" || status === "analyzing") {
        noInsightsNode = (
          <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-color--dustyGray">
            <Loader size="40" color="#44bb66" />
            <p className="u-color--tuna u-fontSize--normal u-fontWeight--bold">We are still analyzing this Support Bundle</p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">This can tak up to a minute, you can refresh the page to see if your analysis is ready.</p>
            {hasAnalysisError && this.renderAnalysisError()}
          </div>
        )
      } else {
        noInsightsNode = (
          <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-color--dustyGray">
            <p className="u-color--tuna u-fontSize--normal u-fontWeight--bold">We were unable to surface any insights for this Support Bundle</p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">It's possible that the file that was uploaded was not a Replicated Support Bundle,<br />or that collection of OS or Docker stats was not enabled in your spec.</p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">We're adding new bundle analyzers all the time, so check back soon.</p>
            <div className="u-marginTop--20">
              <button className="btn secondary" onClick={() => this.reAnalyzeBundle()} disabled={analyzing}>{analyzing ? "Re-analyzing" : "Re-analyze bundle"}</button>
            </div>
            {hasAnalysisError && this.renderAnalysisError()}
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
                    <span className="u-fontWeight--medium u-color--tuna u-fontSize--normal u-marginBottom--5 u-lineHeight--normal u-userSelect--none">Only show errors and warnings</span>
                    <span className="u-fontSize--small u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-userSelect--none">By default we show you everything that was analyzed but you can choose to see only errors and warnings.</span>
                  </div>
                </label>
              </div>
            </div>
            {isEmpty(filteredInsights) ?
              <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-color--dustyGray">
                <p className="u-color--tuna u-fontSize--normal u-fontWeight--bold">There were no errors or warnings found.</p>
                <p className="u-fontSize--small u-fontWeight--regular u-marginTop--10">Turn off "Only show errors and warnings" to see informational analyzers that we were able to surface.</p>
              </div>
              :
              <div className="flex-column flex1 u-overflow--auto">
                  <div className="flex flex-auto flexWrap--wrap">
                  {filteredInsights && filteredInsights.map((tile, i) => (
                    <div key={i} className="insight-tile-wrapper flex-column">
                      <div className={`insight-tile flex-auto u-textAlign--center flex-verticalCenter flex-column ${tile.severity}`}>
                        <div className="flex justifyContent--center u-marginBottom--10">
                          {tile.icon_key ?
                            <span className={`icon analysis-${tile.icon_key} tile-icon`}></span>
                            : tile.icon ?
                              <span className="tile-icon" style={{ backgroundImage: `url(${tile.icon})` }}></span>
                              :
                              <span className={`icon analysis tile-icon`}></span>
                          }
                        </div>
                        <p className={tile.severity === "debug" ? "u-color--dustyGray u-fontSize--smaller u-fontWeight--medium" : "u-color--doveGray u-fontSize--smaller u-fontWeight--medium"}>{tile.detail}</p>
                        <p className={tile.severity === "debug" ? "u-color--dustyGray u-fontSize--normal u-fontWeight--bold u-marginTop--5" : "u-color--tuna u-fontSize--normal u-fontWeight--bold u-marginTop--5"}>{tile.primary}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            }
            <div className="flex-column flex-auto u-paddingTop--20">
              <div className="flex-auto u-paddingLeft--10">
                <button className="btn secondary" onClick={() => this.reAnalyzeBundle()} disabled={analyzing}>{analyzing ? "Re-analyzing" : "Re-analyze bundle"}</button>
              </div>
              {hasAnalysisError && this.renderAnalysisError()}
            </div>
          </div>
        }
      </div>
    );
  }
}

export default AnalyzerInsights;
