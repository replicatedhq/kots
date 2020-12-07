import React, { Component } from "react";
import { withRouter } from "react-router-dom"
import Helmet from "react-helmet";
import ReactTooltip from "react-tooltip"
import isEmpty from "lodash/isEmpty";

import Loader from "../shared/Loader";
import ErrorModal from "../modals/ErrorModal";

import { Utilities } from "../../utilities/utilities";

class IdentityProviders extends Component {
  state = {
    isLoadingConfigSettings: false,
    configSettingsErrMsg: "",
    configSettings: {},
    adminConsoleAddress: "",
    identityServiceAddress: "",
    selectedProvider: "oidcConfig",
    oidcConfig: {
      issuer: "",
      clientId: "",
      clientSecret: "",
      connectorName: "",
      getUserInfo: false,
      insecureEnableGroups: false,
      insecureSkipEmailVerified: false,
      scopes: [],
      userNameKey: "",
      promptType: "",
      userIDKey: "",
      hostedDomains: [],
      claimMapping: {
        preferredUsername: "",
        email: "",
        groups: ""
      }
    },
    geoAxisConfig: {
      issuer: "",
      clientId: "",
      clientSecret: ""
    },
    requiredErrors: {},
    savingProviderSettings: false,
    saveConfirm: false,
    savingProviderErrMsg: "",
    showAdvancedOptions: false,
    displayErrorModal: false
  };

  fetchConfigSettings = async () => {
    this.setState({
      isLoadingConfigSettings: true,
      configSettingsErrMsg: "",
      displayErrorModal: false
    });
    try {
      const res = await fetch(`${window.env.API_ENDPOINT}/identity/config`, {
        method: "GET",
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        }
      })
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        this.setState({
          isLoadingConfigSettings: false,
          configSettingsErrMsg: `Unexpected status code: ${res.status}`,
          displayErrorModal: true
        });
        return;
      }
      const response = await res.json();

