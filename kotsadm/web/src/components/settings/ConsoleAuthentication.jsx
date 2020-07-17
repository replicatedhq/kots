import React, { Component } from "react";
import Helmet from "react-helmet";
import { withRouter } from "react-router-dom";
import Select from "react-select";

class ConsoleAuthentication extends Component {
  state = {
    password: "",
    confirmPassword: "",
    selectedOption: {
      value: "shared-password",
      label: "Shared password"
    }
  }

  saveChanges = () => {
    console.log("save changes");
  }

  handleAuthTypeChange = selectedOption => {
    this.setState({ selectedOption });
  }

  render() {
    const {
      password,
      confirmPassword,
      selectedOption
    } = this.state;

    const selectOptions = [
      {
        value: "shared-password",
        label: "Shared password"
      },
      {
        value: "saml-auth",
        label: "SAML Authentication"
      }
    ]

    return (
      <div className="container flex-column u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>{`Console authentication`}</title>
        </Helmet>
        <div className="ConsoleSettingsSection--wrapper">
          <div className="u-marginTop--15">
            <h2 className="u-fontSize--largest u-fontWeight--bold u-color--tuna">Admin Console Authentication</h2>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-marginTop--10">Configure the type of authentication that will be used to log in to the admin console. You can configure the console to use a Shared Password, Google Auth or SMAL.</p>
          </div>
          <div className="u-width--half u-paddingTop--30">
            <form>
              <div className="flex u-marginBottom--20">
                <div className="flex1">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Authentication type</p>
                  <p className="u-lineHeight--normal u-fontSize--small u-color--dustyGray u-fontWeight--medium u-marginBottom--10">Changing authentication type will require all users to log back in.</p>
                  <Select
                    className="replicated-select-container"
                    classNamePrefix="replicated-select"
                    options={selectOptions}
                    value={selectedOption}
                    getOptionValue={(option) => option.label}
                    isOptionSelected={(option) => { option.value === selectedOption.value }}
                    onChange={this.handleAuthTypeChange}
                  />
                </div>
              </div>
              <div className="flex u-marginBottom--20">
                <div className="flex1 u-paddingRight--5">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Password</p>
                  <input type="password" className="Input" placeholder="password" autoComplete="current-password" value={password || ""} onChange={(e) => { this.setState({ password : e.target.value }) }} />
                </div>
                <div className="flex1 u-paddingLeft--5">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Confirm password</p>
                  <input type="password" className="Input" placeholder="one more time" value={confirmPassword || ""} onChange={(e) => { this.setState({ confirmPassword: e.target.value }) }} />
                </div>
              </div>
              <div className="u-marginTop--20">
                <button className="btn primary blue" onClick={this.saveChanges}>Save changes</button>
              </div>
            </form>
          </div>
        </div>
      </div>
    )
  }
}

export default withRouter(ConsoleAuthentication);