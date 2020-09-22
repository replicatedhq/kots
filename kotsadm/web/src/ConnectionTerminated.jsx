import * as React from "react";
import { Utilities } from "./utilities/utilities";
import fetch from "./utilities/fetchWithTimeout";

export default class ConnectionTerminated extends React.Component {

  state = {
    seconds: 1,
    reconnectAttempts: 1,
  }

  countdown = (seconds) => {
    this.setState({ seconds });
    if (seconds === 0) {
      this.setState({
        reconnectAttempts: this.state.reconnectAttempts + 1
      }, () => {
        this.ping();
      });
    } else {
      const nextCount = seconds - 1;
      setTimeout(() => {
        this.countdown(nextCount);
      }, 1000);
    }
  }

  ping = async () => {
    const { reconnectAttempts } = this.state;
    await fetch(`${window.env.API_ENDPOINT}/ping`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
    }, 10000).then(async (res) => {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      this.props.setTerminatedState(false);
    }).catch(() => {
      this.props.setTerminatedState(true);
      const seconds = reconnectAttempts > 10 ? 10 : reconnectAttempts;
      this.countdown(seconds);
    });
  }

  componentDidMount = () => {
    this.ping();
  }

  render() {
    const { seconds } = this.state;
    return (
      <div className="Modal-body u-textAlign--center">
        <div className="flex u-marginTop--30 u-marginBottom--10 justifyContent--center">
          <span className="icon no-connection-icon" />
          {this.props.appLogo
            ? <img width="60" height="60" className="u-marginLeft--10" src={this.props.appLogo} />
            : <span className="icon onlyAirgapBundleIcon u-marginLeft--10" />
          }
        </div>
        <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal u-userSelect--none">Cannot connect</h2>
        <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--more u-marginTop--10 u-marginBottom--10 u-userSelect--none">We're unable to reach the API right now. If you are using port-forwarding, check that the forwarding process is still active.</p>
        <div className="u-marginBottom--30">
          <span className="u-fontSize--normal u-fontWeight--bold u-color--tundora u-userSelect--none">Trying again in {`${seconds} second${seconds !== 1 ? "s" : ""}`}</span>
        </div>
      </div>
    );
  }
}
