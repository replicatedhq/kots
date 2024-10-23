import { Component, RefObject, createRef } from "react";
import { KotsPageTitle } from "@components/Head";
import { Utilities, dynamicallyResizeText } from "@src/utilities/utilities";
import Loader from "@src/components/shared/Loader";
import ErrorModal from "@src/components/modals/ErrorModal";
import "@src/scss/components/Login.scss";
import { App } from "@types";
import { useNavigate } from "react-router-dom";

type Props = {
  appName: string | null;
  fetchingMetadata: boolean;
  onLoginSuccess: () => Promise<App[]>;
  pendingApp: () => Promise<App>;
  logo: string | null;
  navigate: ReturnType<typeof useNavigate>;
  isEmbeddedClusterWaitingForNodes: boolean;
};

type State = {
  password: string;
  loginErr: boolean;
  loginErrMessage: string;
  authLoading: boolean;
  loginInfo: {
    method: string;
  } | null;
};
type LoginResponse = {
  expires?: number;
  sessionRoles: string;
};
class SecureAdminConsole extends Component<Props, State> {
  loginText: RefObject<HTMLDivElement>;

  constructor(props: Props) {
    super(props);

    this.state = {
      password: "",
      loginErr: false,
      loginErrMessage: "",
      authLoading: false,
      loginInfo: null,
    };

    this.loginText = createRef();
  }

  completeLogin = async (data: LoginResponse) => {
    let loggedIn = false;
    try {
      if (Utilities.localStorageEnabled()) {
        loggedIn = true;
        window.localStorage.setItem("isLoggedIn", "true");

        if (data.sessionRoles) {
          window.localStorage.setItem("session_roles", data.sessionRoles);
        }

        const apps = await this.props.onLoginSuccess();
        const pendingApp = await this.props.pendingApp();
        this.setState({ authLoading: false });

        if (this.props.isEmbeddedClusterWaitingForNodes) {
          this.props.navigate("/cluster/manage", { replace: true });
          return loggedIn;
        }

        if (apps.length > 0) {
          this.props.navigate(`/app/${apps[0].slug}`, { replace: true });
        } else if (pendingApp?.slug && pendingApp?.needsRegistry) {
          this.props.navigate(`/${pendingApp.slug}/airgap`, { replace: true });
        } else if (pendingApp?.slug && !pendingApp?.needsRegistry) {
          this.props.navigate(`/${pendingApp.slug}/airgap-bundle`, {
            replace: true,
          });
        } else {
          this.props.navigate("upload-license", { replace: true });
        }
      } else {
        this.props.navigate("/unsupported");
      }
    } catch (err) {
      console.log(err);
    }

    return loggedIn;
  };

  validatePassword = () => {
    if (!this.state.password || this.state.password.length === 0) {
      this.setState({
        loginErr: true,
        loginErrMessage: `Please provide your password`,
      });
      return false;
    }
    return true;
  };

  loginWithSharedPassword = async () => {
    if (this.validatePassword()) {
      this.setState({
        authLoading: true,
        loginErr: false,
        loginErrMessage: "",
      });
      fetch(`${process.env.API_ENDPOINT}/login`, {
        headers: {
          "Content-Type": "application/json",
        },
        method: "POST",
        body: JSON.stringify({
          password: this.state.password,
        }),
        credentials: "include",
      })
        .then(async (res) => {
          if (res.status >= 400) {
            let body = await res.json();
            let msg = body.error;
            if (!msg) {
              msg =
                res.status === 401
                  ? "Invalid password. Please try again"
                  : "There was an error logging in. Please try again.";
            }
            this.setState({
              authLoading: false,
              loginErr: true,
              loginErrMessage: msg,
            });
            return;
          }
          // TODO: refactor this fetch function to return the result instead of using the callback in the fetch
          // TODO: remove "as" and use type Promise<LoginResponse> on loginWithSharedPassword
          this.completeLogin((await res.json()) as LoginResponse);
        })
        .catch((err) => {
          console.log("Login failed:", err);
          this.setState({
            authLoading: false,
            loginErr: true,
            loginErrMessage: "There was an error logging in. Please try again",
          });
        });
    }
  };

