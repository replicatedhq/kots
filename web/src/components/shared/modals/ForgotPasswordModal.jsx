import * as React from "react";

export default class ForgotPasswordModal extends React.Component {

  state = {
    email: "",
    emailSent: false,
    sendingEmail: false
  }

  onSubmit = () => {
    const { email } = this.state;
    // this.props.sendResetEmail(email);
    console.log(`send email to ${email}`);
  }

  render() {
    return (
      this.state.emailSent ?
        <div>
          <p className="u-fontSize--large u-fontWeight--normal u-color--tuna u-lineHeight--normal u-textAlign--center">
            Check your inbox. If there is an account with the specified address, an email will be sent with instructions for resetting the password.
          </p>
          <div className="u-marginTop--20 u-textAlign--center">
            <button type="button" className="btn primary" onClick={() => this.props.onRequestClose()}>Ok, got it!</button>
          </div>
        </div>
        :
        <div>
          <p className="u-fontWeight--medium u-lineHeight--more u-fontSize--large u-color--tundora u-marginBottom--20">Provide your email address and if an account exists, we'll send an email with a reset link.</p>
          <div className="u-marginBottom--10">
            {this.state.resetError ?
              <div className="ErrorBlock u-marginBottom--small">
                <p>{this.state.resetError.message}</p>
              </div>
              : null}
            <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora u-marginBottom--10 u-marginTop--10">What is your email address?</p>
            <input
              className="Input"
              type="text"
              placeholder="you@example.com"
              value={this.state.email}
              onChange={(e) => { this.setState({ email: e.target.value }); }}
            />
          </div>
          <div className="button-wrapper u-textAlign--right">
            <button
              type="button"
              className="btn secondary u-marginRight--10"
              onClick={() => this.props.onRequestClose()}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn primary"
              disabled={this.state.email === "" || this.state.sendingEmail}
              onClick={this.onSubmit}
            >
              {this.state.sendingEmail ? "Sending" : "Send email"}
            </button>
          </div>
        </div>
    )
  }
}