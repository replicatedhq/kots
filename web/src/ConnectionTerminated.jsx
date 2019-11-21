import * as React from "react";
import { ping } from "@src/queries/AppsQueries";
import Countdown from "react-countdown-now";

export default class ConnectionTerminated extends React.Component {

  state = {
    seconds: 1,
    reconnectAttepts: 0,
  }

  tick = async () => {
    if (!this.props.connectionTerminated) {
      clearInterval(this.timer);
      return;
    }
    if (this.state.seconds > 0) {
      this.setState({ seconds: this.state.seconds - 1 });
    } else {
      this.setState({
        seconds: this.state.reconnectAttepts >= 9 ? 9 : (this.state.reconnectAttepts + 1),
        reconnectAttepts: this.state.reconnectAttepts + 1
      });
    }
  }

  ping = async () => {
    this.timer = setInterval(this.tick, 1000);
    await this.props.gqlClient.query({
      query: ping,
      fetchPolicy: "no-cache"
    }).then(() => {
      this.props.setTerminatedState(false);
    }).catch(() => {
      this.props.setTerminatedState(true);
    });
  }

  componentDidMount = async () => {
    this.ping();
  }

  render() {
    return (
      <div className="ConnectionTerminated--wrapper u-textAlign--center">
        <div className="flex u-marginTop--30 u-marginBottom--10 justifyContent--center">
          <span className="icon no-connection-icon" />
          {this.state.appLogo
            ? <img width="60" height="60" className="u-marginLeft--10" src={this.state.appLogo} />
            : <span className="icon onlyAirgapBundleIcon u-marginLeft--10" />
          }
        </div>
        <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal u-userSelect--none">Cannot connect</h2>
        <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--more u-marginTop--10 u-marginBottom--10 u-userSelect--none">We're unable to reach the API right now. Check to make sure your local server is running.</p>
        <div className="u-marginBottom--30">
          <span className="u-fontSize--normal u-fontWeight--bold u-color--tundora u-userSelect--none">Trying again in <Countdown date={Date.now() + this.state.seconds * 1000} /></span>
        </div>
      </div>
    );
  }
}
