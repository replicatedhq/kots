import * as React from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter, Link } from "react-router-dom";
import "../scss/components/Login.scss";
import { shipAuthSignup } from "../mutations/AuthMutations";

class TraditionalAuth extends React.Component {
  state = {
    email: "",
    password: "",
    firstName: "",
    lastName: ""
  }

  handleLogin = () => {
    console.log("handle login");
  }

  handleSignup = () => {
    this.props.client.mutate({
      mutation: shipAuthSignup,
      variables: {
        input: {
          email: "asdasd",
          password: "asasd",
          firstName: "asdasd",
          lastName: "Asdasd",
        },
      },
    })
    .then((res) => {
      console.log(res);
    })
    .catch((err) => {
      console.log(err);
    });
  }

  render() {
    const { context } = this.props;
    const {
      email,
      password,
      firstName,
      lastName
    } = this.state;

    return (
      <div className="flex1 flex-column traditional-auth-wrapper">
        <div className="flex1 flex-column">
          {context === "signup" ?
            <div>
              <p className="u-lineHeight--normal u-fontSize--large u-color--doveGray u-fontWeight--medium u-marginBottom--20">We just need a few pieces of infomation to get your account created.</p>
            </div>
          :
            <div>
              <p className="u-lineHeight--normal u-fontSize--large u-color--doveGray u-fontWeight--medium u-marginBottom--5">Login with your email and password.</p>
              <p className="u-lineHeight--normal u-fontSize--large u-color--doveGray u-fontWeight--medium u-marginBottom--20">If you do not have an account, you can <Link to="/signup" className="replicated-link">create one here</Link></p>
            </div>
          }
          <form>
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
                <input type="text" className="Input" placeholder="you@example.com" value={email} onChange={(e) => { this.setState({ email: e.target.value }) }}/>
              </div>
              <div className="component-wrapper flex1">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Password</p>
                <input type="password" className="Input" placeholder="password" value={password} onChange={(e) => { this.setState({ password: e.target.value }) }}/>
              </div>
            </div>
          </form>
          <div className="u-marginTop--10 flex">
            <button onClick={context === "signup" ? this.handleSignup : this.handleLogin} className="btn primary">{context === "signup" ? "Create account" : "Log in"}</button>
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter
)(TraditionalAuth);