      this.setState({
        isLoadingConfigSettings: false,
        configSettings: response,
        configSettingsErrMsg: "",
        displayErrorModal: false
      });
    } catch (err) {
      this.setState({
        isLoadingConfigSettings: false,
        configSettingsErrMsg: err.message ? err.message : "There was an error while showing the config settings. Please try again",
        displayErrorModal: true
      })
    }
  }

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  }

  setFields = () => {
    const { configSettings } = this.state;
    if (!configSettings) { return; }

    return this.setState({
      adminConsoleAddress: configSettings?.adminConsoleAddress ? configSettings?.adminConsoleAddress : window.location.origin,
      identityServiceAddress: configSettings?.identityServiceAddress ? configSettings?.identityServiceAddress : `${window.location.origin}/dex`,
      selectedProvider: configSettings?.oidcConfig !== null ? "oidcConfig" : "geoAxisConfig",
      oidcConfig: configSettings?.oidcConfig,
      geoAxisConfig: configSettings?.geoAxisConfig
    });
  }

  componentDidMount() {
    this.fetchConfigSettings();
  }

  componentDidUpdate(lastProps, lastState) {
    if (this.state.configSettings !== lastState.configSettings && this.state.configSettings) {
      this.setFields();
    }
  }

  handleFormChange = (field, e) => {
    const { isKurlEnabled } = this.props;
    let nextState = {};
    if (field === "adminConsoleAddress") {
      nextState[field] = e.target.value;
      if (isKurlEnabled) {
        nextState["identityServiceAddress"] = `${e.target.value.replace(/\/$/, "")}/dex`;
      }
      this.setState(nextState);
    } else if (field === "identityServiceAddress") {
      nextState[field] = e.target.value;
      this.setState(nextState);
    } else {
      if (this.state.selectedProvider === "oidcConfig") {
        if (field === "getUserInfo" || field === "insecureEnableGroups" || field === "insecureSkipEmailVerified") {
          this.setState({ oidcConfig: { ...this.state.oidcConfig, [field]: e.target.checked } });
        } else if (field === "preferredUsername" || field === "email" || field === "groups") {
          this.setState({ oidcConfig: { ...this.state.oidcConfig, claimMapping: { ...this.state.oidcConfig?.claimMapping, [field]: e.target.value } } });
        } else {
          this.setState({ oidcConfig: { ...this.state.oidcConfig, [field]: e.target.value } });
        }
      } else {
        this.setState({ geoAxisConfig: { ...this.state.geoAxisConfig, [field]: e.target.value } })
      }
    }
  }

  handleOnChangeProvider = (provider) => {
    this.setState({ selectedProvider: provider, requiredErrors: {} })
  }

  validateRequiredFields = async (payloadFields) => {
    let requiredErrors = {};

    for (const field in payloadFields) {
      if (field !== "oidcConfig" && field !== "geoAxisConfig") {
        if (isEmpty(payloadFields[field])) {
          requiredErrors = { ...requiredErrors, [field]: true }
        }

        const sharedFields = ["issuer", "clientId", "clientSecret"];
        sharedFields.forEach(f => {
          if (payloadFields?.oidcConfig) {
            if (isEmpty(payloadFields?.oidcConfig?.connectorName)) {
              if (!payloadFields?.oidcConfig?.connectorName || isEmpty(payloadFields?.oidcConfig?.connectorName)) {
                requiredErrors = { ...requiredErrors, connectorName: true }
              }
            }
            if (isEmpty(payloadFields?.oidcConfig?.[f])) {
              requiredErrors = { ...requiredErrors, [f]: true }
            }
          } else {
            if (!payloadFields?.geoAxisConfig?.[f] || isEmpty(payloadFields?.geoAxisConfig?.[f])) {
              requiredErrors = { ...requiredErrors, [f]: true }
            }
          }
        })
      }
    }

    this.setState({ requiredErrors });
    return requiredErrors;
  }

  onSubmit = async (e) => {
    e.preventDefault();

    const oidcConfigPayload = {
      "oidcConfig": {
        issuer: this.state.oidcConfig?.issuer,
        clientId: this.state.oidcConfig?.clientId,
        clientSecret: this.state.oidcConfig?.clientSecret,
        connectorName: this.state.oidcConfig?.connectorName,
        getUserInfo: this.state.oidcConfig?.getUserInfo,
        insecureEnableGroups: this.state.oidcConfig?.insecureEnableGroups,
        insecureSkipEmailVerified: this.state.oidcConfig?.insecureSkipEmailVerified,
        scopes: !Array.isArray(this.state.oidcConfig?.scopes) ? this.state.oidcConfig?.scopes?.split(",") : this.state.oidcConfig?.scopes,
        userNameKey: this.state.oidcConfig?.userNameKey,
        promptType: this.state.oidcConfig?.promptType,
        userIDKey: this.state.oidcConfig?.userIDKey,
        hostedDomains: !Array.isArray(this.state.oidcConfig?.hostedDomains) ? this.state.oidcConfig?.hostedDomains?.split(",") : this.state.oidcConfig?.hostedDomains,
        claimMapping: {
          preferredUsername: this.state.oidcConfig?.claimMapping?.preferredUsername,
          email: this.state.oidcConfig?.claimMapping?.email,
          groups: this.state.oidcConfig?.claimMapping?.groups
        }
      }
    }

    const payload = {
      adminConsoleAddress: this.state.adminConsoleAddress,
      identityServiceAddress: this.state.identityServiceAddress,
      "oidcConfig": this.state.selectedProvider === "oidcConfig" ? oidcConfigPayload.oidcConfig : null,
      "geoAxisConfig": this.state.selectedProvider === "geoAxisConfig" ? this.state.geoAxisConfig : null
    }

    this.setState({ savingProviderErrMsg: "" });

    const errors = await this.validateRequiredFields(payload);

    if (isEmpty(errors)) {
      this.setState({ savingProviderSettings: true });

      fetch(`${window.env.API_ENDPOINT}/identity/config`, {
        method: "POST",
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload)
      })
        .then(async (res) => {
          const configureResponse = await res.json();
          if (!res.ok) {
            this.setState({
              savingProviderSettings: false,
              savingProviderErrMsg: configureResponse.error
            })
            return;
          }

          if (configureResponse.success) {
            this.setState({
              savingProviderSettings: false,
              saveConfirm: true,
              savingProviderErrMsg: ""
            });
            setTimeout(() => {
              this.setState({ saveConfirm: false })
            }, 3000);
          } else {
            this.setState({
              savingProviderSettings: false,
              savingProviderErrMsg: configureResponse.error
            })
          }
        })
        .catch((err) => {
          console.error(err);
          this.setState({
            savingProviderSettings: false,
            savingProviderErrMsg: err ? `Unable to save provider settings: ${err.message}` : "Something went wrong, please try again!"
          });
        });
    }
  }

  toggleAdvancedOptions = () => {
    this.setState({ showAdvancedOptions: !this.state.showAdvancedOptions });
  }

  getRequiredValue = (field) => {
    if (this.state.selectedProvider === "oidcConfig") {
      return this.state.oidcConfig?.[field] || "";
    } else {
      return this.state.geoAxisConfig?.[field] || "";
    }
  }


  render() {
    const { configSettingsErrMsg, isLoadingConfigSettings, requiredErrors, selectedProvider } = this.state;
    const { isKurlEnabled } = this.props;

    if (isLoadingConfigSettings) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    if (configSettingsErrMsg) {
      return (
        <ErrorModal
          errorModal={this.state.displayErrorModal}
          toggleErrorModal={this.toggleErrorModal}
          errMsg={configSettingsErrMsg}
          tryAgain={this.fetchConfigSettings}
          err="Failed to get config settings"
          loading={false}
        />
      )
    }

    return (
      <div className="flex-column flex1 u-position--relative u-overflow--auto u-padding--20 alignItems--center">
        <Helmet>
          <title>Identity Providers</title>
        </Helmet>
        {/* <div className="IdentityProviderWarning--wrapper flex alignItems--center u-marginTop--30">
          <span className="icon small-warning-icon u-marginRight--10" />
          <p>
            To configure an Identity Provider you must have Ingress configured for the Admin Console.
          <Link to="/access/configure-ingress" className="u-color--royalBlue u-textDecoration--underlineOnHover"> Configure Ingress </Link>
          </p>
        </div> */}
        <form className="flex flex-column Identity--wrapper u-marginTop--30">
          <p className="u-fontSize--largest u-lineHeight--default u-fontWeight--bold u-color--tuna"> Configure Identity Provider </p>
          <p className="u-fontSize--normal u-lineHeight--medium u-fontWeight--medium u-color--dustyGray u-marginTop--12"> Configure additional ODIC providers to authenticate in to the Admin Console. </p>

          <div className="u-marginTop--30">
            <div className="flex flex1 alignItems--center">
              <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Admin Console URL </p>
              <span className="required-label"> Required </span>
              {requiredErrors?.adminConsoleAddress && <span className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5"> Admin Console URL is a required field </span>}
            </div>
            <p className="u-fontSize--normal u-lineHeight--medium u-fontWeight--medium u-color--dustyGray u-marginTop--12"> The URL for accessing the KOTS Admin Console. This URL must be accessible from both the browser as well as the KOTS service. </p>
            <input type="text"
              className="Input u-marginTop--12"
              placeholder="https://kots.somebigbankadmin.com"
              value={this.state.adminConsoleAddress}
              onChange={(e) => { this.handleFormChange("adminConsoleAddress", e) }} />
          </div>

          {!isKurlEnabled && (
          <div className="u-marginTop--30">
            <div className="flex flex1 alignItems--center">
              <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> ID Address </p>
              <span className="required-label"> Required </span>
              {requiredErrors?.identityServiceAddress && <span className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5"> ID address is a required field </span>}
            </div>
            <p className="u-fontSize--normal u-lineHeight--medium u-fontWeight--medium u-color--dustyGray u-marginTop--12">
              The address of the Dex identity service, often `&lt;Admin Console URL&gt;/dex`.
              This URL must be accessible from both the browser as well as the KOTS service.
            </p>
            <input type="text"
              className="Input u-marginTop--12"
              placeholder="https://kots.somebigbankadmin.com/dex"
              value={this.state.identityServiceAddress}
              onChange={(e) => { this.handleFormChange("identityServiceAddress", e) }} />
          </div>
          ) }

          <div className="u-marginTop--30">
            <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Select an Identity Provider </p>
            <div className="flex flex1 alignItems--center u-marginTop--15">
              <label htmlFor="oidcConfig" className={`identityProviderBtn flex alignItems--center u-cursor--pointer u-userSelect--none ${this.state.selectedProvider === "oidcConfig" ? "is-active" : ""}`}>
                <input
                  type="radio"
                  id="oidcConfig"
                  style={{ display: "none" }}
                  checked={selectedProvider === "oidcConfig"}
                  onChange={(e) => { this.handleOnChangeProvider("oidcConfig", e) }} />
                <span className="icon openID u-cursor--pointer" />
              </label>
              <label htmlFor="geoAxisConfig" className={`identityProviderBtn flex alignItems--center u-cursor--pointer u-userSelect--none ${this.state.selectedProvider === "geoAxisConfig" ? "is-active" : ""}`} style={{ marginLeft: "15px" }}>
                <input
                  type="radio"
                  id="geoAxisConfig"
                  style={{ display: "none" }}
                  checked={selectedProvider === "geoAxisConfig"}
                  onChange={(e) => { this.handleOnChangeProvider("geoAxisConfig", e) }} />
                <span className="icon geoaxis u-cursor--pointer" />
              </label>
            </div>
          </div>

          {selectedProvider === "oidcConfig" &&
            <div className="u-marginTop--30">
              <div className="flex flex1 alignItems--center">
                <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Connector  name </p>
                <span className="required-label"> Required </span>
                {requiredErrors?.connectorName && <span className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5"> Connector name is a required field</span>}
              </div>
              <input type="text"
                className="Input u-marginTop--12"
                placeholder="OpenID"
                value={this.state.oidcConfig?.connectorName}
                onChange={(e) => { this.handleFormChange("connectorName", e) }} />
            </div>}

          <div className="u-marginTop--30">
            <div className="flex flex1 alignItems--center">
              <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Issuer </p>
              <span className="icon grayOutlineQuestionMark--icon u-marginLeft--10 u-cursor--pointer"
                data-tip="Canonical URL of the provider, also used for configuration discovery. This value MUST match the value returned in the provider config discovery." />
              <ReactTooltip effect="solid" className="replicated-tooltip" />
              <span className="required-label"> Required </span>
              {requiredErrors?.issuer && <span className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5"> Issuer is a required field </span>}
            </div>
            <input type="text"
              className="Input u-marginTop--12"
              value={this.getRequiredValue("issuer")}
              onChange={(e) => { this.handleFormChange("issuer", e) }} />
          </div>

          <div className="u-marginTop--30">
            <div className="flex flex1 alignItems--center">
              <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Client ID </p>
              <span className="required-label"> Required </span>
              {requiredErrors?.clientId && <span className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5"> Client ID is a required field </span>}
            </div>
            <input type="text"
              className="Input u-marginTop--12"
              value={this.getRequiredValue("clientId")}
              onChange={(e) => { this.handleFormChange("clientId", e) }} />
          </div>

          <div className="u-marginTop--30">
            <div className="flex flex1 alignItems--center">
              <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Client secret </p>
              <span className="required-label"> Required </span>
              {requiredErrors?.clientSecret && <span className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5"> Client Secret is a required field </span>}
            </div>
            <input type="password"
              className="Input u-marginTop--12"
              value={this.getRequiredValue("clientSecret")}
              onChange={(e) => { this.handleFormChange("clientSecret", e) }} />
          </div>

          {this.state.selectedProvider === "oidcConfig" &&
            <div className="u-marginTop--20">
              <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-color--royalBlue u-cursor--pointer" onClick={this.toggleAdvancedOptions}> Advanced options
              <span className={`icon ${this.state.showAdvancedOptions ? "up" : "down"}-arrow-icon-blue u-marginLeft--5 u-cursor--pointer`} /> </p>
              {this.state.showAdvancedOptions &&
                <div className="flex flex-column u-marginTop--12">
                  <div className={`flex-auto flex alignItems--center ${this.state.oidcConfig?.getUserInfo ? "is-active" : ""}`}>
                    <input
                      type="checkbox"
                      className="u-cursor--pointer"
                      id="getUserInfo"
                      checked={this.state.oidcConfig?.getUserInfo}
                      onChange={(e) => { this.handleFormChange("getUserInfo", e) }}
                    />
                    <label htmlFor="getUserInfo" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none alignItems--center" style={{ marginLeft: "2px" }}>
                      <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Get user info</p>
                      <span className="icon grayOutlineQuestionMark--icon u-marginLeft--10 u-cursor--pointer"
                        data-tip="When enabled, the OpenID Connector will query the UserInfo endpoint for additional claims. UserInfo claims
                      take priority over claims returned by the IDToken. This option should be used when the IDToken doesn't contain all the claims requested." />
                      <ReactTooltip effect="solid" className="replicated-tooltip" />
                    </label>
                  </div>
                  <div className={`flex-auto flex alignItems--center u-marginTop--5 ${this.state.oidcConfig?.insecureEnableGroups ? "is-active" : ""}`}>
                    <input
                      type="checkbox"
                      className="u-cursor--pointer"
                      id="insecureEnableGroups"
                      checked={this.state.oidcConfig?.insecureEnableGroups}
                      onChange={(e) => { this.handleFormChange("insecureEnableGroups", e) }}
                    />
                    <label htmlFor="insecureEnableGroups" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none alignItems--center" style={{ marginLeft: "2px" }}>
                      <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Enable insecure groups</p>
                      <span className="icon grayOutlineQuestionMark--icon u-marginLeft--10 u-cursor--pointer"
                        data-tip="Groups claims (like the rest of oidc claims through dex) only refresh when the id token is refreshed
                      meaning the regular refresh flow doesn't update the groups claim. As such by default the oidc connector
                      doesn't allow groups claims. If you are okay with having potentially stale group claims you can use
                      this option to enable groups claims through the oidc connector on a per-connector basis." />
                      <ReactTooltip effect="solid" className="replicated-tooltip" />
                    </label>
                  </div>

                  <div className={`flex-auto flex alignItems--center u-marginTop--5  ${this.state.oidcConfig?.insecureSkipEmailVerified ? "is-active" : ""}`}>
                    <input
                      type="checkbox"
                      className="u-cursor--pointer"
                      id="insecureSkipEmailVerified"
                      checked={this.state.oidcConfig?.insecureSkipEmailVerified}
                      onChange={(e) => { this.handleFormChange("insecureSkipEmailVerified", e) }}
                    />
                    <label htmlFor="insecureSkipEmailVerified" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none alignItems--center" style={{ marginLeft: "2px" }}>
                      <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Skip email verification</p>
                      <span className="icon grayOutlineQuestionMark--icon u-marginLeft--10 u-cursor--pointer"
                        data-tip='Some providers return claims without "email_verified", when they had no usage of emails verification in enrollment process
                      or if they are acting as a proxy for another IDP etc AWS Cognito with an upstream SAML IDP' />
                      <ReactTooltip effect="solid" className="replicated-tooltip" />
                    </label>
                  </div>
                  <div className="u-marginTop--20">
                    <div className="flex flex1 alignItems--center">
                      <div className="flex flex-column u-marginRight--30">
                        <div className="flex flex1 alignItems--center">
                          <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> User ID key </p>
                          <span className="icon grayOutlineQuestionMark--icon u-marginLeft--10 u-cursor--pointer"
                            data-tip="The set claim is used as user id." />
                          <ReactTooltip effect="solid" className="replicated-tooltip" />
                        </div>
                        <input type="text"
                          className="Input u-marginTop--12"
                          placeholder="sub"
                          value={this.state.oidcConfig?.userIDKey}
                          onChange={(e) => { this.handleFormChange("userIDKey", e) }} />
                      </div>
                      <div className="flex flex-column">
                        <div className="flex flex1 alignItems--center">
                          <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> User name key </p>
                          <span className="icon grayOutlineQuestionMark--icon u-marginLeft--10 u-cursor--pointer"
                            data-tip="The set claim is used as user name." />
                          <ReactTooltip effect="solid" className="replicated-tooltip" />
                        </div>
                        <input type="text"
                          className="Input u-marginTop--12"
                          placeholder="name"
                          value={this.state.oidcConfig?.userNameKey}
                          onChange={(e) => { this.handleFormChange("userNameKey", e) }} />
                      </div>
                    </div>
                  </div>
                  <div className="u-marginTop--30">
                    <div className="flex flex1 alignItems--center">
                      <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Prompt type </p>
                      <span className="icon grayOutlineQuestionMark--icon u-marginLeft--10 u-cursor--pointer"
                        data-tip='For offline_access, the prompt parameter is set by default to "prompt=consent". 
                      However this is not supported by all OIDC providers, some of them support different value for prompt, like "prompt=login" or "prompt=none"' />
                      <ReactTooltip effect="solid" className="replicated-tooltip" />
                    </div>
                    <input type="text"
                      className="Input u-marginTop--12"
                      placeholder="consent"
                      value={this.state.oidcConfig?.promptType}
                      onChange={(e) => { this.handleFormChange("promptType", e) }} />
                  </div>
                  <div className="u-marginTop--30">
                    <div className="flex flex1 alignItems--center">
                      <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Hosted domains </p>
                      <span className="icon grayOutlineQuestionMark--icon u-marginLeft--10 u-cursor--pointer"
                        data-tip="Google supports whitelisting allowed domains when using G Suite (Google Apps). The following field can be set to a comma-separated list of domains that can log in" />
                      <ReactTooltip effect="solid" className="replicated-tooltip" />
                    </div>
                    <input type="text"
                      className="Input u-marginTop--12"
                      value={this.state.oidcConfig?.hostedDomains}
                      onChange={(e) => { this.handleFormChange("hostedDomains", e) }} />
                  </div>
                  <div className="u-marginTop--30">
                    <div className="flex flex1 alignItems--center">
                      <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Scopes </p>
                      <span className="icon grayOutlineQuestionMark--icon u-marginLeft--10 u-cursor--pointer"
                        data-tip="Comma-separated list of additional scopes to request in token response. Default is profile and email" />
                      <ReactTooltip effect="solid" className="replicated-tooltip" />
                    </div>
                    <input type="text"
                      className="Input u-marginTop--12"
                      placeholder="profile,email,groups..."
                      value={this.state.oidcConfig?.scopes}
                      onChange={(e) => { this.handleFormChange("scopes", e) }} />
                  </div>

                  <div className="u-marginTop--30">
                    <div className="flex flex1 alignItems--center">
                      <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Claim mapping </p>
                      <span className="icon grayOutlineQuestionMark--icon u-marginLeft--10 u-cursor--pointer"
                        data-tip="Some providers return non-standard claims (eg. mail). Use claimMapping to map those claims to standard claims" />
                      <ReactTooltip effect="solid" className="replicated-tooltip" />
                    </div>
                    <p className="u-fontSize--normal u-lineHeight--normal u-fontWeight--normal u-marginTop--5"> claimMapping can only map a non-standard claim to a standard one if it's not returned in the id_token </p>
                    <div className="flex flexWrap--wrap alignItems--center">
                      <div className="flex flex-column u-marginRight--30 u-marginTop--20">
                        <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Preferred username key </p>
                        <input type="text"
                          className="Input u-marginTop--12"
                          placeholder="preferred_username"
                          value={this.state.oidcConfig?.claimMapping?.preferredUsername}
                          onChange={(e) => { this.handleFormChange("preferredUsername", e) }} />
                      </div>
                      <div className="flex flex-column u-marginRight--30 u-marginTop--20">
                        <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Email key </p>
                        <input type="text"
                          className="Input u-marginTop--12"
                          placeholder="email"
                          value={this.state.oidcConfig?.claimMapping?.email}
                          onChange={(e) => { this.handleFormChange("email", e) }} />
                      </div>
                      <div className="flex flex-column u-marginTop--20">
                        <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-color--tuna"> Group keys </p>
                        <input type="text"
                          className="Input u-marginTop--12"
                          placeholder="groups"
                          value={this.state.oidcConfig?.claimMapping?.groups}
                          onChange={(e) => { this.handleFormChange("groups", e) }} />
                      </div>
                    </div>
                  </div>
                </div>
              }
            </div>}

          <div className="flex flex-column u-marginTop--40 flex">
            {this.state.savingProviderErrMsg &&
              <div className="u-marginBottom--10 flex alignItems--center">
                <span className="u-fontSize--small u-fontWeight--medium u-color--chestnut">{this.state.savingProviderErrMsg}</span>
              </div>}
            <div className="flex flex1">
              <button className="btn primary blue" disabled={this.state.savingProviderSettings} onClick={this.onSubmit}>{this.state.savingProviderSettings ? "Saving" : "Save provider settings"}</button>
              {this.state.saveConfirm &&
                <div className="u-marginLeft--10 flex alignItems--center">
                  <span className="icon checkmark-icon" />
                  <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-color--chateauGreen">Settings saved</span>
                </div>
              }
            </div>
          </div>
        </form>
      </div>
    );
  }
}

export default withRouter(IdentityProviders);
