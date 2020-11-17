import * as React from "react";
import Helmet from "react-helmet";
import { Utilities, dynamicallyResizeText } from "../utilities/utilities";
import "../scss/components/Login.scss";

class SecureAdminConsole extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      password: "",
      passwordErr: false,
      passwordErrMessage: "",
      authLoading: false,
    }

    this.loginText = React.createRef();
  }

  completeLogin = (data) => {
    let token = data.token;
    if (Utilities.localStorageEnabled()) {
      window.localStorage.setItem("token", token);
      this.props.onLoginSuccess().then((res) => {
        this.setState({ authLoading: false });
        if (res.length > 0) {
          this.props.history.replace(`/app/${res[0].slug}`);
        } else {
          this.props.history.replace("upload-license");
        }
      });
    } else {
      this.props.history.push("/unsupported");
    }
  }

  validatePassword = () => {
    if (!this.state.password || this.state.password.length === "0") {
      this.setState({
        passwordErr: true,
        passwordErrMessage: `Please provide your password`,
      });
      return false;
    }
    return true;
  }

  loginToConsole = async () => {
    if (this.validatePassword()) {
      this.setState({ authLoading: true, passwordErr: false, passwordErrMessage: "" });
      fetch(`${window.env.API_ENDPOINT}/login`, {
        headers: {
          "Content-Type": "application/json",
        },
        method: "POST",
        body: JSON.stringify({
          password: this.state.password,
        })
      })
      .then(async (res) => {
        if (res.status >= 400) {
          let body = await res.json();
          let msg = body.error;
          if (!msg) {
            msg = res.status === 401 ? "Invalid password. Please try again" : "There was an error logging in. Please try again.";
          }
          this.setState({
            authLoading: false,
            passwordErr: true,
            passwordErrMessage: msg,
          });
          return;
        }
        this.completeLogin(await res.json());
      })
      .catch((err) => {
        console.log("Login failed:", err);
        this.setState({
          authLoading: false,
          passwordErr: true,
          passwordErrMessage: "There was an error logging in. Please try again",
        });
      });
    }
  }

  submitForm = (e) => {
    const enterKey = e.keyCode === 13;
    if (enterKey) {
      e.preventDefault();
      e.stopPropagation();
      this.loginToConsole();
    }
  }

  resizeLoginFont = () => {
    const newFontSize = dynamicallyResizeText(this.loginText.current.innerHTML, this.loginText.current.clientWidth, "32px");
    this.loginText.current.style.fontSize = newFontSize;
  }

  componentDidUpdate(lastProps) {
    const { appName } = this.props;
    if (appName && appName !== lastProps.appName) {
      if (this.loginText) {
        this.resizeLoginFont();
      }
    }
  }

  componentDidMount() {
    fetch(`${window.env.API_ENDPOINT}/oidc/login`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "POST",
      body: JSON.stringify({
      }),
      redirect: "follow",
    }).then(response => {
      console.log(response)
      if (response.status != 303) {
        return
      }
      // console.log(response)
      // const redirectUrl = response.headers["location"]
      // window.location.href = redirectUrl;
    }).catch(err => {
      console.log("hello?", err)
    });
    window.addEventListener("keydown", this.submitForm);
  }

  componentWillUnmount() {
    window.removeEventListener("keydown", this.submitForm);
  }

  render() {
    const {
      appName,
      logo,
      fetchingMetadata,
    } = this.props;
    const {
      password,
      authLoading,
      passwordErr,
      passwordErrMessage,
    } = this.state;

    if (fetchingMetadata) { return null; }

    return (
      <div className="container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <Helmet>
          <title>{`${appName ? `${appName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex-column alignItems--center">
              {logo
              ? <span className="icon brand-login-icon" style={{ backgroundImage: `url(${logo})` }} />
              : !fetchingMetadata ? <span className="icon kots-login-icon" />
              : <span style={{ width: "60px", height: "60px" }} />
              }
              <p ref={this.loginText} style={{ fontSize: "32px" }} className="u-marginTop--10 u-paddingTop--5 u-lineHeight--more u-color--tuna u-fontWeight--bold u-width--full u-textAlign--center">Log in{appName && appName !== "" ? ` to ${appName}` : ""}</p>
            </div>
            <p className="u-marginTop--10 u-marginTop--5 u-fontSize--large u-textAlign--center u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">
              Enter the password to access the {appName} admin console.
            </p>
            <div className="u-marginTop--20 flex-column">
              {passwordErr && <p className="u-fontSize--normal u-fontWeight--medium u-color--chestnut u-lineHeight--normal u-marginBottom--20">{passwordErrMessage}</p>}
              <div>
                <div className="component-wrapper">
                  <input type="password" className="Input" placeholder="password" autoComplete="current-password" value={password} onChange={(e) => { this.setState({ password: e.target.value }) }}/>
                </div>
                <div className="u-marginTop--20 flex">
                  <button type="submit" className="btn primary" disabled={authLoading} onClick={this.loginToConsole}>{authLoading ? "Logging in" : "Log in"}</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default SecureAdminConsole;
