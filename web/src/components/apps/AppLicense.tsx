import { useReducer, useEffect, ReactNode } from "react";
import { KotsPageTitle } from "@components/Head";
// @ts-ignore
import Dropzone from "react-dropzone";
// @ts-ignore
import yaml from "js-yaml";
import classNames from "classnames";
import size from "lodash/size";
import Modal from "react-modal";
import { Link, useOutletContext } from "react-router-dom";
import {
  getFileContent,
  Utilities,
  getLicenseExpiryDate,
  isServiceAccountToken,
  serviceAccountTokensMatch,
} from "../../utilities/utilities";
import Loader from "../shared/Loader";
// @ts-ignore
import styled from "styled-components";

import { App, AppLicense, LicenseFile } from "@src/types";
import "@src/scss/components/apps/AppLicense.scss";
import { LicenseFields } from "@features/Dashboard";
import { useLicenseWithIntercept } from "@features/App";
import Icon from "../Icon";

type Props = {
  app: App;
  isEmbeddedCluster: boolean;
};

type State = {
  appLicense: AppLicense | null;
  loading: boolean;
  message: string;
  messageType: string;
  showNextStepModal: boolean;
  entitlementsToShow: string[];
  showLicenseChangeModalState: boolean;
  licenseChangeFile: LicenseFile | null;
  changingLicense: boolean;
  licenseChangeMessage: string;
  licenseChangeMessageType: string;
  isViewingLicenseEntitlements: boolean;
  showServiceAccountTokenModal: boolean;
  serviceAccountToken: string;
  uploadingServiceAccountToken: boolean;
};

