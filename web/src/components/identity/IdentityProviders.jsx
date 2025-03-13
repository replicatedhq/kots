import { Component } from "react";
import { KotsPageTitle } from "@components/Head";
import ReactTooltip from "react-tooltip";
import isEmpty from "lodash/isEmpty";
import size from "lodash/size";

import Loader from "../shared/Loader";
import ErrorModal from "../modals/ErrorModal";
import RBACGroupPolicyRow from "./RBACGroupPolicyRow";
import DummyRbacRow from "./DummyRbacRow";
import AddRoleGroup from "./AddRoleGroup";

import { Utilities } from "../../utilities/utilities";
import Icon from "../Icon";

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
      connectorId: "",
      connectorName: "",
      getUserInfo: false,
      insecureEnableGroups: false,
      insecureSkipEmailVerified: false,
      scopes: [],
      userNameKey: "",
      promptType: "",
      userIDKey: "",
      claimMapping: {
        preferredUsername: "",
        email: "",
        groups: "",
      },
    },
    geoAxisConfig: {
      issuer: "",
      clientId: "",
      clientSecret: "",
    },
    requiredErrors: {},
    savingProviderSettings: false,
    saveConfirm: false,
    savingProviderErrMsg: "",
    showAdvancedOptions: false,
    displayErrorModal: false,
    syncAppWithGlobal: false,
    roles: [],
    rbacGroupRows: [],
  };

  fetchConfigSettings = async (app) => {
    this.setState({
      isLoadingConfigSettings: true,
      configSettingsErrMsg: "",
      displayErrorModal: false,
    });
    let url;
    if (app && !this.state.syncAppWithGlobal) {
      url = `${process.env.API_ENDPOINT}/app/${app?.slug}/identity/config`;
    } else {
      url = `${process.env.API_ENDPOINT}/identity/config`;
    }
    try {
      const res = await fetch(url, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      });
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        this.setState({
          isLoadingConfigSettings: false,
          configSettingsErrMsg: `Unexpected status code: ${res.status}`,
          displayErrorModal: true,
        });
        return;
      }
      const response = await res.json();

      this.setState({
        isLoadingConfigSettings: false,
        configSettings: response,
        configSettingsErrMsg: "",
        displayErrorModal: false,
      });
    } catch (err) {
      this.setState({
        isLoadingConfigSettings: false,
        configSettingsErrMsg: err.message
          ? err.message
          : "There was an error while showing the config settings. Please try again",
        displayErrorModal: true,
      });
    }
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  buildGroups = (groups) => {
    return groups.map((g) => {
      return {
        id: g.id,
        roles: this.props.isApplicationSettings
          ? g.roleIds?.map((r) => ({ id: r, isChecked: true }))
          : g.roleIds?.map((r) => ({ id: r })),
        isAdded: true,
      };
    });
  };

  setFields = () => {
    const { configSettings } = this.state;
    if (!configSettings) {
      return;
    }

    let rbacGroups;
    if (configSettings?.groups) {
      rbacGroups = this.buildGroups(configSettings?.groups);
    } else {
      rbacGroups = [];
    }

    const nextState = {
      adminConsoleAddress: configSettings?.adminConsoleAddress
        ? configSettings?.adminConsoleAddress
        : window.location.origin,
      identityServiceAddress: configSettings?.identityServiceAddress
        ? configSettings?.identityServiceAddress
        : `${window.location.origin}/dex`,
      selectedProvider:
        configSettings?.oidcConfig !== null
          ? "oidcConfig"
          : configSettings?.geoAxisConfig !== null
          ? "geoAxisConfig"
          : null,
      oidcConfig: configSettings?.oidcConfig,
      geoAxisConfig: configSettings?.geoAxisConfig,
    };

    if (!this.state.syncAppWithGlobal) {
      nextState.roles = configSettings?.roles;
      nextState.rbacGroupRows = rbacGroups;
    }

    return this.setState(nextState);
  };

  componentDidMount() {
    this.fetchConfigSettings(this.props.app);
  }

  componentDidUpdate(lastProps, lastState) {
    if (
      this.state.configSettings !== lastState.configSettings &&
      this.state.configSettings
    ) {
      this.setFields();
    }
    if (this.state.syncAppWithGlobal !== lastState.syncAppWithGlobal) {
      this.fetchConfigSettings(this.props.app);
    }
  }

  handleFormChange = (field, e) => {
    const { isKurlEnabled } = this.props;
    let nextState = {};
    if (field === "adminConsoleAddress") {
      nextState[field] = e.target.value;
      if (isKurlEnabled) {
        nextState["identityServiceAddress"] = `${e.target.value.replace(
          /\/$/,
          ""
        )}/dex`;
      }
      this.setState(nextState);
    } else if (field === "identityServiceAddress") {
      nextState[field] = e.target.value;
      this.setState(nextState);
    } else if (field === "syncAppWithGlobal") {
      nextState[field] = e.target.checked;
      this.setState(nextState);
    } else {
      if (this.state.selectedProvider === "oidcConfig") {
        if (
          field === "getUserInfo" ||
          field === "insecureEnableGroups" ||
          field === "insecureSkipEmailVerified"
        ) {
          this.setState({
            oidcConfig: { ...this.state.oidcConfig, [field]: e.target.checked },
          });
        } else if (
          field === "preferredUsername" ||
          field === "email" ||
          field === "groups"
        ) {
          this.setState({
            oidcConfig: {
              ...this.state.oidcConfig,
              claimMapping: {
                ...this.state.oidcConfig?.claimMapping,
                [field]: e.target.value,
              },
            },
          });
        } else {
          this.setState({
            oidcConfig: { ...this.state.oidcConfig, [field]: e.target.value },
          });
        }
      } else {
        this.setState({
          geoAxisConfig: {
            ...this.state.geoAxisConfig,
            [field]: e.target.value,
          },
        });
      }
    }
  };

  handleFormRoleChange = (field, rowIndex, e) => {
    let rbacGroupRows = [...this.state.rbacGroupRows];
    if (field === "groupName") {
      let row = { ...rbacGroupRows[rowIndex] };
      row.id = e.target.value;
      rbacGroupRows[rowIndex] = row;
    } else {
      let row = { ...rbacGroupRows[rowIndex].roles[0] };
      const idStartPosition = e.target.id.indexOf("=") + 1;
      row.id = e.target.id.slice(idStartPosition);
      rbacGroupRows[rowIndex].roles[0] = row;
    }
    this.setState({ rbacGroupRows });
  };

  handleRoleCheckboxChange = (rowIndex, roleIndex, e) => {
    let rbacGroupRows = [...this.state.rbacGroupRows];
    let row = { ...rbacGroupRows[rowIndex].roles[roleIndex] };
    const idStartPosition = e.target.id.indexOf("=") + 1;
    row.id = e.target.id.slice(idStartPosition);
    row.isChecked = e.target.checked;
    rbacGroupRows[rowIndex].roles[roleIndex] = row;
    this.setState({ rbacGroupRows });
  };

  handleOnChangeProvider = (provider) => {
    this.setState({ selectedProvider: provider, requiredErrors: {} });
  };

  validateRequiredFields = async (payloadFields) => {
    let requiredErrors = {};

    for (const field in payloadFields) {
      if (field === "oidcConfig") {
        continue;
      }
      if (field === "geoAxisConfig") {
        continue;
      }
      if (field === "useAdminConsoleSettings") {
        continue;
      }
      if (field === "groups") {
        continue;
      }

      if (isEmpty(payloadFields[field])) {
        requiredErrors = { ...requiredErrors, [field]: true };
      }

      const sharedFields = ["issuer", "clientId", "clientSecret"];
      sharedFields.forEach((f) => {
        if (payloadFields?.oidcConfig) {
          if (isEmpty(payloadFields?.oidcConfig?.connectorName)) {
            if (
              !payloadFields?.oidcConfig?.connectorName ||
              isEmpty(payloadFields?.oidcConfig?.connectorName)
            ) {
              requiredErrors = { ...requiredErrors, connectorName: true };
            }
          }
          if (isEmpty(payloadFields?.oidcConfig?.[f])) {
            requiredErrors = { ...requiredErrors, [f]: true };
          }
        } else {
          if (
            !payloadFields?.geoAxisConfig?.[f] ||
            isEmpty(payloadFields?.geoAxisConfig?.[f])
          ) {
            requiredErrors = { ...requiredErrors, [f]: true };
          }
        }
      });
    }

    this.setState({ requiredErrors });
    return requiredErrors;
  };

  buildGroupsPayload = () => {
    const { rbacGroupRows } = this.state;
    const { isApplicationSettings } = this.props;

    return rbacGroupRows.map((g) => {
      return {
        id: g.id,
        roleIds: isApplicationSettings
          ? g.roles
              ?.filter((r) => r.isChecked)
              .map((r) => r.id)
              .filter((r) => r !== null)
          : !isEmpty(g.roles)
          ? g.roles?.map((r) => r.id).filter((r) => r !== null)
          : [],
      };
    });
  };

  onSubmit = async (e) => {
    e.preventDefault();
    const { app, isApplicationSettings } = this.props;

    const groups = this.buildGroupsPayload();

    const oidcConfigPayload = {
      oidcConfig: {
        issuer: this.state.oidcConfig?.issuer,
        clientId: this.state.oidcConfig?.clientId,
        clientSecret: this.state.oidcConfig?.clientSecret,
        connectorId: this.state.oidcConfig?.connectorId,
        connectorName: this.state.oidcConfig?.connectorName,
        getUserInfo: this.state.oidcConfig?.getUserInfo,
        insecureEnableGroups: this.state.oidcConfig?.insecureEnableGroups,
        insecureSkipEmailVerified:
          this.state.oidcConfig?.insecureSkipEmailVerified,
        scopes: !Array.isArray(this.state.oidcConfig?.scopes)
          ? this.state.oidcConfig?.scopes?.split(",")
          : this.state.oidcConfig?.scopes,
        userNameKey: this.state.oidcConfig?.userNameKey,
        promptType: this.state.oidcConfig?.promptType,
        userIDKey: this.state.oidcConfig?.userIDKey,
        claimMapping: {
          preferredUsername:
            this.state.oidcConfig?.claimMapping?.preferredUsername,
          email: this.state.oidcConfig?.claimMapping?.email,
          groups: this.state.oidcConfig?.claimMapping?.groups,
        },
      },
    };

    let payload;
    if (isApplicationSettings) {
      payload = {
        oidcConfig:
          this.state.selectedProvider === "oidcConfig"
            ? oidcConfigPayload.oidcConfig
            : null,
        geoAxisConfig:
          this.state.selectedProvider === "geoAxisConfig"
            ? this.state.geoAxisConfig
            : null,
        useAdminConsoleSettings: this.state.syncAppWithGlobal,
        groups: groups,
      };
    } else {
      payload = {
        adminConsoleAddress: this.state.adminConsoleAddress,
        identityServiceAddress: this.state.identityServiceAddress,
        oidcConfig:
          this.state.selectedProvider === "oidcConfig"
            ? oidcConfigPayload.oidcConfig
            : null,
        geoAxisConfig:
          this.state.selectedProvider === "geoAxisConfig"
            ? this.state.geoAxisConfig
            : null,
        groups: groups,
      };
    }

    this.setState({ savingProviderErrMsg: "" });

    const errors = await this.validateRequiredFields(payload);

    if (isEmpty(errors)) {
      this.setState({ savingProviderSettings: true });

      let url;
      if (app) {
        url = `${process.env.API_ENDPOINT}/app/${app?.slug}/identity/config`;
      } else {
        url = `${process.env.API_ENDPOINT}/identity/config`;
      }

      fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(payload),
      })
        .then(async (res) => {
          const configureResponse = await res.json();
          if (!res.ok) {
            this.setState({
              savingProviderSettings: false,
              savingProviderErrMsg: configureResponse.error,
            });
            return;
          }

          if (configureResponse.success) {
            this.setState({
              savingProviderSettings: false,
              saveConfirm: true,
              savingProviderErrMsg: "",
            });
            setTimeout(() => {
              this.setState({ saveConfirm: false });
            }, 3000);
          } else {
            this.setState({
              savingProviderSettings: false,
              savingProviderErrMsg: configureResponse.error,
            });
          }
        })
        .catch((err) => {
          console.error(err);
          this.setState({
            savingProviderSettings: false,
            savingProviderErrMsg: err
              ? `Unable to save provider settings: ${err.message}`
              : "Something went wrong, please try again!",
          });
        });
    }
  };

  toggleAdvancedOptions = () => {
    this.setState({ showAdvancedOptions: !this.state.showAdvancedOptions });
  };

  getRequiredValue = (field) => {
    if (this.state.selectedProvider === "oidcConfig") {
      return this.state.oidcConfig?.[field] || "";
    } else {
      return this.state.geoAxisConfig?.[field] || "";
    }
  };

  onAddGroupRow = () => {
    const { rbacGroupRows } = this.state;

    rbacGroupRows.push({
      id: "",
      roles: [],
      isEditing: true,
      showRoleDetails: true,
    });

    this.setState({ rbacGroupRows });
  };

  onRemoveGroupRow = (rowIndex) => {
    let rbacGroupRows = [...this.state.rbacGroupRows];
    rbacGroupRows.splice(rowIndex, 1);
    this.setState({ rbacGroupRows });
  };

  onCancelGroupRow = (rowIndex) => {
    let rbacGroupRows = [...this.state.rbacGroupRows];
    let row = { ...rbacGroupRows[rowIndex] };
    row.isEditing = false;
    if (!row.isAdded) {
      rbacGroupRows.splice(rowIndex, 1);
    } else {
      rbacGroupRows[rowIndex] = row;
    }
    this.setState({ rbacGroupRows });
  };

  onAddGroup = (rowIndex) => {
    let rbacGroupRows = [...this.state.rbacGroupRows];
    let row = { ...rbacGroupRows[rowIndex] };
    row.isEditing = false;
    row.isAdded = true;
    rbacGroupRows[rowIndex] = row;

    this.setState({ rbacGroupRows });
  };

  onEditGroup = (rowIndex) => {
    let rbacGroupRows = [...this.state.rbacGroupRows];
    let row = { ...rbacGroupRows[rowIndex] };
    row.isEditing = true;
    rbacGroupRows[rowIndex] = row;

    this.setState({ rbacGroupRows });
  };

  showRoleDetails = (rowIndex) => {
    let rbacGroupRows = [...this.state.rbacGroupRows];
    let row = { ...rbacGroupRows[rowIndex] };
    row.showRoleDetails = true;
    rbacGroupRows[rowIndex] = row;

    this.setState({ rbacGroupRows });
  };

  hideRoleDetails = (rowIndex) => {
    let rbacGroupRows = [...this.state.rbacGroupRows];
    let row = { ...rbacGroupRows[rowIndex] };
    row.showRoleDetails = false;
    rbacGroupRows[rowIndex] = row;

    this.setState({ rbacGroupRows });
  };

  render() {
    const {
      configSettingsErrMsg,
      isLoadingConfigSettings,
      requiredErrors,
      selectedProvider,
      syncAppWithGlobal,
    } = this.state;
    const { isKurlEnabled, isApplicationSettings, app, isGeoaxisSupported } =
      this.props;

    if (isLoadingConfigSettings) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    if (configSettingsErrMsg) {
      return (
        <ErrorModal
          errorModal={this.state.displayErrorModal}
          toggleErrorModal={this.toggleErrorModal}
          errMsg={configSettingsErrMsg}
          tryAgain={() => this.fetchConfigSettings(this.props.app)}
          err="Failed to get config settings"
          loading={false}
        />
      );
    }

    return (
      <div className="flex-column flex1 u-position--relative u-overflow--auto u-padding--20 alignItems--center">
        <KotsPageTitle
          pageName="Configure Identity Provider"
          showAppSlug={isApplicationSettings}
        />
        {/* <div className="IdentityProviderWarning--wrapper flex alignItems--center u-marginTop--30">
          <span className="icon small-warning-icon u-marginRight--10" />
          <p>
            To configure an Identity Provider you must have Ingress configured for the Admin Console.
          <Link to="/access/configure-ingress" className="link u-textDecoration--underlineOnHover"> Configure Ingress </Link>
          </p>
        </div> */}
        <form className="flex-auto Identity--wrapper u-marginTop--30" data-testid="identity-provider-form">
          <div className="flex1 flex-column">
            <p className="u-fontSize--largest u-lineHeight--default u-fontWeight--bold u-textColor--primary">
              {" "}
              Configure Identity Provider{" "}
              {isApplicationSettings && `for ${app?.name}`}
            </p>
            <p className="u-fontSize--normal u-lineHeight--medium u-fontWeight--medium u-textColor--bodyCopy u-marginTop--12">
              {" "}
              Configure additional OIDC providers to authenticate in to the
              Admin Console.{" "}
            </p>

            {isApplicationSettings && (
              <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left u-marginTop--20">
                <div
                  className={`flex-auto flex ${
                    syncAppWithGlobal ? "is-active" : ""
                  }`}
                >
                  <input
                    type="checkbox"
                    className="u-cursor--pointer"
                    id="syncAppWithGlobal"
                    checked={syncAppWithGlobal}
                    onChange={(e) => {
                      this.handleFormChange("syncAppWithGlobal", e);
                    }}
                  />
                  <label
                    htmlFor="syncAppWithGlobal"
                    className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                    style={{ marginTop: "2px" }}
                  >
                    <div className="flex flex-column u-marginLeft--5 justifyContent--center">
                      <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium">
                        Use Admin Console settings
                      </p>
                      <p className="u-fontSize--normal u-lineHeight--normal u-fontWeight--normal u-marginTop--5">
                        {" "}
                        Use the settings that you configured for the Admin
                        console.
                      </p>
                    </div>
                  </label>
                </div>
              </div>
            )}

            {!isApplicationSettings && (
              <div className="flex1 flex-column">
                <div className="u-marginTop--30">
                  <div className="flex flex1 alignItems--center">
                    <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                      {" "}
                      {isApplicationSettings ? "App" : "Admin Console"} URL{" "}
                    </p>
                    <span className="required-label"> Required </span>
                    {requiredErrors?.adminConsoleAddress && (
                      <span className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5">
                        {" "}
                        Admin Console URL is a required field{" "}
                      </span>
                    )}
                  </div>
                  <p className="u-fontSize--normal u-lineHeight--medium u-fontWeight--medium u-textColor--bodyCopy u-marginTop--12">
                    {" "}
                    The URL for accessing the KOTS Admin Console. This URL must
                    be accessible from both the browser as well as the KOTS
                    service.{" "}
                  </p>
                  <input
                    type="text"
                    className="Input u-marginTop--12"
                    placeholder="https://kots.somebigbankadmin.com"
                    value={this.state.adminConsoleAddress}
                    disabled={syncAppWithGlobal}
                    onChange={(e) => {
                      this.handleFormChange("adminConsoleAddress", e);
                    }}
                  />
                </div>

                {!isKurlEnabled && (
                  <div className="u-marginTop--30">
                    <div className="flex flex1 alignItems--center">
                      <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                        {" "}
                        ID Address{" "}
                      </p>
                      <span className="required-label"> Required </span>
                      {requiredErrors?.identityServiceAddress && (
                        <span className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5">
                          {" "}
                          ID address is a required field{" "}
                        </span>
                      )}
                    </div>
                    <p className="u-fontSize--normal u-lineHeight--medium u-fontWeight--medium u-textColor--bodyCopy u-marginTop--12">
                      The address of the Dex identity service, often `&lt;Admin
                      Console URL&gt;/dex`. This URL must be accessible from
                      both the browser as well as the KOTS service.
                    </p>
                    <input
                      type="text"
                      className="Input u-marginTop--12"
                      placeholder="https://kots.somebigbankadmin.com/dex"
                      value={this.state.identityServiceAddress}
                      disabled={syncAppWithGlobal}
                      onChange={(e) => {
                        this.handleFormChange("identityServiceAddress", e);
                      }}
                    />
                  </div>
                )}
              </div>
            )}

            <div className="flex1 flex-column u-marginTop--30">
              <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                {" "}
                Select an Identity Provider{" "}
              </p>
              <div className="flex flex1 alignItems--center u-marginTop--15">
                <label
                  htmlFor="oidcConfig"
                  className={`identityProviderBtn flex alignItems--center u-cursor--pointer u-userSelect--none ${
                    this.state.selectedProvider === "oidcConfig"
                      ? "is-active"
                      : ""
                  }`}
                >
                  <input
                    type="radio"
                    id="oidcConfig"
                    data-testid="openid-radio"
                    style={{ display: "none" }}
                    checked={selectedProvider === "oidcConfig"}
                    disabled={syncAppWithGlobal}
                    onChange={(e) => {
                      this.handleOnChangeProvider("oidcConfig", e);
                    }}
                  />
                  <span className="icon openID u-cursor--pointer" />
                </label>
                {(isGeoaxisSupported || app?.isGeoaxisSupported) && (
                  <label
                    htmlFor="geoAxisConfig"
                    className={`identityProviderBtn flex alignItems--center u-cursor--pointer u-userSelect--none ${
                      this.state.selectedProvider === "geoAxisConfig"
                        ? "is-active"
                        : ""
                    }`}
                    style={{ marginLeft: "15px" }}
                  >
                    <input
                      type="radio"
                      id="geoAxisConfig"
                      style={{ display: "none" }}
                      checked={selectedProvider === "geoAxisConfig"}
                      disabled={syncAppWithGlobal}
                      onChange={(e) => {
                        this.handleOnChangeProvider("geoAxisConfig", e);
                      }}
                    />
                    <span className="icon geoaxis u-cursor--pointer" />
                  </label>
                )}
              </div>
            </div>

            <div className="IdentityProvider--info u-marginTop--30 flex1 flex-column">
              <div className="flex flex1">
                {selectedProvider === "oidcConfig" && (
                  <div className="flex1 flex-column u-marginRight--20">
                    <div className="flex flex1 alignItems--center u-marginBottom--5">
                      <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                        {" "}
                        Connector name{" "}
                      </p>
                      <span className="required-label"> Required </span>
                    </div>
                    {requiredErrors?.connectorName && (
                      <span className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
                        {" "}
                        Connector name is a required field
                      </span>
                    )}
                    <input
                      type="text"
                      className="Input u-marginTop--12"
                      data-testid="connector-name-input"
                      placeholder="OpenID"
                      disabled={syncAppWithGlobal}
                      value={this.state.oidcConfig?.connectorName}
                      onChange={(e) => {
                        this.handleFormChange("connectorName", e);
                      }}
                    />
                  </div>
                )}

                <div className="flex1 flex-column">
                  <div className="flex flex1 alignItems--center u-marginBottom--5">
                    <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                      {" "}
                      Issuer{" "}
                    </p>
                    <Icon
                      icon="info-circle-outline"
                      size={16}
                      className="gray-color u-marginLeft--10 clickable"
                      data-tip="Canonical URL of the provider, also used for configuration discovery. This value MUST match the value returned in the provider config discovery."
                    />
                    <ReactTooltip
                      effect="solid"
                      className="replicated-tooltip"
                    />
                    <span className="required-label"> Required </span>
                  </div>
                  {requiredErrors?.issuer && (
                    <span className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
                      {" "}
                      Issuer is a required field{" "}
                    </span>
                  )}
                  <input
                    type="text"
                    className="Input u-marginTop--12"
                    data-testid="issuer-input"
                    disabled={syncAppWithGlobal}
                    value={this.getRequiredValue("issuer")}
                    onChange={(e) => {
                      this.handleFormChange("issuer", e);
                    }}
                  />
                </div>
              </div>

              <div className="flex flex1 u-marginTop--30">
                <div className="flex1 flex-column u-marginRight--20">
                  <div className="flex flex1 alignItems--center u-marginBottom--5">
                    <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                      {" "}
                      Client ID{" "}
                    </p>
                    <span className="required-label"> Required </span>
                  </div>
                  {requiredErrors?.clientId && (
                    <span className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
                      {" "}
                      Client ID is a required field{" "}
                    </span>
                  )}
                  <input
                    type="text"
                    className="Input u-marginTop--12"
                    data-testid="client-id-input"
                    value={this.getRequiredValue("clientId")}
                    disabled={syncAppWithGlobal}
                    onChange={(e) => {
                      this.handleFormChange("clientId", e);
                    }}
                  />
                </div>

                <div className="flex1 flex-column">
                  <div className="flex flex1 alignItems--center u-marginBottom--5">
                    <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                      {" "}
                      Client secret{" "}
                    </p>
                    <span className="required-label"> Required </span>
                  </div>
                  {requiredErrors?.clientSecret && (
                    <span className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
                      {" "}
                      Client Secret is a required field{" "}
                    </span>
                  )}
                  <input
                    type="password"
                    className="Input u-marginTop--12"
                    data-testid="client-secret-input"
                    value={this.getRequiredValue("clientSecret")}
                    disabled={syncAppWithGlobal}
                    onChange={(e) => {
                      this.handleFormChange("clientSecret", e);
                    }}
                  />
                </div>
              </div>
            </div>

            {size(this.state.roles) > 0 && (
              <div className="RbacPolicy--wrapper flex1 flex-column">
                <div
                  className={`u-marginTop--30 ${
                    size(this.state.rbacGroupRows) > 0 &&
                    "u-borderBottom--gray darker"
                  }`}
                >
                  <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                    {" "}
                    Role Based Access Control Group Policy{" "}
                  </p>
                  <p className="u-fontSize--normal u-lineHeight--medium u-fontWeight--medium u-textColor--bodyCopy u-marginTop--12 u-marginBottom--10">
                    {" "}
                    Groups are defined by your identity provider as is adding
                    and removing of team members to those groups.{" "}
                  </p>
                </div>
                {size(this.state.rbacGroupRows) > 0 ? (
                  <div className="flex-1-auto flex-column">
                    {this.state.rbacGroupRows?.map((g, i) => (
                      <RBACGroupPolicyRow
                        index={i}
                        key={i}
                        groupName={g.id}
                        roles={this.state.roles}
                        checkedRoles={g.roles?.filter((r) => r.isChecked)}
                        groupRoles={g.roles}
                        isEditing={g.isEditing}
                        onAddGroupRow={this.onAddGroupRow}
                        onAddGroup={this.onAddGroup}
                        handleFormChange={this.handleFormRoleChange}
                        onRemoveGroupRow={this.onRemoveGroupRow}
                        onEdit={this.onEditGroup}
                        showRoleDetails={g.showRoleDetails}
                        onShowRoleDetails={this.showRoleDetails}
                        onHideRoleDetails={this.hideRoleDetails}
                        isApplicationSettings={isApplicationSettings}
                        handleRoleCheckboxChange={this.handleRoleCheckboxChange}
                        onCancelGroupRow={this.onCancelGroupRow}
                      />
                    ))}
                    <p
                      className="u-fontSize--small u-lineHeight--normal u-marginTop--15 link"
                      onClick={this.onAddGroupRow}
                    >
                      {" "}
                      + Add a group{" "}
                    </p>
                  </div>
                ) : (
                  <div className="flex flex-column u-position--relative">
                    {[0, 1, 2, 3].map((el) => (
                      <DummyRbacRow key={el} />
                    ))}
                    <AddRoleGroup
                      addGroup={this.onAddGroupRow}
                      isApplicationSettings={isApplicationSettings}
                    />
                  </div>
                )}
              </div>
            )}

            {this.state.selectedProvider === "oidcConfig" && (
              <div className="flex1 flex-column u-marginTop--20">
                <p
                  className="u-fontSize--small u-lineHeight--normal link"
                  onClick={this.toggleAdvancedOptions}
                  data-testid="advanced-options-toggle"
                >
                  {" "}
                  Advanced options
                  <Icon
                    icon={
                      this.state.showAdvancedOptions ? "up-arrow" : "down-arrow"
                    }
                    size={12}
                    className="u-marginLeft--5 clickable"
                  />
                </p>
                {this.state.showAdvancedOptions && (
                  <div className="flex flex-column u-marginTop--12" data-testid="advanced-options-form">
                    <div className="flex flex1 justifyContent--spaceBetween">
                      <div
                        className="flex flex-column justifyContent--flexStart"
                        style={{ marginRight: "40px" }}
                      >
                        <div
                          className={`flex-auto flex alignItems--center ${
                            this.state.oidcConfig?.getUserInfo
                              ? "is-active"
                              : ""
                          }`}
                        >
                          <input
                            type="checkbox"
                            className="u-cursor--pointer"
                            id="getUserInfo"
                            disabled={syncAppWithGlobal}
                            checked={this.state.oidcConfig?.getUserInfo}
                            onChange={(e) => {
                              this.handleFormChange("getUserInfo", e);
                            }}
                          />
                          <label
                            htmlFor="getUserInfo"
                            className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none alignItems--center"
                            style={{ marginLeft: "2px" }}
                          >
                            <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium">
                              Get user info
                            </p>
                            <Icon
                              icon="info-circle-outline"
                              size={16}
                              className="gray-color u-marginLeft--10 clickable"
                              data-tip="When enabled, the OpenID Connector will query the UserInfo endpoint for additional claims. UserInfo claims
          take priority over claims returned by the IDToken. This option should be used when the IDToken doesn't contain all the claims requested."
                            />
                            <ReactTooltip
                              effect="solid"
                              className="replicated-tooltip"
                            />
                          </label>
                        </div>
                        <div
                          className={`flex-auto flex alignItems--center u-marginTop--20 ${
                            this.state.oidcConfig?.insecureEnableGroups
                              ? "is-active"
                              : ""
                          }`}
                        >
                          <input
                            type="checkbox"
                            className="u-cursor--pointer"
                            id="insecureEnableGroups"
                            disabled={syncAppWithGlobal}
                            checked={
                              this.state.oidcConfig?.insecureEnableGroups
                            }
                            onChange={(e) => {
                              this.handleFormChange("insecureEnableGroups", e);
                            }}
                          />
                          <label
                            htmlFor="insecureEnableGroups"
                            className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none alignItems--center"
                            style={{ marginLeft: "2px" }}
                          >
                            <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium">
                              Enable insecure groups
                            </p>
                            <Icon
                              icon="info-circle-outline"
                              size={16}
                              className="gray-color u-marginLeft--10 clickable"
                              data-tip="Groups claims (like the rest of oidc claims through dex) only refresh when the id token is refreshed
          meaning the regular refresh flow doesn't update the groups claim. As such by default the oidc connector
          doesn't allow groups claims. If you are okay with having potentially stale group claims you can use
          this option to enable groups claims through the oidc connector on a per-connector basis."
                            />
                            <ReactTooltip
                              effect="solid"
                              className="replicated-tooltip"
                            />
                          </label>
                        </div>

                        <div
                          className={`flex-auto flex alignItems--center u-marginTop--20  ${
                            this.state.oidcConfig?.insecureSkipEmailVerified
                              ? "is-active"
                              : ""
                          }`}
                        >
                          <input
                            type="checkbox"
                            className="u-cursor--pointer"
                            id="insecureSkipEmailVerified"
                            disabled={syncAppWithGlobal}
                            checked={
                              this.state.oidcConfig?.insecureSkipEmailVerified
                            }
                            onChange={(e) => {
                              this.handleFormChange(
                                "insecureSkipEmailVerified",
                                e
                              );
                            }}
                          />
                          <label
                            htmlFor="insecureSkipEmailVerified"
                            className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none alignItems--center"
                            style={{ marginLeft: "2px" }}
                          >
                            <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium">
                              Skip email verification
                            </p>
                            <Icon
                              icon="info-circle-outline"
                              size={16}
                              className="gray-color u-marginLeft--10 clickable"
                              data-tip='Some providers return claims without "email_verified", when they had no usage of emails verification in enrollment process
                              or if they are acting as a proxy for another IDP etc AWS Cognito with an upstream SAML IDP'
                            />
                            <ReactTooltip
                              effect="solid"
                              className="replicated-tooltip"
                            />
                          </label>
                        </div>
                      </div>
                      <div className="IdentityProviderAdvanced--info flex flex1 alignItems--center justifyContent--center">
                        <div className="flex1 flex-column u-marginRight--30">
                          <div className="flex flex1">
                            <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                              {" "}
                              User ID key{" "}
                            </p>
                            <Icon
                              icon="info-circle-outline"
                              size={16}
                              className="gray-color u-marginLeft--10 clickable"
                              data-tip="The set claim is used as user id."
                            />
                            <ReactTooltip
                              effect="solid"
                              className="replicated-tooltip"
                            />
                          </div>
                          <input
                            type="text"
                            className="Input u-marginTop--12"
                            placeholder="sub"
                            disabled={syncAppWithGlobal}
                            value={this.state.oidcConfig?.userIDKey}
                            onChange={(e) => {
                              this.handleFormChange("userIDKey", e);
                            }}
                          />
                        </div>

                        <div className="flex1 flex-column">
                          <div className="flex flex1">
                            <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                              {" "}
                              User name key{" "}
                            </p>
                            <Icon
                              icon="info-circle-outline"
                              size={16}
                              className="gray-color u-marginLeft--10 clickable"
                              data-tip="The set claim is used as user name."
                            />
                            <ReactTooltip
                              effect="solid"
                              className="replicated-tooltip"
                            />
                          </div>
                          <input
                            type="text"
                            className="Input u-marginTop--12"
                            data-testid="user-name-key-input"
                            placeholder="name"
                            value={this.state.oidcConfig?.userNameKey}
                            disabled={syncAppWithGlobal}
                            onChange={(e) => {
                              this.handleFormChange("userNameKey", e);
                            }}
                          />
                        </div>
                      </div>
                    </div>
                    <div className="IdentityProviderAdvanced--info flex">
                      <div className="flex1 flex-column u-marginTop--30 u-marginRight--30">
                        <div className="flex flex1 alignItems--center">
                          <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                            {" "}
                            Prompt type{" "}
                          </p>
                          <Icon
                            icon="info-circle-outline"
                            size={16}
                            className="gray-color u-marginLeft--10 clickable"
                            data-tip='For offline_access, the prompt parameter is set by default to "prompt=consent".
                              However this is not supported by all OIDC providers, some of them support different value for prompt, like "prompt=login" or "prompt=none"'
                          />
                          <ReactTooltip
                            effect="solid"
                            className="replicated-tooltip"
                          />
                        </div>
                        <input
                          type="text"
                          className="Input u-marginTop--12"
                          placeholder="consent"
                          value={this.state.oidcConfig?.promptType}
                          disabled={syncAppWithGlobal}
                          onChange={(e) => {
                            this.handleFormChange("promptType", e);
                          }}
                        />
                      </div>
                      <div className="flex1 flex-column u-marginTop--30">
                        <div className="flex flex1 alignItems--center">
                          <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                            {" "}
                            Scopes{" "}
                          </p>
                          <Icon
                            icon="info-circle-outline"
                            size={16}
                            className="gray-color u-marginLeft--10 clickable"
                            data-tip="Comma-separated list of additional scopes to request in token response. Default is profile and email"
                          />
                          <ReactTooltip
                            effect="solid"
                            className="replicated-tooltip"
                          />
                        </div>
                        <input
                          type="text"
                          className="Input u-marginTop--12"
                          placeholder="profile,email,groups..."
                          value={this.state.oidcConfig?.scopes}
                          disabled={syncAppWithGlobal}
                          onChange={(e) => {
                            this.handleFormChange("scopes", e);
                          }}
                        />
                      </div>
                    </div>

                    <div className="flex1 flex-column u-marginTop--30">
                      <div className="flex flex1 alignItems--center">
                        <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                          {" "}
                          Claim mapping{" "}
                        </p>
                        <Icon
                          icon="info-circle-outline"
                          size={16}
                          className="gray-color u-marginLeft--10 clickable"
                          data-tip="Some providers return non-standard claims (eg. mail). Use claimMapping to map those claims to standard claims"
                        />
                        <ReactTooltip
                          effect="solid"
                          className="replicated-tooltip"
                        />
                      </div>
                      <p className="u-fontSize--normal u-lineHeight--normal u-fontWeight--normal u-marginTop--5">
                        {" "}
                        claimMapping can only map a non-standard claim to a
                        standard one if it's not returned in the id_token{" "}
                      </p>
                      <div className="IdentityProviderAdvanced--info flex">
                        <div className="flex1 flex-column u-marginRight--30 u-marginTop--20">
                          <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                            {" "}
                            Preferred username key{" "}
                          </p>
                          <input
                            type="text"
                            className="Input u-marginTop--12"
                            placeholder="preferred_username"
                            value={
                              this.state.oidcConfig?.claimMapping
                                ?.preferredUsername
                            }
                            disabled={syncAppWithGlobal}
                            onChange={(e) => {
                              this.handleFormChange("preferredUsername", e);
                            }}
                          />
                        </div>
                        <div className="flex1 flex-column u-marginRight--30 u-marginTop--20">
                          <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                            {" "}
                            Email key{" "}
                          </p>
                          <input
                            type="text"
                            className="Input u-marginTop--12"
                            placeholder="email"
                            value={this.state.oidcConfig?.claimMapping?.email}
                            disabled={syncAppWithGlobal}
                            onChange={(e) => {
                              this.handleFormChange("email", e);
                            }}
                          />
                        </div>
                        <div className="flex1 flex-column u-marginTop--20">
                          <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                            {" "}
                            Group keys{" "}
                          </p>
                          <input
                            type="text"
                            className="Input u-marginTop--12"
                            placeholder="groups"
                            value={this.state.oidcConfig?.claimMapping?.groups}
                            disabled={syncAppWithGlobal}
                            onChange={(e) => {
                              this.handleFormChange("groups", e);
                            }}
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}

            <div className="flex flex-column u-marginTop--40">
              {this.state.savingProviderErrMsg && (
                <div className="u-marginBottom--10 flex alignItems--center">
                  <span className="u-fontSize--small u-fontWeight--medium u-textColor--error">
                    {this.state.savingProviderErrMsg}
                  </span>
                </div>
              )}
              <div className="flex flex1">
                <button
                  className="btn primary blue"
                  data-testid="save-provider-settings-button"
                  disabled={this.state.savingProviderSettings}
                  onClick={this.onSubmit}
                >
                  {this.state.savingProviderSettings
                    ? "Saving"
                    : "Save provider settings"}
                </button>
                {this.state.saveConfirm && (
                  <div className="u-marginLeft--10 flex alignItems--center" data-testid="provider-settings-saved-confirmation">
                    <Icon
                      icon="check-circle-filled"
                      size={16}
                      className="success-color"
                    />
                    <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-textColor--success">
                      Settings saved
                    </span>
                  </div>
                )}
              </div>
            </div>
          </div>
        </form>
      </div>
    );
  }
}

export default IdentityProviders;
