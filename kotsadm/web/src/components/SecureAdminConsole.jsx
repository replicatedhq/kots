import * as React from "react";
import Helmet from "react-helmet";
import { Utilities, dynamicallyResizeText } from "../utilities/utilities";
import Loader from "./shared/Loader";
import "../scss/components/Login.scss";

class SecureAdminConsole extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      password: "",
      passwordErr: false,
      passwordErrMessage: "",
      authLoading: false,
      loginInfo: null,
    }

    this.loginText = React.createRef();
  }

  completeLogin = (data) => {
    let token = data.token;
    if (Utilities.localStorageEnabled()) {
      window.localStorage.setItem("token", token);
      setTimeout(() => {
        Utilities.removeCookie("token");
      }, 5000); // was removing the local storage token without a delay for some reason

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

  loginWithSharedPassword = async () => {
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

  loginWithIdentityProvider = async () => {
    try {
      this.setState({ passwordErr: false, passwordErrMessage: "" });

      const res = await fetch(`${window.env.API_ENDPOINT}/oidc/login`, {
        headers: {
          "Content-Type": "application/json",
        },
        method: "GET",
      });

      if (res.status >= 400) {
        const body = await res.json();
        let msg = body.error;
        if (!msg) {
          msg = "There was an error logging in. Please try again.";
        }
        this.setState({
          passwordErr: true,
          passwordErrMessage: msg,
        });
        return;
      }

      const body = await res.json();
      window.location = body.authCodeURL
    } catch(err) {
      console.log("Login failed:", err);
      this.setState({
        passwordErr: true,
        passwordErrMessage: "There was an error logging in. Please try again",
      });
    }
  }

  submitForm = (e) => {
    const enterKey = e.keyCode === 13;
    if (enterKey) {
      e.preventDefault();
      e.stopPropagation();
      this.loginWithSharedPassword();
    }
  }

  resizeLoginFont = () => {
    if (!this.loginText?.current) {
      return;
    }
    const newFontSize = dynamicallyResizeText(this.loginText.current.innerHTML, this.loginText.current.clientWidth, "32px");
    this.loginText.current.style.fontSize = newFontSize;
  }

  getLoginInfo = async () => {
    try {
      const response = await fetch(`${window.env.API_ENDPOINT}/login/info`, {
        headers: {
          "Content-Type": "application/json",
        },
        method: "GET",
      });
  
      if (!response.ok) {
        console.log(`Unexpected status code ${response.status}`)
        return;
      }

      const res = await response.json();
      this.setState({ loginInfo: res });
    } catch(err) {
      console.log(err);
    }
  }

  componentDidUpdate(lastProps) {
    const { appName } = this.props;
    if (appName && appName !== lastProps.appName) {
      if (this.loginText) {
        this.resizeLoginFont();
      }
    }
  }

  async componentWillMount() {
    const token = Utilities.getCookie("token");
    if (token) {
      // this is a redirect from identity service login
      // strip quotes from token (golang adds them when the cookie value has spaces, commas, etc..)
      const loginData = {
        token: token.replace(/"/g, ""),
      };
      this.completeLogin(loginData);
      return;
    }

    await this.getLoginInfo();
  }

  componentDidMount() {
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
      loginInfo,
    } = this.state;

    if (fetchingMetadata || !loginInfo) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

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
            {loginInfo?.method === "identity-service" ?
              <button type="submit" className="btn primary u-marginTop--20" onClick={this.loginWithIdentityProvider}>{`Log in with ${loginInfo?.identityConnector}`}</button>
              : 
              <div className="flex-auto flex-column justifyContent--center">
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
                      <button type="submit" className="btn primary" disabled={authLoading} onClick={this.loginWithSharedPassword}>{authLoading ? "Logging in" : "Log in"}</button>
                    </div>
                  </div>
                </div>
              </div>
            }
          </div>
        </div>
      </div>
    );
  }
}

export default SecureAdminConsole;