  loginWithIdentityProvider = async () => {
    try {
      this.setState({ loginErr: false, loginErrMessage: "" });

      const res = await fetch(`${process.env.API_ENDPOINT}/oidc/login`, {
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
          loginErr: true,
          loginErrMessage: msg,
        });
        return;
      }

      const body = await res.json();
      window.location = body.authCodeURL;
    } catch (err) {
      console.log("Login failed:", err);
      this.setState({
        loginErr: true,
        loginErrMessage: "There was an error logging in. Please try again",
      });
    }
  };

  submitForm = (e: KeyboardEvent) => {
    // TODO: keyCode is deprecated
    const enterKey = e.keyCode === 13;
    if (enterKey) {
      e.preventDefault();
      e.stopPropagation();
      this.loginWithSharedPassword();
    }
  };

  resizeLoginFont = () => {
    if (!this.loginText?.current) {
      return;
    }
    const newFontSize = dynamicallyResizeText(
      this.loginText.current.innerHTML,
      this.loginText.current.clientWidth,
      "32px"
    );
    this.loginText.current.style.fontSize = newFontSize;
  };

  getLoginInfo = async () => {
    try {
      const response = await fetch(`${process.env.API_ENDPOINT}/login/info`, {
        headers: {
          "Content-Type": "application/json",
        },
        method: "GET",
        credentials: "include",
      });

      if (!response.ok) {
        const res = await response.json();
        if (res.error) {
          throw new Error(
            `Unexpected status code ${response.status}: ${res.error}`
          );
        }
        throw new Error(`Unexpected status code ${response.status}`);
      }

      const loginInfo = await response.json();
      this.setState({ loginInfo, loginErr: false, loginErrMessage: "" });

      return loginInfo;
    } catch (err) {
      console.log(err);
    }

    return null;
  };

  componentDidUpdate() {
    const { appName } = this.props;
    if (appName) {
      if (this.loginText) {
        this.resizeLoginFont();
      }
    }
  }

  redirectLoginIfNeeded = async () => {
    const loginInfo = await this.getLoginInfo();
    if (loginInfo?.method === "identity-service") {
      await this.loginWithIdentityProvider();
    }
  };

  async componentDidMount() {
    if (Utilities.isLoggedIn()) {
      this.props.navigate("/apps");
    }
    window.addEventListener("keydown", this.submitForm);

    const isIdentityServiceLogin = Utilities.getCookie(
      "identity-service-login"
    );
    if (isIdentityServiceLogin) {
      // this is a redirect from identity service login
      const loginData = {
        sessionRoles: Utilities.getCookie("session_roles"),
      };
      const loggedIn = await this.completeLogin(loginData);
      if (loggedIn) {
        Utilities.removeCookie("identity-service-login");
        Utilities.removeCookie("session_roles");
      }
      return;
    }

    const urlParams = new URLSearchParams(window.location.search);
    const encodedMessage = urlParams.get("message");
    if (encodedMessage) {
      try {
        const message = JSON.parse(atob(encodedMessage));
        if (message.error) {
          this.setState({ loginErr: true, loginErrMessage: message.error });
          return;
        }
      } catch (err) {
        console.log("failed to decode message:", err);
      }
    }

    this.redirectLoginIfNeeded();
  }

  componentWillUnmount() {
    window.removeEventListener("keydown", this.submitForm);
  }

  render() {
    const { appName, logo, fetchingMetadata } = this.props;
    const { password, authLoading, loginErr, loginErrMessage, loginInfo } =
      this.state;

    if (fetchingMetadata || !loginInfo) {
      // secure-console url can receive an error message as url parameter.
      // When this happens, the spinner clause will always evaluate to true,
      // so we show this special error here with the "try again" option.
      // Right now this is how OIDC redirects pass errors back to the user.
      if (loginErr && loginErrMessage) {
        return (
          <ErrorModal
            errorModal={true}
            errMsg={loginErrMessage}
            err="Failed to attempt login"
            tryAgain={this.redirectLoginIfNeeded}
          />
        );
      }

      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    if (loginInfo?.method === "identity-service") {
      if (!loginErr) {
        return null;
      }
      return (
        <ErrorModal
          errorModal={true}
          errMsg={loginErrMessage}
          err="Failed to attempt login"
        />
      );
    }

    return (
      <div className="container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <KotsPageTitle pageName="Login" showAppSlug />
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex-column alignItems--center">
              {logo ? (
                <span
                  className="icon brand-login-icon"
                  style={{ backgroundImage: `url(${logo})` }}
                />
              ) : !fetchingMetadata ? (
                <span className="icon kots-login-icon" />
              ) : (
                <span style={{ width: "60px", height: "60px" }} />
              )}
              <p
                ref={this.loginText}
                style={{ fontSize: "32px" }}
                className="u-marginTop--10 u-paddingTop--5 u-lineHeight--more u-textColor--primary u-fontWeight--bold u-width--full u-textAlign--center break-word"
              >
                Log in
                {appName && appName !== ""
                  ? ` to ${appName} Admin Console`
                  : " to Admin Console"}
              </p>
            </div>
            <div className="flex-auto flex-column justifyContent--center">
              <p className="u-marginTop--10 u-marginTop--5 u-fontSize--large u-textAlign--center u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy break-word">
                Enter the password to access the {appName} Admin Console.
              </p>
              <div className="u-marginTop--20 flex-column">
                {loginErr && (
                  <p className="u-fontSize--normal u-fontWeight--medium u-textColor--error u-lineHeight--normal u-marginBottom--20">
                    {loginErrMessage}
                  </p>
                )}
                <div>
                  <div className="component-wrapper">
                    <input
                      type="password"
                      className="Input"
                      placeholder="password"
                      autoComplete="current-password"
                      value={password}
                      onChange={(e) => {
                        this.setState({ password: e.target.value });
                      }}
                    />
                  </div>
                  <div className="u-marginTop--20 flex justifyContent--center">
                    <button
                      type="submit"
                      className="btn primary"
                      disabled={authLoading}
                      onClick={this.loginWithSharedPassword}
                    >
                      {authLoading ? "Logging in" : "Log in"}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export { SecureAdminConsole };