const AppLicenseComponent = () => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      appLicense: null,
      loading: false,
      message: "",
      messageType: "info",
      showNextStepModal: false,
      entitlementsToShow: [],
      showLicenseChangeModalState: false,
      licenseChangeFile: null,
      changingLicense: false,
      licenseChangeMessage: "",
      licenseChangeMessageType: "info",
      isViewingLicenseEntitlements: false,
      showServiceAccountTokenModal: false,
      serviceAccountToken: "",
      uploadingServiceAccountToken: false,
    }
  );
  const outletContext: Props = useOutletContext();

  const { data: licenseWithInterceptResponse } = useLicenseWithIntercept();
  useEffect(() => {
    if (!licenseWithInterceptResponse) {
      setState({ appLicense: null });
    } else {
      setState({
        appLicense: licenseWithInterceptResponse.license,
        isViewingLicenseEntitlements:
          size(licenseWithInterceptResponse.license?.entitlements) <= 5
            ? false
            : true,
      });
    }
  }, [licenseWithInterceptResponse]);

  const syncAppLicense = (licenseData: string) => {
    const { app } = outletContext;
    setState({
      loading: true,
      message: "",
      messageType: "info",
    });

    const payload = {
      licenseData,
    };

    fetch(
      `${process.env.API_ENDPOINT}/app/${outletContext?.app?.slug}/license`,
      {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(payload),
      }
    )
      .then(async (response) => {
        if (!response.ok) {
          if (response.status == 401) {
            Utilities.logoutUser();
            return;
          }
          const res = await response.json();
          throw new Error(res?.error);
        }
        return response.json();
      })
      .then(async (licenseResponse) => {
        let message;
        if (!licenseResponse.synced) {
          message = "License is already up to date";
        } else if (app.isAirgap) {
          message = "License uploaded successfully";
        } else {
          message = "License synced successfully";
        }

        setState({
          appLicense: licenseResponse.license,
          message,
          messageType: "info",
          showNextStepModal: licenseResponse.synced,
        });
      })
      .catch((err) => {
        console.log(err);
        setState({
          message: err ? err.message : "Something went wrong",
          messageType: "error",
        });
      })
      .finally(() => {
        setState({ loading: false });
      });
  };

  const uploadServiceAccountToken = (token: string) => {
    setState({
      uploadingServiceAccountToken: true,
      message: "",
      messageType: "info",
    });

    const payload = {
      serviceAccountToken: token,
    };

    fetch(
      `${process.env.API_ENDPOINT}/app/${outletContext?.app?.slug}/service-account-token`,
      {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(payload),
      }
    )
      .then(async (response) => {
        if (!response.ok) {
          if (response.status == 401) {
            Utilities.logoutUser();
            return;
          }
          const res = await response.json();
          throw new Error(res?.error);
        }
        return response.json();
      })
      .then(async (tokenResponse) => {
        let message;
        if (tokenResponse.synced) {
          message = "Service account token uploaded and license synced successfully";
        } else {
          message = "Service account token uploaded but matched the current license - no change was made";
        }

        setState({
          appLicense: tokenResponse.license,
          message,
          messageType: "info",
          showServiceAccountTokenModal: false,
          serviceAccountToken: "",
          showNextStepModal: tokenResponse.synced,
        });
      })
      .catch((err) => {
        console.log(err);
        setState({
          message: err?.message || "Failed to upload service account token",
          messageType: "error",
        });
      })
      .finally(() => {
        setState({ uploadingServiceAccountToken: false });
      });
  };

  const onDrop = async (files: LicenseFile[]) => {
    // TODO: TextDecoder.decode() expects arg of BufferSource | undefined
    // getFileContent returns string, ArrayBuffer, or null. Need to figure out
    // eslint-disable-next-line
    const content: any = await getFileContent(files[0]);
    const contentStr = new TextDecoder("utf-8").decode(content);
    const airgapLicense = await yaml.safeLoad(contentStr);
    const { appLicense } = state;

    // TODO: FIX THIS
    // @ts-ignore
    if (airgapLicense?.spec?.licenseID !== appLicense?.id) {
      // if the license ID has changed, but the service account token is the same, we can sync the license
      // @ts-ignore
      if (serviceAccountTokensMatch(airgapLicense.spec?.licenseID, appLicense?.id)) {
        return syncAppLicense(contentStr);
      }
      setState({
        message: "Licenses do not match",
        messageType: "error",
      });
      return;
    }

    // TODO: FIX THIS
    // @ts-ignore
    if (airgapLicense?.spec?.licenseSequence === appLicense?.licenseSequence) {
      setState({
        message: "License is already up to date",
        messageType: "info",
      });
      return;
    }

    syncAppLicense(contentStr);
  };

  const onLicenseChangeDrop = async (files: LicenseFile[]) => {
    setState({
      licenseChangeFile: files[0],
      licenseChangeMessage: "",
    });
  };

  const clearLicenseChangeFile = () => {
    setState({ licenseChangeFile: null, licenseChangeMessage: "" });
  };

  const changeAppLicense = async () => {
    if (!state.licenseChangeFile) {
      return;
    }

    // TODO: TextDecoder.decode() expects arg of BufferSource | undefined
    // getFileContent returns string, ArrayBuffer, or null. Need to figure out
    // eslint-disable-next-line
    const content: any = await getFileContent(state.licenseChangeFile);

    const licenseData = new TextDecoder("utf-8").decode(content);

    setState({
      changingLicense: true,
      licenseChangeMessage: "",
      licenseChangeMessageType: "info",
    });

    const { app } = outletContext;

    const payload = {
      licenseData,
    };

    fetch(`${process.env.API_ENDPOINT}/app/${app.slug}/change-license`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify(payload),
    })
      .then(async (response) => {
        if (!response.ok) {
          if (response.status == 401) {
            Utilities.logoutUser();
            return;
          }
          const res = await response.json();
          throw new Error(res?.error);
        }
        return response.json();
      })
      .then(async (licenseResponse) => {
        setState({
          appLicense: licenseResponse.license,
          showNextStepModal: true,
          showLicenseChangeModalState: false,
          licenseChangeFile: null,
          licenseChangeMessage: "",
        });
      })
      .catch((err) => {
        console.log(err);
        setState({
          licenseChangeMessage: err ? err.message : "Something went wrong",
          licenseChangeMessageType: "error",
        });
      })
      .finally(() => {
        setState({ changingLicense: false });
      });
  };

  const hideNextStepModal = () => {
    setState({ showNextStepModal: false });
  };

  const hideLicenseChangeModal = () => {
    setState({
      showLicenseChangeModalState: false,
      licenseChangeFile: null,
      licenseChangeMessage: "",
    });
  };

  const showLicenseChangeModal = () => {
    setState({ showLicenseChangeModalState: true });
  };

  const toggleShowDetails = (entitlement: string) => {
    setState({
      entitlementsToShow: [...state.entitlementsToShow, entitlement],
    });
  };

  const toggleHideDetails = (entitlement: string) => {
    let entitlementsToShow = [...state.entitlementsToShow];
    const index = state.entitlementsToShow.indexOf(entitlement);
    entitlementsToShow.splice(index, 1);
    setState({ entitlementsToShow });
  };

  const viewLicenseEntitlements = () => {
    setState({
      isViewingLicenseEntitlements: !state.isViewingLicenseEntitlements,
    });
  };

  const {
    appLicense,
    loading,
    message,
    messageType,
    showNextStepModal,
    showLicenseChangeModalState,
    licenseChangeFile,
    changingLicense,
    licenseChangeMessage,
    licenseChangeMessageType,
  } = state;

  if (!appLicense) {
    return (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );
  }

  const { app, isEmbeddedCluster } = outletContext;
  const expiresAt = getLicenseExpiryDate(appLicense);
  const gitops = app.downstream?.gitops;
  const appName = app?.name || "Your application";

  let nextModalBody: ReactNode;
  if (gitops?.isConnected) {
    nextModalBody = (
      <div className="Modal-body" data-testid="license-next-step-modal-gitops">
        <p className="u-fontSize--large u-textColor--primary u-lineHeight--medium u-marginBottom--20">
          The license for {appName} has been updated. A new commit has been made
          to the gitops repository with these changes. Please head to the{" "}
          <a
            className="link"
            target="_blank"
            href={gitops?.uri}
            rel="noopener noreferrer"
          >
            repo
          </a>{" "}
          to see the diff.
        </p>
        <div className="flex justifyContent--flexEnd">
          <button
            type="button"
            className="btn blue primary"
            onClick={hideNextStepModal}
          >
            Ok, got it!
          </button>
        </div>
      </div>
    );
  } else {
    nextModalBody = (
      <div className="Modal-body" data-testid="license-next-step-modal">
        <p className="u-fontSize--large u-textColor--primary u-lineHeight--medium u-marginBottom--20">
          The license for {appName} has been updated. A new version is available
          on the version history page with these changes.
        </p>
        <div className="flex justifyContent--flexEnd">
          <button
            type="button"
            className="btn blue secondary u-marginRight--10"
            onClick={hideNextStepModal}
          >
            Cancel
          </button>
          <Link to={`/app/${app?.slug}/version-history`}>
            <button type="button" className="btn blue primary">
              Go to new version
            </button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-column justifyContent--center alignItems--center">
      <KotsPageTitle pageName="License" showAppSlug />
      {size(appLicense) > 0 ? (
        <div
          className="License--wrapper flex-column card-bg"
          data-testid="license-card"
        >
          <div className="flex flex-auto alignItems--center">
            <span className="u-fontSize--large u-fontWeight--bold u-lineHeight--normal card-title">
              {" "}
              License{" "}
            </span>
            {appLicense?.licenseType === "community" && (
              <div className="flex-auto">
                <span className="CommunityEditionTag u-marginLeft--10">
                  Community Edition
                </span>
                <span
                  className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-marginLeft--10"
                  style={{ color: "#A5A5A5" }}
                >
                  {" "}
                  To change your license, please contact your account
                  representative.{" "}
                </span>
              </div>
            )}
          </div>
          <div className="LicenseDetails flex-row card-item">
            <div className=" flex flex1 justifyContent--spaceBetween">
              <div className="flex1 flex-column u-paddingRight--20">
                <div className="flex flex-auto alignItems--center">
                  <span
                    className="u-fontSize--larger u-fontWeight--bold u-lineHeight--normal card-item-title break-word"
                    data-testid="license-customer-name"
                  >
                    {" "}
                    {appLicense.assignee}{" "}
                  </span>
                  {appLicense?.channelName && (
                    <span
                      className="channelTag flex-auto alignItems--center u-fontWeight--medium u-marginLeft--10"
                      data-testid="license-channel-name"
                    >
                      {" "}
                      {appLicense.channelName}{" "}
                    </span>
                  )}
                </div>
                <div className="flex flex1 alignItems--center u-marginTop--5">
                  <div
                    className={`LicenseTypeTag ${appLicense?.licenseType} flex-auto flex-verticalCenter alignItems--center`}
                    data-testid="license-type"
                  >
                    <Icon
                      icon={
                        Utilities.licenseTypeTag(appLicense?.licenseType)
                          .iconName
                      }
                      size={12}
                      style={{ marginRight: "2px" }}
                      className={
                        Utilities.licenseTypeTag(appLicense?.licenseType)
                          .iconColor
                      }
                    />
                    {appLicense?.licenseType !== "---"
                      ? `${Utilities.toTitleCase(
                          appLicense.licenseType
                        )} license`
                      : `---`}
                  </div>
                  <p
                    className={`u-fontWeight--medium u-fontSize--small u-lineHeight--normal u-marginLeft--10 ${
                      Utilities.checkIsDateExpired(expiresAt)
                        ? "u-textColor--error"
                        : "u-textColor--bodyCopy"
                    }`}
                    data-testid="license-expiration-date"
                  >
                    {expiresAt === "Never"
                      ? "Does not expire"
                      : Utilities.checkIsDateExpired(expiresAt)
                      ? `Expired ${expiresAt}`
                      : `Expires ${expiresAt}`}
                  </p>
                </div>

                <div className="flexWrap--wrap flex alignItems--center entitlementItems">
                  {appLicense?.isAirgapSupported ? (
                    <span className="flex alignItems--center">
                      <span className="icon licenseAirgapIcon" /> Airgap enabled{" "}
                    </span>
                  ) : null}
                  {isEmbeddedCluster &&
                  appLicense?.isDisasterRecoverySupported ? (
                    <span className="flex alignItems--center">
                      <span className="icon licenseVeleroIcon" /> Disaster
                      Recovery enabled{" "}
                    </span>
                  ) : null}
                  {!isEmbeddedCluster && appLicense?.isSnapshotSupported ? (
                    <span className="flex alignItems--center">
                      <span className="icon licenseVeleroIcon" /> Snapshots
                      enabled{" "}
                    </span>
                  ) : null}
                  {isEmbeddedCluster &&
                  appLicense?.isEmbeddedClusterMultiNodeEnabled ? (
                    <span className="flex alignItems--center">
                      <span className="icon licenseMultiNodeIcon" />
                      Multi-node cluster enabled{" "}
                    </span>
                  ) : null}
                  {appLicense?.isGitOpsSupported ? (
                    <span className="flex alignItems--center">
                      <Icon
                        icon="github-icon"
                        size={22}
                        className="u-marginRight--5 github-icon"
                        color={""}
                        style={{}}
                        disableFill={false}
                        removeInlineStyle={false}
                      />{" "}
                      GitOps enabled{" "}
                    </span>
                  ) : null}
                  {appLicense?.isIdentityServiceSupported ? (
                    <span className="flex alignItems--center">
                      <span className="icon licenseIdentityIcon" /> Identity
                      Service enabled{" "}
                    </span>
                  ) : null}
                  {appLicense?.isGeoaxisSupported ? (
                    <span className="flex alignItems--center">
                      <span className="icon licenseGeoaxisIcon" /> GEOAxIS
                      Provider enabled{" "}
                    </span>
                  ) : null}
                </div>
              </div>
              <div className="flex-column flex-auto alignItems--flexEnd justifyContent--center">
                <div className="flex alignItems--center">
                  {appLicense?.licenseType === "community" && (
                    <button
                      className="btn secondary blue u-marginRight--10"
                      disabled={changingLicense}
                      onClick={showLicenseChangeModal}
                    >
                      {changingLicense ? "Changing" : "Change license"}
                    </button>
                  )}
                  {app.isAirgap ? (
                    <Dropzone
                      className="Dropzone-wrapper"
                      data-testid="license-upload-dropzone"
                      accept={["application/x-yaml", ".yaml", ".yml"]}
                      onDropAccepted={onDrop}
                      multiple={false}
                    >
                      <button className="btn primary blue" disabled={loading}>
                        {loading ? "Uploading" : "Upload license"}
                      </button>
                    </Dropzone>
                  ) : (
                    <div className="flex flex-column">
                      <button
                        className="btn primary blue"
                        disabled={loading}
                        onClick={() => syncAppLicense("")}
                      >
                        {loading ? "Syncing" : "Sync license"}
                      </button>
                      {appLicense?.id && isServiceAccountToken(appLicense.id) && (
                        <button
                          className="btn secondary blue u-marginTop--10"
                          disabled={loading || state.uploadingServiceAccountToken}
                          onClick={() => setState({ showServiceAccountTokenModal: true })}
                        >
                          Replace service account token
                        </button>
                      )}
                    </div>
                  )}
                </div>
                {message && (
                  <p
                    className={classNames(
                      "u-fontWeight--bold u-fontSize--small u-marginTop--10",
                      {
                        "u-textColor--error": messageType === "error",
                        "u-textColor--primary": messageType === "info",
                      }
                    )}
                  >
                    {message}
                  </p>
                )}
                {appLicense?.lastSyncedAt && (
                  <p className="u-fontWeight--bold u-fontSize--small u-textColor--info u-lineHeight--default u-marginTop--10">
                    Last synced {Utilities.dateFromNow(appLicense.lastSyncedAt)}
                  </p>
                )}
              </div>
            </div>
            {size(appLicense?.entitlements) >= 5 && (
              <span
                className="flexWrap--wrap flex u-fontSize--small u-lineHeight--normal u-color--doveGray u-fontWeight--medium u-marginRight--normal alignItems--center"
                style={{ margin: "10px 0" }}
              >
                <span
                  className={`u-fontWeight--bold u-cursor--pointer`}
                  data-testid="view-license-entitlements-button"
                  style={{ whiteSpace: "pre" }}
                  onClick={(e) => {
                    e.stopPropagation();
                    viewLicenseEntitlements();
                  }}
                >
                  View {size(appLicense?.entitlements)} license entitlements
                  <Icon
                    icon={
                      state.isViewingLicenseEntitlements
                        ? "up-arrow"
                        : "down-arrow"
                    }
                    size={12}
                    className="clickable u-marginLeft--5 gray-color"
                    color={""}
                    style={{}}
                    disableFill={false}
                    removeInlineStyle={false}
                  />
                </span>
              </span>
            )}

            {state.isViewingLicenseEntitlements ? (
              <LicenseFields
                entitlements={appLicense?.entitlements}
                entitlementsToShow={state.entitlementsToShow}
                toggleHideDetails={toggleHideDetails}
                toggleShowDetails={toggleShowDetails}
              />
            ) : (
              appLicense.entitlements.length > 0 &&
              appLicense.entitlements.length < 5 && (
                <div style={{ marginTop: "15px" }}>
                  <LicenseFields
                    entitlements={appLicense?.entitlements}
                    entitlementsToShow={state.entitlementsToShow}
                    toggleHideDetails={toggleHideDetails}
                    toggleShowDetails={toggleShowDetails}
                  />
                </div>
              )
            )}
          </div>
        </div>
      ) : (
        <div>
          <p className="u-fontSize--large u-textColor--bodyCopy u-marginTop--15 u-lineHeight--more">
            {" "}
            License data is not available on this application because it was
            installed via Helm{" "}
          </p>
        </div>
      )}
      <Modal
        isOpen={showNextStepModal}
        onRequestClose={hideNextStepModal}
        shouldReturnFocusAfterClose={false}
        contentLabel="Next step"
        ariaHideApp={false}
        className="Modal MediumSize"
      >
        {nextModalBody}
      </Modal>

      {showLicenseChangeModalState && (
        <Modal
          isOpen={showLicenseChangeModalState}
          onRequestClose={hideLicenseChangeModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Change License"
          ariaHideApp={false}
          className="Modal SmallSize"
        >
          <div className="u-marginTop--10 u-padding--20">
            <p className="u-fontSize--larger u-fontWeight--bold u-textColor--primary u-marginBottom--10">
              Change your license
            </p>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--10">
              The new license must be for the same application as your current
              license.
            </p>
            <div
              className={`FileUpload-wrapper flex1 ${
                licenseChangeFile ? "has-file" : ""
              }`}
            >
              {licenseChangeFile ? (
                <div className="has-file-wrapper">
                  <div className="flex">
                    <Icon
                      icon="yaml-icon"
                      size={24}
                      className="u-marginRight--10 gray-color"
                      color={""}
                      style={{}}
                      disableFill={false}
                      removeInlineStyle={false}
                    />
                    <div>
                      <p className="u-fontSize--normal u-textColor--primary u-fontWeight--medium">
                        {licenseChangeFile.name}
                      </p>
                      <span
                        className="link u-fontSize--small"
                        onClick={clearLicenseChangeFile}
                      >
                        Select a different file
                      </span>
                    </div>
                  </div>
                </div>
              ) : (
                <Dropzone
                  className="Dropzone-wrapper"
                  accept={["application/x-yaml", ".yaml", ".yml"]}
                  onDropAccepted={onLicenseChangeDrop}
                  multiple={false}
                >
                  <div className="u-textAlign--center">
                    <Icon
                      icon="yaml-icon"
                      size={40}
                      className="u-marginBottom--10 gray-color"
                      color={""}
                      style={{}}
                      disableFill={false}
                      removeInlineStyle={false}
                    />
                    <p className="u-fontSize--normal u-textColor--secondary u-fontWeight--medium u-lineHeight--normal">
                      Drag your new license here or{" "}
                      <span className="link u-textDecoration--underlineOnHover">
                        choose a file
                      </span>
                    </p>
                    <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--normal u-lineHeight--normal u-marginTop--10">
                      This will be a .yaml file. Please contact your account rep
                      if you are unable to locate your new license file.
                    </p>
                  </div>
                </Dropzone>
              )}
            </div>
            {licenseChangeMessage && (
              <p
                className={classNames(
                  "u-fontWeight--bold u-fontSize--small u-marginTop--10 u-marginBottom--20",
                  {
                    "u-textColor--error": licenseChangeMessageType === "error",
                    "u-textColor--primary": licenseChangeMessageType === "info",
                  }
                )}
              >
                {licenseChangeMessage}
              </p>
            )}
            <div className="flex flex-auto">
              <button
                type="button"
                className="btn secondary large u-marginRight--10"
                onClick={hideLicenseChangeModal}
              >
                Cancel
              </button>
              {licenseChangeFile && (
                <button
                  type="button"
                  className="btn primary large"
                  onClick={changeAppLicense}
                  disabled={changingLicense}
                >
                  {changingLicense ? "Changing" : "Change license"}
                </button>
              )}
            </div>
          </div>
        </Modal>
      )}

      {state.showServiceAccountTokenModal && (
        <Modal
          isOpen={state.showServiceAccountTokenModal}
          onRequestClose={() => setState({ 
            showServiceAccountTokenModal: false, 
            serviceAccountToken: "" 
          })}
          shouldReturnFocusAfterClose={false}
          contentLabel="Replace service account token"
          ariaHideApp={false}
          className="Modal SmallSize"
        >
          <div className="u-marginTop--10 u-padding--20">
            <p className="u-fontSize--larger u-fontWeight--bold u-textColor--primary u-marginBottom--10">
              Replace service account token
            </p>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--10">
              Enter your new service account token to update the license credentials.
            </p>
            <div className="u-marginBottom--20">
              <div className="FormGroup">
                <label className="field-label">Service Account Token</label>
                <textarea
                  className="Input"
                  placeholder="Paste your service account token here..."
                  value={state.serviceAccountToken}
                  onChange={(e) => setState({ serviceAccountToken: e.target.value })}
                  rows={4}
                />
              </div>
            </div>
            {state.message && (
              <p
                className={classNames(
                  "u-fontWeight--bold u-fontSize--small u-marginTop--10 u-marginBottom--20",
                  {
                    "u-textColor--error": state.messageType === "error",
                    "u-textColor--primary": state.messageType === "info",
                  }
                )}
              >
                {state.message}
              </p>
            )}
            <div className="flex flex-auto">
              <button
                type="button"
                className="btn secondary large u-marginRight--10"
                onClick={() => setState({ 
                  showServiceAccountTokenModal: false, 
                  serviceAccountToken: "" 
                })}
              >
                Cancel
              </button>
              <button
                type="button"
                className="btn primary large"
                disabled={!state.serviceAccountToken.trim() || state.uploadingServiceAccountToken}
                onClick={() => uploadServiceAccountToken(state.serviceAccountToken)}
              >
                {state.uploadingServiceAccountToken ? "Uploading..." : "Upload Token"}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};

export default AppLicenseComponent;

export const CustomerLicenseFields = styled.div`
  background: #f5f8f9;
  border-radius: 6px;
  border: 1px solid #bccacd;
  padding: 10px;
  line-height: 25px;
`;

export const CustomerLicenseField = styled.span`
  margin-right: 15px;
  display: block;
  overflow-wrap: anywhere;
  max-width: 100%;
`;

export const ExpandButton = styled.button`
  background: none;
  border: none;
  color: #007cbb;
  cursor: pointer;
  font-size: 12px;
  padding-left: 0;
`;
