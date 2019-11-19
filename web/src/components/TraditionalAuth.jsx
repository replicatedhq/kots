import * as React from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter, Link } from "react-router-dom";
import { shipAuthSignup, shipAuthLogin } from "../mutations/AuthMutations";

import "../scss/components/Login.scss";
import { Utilities } from "../utilities/utilities";

class TraditionalAuth extends React.Component {
  state = {
    email: "",
    password: "",
    firstName: "",
    lastName: "",
    signup: {
      buttonText: "Create account",
      buttonLoadingText: "Creating",
      secondaryAction: "I already have an account"
    },
    login: {
      buttonText: "Log in",
      buttonLoadingText: "Loggin in",
      secondaryAction: "Create an account"
    },
    authLoading: false
  }

  handleLogin = () => {
    const { onLoginSuccess } = this.props;
    this.setState({ authLoading: true });
    this.props.client.mutate({
      mutation: shipAuthLogin,
      variables: {
        input: {
          email: this.state.email,
          password: this.state.password,
        },
      },
    })
    .then((res) => {
      this.setState({ authLoading: false });
      if (Utilities.localStorageEnabled()) {
        window.localStorage.setItem("token", res.data.login.token);
        onLoginSuccess().then(() => {
          this.props.history.push("/apps");
        });
      } else {
        this.props.history.push("/unsupported");
      }
    })
    .catch((err) => {
      console.log(err);
    });
  }

  handleSignup = () => {
    this.setState({ authLoading: true });
    this.props.client.mutate({
      mutation: shipAuthSignup,
      variables: {
        input: {
          email: this.state.email,
          password: this.state.password,
          firstName: this.state.firstName,
          lastName: this.state.lastName,
        },
      },
    })
    .then((res) => {
      this.setState({ authLoading: false });
      if (Utilities.localStorageEnabled()) {
        window.localStorage.setItem("token", res.data.signup.token);
        this.props.history.push("/apps");
      } else {
        this.props.history.push("/unsupported");
      }
    })
    .catch((err) => {
      console.log(err);
    });
  }

  onSubmit = (e) => {
    e.preventDefault();
    if (this.props.context === "signup") {
      this.handleSignup();
    } else {
      this.handleLogin();
    }
  }

  render() {
    const { context } = this.props;
    const {
      email,
      password,
      firstName,
      lastName,
      authLoading
    } = this.state;

    return (
      <div className="flex1 flex-column traditional-auth-wrapper">
        <div className="flex1 flex-column">
          {context === "signup" ?
            <p className="u-lineHeight--normal u-fontSize--large u-color--doveGray u-fontWeight--medium u-marginBottom--20">We just need a few pieces of infomation to get your account created.</p>
          :
            <p className="u-lineHeight--normal u-fontSize--large u-color--doveGray u-fontWeight--medium u-marginBottom--5">Login with your email and password.</p>
          }
          <form onSubmit={(e) => this.onSubmit(e)}>
            {context === "signup" &&
              <div className="u-flexTabletReflow">
                <div className="component-wrapper flex1 u-paddingRight--10">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">First name</p>
                  <input type="text" className="Input" placeholder="John" value={firstName} onChange={(e) => { this.setState({ firstName: e.target.value }) }}/>
                </div>
                <div className="component-wrapper flex1 u-paddingLeft--10">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Last name</p>
                  <input type="text" className="Input" placeholder="Doe" value={lastName} onChange={(e) => { this.setState({ lastName: e.target.value }) }}/>
                </div>
              </div>
            }
            <div className="flex-column u-marginBottom--10">
              <div className="component-wrapper flex1">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Email address</p>
                <input type="text" className="Input" placeholder="you@example.com" value={email} autoComplete="username" onChange={(e) => { this.setState({ email: e.target.value }) }}/>
              </div>
              <div className="component-wrapper flex1">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Password</p>
                <input type="password" className="Input" placeholder="password" autoComplete="current-password" value={password} onChange={(e) => { this.setState({ password: e.target.value }) }}/>
                {context === "login" && <p className="replicated-link u-fontSize--small u-marginTop--10" onClick={this.onForgotPasswordClick}>Forgot password?</p>}
              </div>
            </div>
            <div className="u-marginTop--10 flex alignItems--center">
              <button type="submit" className="btn primary" disabled={authLoading}>{authLoading ? this.state[context].buttonLoadingText : this.state[context].buttonText}</button>
              <Link to={context === "signup" ? "/login?ta=1" : "/signup"} className="replicated-link u-fontSize--small u-marginLeft--10">{this.state[context].secondaryAction}</Link>
            </div>
          </form>
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter
)(TraditionalAuth);
