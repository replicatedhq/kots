import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { isSecured } from "../queries/UserQueries";
import { createAdminConsolePassword, loginToAdminConsole } from "../mutations/AuthMutations";
import "../scss/components/Login.scss";

class SecureAdminConsole extends React.Component {

  state = {
    password: "",
    confirmPassword: "",
    passwordErr: false,
    passwordErrMessage: "",
  }

  createPassword = async () => {
    const { password, confirmPassword } = this.state;
    if (password !== confirmPassword) {
      return this.setState({
        passwordErr: true,
        passwordErrMessage: "Password's did not match",
      });
    }
    this.setState({ createLoading: true });
    await this.props.createAdminConsolePassword(password)
    .then(res => {
      this.setState({ createLoading: false });
      console.log(res.data);
      // TODO: set token and get user?
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

  loginToConsole = async () => {
    const { password } = this.state;
    if (!password || password.length === "0") {
      return this.setState({
        passwordErr: true,
        passwordErrMessage: "Please provide your password",
      });
    }
    this.setState({ authLoading: true });
    await this.props.loginToAdminConsole(password)
    .then(res => {
      this.setState({ authLoading: false });
      console.log(res.data);
      // TODO: set token and get user?
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

  render() {
    const { isSecured } = this.props;
    const {
      password, 
      confirmPassword,
      authLoading,
      createLoading,
    } = this.state;

    const hasPassword = isSecured?.isSecured;

    return (
      <div className="container flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center">
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex">
              <span className="icon ship-login-icon"></span>
              <p className="login-text u-color--tuna u-fontWeight--bold">Secure admin console</p>
            </div>
            <p className="u-marginTop--20 u-fontSize--large u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">
              {hasPassword
              ? "Enter the password to access the admin console."
              : "Set a shared password to secure the admin console. This can only be done once and cannot be changed."}
            </p>
            <div className="u-marginTop--20 flex-column">
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
