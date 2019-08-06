import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { isSecured } from "../queries/UserQueries";
import { Utilities } from "../utilities/utilities";
import { createAdminConsolePassword, loginToAdminConsole } from "../mutations/AuthMutations";
import "../scss/components/Login.scss";

class SecureAdminConsole extends React.Component {

  state = {
    password: "",
    confirmPassword: "",
    passwordErr: false,
    passwordErrMessage: "",
    authLoading: false,
    createLoading: false,
  }

  completeLogin = (data, create = false) => {
    const { onLoginSuccess } = this.props;
    let token;
    if (create) {
      token = data.createAdminConsolePassword.token
    } else {
      token = data.loginToAdminConsole.token
    }
    if (Utilities.localStorageEnabled()) {
      window.localStorage.setItem("token", token);
      onLoginSuccess().then(() => {
        this.setState({
          authLoading: false,
          createLoading: false,
        });
        this.props.history.push("/watches");
      });
    } else {
      this.props.history.push("/unsupported");
    }
  }

  validatePassword = (create = false) => {
    const { password, confirmPassword } = this.state;
    if (!password || password.length === "0") {
      this.setState({
        passwordErr: true,
        passwordErrMessage: `Please ${create ? "create a" : "provide your"} password`,
      });
      return false;
    }
    if (create) {
      if (password !== confirmPassword) {
        this.setState({
          passwordErr: true,
          passwordErrMessage: "Password's did not match",
        });
        return false;
      }
    }
    return true;
  }

  createPassword = async () => {
    const { password } = this.state;
    if (this.validatePassword(true)) {
      this.setState({ createLoading: true });
      await this.props.createAdminConsolePassword(password)
      .then(res => {
        this.setState({ createLoading: false });
        this.completeLogin(res.data, true);
      })
      .catch(err => {
        err.graphQLErrors.map(({ message }) => {
          this.setState({
            createLoading: false,
            passwordErr: true,
            passwordErrMessage: message,
          })
        })
      });
    }
  }

  loginToConsole = async () => {
    const { password } = this.state;
    if (this.validatePassword()) {
      this.setState({ authLoading: true });
      await this.props.loginToAdminConsole(password)
      .then(res => {
        this.completeLogin(res.data);
      })
      .catch(err => {
        err.graphQLErrors.map(({ message }) => {
          this.setState({
            authLoading: false,
            passwordErr: true,
            passwordErrMessage: message,
          })
        })
      });
    }
  }

  submitForm = (e) => {
    const enterKey = e.keyCode === 13;
    const hasPassword = this.props.isSecured?.isSecured;
    if (enterKey) {
      e.preventDefault();
      e.stopPropagation();
      if (hasPassword) {
        this.loginToConsole();
      } else {
        this.createPassword();
      }
    }
  }

  componentDidMount() {
    window.addEventListener("keydown", this.submitForm);
  }

  componentWillUnmount() {
    window.removeEventListener("keydown", this.submitForm);
  }

  render() {
    const { isSecured } = this.props;
    const {
      password, 
      confirmPassword,
      authLoading,
      createLoading,
      passwordErr,
      passwordErrMessage,
    } = this.state;

    const hasPassword = isSecured?.isSecured;

    return (
      <div className="container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex">
              <span className="icon ship-login-icon"></span>
              <p className="login-text u-color--tuna u-fontWeight--bold">{hasPassword ? "Log in" : "Secure admin console"}</p>
            </div>
            <p className="u-marginTop--20 u-fontSize--large u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">
              {hasPassword
              ? "Enter the password to access the admin console."
              : "Set a shared password to secure the admin console."}
            </p>
            <div className="u-marginTop--20 flex-column">
              {passwordErr && <p className="u-fontSize--normal u-fontWeight--medium u-color--chestnut u-lineHeight--normal u-marginBottom--20">{passwordErrMessage}</p>}
              {hasPassword
              ?
                <div>
                  <div className="component-wrapper">
                    <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Password</p>
                    <input type="password" className="Input" placeholder="password" autoComplete="current-password" value={password} onChange={(e) => { this.setState({ password: e.target.value }) }}/>
                  </div>
                  <div className="u-marginTop--20 flex">
                    <button type="submit" className="btn primary" disabled={authLoading} onClick={this.loginToConsole}>{authLoading ? "Logging in" : "Log in"}</button>
                  </div>
                </div>
              :
                <div>
                  <div className="component-wrapper">
                    <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Password</p>
                    <input type="password" className="Input" placeholder="Password" autoComplete="current-password" value={password} onChange={(e) => { this.setState({ password: e.target.value }) }}/>
                  </div>
                  <div className="component-wrapper u-marginTop--20">
                    <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Confirm password</p>
                    <input type="password" className="Input" placeholder="Confirm password" autoComplete="" value={confirmPassword} onChange={(e) => { this.setState({ confirmPassword: e.target.value }) }}/>
                  </div>
                  <div className="u-marginTop--20 flex">
                    <button type="submit" className="btn primary" disabled={createLoading} onClick={this.createPassword}>{createLoading ? "Securing console" : "Secure console"}</button>
                  </div>
                </div>
              }
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(isSecured, {
    name: "isSecured"
  }),
  graphql(createAdminConsolePassword, {
    props: ({ mutate }) => ({
      createAdminConsolePassword: (password) => mutate({ variables: { password } })
    })
  }),
  graphql(loginToAdminConsole, {
    props: ({ mutate }) => ({
      loginToAdminConsole: (password) => mutate({ variables: { password } })
    })
  }),
)(SecureAdminConsole);
