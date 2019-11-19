import * as React from "react";
import { withRouter } from "react-router-dom";
import "../scss/components/Login.scss";
import Modal from "react-modal";
import { Helmet } from "react-helmet";
import { Utilities } from "../utilities/utilities";
import TrackSCMLeads from "./TrackSCMLeads";
import TraditionalAuth from "./TraditionalAuth";
import ForgotPasswordModal from "./shared/modals/ForgotPasswordModal";

class Login extends React.Component {

  state = {
    traditionalAuth: false,
    displayForgotPasswordModal: false
  }

  componentDidMount() {
    if (!Utilities.localStorageEnabled()) {
      this.props.history.push("/unsupported")
    }

    if (window.env.SECURE_ADMIN_CONSOLE) {
      return this.props.history.replace("/secure-console");
    }

    const { search } = this.props.location;
    const URLParams = new URLSearchParams(search);
    if (URLParams.get("next") && Utilities.localStorageEnabled()) {
      localStorage.setItem("next", URLParams.get("next"));
    }
    const traditional = URLParams.get("ta");
    if (traditional === "1") {
      this.setState({ traditionalAuth: true });
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

  handleLogIn = (type) => {
    if (type === "github") {
      this.props.history.push("/auth/github");
    } else if (type === "gitlab") {
      console.log("GitLab to be implemented");
    } else if (type === "bitbucket") {
      console.log("Bitbucket to be implemented");
    } else {
      this.setState({ traditionalAuth: true });
    }
  }

  render() {
    const { onLoginSuccess, appName } = this.props;
    const { traditionalAuth } = this.state;
    const showSCM = window.env.SHOW_SCM_LEADS;
    const allowedLogins = window.env.AVAILABLE_LOGIN_TYPES;
    const scmLeadsStyle = showSCM ? { width: "100%", maxWidth: "960px"} : {};
    return (
      <div className="container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <Helmet>
          <title>{`${appName ? `${appName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto" style={scmLeadsStyle}>
          <div className="flex-auto flex-column login-form-wrapper justifyContent--center">
            <div className="flex-column">
              <span className="icon kots-login-icon"></span>
              <p className="login-text u-color--tuna u-fontWeight--bold">Log in</p>
            </div>
            {traditionalAuth ?
              <button type="button" className={`btn auth traditional u-marginTop--20`} onClick={() => this.setState({ traditionalAuth: false })}>
                <span className="icon clickable backArrow-icon" style={{ verticalAlign: "0" }}></span> Use a different auth type
              </button>
            : allowedLogins && allowedLogins.map((type) => {
              const readableType = Utilities.getReadableLoginType(type);
              return (
                <button key={type} type="button" className={`btn auth ${type} u-marginTop--20`} onClick={() => this.handleLogIn(type)}>
                  <span className={`icon clickable ${type}-button-icon`}></span> {type === "traditional" ? "Use email & password" : `Login with ${readableType}`}
                </button>
              )
            })
            }
          </div>
          {traditionalAuth &&
            <TraditionalAuth
              onLoginSuccess={onLoginSuccess}
              context="login"
              handleForgotPasswordClick={() => this.setState({ displayForgotPasswordModal: true })}
            />
          }
          {showSCM && !traditionalAuth ?
            <TrackSCMLeads />
          : null}
        </div>
        {this.state.displayForgotPasswordModal &&
          <Modal
            isOpen={this.state.displayForgotPasswordModal}
            onRequestClose={() => this.setState({ displayForgotPasswordModal: false })}
            shouldReturnFocusAfterClose={false}
            contentLabel="Forgot password modal"
            ariaHideApp={false}
            className="ForgotPasswordModal--wrapper Modal DefaultSize"
          >
            <div className="Modal-body">
              <ForgotPasswordModal onRequestClose={() => this.setState({ displayForgotPasswordModal: false })} />
            </div>
          </Modal>
        }
      </div>
    );
  }
}

export default withRouter(Login);
