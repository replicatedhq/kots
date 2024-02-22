import { ReactNode, useReducer } from "react";
import size from "lodash/size";
// @ts-ignore
import yaml from "js-yaml";
import classNames from "classnames";
import Loader from "@src/components/shared/Loader";
// @ts-ignore
import Dropzone from "react-dropzone";
import Modal from "react-modal";
import {
  getFileContent,
  getLicenseExpiryDate,
  Utilities,
} from "@src/utilities/utilities";
import "@src/scss/components/watches/DashboardCard.scss";
import "@src/scss/components/apps/AppLicense.scss";
import { Link } from "react-router-dom";
import LicenseFields from "./LicenseFields";
import Icon from "@src/components/Icon";
import { App, LicenseFile, AppLicense } from "@src/types/index";

type Props = {
  app: App;
  appLicense: AppLicense | null;
  gettingAppLicenseErrMsg: string | null;
  syncCallback: () => void;
  refetchLicense: () => void;
};

type State = {
  appLicense: AppLicense | null;
  entitlementsToShow: string[];
  isViewingLicenseEntitlements: boolean;
  message: string;
  messageType: string;
  showNextStepModal: boolean;
  syncingLicense: boolean;
};

const DashboardLicenseCard = (props: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      appLicense: null,
      entitlementsToShow: [],
      isViewingLicenseEntitlements: false,
      message: "",
      messageType: "",
      showNextStepModal: false,
      syncingLicense: false,
    }
  );

  const syncLicense = (licenseData: string) => {
    setState({
      syncingLicense: true,
      message: "",
      messageType: "info",
    });

    const { app } = props;

    const payload = {
      licenseData,
    };

    fetch(`${process.env.API_ENDPOINT}/app/${app?.slug}/license`, {
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
        let message;
        if (!licenseResponse.synced) {
          message = "License is already up to date";
        } else if (app?.isAirgap) {
          message = "License uploaded successfully";
        } else {
          message = "License synced successfully";
        }

        props.refetchLicense();
        setState({
          appLicense: licenseResponse.license,
          isViewingLicenseEntitlements:
            size(licenseResponse.license?.entitlements) < 5 ? false : true,
          message,
          messageType: "info",
          showNextStepModal: licenseResponse.synced,
        });

        if (props.syncCallback) {
          props.syncCallback();
        }
      })
      .catch((err) => {
        console.log(err);
        setState({
          message: err ? err.message : "Something went wrong",
          messageType: "error",
        });
      })
      .finally(() => {
        setState({ syncingLicense: false });
        setTimeout(() => {
          setState({
            message: "",
            messageType: "",
          });
        }, 3000);
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

    // @ts-ignore
    // TODO: fix this
    if (airgapLicense.spec?.licenseID !== appLicense?.id) {
      setState({
        message: "Licenses do not match",
        messageType: "error",
      });
      return;
    }

    // @ts-ignore
    // TODO: fix this
    if (airgapLicense.spec?.licenseSequence === appLicense?.licenseSequence) {
      setState({
        message: "License is already up to date",
        messageType: "info",
      });
      return;
    }

    syncLicense(contentStr);
  };

  const hideNextStepModal = () => {
    setState({ showNextStepModal: false });
  };

  const toggleShowDetails = (entitlement: string) => {
    setState({
      entitlementsToShow: [...state.entitlementsToShow, entitlement],
    });
  };

  const toggleHideDetails = (entitlement: string) => {
    let entitlementsToShow = [...state.entitlementsToShow];
    const index = state.entitlementsToShow?.indexOf(entitlement);
    entitlementsToShow.splice(index, 1);
    setState({ entitlementsToShow });
  };

  const viewLicenseEntitlements = () => {
    setState({
      isViewingLicenseEntitlements: !state.isViewingLicenseEntitlements,
    });
  };

  const { app, appLicense, gettingAppLicenseErrMsg } = props;
  const { syncingLicense, showNextStepModal, message, messageType } = state;
  const expiresAt = getLicenseExpiryDate(appLicense);
  const isCommunityLicense = appLicense?.licenseType === "community";
  const gitops = app.downstream?.gitops;
  const appName = app?.name || "Your application";
  const expiredLicenseClassName = Utilities.checkIsDateExpired(expiresAt)
    ? "expired-license"
    : "";
  const appLicenseClassName =
    appLicense && size(appLicense) === 0
      ? "no-license"
      : "dashboard-card card-bg";

  return (
    <div
      className={`${
        isCommunityLicense ? "CommunityLicense--wrapper" : appLicenseClassName
      } ${expiredLicenseClassName} flex-column`}
    >
      <div className="flex flex1 justifyContent--spaceBetween alignItems--center">
        <div className="tw-flex alignItems--center">
          <p
            className={`card-title ${
              Utilities.checkIsDateExpired(expiresAt)
                ? "u-textColor--error"
                : ""
            }`}
          >
            License {Utilities.checkIsDateExpired(expiresAt) && "is expired"}
          </p>
          {isCommunityLicense && (
            <span className="CommunityEditionTag tw-ml-2.5">
              Community Edition
            </span>
          )}
        </div>
        {app?.isAirgap ? (
          <Dropzone
            className="Dropzone-wrapper flex alignItems--center"
            accept={["application/x-yaml", ".yaml", ".yml"]}
            onDropAccepted={onDrop}
            multiple={false}
          >
            <span className="icon clickable dashboard-card-upload-version-icon u-marginRight--5" />
            <span
              className="link u-fontSize--small"
              onClick={() => syncLicense("")}
            >
              Upload license
            </span>
          </Dropzone>
        ) : syncingLicense ? (
          <div className="flex alignItems--center">
            <Loader className="u-marginRight--5" size="15" />
            <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default">
              Syncing license
            </span>
          </div>
        ) : (
          <div className="flex alignItems--center">
            {message && (
              <p
                className={classNames(
                  "u-fontWeight--bold u-fontSize--small u-marginRight--10",
                  {
                    "u-textColor--error": messageType === "error",
                    "u-textColor--info": messageType === "info",
                  }
                )}
              >
                {message}
              </p>
            )}
            {appLicense?.lastSyncedAt && !message ? (
              <span className="u-fontSize--small u-textColor--info u-fontWeight--medium u-lineHeight--normal u-marginRight--10">
                Last synced {Utilities.dateFromNow(appLicense.lastSyncedAt)}
              </span>
            ) : null}
            <div className="flex alignItems--center u-marginRight--20 link">
              <Icon
                icon="sync-license"
                className="clickable u-marginRight--5"
                size={16}
              />
              <span
                className="u-fontSize--small"
                onClick={() => syncLicense("")}
              >
                Sync license
              </span>
            </div>
          </div>
        )}
      </div>
      <div className="card-item u-marginTop--10">
        {size(appLicense) > 0 ? (
          <div className="flex">
            <div className="flex-column flex1">
              <div className="flex alignItems--center">
                <p className="u-fontSize--large u-fontWeight--medium card-item-title break-word">
                  {" "}
                  {appLicense?.assignee}
                </p>
                {appLicense?.channelName && (
                  <span className="channelTag alignItems--center u-fontWeight--medium u-marginLeft--10">
                    {" "}
                    {appLicense.channelName}{" "}
                  </span>
                )}
              </div>
              <div className="flex flex1 alignItems--center u-marginTop--15">
                <div
                  className={`LicenseTypeTag ${appLicense?.licenseType} flex-auto justifyContent--center alignItems--center`}
                >
                  {/* TODO: refactor logic */}
                  <Icon
                    icon={
                      Utilities.licenseTypeTag(appLicense?.licenseType).iconName
                    }
                    size={12}
                    style={{ marginRight: "2px" }}
                    className={
                      Utilities.licenseTypeTag(appLicense?.licenseType)
                        .iconColor
                    }
                  />
                  {appLicense !== null && appLicense?.licenseType !== "---"
                    ? `${Utilities.toTitleCase(appLicense.licenseType)} license`
                    : `---`}
                </div>
                <p
                  className={`u-fontWeight--medium u-fontSize--small u-lineHeight--default u-marginLeft--10 ${
                    Utilities.checkIsDateExpired(expiresAt)
                      ? "u-textColor--error"
                      : "u-textColor--bodyCopy"
                  }`}
                >
                  {expiresAt === "Never"
                    ? "Does not expire"
                    : Utilities.checkIsDateExpired(expiresAt)
                    ? `Expired ${expiresAt}`
                    : `Expires ${expiresAt}`}
                </p>
              </div>
              {size(appLicense?.entitlements) >= 5 && (
                <span
                  className="flexWrap--wrap flex u-fontSize--small u-lineHeight--normal u-color--doveGray u-fontWeight--medium u-marginRight--normal alignItems--center"
                  style={{ margin: "10px 0" }}
                >
                  <span
                    className={`u-fontWeight--bold u-cursor--pointer`}
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
                    />
                  </span>
                </span>
              )}
              {appLicense !== null &&
              appLicense.entitlements.length > 0 &&
              appLicense.entitlements.length < 5 ? (
                <div style={{ marginTop: "15px" }}>
                  <LicenseFields
                    entitlements={appLicense?.entitlements}
                    entitlementsToShow={state.entitlementsToShow}
                    toggleHideDetails={toggleHideDetails}
                    toggleShowDetails={toggleShowDetails}
                  />
                </div>
              ) : (
                appLicense !== null &&
                state.isViewingLicenseEntitlements && (
                  <LicenseFields
                    entitlements={appLicense?.entitlements}
                    entitlementsToShow={state.entitlementsToShow}
                    toggleHideDetails={toggleHideDetails}
                    toggleShowDetails={toggleShowDetails}
                  />
                )
              )}
            </div>
          </div>
        ) : (
          <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal flex">
            {gettingAppLicenseErrMsg}
          </p>
        )}
      </div>
      <div className="u-marginTop--10">
        <Link
          to={`/app/${app?.slug}/license`}
          className="link u-fontSize--small"
        >
          See license details
          <Icon
            icon="next-arrow"
            size={10}
            className="has-arrow u-marginLeft--5"
          />
        </Link>
      </div>
      <Modal
        isOpen={showNextStepModal}
        onRequestClose={hideNextStepModal}
        shouldReturnFocusAfterClose={false}
        contentLabel="Next step"
        ariaHideApp={false}
        className="Modal SmallSize"
      >
        {gitops?.isConnected ? (
          <div className="Modal-body">
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              License synced
            </p>
            <p className="u-fontSize--large u-textColor--bodyCopy u-lineHeight--medium u-marginBottom--20">
              The license for {appName} has been updated. A new commit has been
              made to the gitops repository with these changes. Please head to
              the{" "}
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
            <div>
              <button
                type="button"
                className="btn blue primary"
                onClick={hideNextStepModal}
              >
                Ok, got it!
              </button>
            </div>
          </div>
        ) : (
          <div className="Modal-body">
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              License synced
            </p>
            <p className="u-fontSize--large u-textColor--bodyCopy u-lineHeight--medium u-marginBottom--20">
              The license for {appName} has been updated. A new version is
              available for deploy with these changes from the Version card on
              the dashboard. To see a full list of versions visit the{" "}
              <Link to={`/app/${app?.slug}/version-history`}>
                version history
              </Link>{" "}
              page.
            </p>
            <div>
              <button
                type="button"
                className="btn blue primary"
                onClick={hideNextStepModal}
              >
                Ok, got it!
              </button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default DashboardLicenseCard;
