import React, { ChangeEvent, Component } from "react";
import Select from "react-select";
import MonacoEditor from "@monaco-editor/react";
import find from "lodash/find";
import Modal from "react-modal";

import ConfigureSnapshots from "./ConfigureSnapshots";
import CodeSnippet from "../shared/CodeSnippet";
import Loader from "../shared/Loader";
import { App } from "@types";

import "../../scss/components/shared/SnapshotForm.scss";

import SnapshotSchedule from "./SnapshotSchedule";
import UploadCACertificate from "./UploadCACertificate";
import {
  DESTINATIONS,
  AZURE_CLOUD_NAMES,
  FILE_SYSTEM_NFS_TYPE,
  FILE_SYSTEM_HOSTPATH_TYPE,
} from "./SnapshotStorageDestination.data";
import InputField from "@components/shared/forms/InputField";
import { Utilities } from "@src/utilities/utilities";

type ValueType = {
  value?: string;
  label?: string;
};

type CACertificate = {
  name: string;
  data: Array<string>;
};

type FileSystemProviderInstructionType = "link" | "command";

type FileSystemProviderInstruction = {
  title: string;
  action: string;
  type: FileSystemProviderInstructionType;
};

type State = {
  azureBucket?: string;
  azureClientId: string;
  azureClientSecret: string;
  azurePath?: string;
  azureResourceGroupName: string;
  azureStorageAccountId: string;
  azureSubscriptionId: string;
  azureTenantId: string;
  caCertificate?: CACertificate;
  gettingFileSystemProviderInstructionsErrorMsg?: string;
  fileSystemProviderInstructions?: FileSystemProviderInstruction[];
  gettingFileSystemProviderInstructions?: boolean;
  determiningDestination?: boolean;
  fileSystemHostPath?: string;
  fileSystemNFSPath?: string;
  fileSystemNFSServer?: string;
  fileSystemType?: string;
  gcsBucket?: string;
  gcsJsonFile: string;
  gcsPath?: string;
  gcsServiceAccount: string;
  gcsUseIam: boolean;
  s3bucket?: string;
  s3CompatibleBucket?: string;
  s3CompatibleEndpoint: string;
  s3CompatibleFieldErrors?: { endpoint?: string };
  s3CompatibleKeyId: string;
  s3CompatibleKeySecret: string;
  s3CompatiblePath?: string;
  s3CompatibleRegion: string;
  s3KeyId: string;
  s3KeySecret: string;
  s3Path?: string;
  s3Region: string;
  selectedAzureCloudName: ValueType;
  selectedDestination?: ValueType & {};
  showCACertificateField?: boolean;
  showConfigureFileSystemProviderModal?: boolean;
  showFileSystemProviderInstructionsModal?: boolean;
  tmpFileSystemHostPath?: string;
  tmpFileSystemNFSPath?: string;
  tmpFileSystemNFSServer?: string;
  tmpFileSystemType?: string;
  updatingSettings?: boolean;
  useIamAws: boolean;
  isFirstChange: boolean;
};

type StoreProviderName = "aws" | "gcp" | "azure" | "other";

type AWSStoreProvider = {
  aws: {
    region: string;
    accessKeyID: string;
    secretAccessKey: string;
    useInstanceRole: boolean;
  };
  gcp?: undefined;
  azure?: undefined;
  other?: undefined;
  path?: string;
  bucket?: string;
  internal?: undefined;
  fileSystem?: undefined;
};

type GCPStoreProvider = {
  gcp: {
    jsonFile: string;
    serviceAccount: string;
    useInstanceRole: boolean;
  };
  bucket?: undefined;
  aws?: undefined;
  azure?: undefined;
  other?: undefined;
  path?: undefined;
  internal?: undefined;
  fileSystem?: undefined;
};

type AzureStoreProvider = {
  azure: {
    clientId: string;
    clientSecret: string;
    cloudName: string;
    resourceGroup: string;
    storageAccount: string;
    subscriptionId: string;
    tenantId: string;
  };
  bucket?: undefined;
  aws?: undefined;
  gcp?: undefined;
  other?: undefined;
  path?: undefined;
  internal?: undefined;
  fileSystem?: undefined;
};

type OtherStoreProvider = {
  other: {
    region: string;
    accessKeyID: string;
    accessKeySecret?: string;
    secretAccessKey: string;
    endpoint: string;
  };
  bucket?: undefined;
  aws?: undefined;
  gcp?: undefined;
  azure?: undefined;
  path?: undefined;
  internal?: undefined;
  fileSystem?: undefined;
};

type StoreMetadata = {
  aws?: undefined;
  gcp?: undefined;
  azure?: undefined;
  other?: undefined;
  bucket?: string;
  internal?: undefined;
  fileSystem?: undefined;
  path?: string;
};

type StoreProvider =
  | StoreMetadata
  | AWSStoreProvider
  | GCPStoreProvider
  | AzureStoreProvider
  | OtherStoreProvider;

type FileSystemConfig = {
  nfs?: {
    path: string;
    server: string;
  };
  hostPath?: string;
};

type FileSystemOptions = {
  forceReset?: boolean;
  hostPath?: string;
  nfs?: {
    path?: string;
    server?: string;
  };
};

type ProviderPayload =
  | {
      bucket?: string;
      caCertData?: string;
      fileSystem?: FileSystemOptions;
      path?: string;
      provider?: StoreProviderName;
      internal?: boolean;
    }
  | StoreProvider;

type Props = {
  apps: App[];
  checkForVeleroAndNodeAgent: boolean;
  fetchSnapshotSettings: () => void;
  hideCheckVeleroButton: boolean;
  toggleSnapshotView: (isEmptyView?: boolean) => void;
  hideResetFileSystemWarningModal: () => void;
  isEmptyView?: boolean;
  isLicenseUpload: boolean;
  isKurlEnabled?: boolean;
  isEmbeddedCluster?: boolean;
  kotsadmRequiresVeleroAccess: boolean;
  minimalRBACKotsadmNamespace: string;
  openConfigureSnapshotsMinimalRBACModal: (
    kotsadmRequiresVeleroAccess: boolean,
    minimalRBACKotsadmNamespace: string
  ) => void;
  renderNotVeleroMessage: () => void;
  resetFileSystemWarningMessage: string;
  showConfigureSnapshotsModal: boolean;
  showResetFileSystemWarningModal: boolean;
  snapshotSettings?: {
    fileSystemConfig?: FileSystemConfig;
    isKurl?: boolean;
    isMinioDisabled?: boolean;
    isVeleroRunning?: boolean;
    store?: StoreProvider;
    veleroPlugins?: string[];
    veleroVersion?: string;
  };
  toggleConfigureSnapshotsModal: () => void;
  updateConfirm: boolean;
  updateErrorMsg: string;
  updateSettings: (payload: ProviderPayload) => void;
  updatingSettings: boolean;
};

