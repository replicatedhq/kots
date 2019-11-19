import * as React from "react";
import { withRouter } from "react-router-dom";
import "../scss/components/Login.scss";
import { Utilities } from "../utilities/utilities";
import TraditionalAuth from "./TraditionalAuth";

class Signup extends React.Component {
  
  state = {
    traditionalAuth: false
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
        localStorage.removeItem("next");
        const decodedNext = decodeURI(next);
        this.props.history.push(decodedNext);
      } else {
        this.props.history.push("/apps");
      }
    }
  }

  render() {
    return (
      <div className="container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper justifyContent--center">
            <div className="flex-column">
              <span className="icon kots-login-icon"></span>
              <p className="login-text u-color--tuna u-fontWeight--bold">Sign up</p>
            </div>
            <button type="button" className={`btn auth traditional u-marginTop--20`} onClick={() => this.props.history.push("/login")}>
              <span className="icon clickable backArrow-icon" style={{ verticalAlign: "0" }}></span> Use a different auth type
            </button>
          </div>
          <TraditionalAuth context="signup" />
        </div>
      </div>
    );
  }
}

export default withRouter(Signup);
