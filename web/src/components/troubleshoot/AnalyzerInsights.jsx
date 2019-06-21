import * as React from "react";
import autoBind from "react-autobind";
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
      analysisError: false
    };
    autoBind(this);
  }

  componentDidUpdate(lastProps) {
    let isError, isWarn;
    if (this.props.insights !== lastProps.insights && this.props.insights) {
      isError = this.props.insights.some(i => i.level === "error");
      isWarn = this.props.insights.some(i => i.level === "warn");
      this.setState({
        insights: sortAnalyzers(this.props.insights),
      });
      if (isWarn || isError) {
        const insights = filter(this.props.insights, (i) => { return i.level !== "debug" && i.level !== "info" });
        this.setState({ filterTiles: "1", insights: insights })
      }
    }
  }

  componentDidMount() {
    let isError, isWarn;
    if (this.props.insights) {
      isError = this.props.insights.some(i => i.level === "error");
      isWarn = this.props.insights.some(i => i.level === "warn");
      this.setState({
        insights: sortAnalyzers(this.props.insights),
      });
      if (isError || isWarn) {
        const insights = filter(this.props.insights, (i) => { return i.level !== "debug" && i.level !== "info" });
        this.setState({ filterTiles: "1", insights: insights })
      }
    }
  }

  handleFilterTiles(field, e) {
    let nextState = {};
    const val = e.target.checked ? "1" : "0";
    nextState[field] = val;
    let insights;
    if (val === "1") {
      insights = filter(this.props.insights, (i) => { return i.level !== "debug" && i.level !== "info" });
    } else {
      insights = sortBy(this.props.insights, (item) => {
        if (item.level === "error") {
          return 1
        }
        if (item.level === "warn") {
          return 2
        }
        if (item.level === "info") {
          return 3
        }
        if (item.level === "debug") {
          return 4
        }
      })
    }
    this.setState({
      ...nextState,
      insights
    });
  }

  reAnalyzeBundle() {
    this.setState({ analyzing: true });
    this.props.reAnalyzeBundle((response, analysisError) => {
      this.setState({ analyzing: false, analysisError });
    });
  }

  render() {
    const { insights } = this.props;
    const { filterTiles, analyzing, analysisError } = this.state;
    const filteredInsights = this.state.insights;

    const analysisErrorExists = analysisError && analysisError.graphQLErrors && analysisError.graphQLErrors.length;

    return (
      <div className="flex flex1">
        {isEmpty(insights) ?
          <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-color--dustyGray">
            <p className="u-color--tuna u-fontSize--normal u-fontWeight--bold">We were unable to surface any insights for this Support Bundle</p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--normal">It's possible that the file that was uploaded was not a Replicated Support Bundle,<br />or that collection of OS or Docker stats was not enabled in your spec.</p>
            <p className="u-fontSize--small u-fontWeight--regular u-marginTop--normal">We're adding new bundle analyzers all the time, so check back soon.</p>
            <div className="u-marginTop--most">
              <button className="Button primary" onClick={() => this.reAnalyzeBundle()} disabled={analyzing}>{analyzing ? "Re-analyzing" : "Re-analyze bundle"}</button>
            </div>
            {analysisErrorExists && <span style={{ maxWidth: 420 }} className="u-fontSize--small u-fontWeight--bold u-color--error u-marginTop--more u-textAlign--center">{analysisError.graphQLErrors[0].message}</span>}
          </div>
          :
          <div className="flex-column flex1">
            <div className="flex-auto">
              <div className="u-position--relative flex u-marginBottom--more u-paddingLeft--10 u-paddingRight--10">
                <input
                  type="checkbox"
                  className=""
                  id="filterTiles"
                  checked={filterTiles === "1"}
                  value={filterTiles}
                  onChange={(e) => { this.handleFilterTiles("filterTiles", e) }}
                />
                <label htmlFor="filterTiles" className="flex1 u-width--full u-position--relative u-marginLeft--small u-cursor--pointer">
                  <div className="flex-column">
                    <span className="u-fontWeight--medium u-color--tuna u-fontSize--normal u-marginBottom--small u-lineHeight--normal u-userSelect--none">Only show errors and warnings</span>
                    <span className="u-fontSize--small u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-userSelect--none">By default we show you everything that was analyzed but you can choose to see only errors and warnings.</span>
                  </div>
                </label>
              </div>
            </div>
            {isEmpty(filteredInsights) ?
              <div className="flex-column flex1 justifyContent--center alignItems--center u-textAlign--center u-lineHeight--normal u-color--dustyGray">
                <p className="u-color--tuna u-fontSize--normal u-fontWeight--bold">There were no errors or warnings found.</p>
                <p className="u-fontSize--small u-fontWeight--regular u-marginTop--normal">Turn off "Only show errors and warnings" to see informational analyzers that we were able to surface.</p>
              </div>
              :
              <div className="flex flexWrap--wrap u-overflow--auto">
                {filteredInsights && filteredInsights.map((tile, i) => (
                  <div key={i} className="insight-tile-wrapper flex-column">
                    <div className={`insight-tile flex1 u-textAlign--center flex-verticalCenter flex-column ${tile.level}`}>
                      <div className="flex justifyContent--center u-marginBottom--normal">
                        {tile.icon_key ?
                          <span className={`icon analysis-${tile.icon_key} tile-icon`}></span>
                          : tile.icon ?
                            <span className="tile-icon" style={{ backgroundImage: `url(${tile.icon})` }}></span>
                            :
                            <span className={`icon analysis tile-icon`}></span>
                        }
                      </div>
                      <p className={tile.level === "debug" ? "u-color--dustyGray u-fontSize--smaller u-fontWeight--medium" : "u-color--doveGray u-fontSize--smaller u-fontWeight--medium"}>{tile.detail}</p>
                      <p className={tile.level === "debug" ? "u-color--dustyGray u-fontSize--normal u-fontWeight--bold u-marginTop--small" : "u-color--tuna u-fontSize--normal u-fontWeight--bold u-marginTop--small"}>{tile.primary}</p>
                    </div>
                  </div>
                ))}
              </div>
            }
            <div className="flex-column flex1">
              <div className="flex-auto u-paddingLeft--10">
                <button className="Button primary" onClick={() => this.reAnalyzeBundle()} disabled={analyzing}>{analyzing ? "Re-analyzing" : "Re-analyze bundle"}</button>
              </div>
              {analysisErrorExists && <span style={{ maxWidth: 420 }} className="u-fontSize--small u-fontWeight--bold u-color--error u-marginTop--more u-textAlign--center">{analysisError.graphQLErrors[0].message}</span>}
            </div>
          </div>
        }
      </div>
    );
  }
}

export default AnalyzerInsights;