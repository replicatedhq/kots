import * as React from "react";
import { withRouter } from "react-router-dom";

export class PodAnalyzerDetails extends React.Component {

  state = {
    activeTab: "podDefinition"
  }
  
  togglePodDetailView = (active) => {
    this.setState({ activeTab: active });
  }

  renderPodDetailView = () => {
    switch (this.state.activeTab) {
    case "podDefinition":
      return (
        <p>render pod definition</p>
      )
    case "podLogs":
      return (
        <p>render pod logs</p>
      )
    case "podEvents":
      return (
        <p>render pod events</p>
      )
    default:
      return <div>nothing selected</div>
    }
  }

  render() {
    const { pod } = this.props;
    return (
        <div className="flex flex-column">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">Details for {pod.name}</p>
          <div className="SupportBundleTabs--wrapper flex-column flex1">
            <div className="flex tab-items">
              <span className={`${this.state.activeTab === "podDefinition" ? "is-active" : ""} tab-item blue`} onClick={() => this.togglePodDetailView("podDefinition")}>Pod definition</span>
              <span className={`${this.state.activeTab === "podLogs" ? "is-active" : ""} tab-item blue`} onClick={() => this.togglePodDetailView("podLogs")}>Pod logs</span>
              <span className={`${this.state.activeTab === "podEvents" ? "is-active" : ""} tab-item blue`} onClick={() => this.togglePodDetailView("podEvents")}>Pod events</span>
            </div>
            <div className="flex flex1 action-content">
              {this.renderPodDetailView()}
            </div>
        </div>
        </div>

    );
  }
}

export default withRouter(PodAnalyzerDetails);
