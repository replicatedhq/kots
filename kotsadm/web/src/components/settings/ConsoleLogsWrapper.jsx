import React, { Component } from "react";
import { withRouter, Switch, Route } from "react-router-dom";

class ConsoleLogsWrapper extends Component {
  render() {
    return (
      <div className="u-paddingLeft--20">
        <div>
          <div className="toggle-wrapper">
            <span className="toggle-item" onClick={() => this.props.history.push("/settings/logs/view")}>View logs</span>
            <span className="toggle-item" onClick={() => this.props.history.push("/settings/logs/configure")}>Configure Fluentd</span>
          </div>
        </div>
        <Switch>
          <Route exact path="/settings/logs/view" render={(props) => <div>View logs</div> }/>
          <Route exact path="/settings/logs/configure" render={(props) => <div>Configure Fluentd</div>} />
        </Switch>
      </div>
    )
  }
}

export default withRouter(ConsoleLogsWrapper);