import * as React from "react";
import { withRouter } from "react-router-dom";
import "../scss/components/Login.scss";
import { Utilities } from "../utilities/utilities";
import TrackSCMLeads from "./TrackSCMLeads";

class Login extends React.Component {
  constructor(props) {
    super(props);
  }

  componentDidMount() {
    if (!Utilities.localStorageEnabled()) {
      this.props.history.push("/unsupported") 
    }

    const { search } = this.props.location;
    const URLParams = new URLSearchParams(search);
    if (URLParams.get("next") && Utilities.localStorageEnabled()) {
      localStorage.setItem("next", URLParams.get("next"));
    }
    if (Utilities.getToken()) {
      const next = URLParams.get("next");
      if (next) {
        const decodedNext = decodeURI(next);
        this.props.history.push(decodedNext);
      } else {
        this.props.history.push("/watches");
      }
    }
  }

  handleLogIn = () => {
    this.props.history.push("/auth/github");
  }

  render() {
    const showSCM = window.env.SHOW_SCM_LEADS;
    const scmLeadsStyle = showSCM ? { width: "100%", maxWidth: "960px"} : {};
    return (
      <div className="container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto" style={scmLeadsStyle}>
          <div className="flex-auto flex-column login-form-wrapper justifyContent--center">
            <div className="flex">
              <span className="icon ship-login-icon"></span>
              <p className="login-text u-color--tuna u-fontWeight--bold">Log in</p>
            </div>
            <p className="u-lineHeight--normal u-fontSize--larger u-color--tuna u-fontWeight--bold u-marginBottom--20">Connect your GitHub account to get started using Replicated Ship</p>
            <button type="button" className="btn auth github" onClick={this.handleLogIn}>
              <span className="icon clickable github-button-icon"></span> Login with GitHub
            </button>
          </div>
          {showSCM &&
            <TrackSCMLeads />
          }
        </div>
      </div>
    );
  }
}

export default withRouter(Login);