type FieldName =
  | "azureBucket"
  | "azureClientId"
  | "azureClientSecret"
  | "azurePath"
  | "azureResourceGroupName"
  | "azureStorageAccountId"
  | "azureSubscriptionId"
  | "azureTenantId"
  | "fileSystemHostPath"
  | "fileSystemNFSPath"
  | "fileSystemNFSServer"
  | "gcsBucket"
  | "gcsPath"
  | "gcsServiceAccount"
  | "gcsUseIam"
  | "s3bucket"
  | "s3CompatibleBucket"
  | "s3CompatiblePath"
  | "s3CompatibleKeyId"
  | "s3CompatibleKeySecret"
  | "s3CompatibleEndpoint"
  | "s3CompatibleRegion"
  | "s3KeyId"
  | "s3KeySecret"
  | "s3Region"
  | "s3Path"
  | "useIamAws";

class SnapshotStorageDestination extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      determiningDestination: true,
      selectedDestination: {},
      updatingSettings: false,
      s3bucket: "",
      s3Region: "",
      s3Path: "",
      useIamAws: false,
      s3KeyId: "",
      s3KeySecret: "",
      azureBucket: "",
      azurePath: "",
      azureSubscriptionId: "",
      azureTenantId: "",
      azureClientId: "",
      azureClientSecret: "",
      azureResourceGroupName: "",
      azureStorageAccountId: "",
      selectedAzureCloudName: {
        value: "AzurePublicCloud",
        label: "Public",
      },
      caCertificate: {
        name: "",
        data: [],
      },
      showCACertificateField: false,

      gcsBucket: "",
      gcsPath: "",
      gcsServiceAccount: "",
      gcsJsonFile: "",
      gcsUseIam: false,

      s3CompatibleBucket: "",
      s3CompatiblePath: "",
      s3CompatibleKeyId: "",
      s3CompatibleKeySecret: "",
      s3CompatibleEndpoint: "",
      s3CompatibleRegion: "",
      s3CompatibleFieldErrors: {},

      gettingFileSystemProviderInstructions: false,
      gettingFileSystemProviderInstructionsErrorMsg: "",
      fileSystemProviderInstructions: [],
      showFileSystemProviderInstructionsModal: false,
      showConfigureFileSystemProviderModal: false,

      fileSystemType: "",
      fileSystemNFSPath: "",
      fileSystemNFSServer: "",
      fileSystemHostPath: "",

      tmpFileSystemType: "",
      tmpFileSystemNFSPath: "",
      tmpFileSystemNFSServer: "",
      tmpFileSystemHostPath: "",

      isFirstChange: true,
    };
  }

  static defaultProps = {
    snapshotSettings: {
      store: {},
    },
  };

  componentDidMount() {
    if (this.props.snapshotSettings && !this.props.checkForVeleroAndNodeAgent) {
      this.setFields();
    }
  }

  componentDidUpdate(lastProps: Props) {
    if (
      this.props.snapshotSettings !== lastProps.snapshotSettings &&
      this.props.snapshotSettings &&
      !this.props.checkForVeleroAndNodeAgent
    ) {
      this.setFields();
    }
  }

  checkForStoreChanges = (provider: StoreProviderName) => {
    const {
      s3Region,
      s3KeyId,
      s3KeySecret,
      useIamAws,
      gcsUseIam,
      gcsServiceAccount,
      gcsJsonFile,
      azureResourceGroupName,
      azureStorageAccountId,
      azureSubscriptionId,
      azureTenantId,
      azureClientId,
      azureClientSecret,
      selectedAzureCloudName,
      s3CompatibleRegion,
      s3CompatibleKeyId,
      s3CompatibleKeySecret,
      s3CompatibleEndpoint,
    } = this.state;

    const { snapshotSettings } = this.props;

    if (provider === "aws") {
      if (snapshotSettings?.store?.aws) {
        return (
          snapshotSettings?.store?.aws.region !== s3Region ||
          snapshotSettings?.store?.aws.accessKeyID !== s3KeyId ||
          snapshotSettings?.store?.aws.secretAccessKey !== s3KeySecret ||
          snapshotSettings?.store?.aws.useInstanceRole !== useIamAws
        );
      }
      return true;
    }
    if (provider === "gcp") {
      if (snapshotSettings?.store?.gcp) {
        return (
          snapshotSettings?.store?.gcp.useInstanceRole !== gcsUseIam ||
          snapshotSettings?.store?.gcp.serviceAccount !== gcsServiceAccount ||
          snapshotSettings?.store?.gcp.jsonFile !== gcsJsonFile
        );
      }
      return true;
    }
    if (provider === "azure") {
      return (
        snapshotSettings?.store?.azure?.resourceGroup !==
          azureResourceGroupName ||
        snapshotSettings?.store?.azure?.storageAccount !==
          azureStorageAccountId ||
        snapshotSettings?.store?.azure?.subscriptionId !==
          azureSubscriptionId ||
        snapshotSettings?.store?.azure?.tenantId !== azureTenantId ||
        snapshotSettings?.store?.azure?.clientId !== azureClientId ||
        snapshotSettings?.store?.azure?.clientSecret !== azureClientSecret ||
        snapshotSettings?.store?.azure?.cloudName !==
          selectedAzureCloudName.value
      );
    }
    if (provider === "other") {
      return (
        snapshotSettings?.store?.other?.region !== s3CompatibleRegion ||
        snapshotSettings?.store?.other?.accessKeyID !== s3CompatibleKeyId ||
        snapshotSettings?.store?.other?.secretAccessKey !==
          s3CompatibleKeySecret ||
        snapshotSettings?.store?.other?.endpoint !== s3CompatibleEndpoint
      );
    }
  };

  getCurrentProviderStores = (
    provider: StoreProviderName
  ): StoreProvider | null => {
    const hasChanges = this.checkForStoreChanges(provider);
    if (hasChanges) {
      switch (provider) {
        case "aws":
          return {
            aws: {
              region: this.state.s3Region,
              accessKeyID: !this.state.useIamAws ? this.state.s3KeyId : "",
              secretAccessKey: !this.state.useIamAws
                ? this.state.s3KeySecret
                : "",
              useInstanceRole: this.state.useIamAws,
            },
          };
        case "azure":
          return {
            azure: {
              resourceGroup: this.state.azureResourceGroupName,
              storageAccount: this.state.azureStorageAccountId,
              subscriptionId: this.state.azureSubscriptionId,
              tenantId: this.state.azureTenantId,
              clientId: this.state.azureClientId,
              clientSecret: this.state.azureClientSecret,
              cloudName: this.state.selectedAzureCloudName.value || "",
            },
          };
        case "gcp":
          return {
            gcp: {
              serviceAccount: this.state.gcsUseIam
                ? this.state.gcsServiceAccount
                : "",
              jsonFile: !this.state.gcsUseIam ? this.state.gcsJsonFile : "",
              useInstanceRole: this.state.gcsUseIam,
            },
          };
        case "other":
          return {
            other: {
              region: this.state.s3CompatibleRegion,
              accessKeyID: this.state.s3CompatibleKeyId,
              secretAccessKey: this.state.s3CompatibleKeySecret,
              endpoint: this.state.s3CompatibleEndpoint,
            },
          };
        default:
          console.error(new Error("Unknown provider"));
          return null;
      }
    }
    return null;
  };

  setFields = () => {
    const { snapshotSettings } = this.props;
    if (!snapshotSettings) {
      return;
    }
    const { store } = snapshotSettings;

    if (store?.aws) {
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "aws"]),
        s3bucket: store?.bucket || "",
        s3Region: store.aws.region,
        s3Path: store.path,
        useIamAws: store.aws.useInstanceRole,
        s3KeyId: store.aws.accessKeyID || "",
        s3KeySecret: store.aws.secretAccessKey || "",
      });
    }

    if (store?.azure) {
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "azure"]),
        azureBucket: store.bucket,
        azurePath: store.path,
        azureSubscriptionId: store?.azure?.subscriptionId,
        azureTenantId: store?.azure?.tenantId,
        azureClientId: store?.azure?.clientId,
        azureClientSecret: store?.azure?.clientSecret,
        azureResourceGroupName: store?.azure?.resourceGroup,
        azureStorageAccountId: store?.azure?.storageAccount,
        selectedAzureCloudName: find(AZURE_CLOUD_NAMES, [
          "value",
          store?.azure?.cloudName,
        ]) || {
          value: "AzurePublicCloud",
          label: "Public",
        },
      });
    }

    if (store?.gcp) {
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "gcp"]),
        gcsBucket: store.bucket,
        gcsPath: store.path,
        gcsServiceAccount: store?.gcp?.serviceAccount || "",
        gcsJsonFile: store?.gcp?.jsonFile || "",
        gcsUseIam: store?.gcp?.useInstanceRole,
      });
    }

    if (store?.other) {
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "other"]),
        s3CompatibleBucket: store.bucket,
        s3CompatiblePath: store.path,
        s3CompatibleKeyId: store?.other?.accessKeyID,
        s3CompatibleKeySecret: store?.other?.accessKeySecret || "",
        s3CompatibleEndpoint: store?.other?.endpoint,
        s3CompatibleRegion: store?.other?.region,
      });
    }

    if (store?.internal) {
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "internal"]),
      });
    }

    if (store?.fileSystem) {
      const { fileSystemConfig } = snapshotSettings;
      return this.setState({
        determiningDestination: false,
        selectedDestination: fileSystemConfig?.hostPath
          ? find(DESTINATIONS, ["value", "hostpath"])
          : find(DESTINATIONS, ["value", "nfs"]),
        fileSystemType: fileSystemConfig?.hostPath
          ? FILE_SYSTEM_HOSTPATH_TYPE
          : FILE_SYSTEM_NFS_TYPE,
        fileSystemNFSPath: fileSystemConfig?.nfs?.path,
        fileSystemNFSServer: fileSystemConfig?.nfs?.server,
        fileSystemHostPath: fileSystemConfig?.hostPath,
      });
    }

    // if nothing exists yet, we've determined default state is good
    let defaultDestination = "aws";
    if (this.props.isEmbeddedCluster) {
      defaultDestination = "other";
    }
    this.setState({
      determiningDestination: false,
      selectedDestination: find(DESTINATIONS, ["value", defaultDestination]),
    });
  };

  handleFormChange = (field: FieldName, e: ChangeEvent<HTMLInputElement>) => {
    let nextState: {
      [K in FieldName]?: string | boolean;
    } = {};
    if (field === "useIamAws" || field === "gcsUseIam") {
      nextState[field] = e.target.checked;
    } else {
      nextState[field] = e.target.value;
    }
    // TODO: make this more explicit
    // @ts-ignore
    this.setState(nextState);
  };

  // TODO: upgrade react-select and use it's latest Option type
  handleDestinationChange = (destination: { value: string }) => {
    const fileSystemType =
      destination?.value === "hostpath"
        ? FILE_SYSTEM_HOSTPATH_TYPE
        : destination?.value === "nfs"
        ? FILE_SYSTEM_NFS_TYPE
        : "";
    this.setState({
      selectedDestination: destination,
      fileSystemType: fileSystemType,
    });
  };

  handleAzureCloudNameChange = (azureCloudName: ValueType) => {
    this.setState({ selectedAzureCloudName: azureCloudName });
  };

  handleCACertificateFieldClick = () => {
    this.setState({ showCACertificateField: true });
  };

  onGcsEditorChange = (value: string | undefined) => {
    this.setState({ gcsJsonFile: value || "" });
  };

  onSubmit = async (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    let s3CompatibleFieldErrors = this.state.s3CompatibleFieldErrors;
    switch (this.state.selectedDestination?.value) {
      case "aws":
        await this.snapshotProviderAWS();
        break;
      case "azure":
        await this.snapshotProviderAzure();
        break;
      case "gcp":
        await this.snapshotProviderGoogle();
        break;
      case "other":
        s3CompatibleFieldErrors = this.validateSnapshotProviderS3Compatible();
        this.setState({ s3CompatibleFieldErrors });
        if (Object.keys(s3CompatibleFieldErrors).length > 0) {
          break;
        }
        await this.snapshotProviderS3Compatible();
        break;
      case "internal":
        await this.snapshotProviderInternal();
        break;
      case "nfs":
      case "hostpath":
        await this.snapshotProviderFileSystem(false);
        break;
    }
    this.setState({
      isFirstChange: false,
    });
  };

  validateSnapshotProviderS3Compatible = () => {
    const urlRe =
      /\b(https?):\/\/[-A-Za-z0-9+&@#/%?=~_|!:,.;]*[-A-Za-z0-9+&@#/%=~_|]/;

    // TODO: clean up the state so we don't need to check for string here
    if (
      typeof this.state?.s3CompatibleEndpoint === "string" &&
      !urlRe.test(this.state?.s3CompatibleEndpoint)
    ) {
      return { endpoint: "Please enter a valid endpoint with protocol" };
    }
    return {};
  };

  getProviderPayload = (
    provider: StoreProviderName,
    bucket?: string,
    path?: string
  ): ProviderPayload => {
    const caCertData = this.state?.caCertificate?.data;
    return Object.assign(
      {
        provider,
        bucket,
        path,
        caCertData,
      },
      this.getCurrentProviderStores(provider)
    );
  };

  handleSetCACert = (caCertificate: CACertificate) => {
    this.setState({ caCertificate });
  };

  snapshotProviderAWS = async () => {
    const payload = this.getProviderPayload(
      "aws",
      this.state?.s3bucket,
      this.state?.s3Path
    );
    this.props.updateSettings(payload);
  };

  snapshotProviderAzure = async () => {
    const payload = this.getProviderPayload(
      "azure",
      this.state.azureBucket,
      this.state.azurePath
    );
    this.props.updateSettings(payload);
  };

  snapshotProviderGoogle = async () => {
    const payload = this.getProviderPayload(
      "gcp",
      this.state.gcsBucket,
      this.state.gcsPath
    );
    this.props.updateSettings(payload);
  };

  snapshotProviderS3Compatible = async () => {
    const payload = this.getProviderPayload(
      "other",
      this.state.s3CompatibleBucket,
      this.state.s3CompatiblePath
    );
    this.props.updateSettings(payload);
  };

  snapshotProviderInternal = async () => {
    const payload = { internal: true };
    this.props.updateSettings(payload);
  };

  snapshotProviderFileSystem = async (forceReset = false) => {
    if (forceReset) {
      this.props.hideResetFileSystemWarningModal();
    }

    const type = this.state.fileSystemType;
    const path = this.state.fileSystemNFSPath;
    const server = this.state.fileSystemNFSServer;
    const hostPath = this.state.fileSystemHostPath;

    const payload = {
      fileSystem: this.buildFileSystemOptions(
        type,
        path,
        server,
        hostPath,
        forceReset
      ),
    };
    this.props.updateSettings(payload);
  };

  buildFileSystemOptions = (
    type?: string,
    path?: string,
    server?: string,
    hostPath?: string,
    forceReset?: boolean
  ): FileSystemOptions => {
    const options: FileSystemOptions = {
      forceReset: forceReset,
    };
    if (type === FILE_SYSTEM_HOSTPATH_TYPE) {
      options.hostPath = hostPath;
    } else if (type === FILE_SYSTEM_NFS_TYPE) {
      options.nfs = {
        path: path,
        server: server,
      };
    }
    return options;
  };

  openConfigureFileSystemProviderModal = (fileSystemType: string) => {
    this.setState({
      showConfigureFileSystemProviderModal:
        !this.state.showConfigureFileSystemProviderModal,
      tmpFileSystemType: fileSystemType,
    });
  };

  hideConfigureFileSystemProviderModal = () => {
    this.setState({ showConfigureFileSystemProviderModal: false });
  };

  hideConfigureFileSystemProviderInstructionsModal = () => {
    this.setState({ showFileSystemProviderInstructionsModal: false });
  };

  getFileSystemProviderInstructions = () => {
    const type = this.state.tmpFileSystemType;
    const path = this.state.tmpFileSystemNFSPath;
    const server = this.state.tmpFileSystemNFSServer;
    const hostPath = this.state.tmpFileSystemHostPath;
    const fileSystemOptions = this.buildFileSystemOptions(
      type,
      path,
      server,
      hostPath,
      false
    );

    this.setState({
      gettingFileSystemProviderInstructions: true,
      gettingFileSystemProviderInstructionsErrorMsg: "",
    });

    fetch(`${process.env.API_ENDPOINT}/snapshots/filesystem/instructions`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify({
        fileSystemOptions: fileSystemOptions,
      }),
    })
      .then(async (res) => {
        const response = await res.json();
        if (!res.ok) {
          this.setState({
            gettingFileSystemProviderInstructions: false,
            gettingFileSystemProviderInstructionsErrorMsg: response.error,
          });
          return;
        }

        if (response.success) {
          this.setState({
            gettingFileSystemProviderInstructions: false,
            showConfigureFileSystemProviderModal: false,
            showFileSystemProviderInstructionsModal: true,
            gettingFileSystemProviderInstructionsErrorMsg: "",
            fileSystemProviderInstructions: response.instructions,
          });
          return;
        }

        this.setState({
          gettingFileSystemProviderInstructions: false,
          gettingFileSystemProviderInstructionsErrorMsg: response.error,
        });
      })
      .catch((err) => {
        console.error(err);
        this.setState({
          gettingFileSystemProviderInstructions: false,
          gettingFileSystemProviderInstructionsErrorMsg:
            "Something went wrong, please try again.",
        });
      });
  };

  renderIcons = (destination: ValueType) => {
    if (destination) {
      return (
        <span className={`icon snapshotDestination--${destination.value}`} />
      );
    }
    return;
  };

  getDestinationLabel = (destination: ValueType, label: string) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span
          style={{
            fontSize: 18,
            marginRight: "10px",
            minWidth: 16,
            textAlign: "center",
          }}
        >
          {this.renderIcons(destination)}
        </span>
        <span style={{ fontSize: 14, lineHeight: "16px" }}>{label}</span>
      </div>
    );
  };

  renderDestinationFields = () => {
    const { selectedDestination, useIamAws, gcsUseIam } = this.state;
    const selectedAzureCloudName = AZURE_CLOUD_NAMES.find((cn) => {
      return cn.value === this.state?.selectedAzureCloudName?.value;
    });
    switch (selectedDestination?.value) {
      case "aws":
        return (
          <>
            <div className="flex u-marginBottom--15">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Bucket
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="Bucket name"
                  value={this.state.s3bucket}
                  onChange={(e) => this.handleFormChange("s3bucket", e)}
                />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Region
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="Bucket region"
                  value={this.state.s3Region}
                  onChange={(e) => this.handleFormChange("s3Region", e)}
                />
              </div>
            </div>
            <div className="flex flex-column u-marginBottom--30">
              <div className="u-marginBottom--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Prefix
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="/path/to/destination"
                  value={this.state.s3Path}
                  onChange={(e) => this.handleFormChange("s3Path", e)}
                  style={{ width: "49%" }}
                />
              </div>
              <div>
                <div
                  className={`flex-auto flex alignItems--center ${
                    this.state.useIamAws ? "is-active" : ""
                  }`}
                >
                  <input
                    type="checkbox"
                    className="u-cursor--pointer"
                    id="useIamAws"
                    checked={this.state.useIamAws}
                    onChange={(e) => this.handleFormChange("useIamAws", e)}
                  />
                  <label
                    htmlFor="useIamAws"
                    className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                  >
                    <div className="flex1">
                      <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium">
                        Use IAM Role
                      </p>
                    </div>
                  </label>
                </div>
              </div>
            </div>

            {!useIamAws && (
              <div className="flex u-marginBottom--5">
                <div className="flex1 u-paddingRight--5">
                  <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                    Access Key ID
                  </p>
                  <input
                    type="text"
                    className="Input"
                    placeholder="key ID"
                    value={this.state.s3KeyId}
                    onChange={(e) => this.handleFormChange("s3KeyId", e)}
                  />
                </div>
                <div className="flex1 u-paddingLeft--5">
                  <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                    Access Key Secret
                  </p>
                  <InputField
                    type="password"
                    placeholder="access key"
                    value={this.state.s3KeySecret}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      this.handleFormChange("s3KeySecret", e)
                    }
                    onFocus={undefined}
                    onBlur={undefined}
                    className={"tw-gap-0"}
                    isFirstChange={this.state.isFirstChange}
                    label={undefined}
                    id={"access-key"}
                    autoFocus={undefined}
                    helperText={undefined}
                  />
                </div>
              </div>
            )}
          </>
        );

      case "azure":
        return (
          <>
            <div className="flex1 u-paddingRight--5 u-marginBottom--15">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Bucket
              </p>
              <input
                type="text"
                className="Input"
                placeholder="Bucket name"
                value={this.state.azureBucket}
                onChange={(e) => this.handleFormChange("azureBucket", e)}
              />
            </div>
            <div className="flex u-marginBottom--15">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Prefix
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="/path/to/destination"
                  value={this.state.azurePath}
                  onChange={(e) => this.handleFormChange("azurePath", e)}
                />
              </div>
            </div>
            <div className="flex u-marginBottom--15">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Subscription ID
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="Subscription ID"
                  value={this.state.azureSubscriptionId}
                  onChange={(e) =>
                    this.handleFormChange("azureSubscriptionId", e)
                  }
                />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Tenant ID
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="Tenant ID"
                  value={this.state.azureTenantId}
                  onChange={(e) => this.handleFormChange("azureTenantId", e)}
                />
              </div>
            </div>
            <div className="flex u-marginBottom--15">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Client ID
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="Client ID"
                  value={this.state.azureClientId}
                  onChange={(e) => this.handleFormChange("azureClientId", e)}
                />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Client Secret
                </p>

                <InputField
                  type="password"
                  placeholder="Client Secret"
                  value={this.state.azureClientSecret}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    this.handleFormChange("azureClientSecret", e)
                  }
                  onFocus={undefined}
                  onBlur={undefined}
                  className={"tw-gap-0"}
                  isFirstChange={this.state.isFirstChange}
                  label={undefined}
                  id={"client-secret"}
                  autoFocus={undefined}
                  helperText={undefined}
                />
              </div>
            </div>

            <div className="flex-column u-marginBottom--15">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Cloud Name
              </p>
              <div className="flex1">
                <Select
                  className="replicated-select-container"
                  classNamePrefix="replicated-select"
                  placeholder="Select unit"
                  options={AZURE_CLOUD_NAMES}
                  isSearchable={false}
                  getOptionValue={(cloudName) => cloudName.label}
                  value={selectedAzureCloudName}
                  // TODO: upgrade react-select and fix this
                  // @ts-ignore
                  onChange={this.handleAzureCloudNameChange}
                  isOptionSelected={(option) =>
                    // TODO: fix this
                    // @ts-ignore
                    option.value === selectedAzureCloudName
                  }
                />
              </div>
            </div>
            <div className="flex u-marginBottom--5">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Resource Group Name
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="Resource Group Name"
                  value={this.state.azureResourceGroupName}
                  onChange={(e) =>
                    this.handleFormChange("azureResourceGroupName", e)
                  }
                />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Storage Account ID
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="Storage Account ID"
                  value={this.state.azureStorageAccountId}
                  onChange={(e) =>
                    this.handleFormChange("azureStorageAccountId", e)
                  }
                />
              </div>
            </div>
          </>
        );

      case "gcp":
        return (
          <div>
            <div className="flex1 u-paddingRight--5 u-marginBottom--15">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Bucket
              </p>
              <input
                type="text"
                className="Input"
                placeholder="Bucket name"
                value={this.state.gcsBucket}
                onChange={(e) => this.handleFormChange("gcsBucket", e)}
              />
            </div>
            <div className="flex u-marginBottom--5">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Prefix
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="/path/to/destination"
                  value={this.state.gcsPath}
                  onChange={(e) => this.handleFormChange("gcsPath", e)}
                />
              </div>
            </div>
            <div className="BoxedCheckbox-wrapper u-textAlign--left u-marginBottom--15">
              <div
                className={`flex-auto flex alignItems--center u-width--half ${
                  this.state.gcsUseIam ? "is-active" : ""
                }`}
              >
                <input
                  type="checkbox"
                  className="u-cursor--pointer"
                  id="gcsUseIam"
                  checked={this.state.gcsUseIam}
                  onChange={(e) => this.handleFormChange("gcsUseIam", e)}
                />
                <label
                  htmlFor="gcsUseIam"
                  className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                >
                  <div className="flex1">
                    <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium">
                      Use IAM Instance Role
                    </p>
                  </div>
                </label>
              </div>
            </div>

            {gcsUseIam && (
              <div className="flex u-marginBottom--5">
                <div className="flex1 u-paddingRight--5">
                  <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                    Service Account
                  </p>
                  <input
                    type="text"
                    className="Input"
                    placeholder=""
                    value={this.state.gcsServiceAccount}
                    onChange={(e) =>
                      this.handleFormChange("gcsServiceAccount", e)
                    }
                  />
                </div>
              </div>
            )}

            {!gcsUseIam && (
              <div className="flex u-marginBottom--5">
                <div className="flex1">
                  <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                    JSON File
                  </p>
                  <div className="gcs-editor">
                    <MonacoEditor
                      language="json"
                      value={this.state.gcsJsonFile}
                      height="420px"
                      onChange={this.onGcsEditorChange}
                      options={{
                        contextmenu: false,
                        minimap: {
                          enabled: false,
                        },
                        scrollBeyondLastLine: false,
                      }}
                    />
                  </div>
                </div>
              </div>
            )}
          </div>
        );

      case "other":
        return (
          <div>
            <div className="flex1 u-paddingRight--5 u-marginBottom--15">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Bucket
              </p>
              <input
                type="text"
                className="Input"
                placeholder="Bucket name"
                value={this.state.s3CompatibleBucket}
                onChange={(e) => this.handleFormChange("s3CompatibleBucket", e)}
              />
            </div>
            <div className="flex u-marginBottom--15">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Prefix
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="/path/to/destination"
                  value={this.state.s3CompatiblePath}
                  onChange={(e) => this.handleFormChange("s3CompatiblePath", e)}
                />
              </div>
            </div>
            <div className="flex u-marginBottom--15">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Access Key ID
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="key ID"
                  value={this.state.s3CompatibleKeyId}
                  onChange={(e) =>
                    this.handleFormChange("s3CompatibleKeyId", e)
                  }
                />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Access Key Secret
                </p>

                <InputField
                  type="password"
                  placeholder="access key"
                  value={this.state.s3CompatibleKeySecret}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    this.handleFormChange("s3CompatibleKeySecret", e)
                  }
                  onFocus={undefined}
                  onBlur={undefined}
                  className={"tw-gap-0"}
                  isFirstChange={this.state.isFirstChange}
                  label={undefined}
                  id={"s3-access-key"}
                  autoFocus={undefined}
                  helperText={undefined}
                />
              </div>
            </div>
            <div className="u-marginBottom--5">
              <div className="flex">
                <div className="flex1 u-paddingRight--5">
                  <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                    Endpoint
                  </p>
                  <input
                    type="text"
                    className="Input"
                    placeholder="http[s]://hostname[:port]"
                    value={this.state.s3CompatibleEndpoint}
                    onChange={(e) =>
                      this.handleFormChange("s3CompatibleEndpoint", e)
                    }
                  />
                </div>
                <div className="flex1 u-paddingLeft--5">
                  <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                    Region
                  </p>
                  <input
                    type="text"
                    className="Input"
                    placeholder="us-east-1"
                    value={this.state.s3CompatibleRegion}
                    onChange={(e) =>
                      this.handleFormChange("s3CompatibleRegion", e)
                    }
                  />
                </div>
              </div>
              {this.state?.s3CompatibleFieldErrors?.endpoint && (
                <div className="u-fontWeight--bold u-fontSize--small u-textColor--error u-marginBottom--10 u-marginTop--10">
                  {this.state?.s3CompatibleFieldErrors?.endpoint}
                </div>
              )}
            </div>
          </div>
        );

      case "internal":
        return null;

      case "nfs":
        return (
          <div className="flex u-marginBottom--5">
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Server
              </p>
              <input
                key="filesystem-nfs-server"
                type="text"
                className="Input"
                placeholder="NFS server hostname/IP"
                value={this.state.fileSystemNFSServer}
                onChange={(e) =>
                  this.handleFormChange("fileSystemNFSServer", e)
                }
              />
            </div>
            <div className="flex1 u-paddingLeft--5">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Path
              </p>
              <input
                key="filesystem-nfs-path"
                type="text"
                className="Input"
                placeholder="/path/to/nfs-directory"
                value={this.state.fileSystemNFSPath}
                onChange={(e) => this.handleFormChange("fileSystemNFSPath", e)}
              />
            </div>
          </div>
        );

      case "hostpath":
        return (
          <div className="flex u-marginBottom--5">
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Host Path
              </p>
              <input
                key="filesystem-hostpath"
                type="text"
                className="Input"
                placeholder="/path/to/host-directory"
                value={this.state.fileSystemHostPath}
                onChange={(e) => this.handleFormChange("fileSystemHostPath", e)}
              />
            </div>
          </div>
        );

      default:
        return <div>No snapshot destination is selected</div>;
    }
  };

  renderConfigureFileSystemProviderModalContent = () => {
    if (this.state.tmpFileSystemType === FILE_SYSTEM_HOSTPATH_TYPE) {
      return (
        <div className="Modal-body">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--secondary u-marginBottom--10">
            Configure Host Path
          </p>
          <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--10">
            Enter the host path for the directory in which you would like to
            store the snapshots.
          </p>
          <div className="flex u-marginBottom--30">
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Host Path
              </p>
              <input
                type="text"
                className="Input"
                placeholder="/path/to/host-directory"
                value={this.state.tmpFileSystemHostPath}
                onChange={(e) =>
                  this.setState({ tmpFileSystemHostPath: e.target.value })
                }
              />
            </div>
          </div>
          <div className="flex justifyContent--flexStart alignItems-center">
            {this.state.gettingFileSystemProviderInstructions && (
              <Loader className="u-marginRight--5" size="32" />
            )}
            <button
              type="button"
              className="btn blue primary u-marginRight--10"
              onClick={this.getFileSystemProviderInstructions}
              disabled={
                !this.state.tmpFileSystemHostPath ||
                this.state.gettingFileSystemProviderInstructions
              }
            >
              {this.state.gettingFileSystemProviderInstructions
                ? "Getting instructions"
                : "Get instructions"}
            </button>
            <button
              type="button"
              className="btn secondary"
              onClick={this.hideConfigureFileSystemProviderModal}
            >
              Cancel
            </button>
          </div>
          {this.state.gettingFileSystemProviderInstructionsErrorMsg && (
            <div className="flex u-fontWeight--bold u-fontSize--small u-textColor--error u-marginBottom--10 u-marginTop--10">
              {this.state.gettingFileSystemProviderInstructionsErrorMsg}
            </div>
          )}
        </div>
      );
    }

    if (this.state.tmpFileSystemType === FILE_SYSTEM_NFS_TYPE) {
      return (
        <div className="Modal-body">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--secondary u-marginBottom--10">
            Configure NFS
          </p>
          <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--10">
            Enter the NFS server hostname or IP address, and the exported
            directory path in which you would like to store the snapshots.
          </p>
          <div className="flex u-marginBottom--30">
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Server
              </p>
              <input
                type="text"
                className="Input"
                placeholder="NFS server hostname/IP"
                value={this.state.tmpFileSystemNFSServer}
                onChange={(e) =>
                  this.setState({ tmpFileSystemNFSServer: e.target.value })
                }
              />
            </div>
            <div className="flex1 u-paddingLeft--5">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Path
              </p>
              <input
                type="text"
                className="Input"
                placeholder="/path/to/nfs-directory"
                value={this.state.tmpFileSystemNFSPath}
                onChange={(e) =>
                  this.setState({ tmpFileSystemNFSPath: e.target.value })
                }
              />
            </div>
          </div>
          <div className="flex justifyContent--flexStart alignItems-center">
            {this.state.gettingFileSystemProviderInstructions && (
              <Loader className="u-marginRight--5" size="32" />
            )}
            <button
              type="button"
              className="btn blue primary u-marginRight--10"
              disabled={
                !this.state.tmpFileSystemNFSServer ||
                !this.state.tmpFileSystemNFSPath ||
                this.state.gettingFileSystemProviderInstructions
              }
              onClick={this.getFileSystemProviderInstructions}
            >
              {this.state.gettingFileSystemProviderInstructions
                ? "Getting instructions"
                : "Get instructions"}
            </button>
            <button
              type="button"
              className="btn secondary"
              onClick={this.hideConfigureFileSystemProviderModal}
            >
              Cancel
            </button>
          </div>
          {this.state.gettingFileSystemProviderInstructionsErrorMsg && (
            <div className="flex u-fontWeight--bold u-fontSize--small u-textColor--error u-marginBottom--10 u-marginTop--10">
              {this.state.gettingFileSystemProviderInstructionsErrorMsg}
            </div>
          )}
        </div>
      );
    }

    return null;
  };

  renderFileSystemProviderInstructions = () => {
    const instructions = this.state.fileSystemProviderInstructions;
    if (!instructions?.length) {
      return null;
    }

    return instructions.map((instruction, index) => {
      let action;
      if (instruction.type === "link") {
        action = (
          <span className="link u-fontSize--small u-cursor--pointer">
            <a href={instruction.action} target="_blank" className="link">
              {instruction.action}
            </a>
          </span>
        );
      } else {
        action = (
          <CodeSnippet
            language="bash"
            canCopy={true}
            onCopyText={
              <span className="u-textColor--success">
                Snippet has been copied to your clipboard
              </span>
            }
          >
            {instruction.action}
          </CodeSnippet>
        );
      }
      return (
        <div key={`${index}`} className="flex flex1 u-marginTop--20">
          <div className="flex">
            <span className="circleNumberGray u-marginRight--10">
              {" "}
              {index + 1}{" "}
            </span>
          </div>
          <div className="flex flex-column">
            <p className="u-fontSize--small flex alignItems--center u-fontWeight--medium u-lineHeight--medium u-textColor--bodyCopy">
              {" "}
              {instruction.title}{" "}
            </p>
            <div className="flex u-marginTop--5">{action}</div>
          </div>
        </div>
      );
    });
  };

  render() {
    const {
      snapshotSettings,
      updatingSettings,
      updateConfirm,
      updateErrorMsg,
      isEmbeddedCluster,
      isKurlEnabled,
      checkForVeleroAndNodeAgent,
    } = this.props;

    const availableDestinations = [];
    if (snapshotSettings?.veleroPlugins) {
      for (const veleroPlugin of snapshotSettings?.veleroPlugins) {
        if (isEmbeddedCluster) {
          if (veleroPlugin.includes("velero-plugin-for-aws")) {
            availableDestinations.push({
              value: "other",
              label: "S3-Compatible Storage",
            });
          }
          continue;
        }

        if (veleroPlugin.includes("velero-plugin-for-gcp")) {
          availableDestinations.push({
            value: "gcp",
            label: "Google Cloud Storage",
          });
        } else if (veleroPlugin.includes("velero-plugin-for-aws")) {
          availableDestinations.push({
            value: "aws",
            label: "Amazon S3",
          });
          availableDestinations.push({
            value: "other",
            label: "S3-Compatible Storage",
          });
          if (snapshotSettings.isKurl && !snapshotSettings?.isMinioDisabled) {
            availableDestinations.push({
              value: "internal",
              label: "Internal Storage (Default)",
            });
          }
          // Checks for legacy behavior where minio was used for hostpath and nfs
          if (!snapshotSettings?.isMinioDisabled) {
            availableDestinations.push({
              value: "nfs",
              label: "Network File System (NFS)",
            });
            availableDestinations.push({
              value: "hostpath",
              label: "Host Path",
            });
          }
        } else if (veleroPlugin.includes("velero-plugin-for-microsoft-azure")) {
          availableDestinations.push({
            value: "azure",
            label: "Azure Blob Storage",
          });
        } else if (
          veleroPlugin.includes("local-volume-provider") &&
          snapshotSettings?.isMinioDisabled
        ) {
          availableDestinations.push({
            value: "nfs",
            label: "Network File System (NFS)",
          });
          availableDestinations.push({
            value: "hostpath",
            label: "Host Path",
          });
          if (snapshotSettings.isKurl) {
            availableDestinations.push({
              value: "internal",
              label: "Internal Storage (Default)",
            });
          }
        }
      }
      availableDestinations.sort((a, b) => a.label.localeCompare(b.label));
    }

    const selectedDestination = availableDestinations.find(
      (d) => d.value === this.state?.selectedDestination?.value
    );

    const showResetFileSystemWarningModal =
      this.props.showResetFileSystemWarningModal;
    const resetFileSystemWarningMessage =
      this.props.resetFileSystemWarningMessage;

    let featureName = "snapshot";
    if (isEmbeddedCluster) {
      featureName = "backup";
    }

    return (
      <div className="flex1 flex-column u-marginTop--40">
        <div className="flex" style={{ gap: "30px" }}>
          <div
            className="flex flex-column card-bg u-padding--15"
            style={{ maxWidth: "400px" }}
          >
            <div className="flex justifyContent--spaceBetween">
              <p className="card-title u-paddingBottom--15">
                {Utilities.toTitleCase(featureName)} settings
              </p>
              <div>
                {!isEmbeddedCluster && (
                  <span
                    className="link u-fontSize--small flex justifyContent--flexEnd u-cursor--pointer"
                    onClick={this.props.toggleConfigureSnapshotsModal}
                  >
                    + Add a new destination
                  </span>
                )}
              </div>
            </div>
            {!isEmbeddedCluster && (
              <div className="flex flex-auto u-marginBottom--15">
                <div className="flex flex-column">
                  <span className="u-fontSize--normal u-fontWeight--normal u-lineHeight--normal u-textColor--bodyCopy">
                    Full (Instance) and Partial (Application) snapshots share
                    the same Velero configuration and storage destination.
                  </span>
                </div>
              </div>
            )}
            <div className="flex flex-column card-item u-padding--15">
              <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Destination
              </p>
              <form className="flex flex-column">
                {updateErrorMsg && (
                  <div className="flex-auto u-fontWeight--bold u-fontSize--small u-textColor--error u-marginBottom--10">
                    {updateErrorMsg}
                  </div>
                )}
                <div className="flex flex-column u-marginBottom--15">
                  {!snapshotSettings?.isVeleroRunning &&
                    !checkForVeleroAndNodeAgent &&
                    isKurlEnabled && (
                      <div className="flex-auto u-fontWeight--bold u-fontSize--small u-textColor--error u-marginBottom--10">
                        Please fix Velero so that the deployment is running. For
                        help troubleshooting this issue visit{" "}
                        <a
                          href="https://velero.io/docs/main/troubleshooting/"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="link u-marginLeft--5"
                        >
                          https://velero.io/docs/main/troubleshooting/
                        </a>
                        .
                      </div>
                    )}
                  <div className="flex1">
                    {availableDestinations.length > 1 ? (
                      <Select
                        className="replicated-select-container"
                        classNamePrefix="replicated-select"
                        placeholder="Select unit"
                        options={availableDestinations}
                        isSearchable={false}
                        getOptionLabel={(destination) =>
                          // TODO: upgrade react-select and use the current typing
                          // We want to display element instead of string
                          // @ts-ignore
                          this.getDestinationLabel(
                            destination,
                            destination.label
                          )
                        }
                        getOptionValue={(destination) => destination.label}
                        value={selectedDestination}
                        onChange={this.handleDestinationChange}
                        isOptionSelected={(option) => {
                          // TODO: fix this is probably a bug
                          // @ts-ignore
                          return option.value === selectedDestination;
                        }}
                      />
                    ) : availableDestinations.length === 1 ? (
                      <div className="u-textColor--primary u-fontWeight--medium flex alignItems--center">
                        {this.getDestinationLabel(
                          availableDestinations[0],
                          availableDestinations[0].label
                        )}
                      </div>
                    ) : null}
                  </div>
                </div>
                {!this.state.determiningDestination && (
                  <>
                    {this.renderDestinationFields()}
                    {this.state.showCACertificateField && (
                      <UploadCACertificate
                        certificate={this.state.caCertificate}
                        handleSetCACert={this.handleSetCACert}
                      />
                    )}
                    {!this.state.showCACertificateField && (
                      <button
                        className="AddCAButton link u-fontSize--small"
                        onClick={this.handleCACertificateFieldClick}
                      >
                        + Add a CA Certificate
                      </button>
                    )}
                    <div className="flex">
                      <button
                        className="btn primary blue"
                        disabled={updatingSettings}
                        onClick={this.onSubmit}
                      >
                        {updatingSettings
                          ? "Updating"
                          : "Update storage settings"}
                      </button>
                      {updatingSettings && (
                        <Loader className="u-marginLeft--10" size="32" />
                      )}
                      {updateConfirm && (
                        <div className="u-marginLeft--10 flex alignItems--center">
                          <span className="icon checkmark-icon" />
                          <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-textColor--success">
                            Settings updated
                          </span>
                        </div>
                      )}
                    </div>
                  </>
                )}
                {!isEmbeddedCluster && (
                  <span className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-textColor--bodyCopy u-marginTop--15">
                    All data in your snapshots will be deduplicated. Snapshots
                    makes use of Restic, a fast and secure backup technology
                    with native deduplication.
                  </span>
                )}
              </form>
            </div>
          </div>
          <SnapshotSchedule
            apps={this.props.apps}
            isKurlEnabled={this.props.isKurlEnabled}
            isEmbeddedCluster={isEmbeddedCluster}
            isVeleroRunning={snapshotSettings?.isVeleroRunning}
            isVeleroInstalled={!!snapshotSettings?.veleroVersion}
            updatingSettings={updatingSettings}
            openConfigureSnapshotsMinimalRBACModal={
              this.props.openConfigureSnapshotsMinimalRBACModal
            }
          />
        </div>
        {this.props.showConfigureSnapshotsModal && (
          <ConfigureSnapshots
            snapshotSettings={this.props.snapshotSettings}
            fetchSnapshotSettings={this.props.fetchSnapshotSettings}
            renderNotVeleroMessage={this.props.renderNotVeleroMessage}
            hideCheckVeleroButton={this.props.hideCheckVeleroButton}
            showConfigureSnapshotsModal={this.props.showConfigureSnapshotsModal}
            toggleConfigureSnapshotsModal={
              this.props.toggleConfigureSnapshotsModal
            }
            kotsadmRequiresVeleroAccess={this.props.kotsadmRequiresVeleroAccess}
            minimalRBACKotsadmNamespace={this.props.minimalRBACKotsadmNamespace}
            openConfigureFileSystemProviderModal={
              this.openConfigureFileSystemProviderModal
            }
            isKurlEnabled={isKurlEnabled}
          />
        )}
        {this.state.showConfigureFileSystemProviderModal && (
          <Modal
            isOpen={this.state.showConfigureFileSystemProviderModal}
            onRequestClose={this.hideConfigureFileSystemProviderModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="Configure File System backend"
            ariaHideApp={false}
            className="Modal SmallSize"
          >
            {this.renderConfigureFileSystemProviderModalContent()}
          </Modal>
        )}
        {this.state.showFileSystemProviderInstructionsModal && (
          <Modal
            isOpen={this.state.showFileSystemProviderInstructionsModal}
            onRequestClose={
              this.hideConfigureFileSystemProviderInstructionsModal
            }
            shouldReturnFocusAfterClose={false}
            contentLabel="File system next steps"
            ariaHideApp={false}
            className="Modal SmallSize"
          >
            <div className="Modal-body">
              <p className="u-fontSize--largest u-fontWeight--bold u-textColor--secondary u-marginBottom--10">
                Velero installation instructions
              </p>
              {this.renderFileSystemProviderInstructions()}
              <div className="u-marginTop--20 flex justifyContent--flexStart">
                <button
                  type="button"
                  className="btn blue primary"
                  onClick={
                    this.hideConfigureFileSystemProviderInstructionsModal
                  }
                >
                  Ok, got it!
                </button>
              </div>
            </div>
          </Modal>
        )}
        {showResetFileSystemWarningModal && (
          <Modal
            isOpen={showResetFileSystemWarningModal}
            onRequestClose={this.props.hideResetFileSystemWarningModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="Reset file system config"
            ariaHideApp={false}
            className="Modal MediumSize"
          >
            <div className="Modal-body">
              <p className="u-fontSize--large u-textColor--error u-marginBottom--20">
                {resetFileSystemWarningMessage} Would you like to continue?
              </p>
              <div className="u-marginTop--10 flex justifyContent--flexStart">
                <button
                  type="button"
                  className="btn blue primary u-marginRight--10"
                  onClick={() => this.snapshotProviderFileSystem(true)}
                >
                  Yes
                </button>
                <button
                  type="button"
                  className="btn secondary"
                  onClick={this.props.hideResetFileSystemWarningModal}
                >
                  No
                </button>
              </div>
            </div>
          </Modal>
        )}
      </div>
    );
  }
}

export default SnapshotStorageDestination;
