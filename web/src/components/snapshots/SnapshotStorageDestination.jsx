import React, { Component } from "react";
import Select from "react-select";
import { withRouter } from "react-router-dom";
import MonacoEditor from "@monaco-editor/react";
import find from "lodash/find";
import Modal from "react-modal";

import ConfigureSnapshots from "./ConfigureSnapshots";
import CodeSnippet from "../shared/CodeSnippet";
import Loader from "../shared/Loader";
import { Utilities } from "../../utilities/utilities";

import "../../scss/components/shared/SnapshotForm.scss";

import SnapshotSchedule from "./SnapshotSchedule";
import UploadCACertificate from "./UploadCACertificate";
import {
  DESTINATIONS,
  AZURE_CLOUD_NAMES,
  FILE_SYSTEM_NFS_TYPE,
  FILE_SYSTEM_HOSTPATH_TYPE,
} from "./SnapshotStorageDestination.data";

class SnapshotStorageDestination extends Component {
  state = {
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

    configuringFileSystemProvider: false,
    configureFileSystemProviderErrorMsg: "",
    configureFileSystemProviderNamespace: "",
    showConfigureFileSystemProviderNextStepsModal: false,
    showConfigureFileSystemProviderModal: false,
    showResetFileSystemWarningModal: false,
    resetFileSystemWarningMessage: "",

    fileSystemType: "",
    fileSystemNFSPath: "",
    fileSystemNFSServer: "",
    fileSystemHostPath: "",

    tmpFileSystemType: "",
    tmpFileSystemNFSPath: "",
    tmpFileSystemNFSServer: "",
    tmpFileSystemHostPath: "",
  };

  componentDidMount() {
    if (this.props.snapshotSettings && !this.props.checkForVeleroAndRestic) {
      this.setFields();
    }
  }

  componentDidUpdate(lastProps) {
    if (
      this.props.snapshotSettings !== lastProps.snapshotSettings &&
      this.props.snapshotSettings &&
      !this.props.checkForVeleroAndRestic
    ) {
      this.setFields();
    }
  }

  checkForStoreChanges = (provider) => {
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
      return (
        snapshotSettings?.store?.aws?.region !== s3Region ||
        snapshotSettings?.store?.aws?.accessKeyID !== s3KeyId ||
        snapshotSettings?.store?.aws?.secretAccessKey !== s3KeySecret ||
        snapshotSettings?.store?.aws?.useInstanceRole !== useIamAws
      );
    }
    if (provider === "gcp") {
      return (
        snapshotSettings?.store?.gcp?.useInstanceRole !== gcsUseIam ||
        snapshotSettings?.store?.gcp?.serviceAccount !== gcsServiceAccount ||
        snapshotSettings?.store?.gcp?.jsonFile !== gcsJsonFile
      );
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

  getCurrentProviderStores = (provider) => {
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
              cloudName: this.state.selectedAzureCloudName.value,
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
      }
    }
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
        s3bucket: store.bucket,
        s3Region: store?.aws?.region,
        s3Path: store.path,
        useIamAws: store?.aws?.useInstanceRole,
        s3KeyId: store?.aws?.accessKeyID || "",
        s3KeySecret: store?.aws?.secretAccessKey || "",
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
        ]),
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
        s3CompatibleKeySecret: store?.other?.accessKeySecret,
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
    this.setState({
      determiningDestination: false,
      selectedDestination: find(DESTINATIONS, ["value", "aws"]),
    });
  };

  handleFormChange = (field, e) => {
    let nextState = {};
    if (field === "useIamAws" || field === "gcsUseIam") {
      nextState[field] = e.target.checked;
    } else {
      nextState[field] = e.target.value;
    }
    this.setState(nextState);
  };

  handleDestinationChange = (destination) => {
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

  handleAzureCloudNameChange = (azureCloudName) => {
    this.setState({ selectedAzureCloudName: azureCloudName });
  };

  handleCACertificateFieldClick = () => {
    this.setState({ showCACertificateField: true });
  };

  onGcsEditorChange = (value) => {
    this.setState({ gcsJsonFile: value });
  };

  onSubmit = async (e) => {
    e.preventDefault();
    let s3CompatibleFieldErrors = this.state.s3CompatibleFieldErrors;
    switch (this.state.selectedDestination.value) {
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
  };

  validateSnapshotProviderS3Compatible = () => {
    const urlRe =
      /\b(https?):\/\/[-A-Za-z0-9+&@#/%?=~_|!:,.;]*[-A-Za-z0-9+&@#/%=~_|]/;

    if (!urlRe.test(this.state.s3CompatibleEndpoint)) {
      return { endpoint: "Please enter a valid endpoint with protocol" };
    }
    return {};
  };

  getProviderPayload = (provider, bucket, path) => {
    const caCertData = this.state.caCertificate.data;
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

  handleSetCACert = (caCertificate) => {
    this.setState({ caCertificate });
  };

  snapshotProviderAWS = async () => {
    const payload = this.getProviderPayload(
      "aws",
      this.state.s3bucket,
      this.state.s3Path
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
      this.hideResetFileSystemWarningModal();
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

  buildFileSystemOptions = (type, path, server, hostPath, forceReset) => {
    const options = {
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

  openConfigureFileSystemProviderModal = (fileSystemType) => {
    this.setState({
      showConfigureFileSystemProviderModal:
        !this.state.showConfigureFileSystemProviderModal,
      tmpFileSystemType: fileSystemType,
    });
  };

  hideConfigureFileSystemProviderModal = () => {
    this.setState({ showConfigureFileSystemProviderModal: false });
  };

  hideConfigureFileSystemProviderNextStepsModal = () => {
    this.setState({ showConfigureFileSystemProviderNextStepsModal: false });
  };

  hideResetFileSystemWarningModal = () => {
    this.setState({ showResetFileSystemWarningModal: false });
    if (this.props.hideResetFileSystemWarningModal) {
      this.props.hideResetFileSystemWarningModal();
    }
  };

  configureFileSystemProvider = (forceReset = false) => {
    if (forceReset) {
      this.hideResetFileSystemWarningModal();
    }

    const type = this.state.tmpFileSystemType;
    const path = this.state.tmpFileSystemNFSPath;
    const server = this.state.tmpFileSystemNFSServer;
    const hostPath = this.state.tmpFileSystemHostPath;
    const fileSystemOptions = this.buildFileSystemOptions(
      type,
      path,
      server,
      hostPath,
      forceReset
    );

    this.setState({
      configuringFileSystemProvider: true,
      configureFileSystemProviderErrorMsg: "",
    });

    fetch(`${process.env.API_ENDPOINT}/snapshots/filesystem`, {
      method: "PUT",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        fileSystemOptions: fileSystemOptions,
      }),
    })
      .then(async (res) => {
        if (res.status === 409) {
          const response = await res.json();
          this.setState({
            configuringFileSystemProvider: false,
            showResetFileSystemWarningModal: true,
            resetFileSystemWarningMessage: response.error,
          });
          return;
        }

        const response = await res.json();
        if (!res.ok) {
          this.setState({
            configuringFileSystemProvider: false,
            configureFileSystemProviderErrorMsg: response.error,
          });
          return;
        }

        if (response.success) {
          this.setState({
            configuringFileSystemProvider: false,
            showConfigureFileSystemProviderModal: false,
            showConfigureFileSystemProviderNextStepsModal: true,
            configureFileSystemProviderErrorMsg: "",
            configureFileSystemProviderNamespace: response.namespace,
          });
          return;
        }

        this.setState({
          configuringFileSystemProvider: false,
          configureFileSystemProviderErrorMsg: response.error,
        });
      })
      .catch((err) => {
        console.error(err);
        this.setState({
          configuringFileSystemProvider: false,
          configureFileSystemProviderErrorMsg:
            "Something went wrong, please try again.",
        });
      });
  };

  renderIcons = (destination) => {
    if (destination) {
      return (
        <span className={`icon snapshotDestination--${destination.value}`} />
      );
    }
    return;
  };

  getDestinationLabel = (destination, label) => {
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
      return cn.value === this.state.selectedAzureCloudName.value;
    });
    switch (selectedDestination.value) {
      case "aws":
        return (
          <>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Path
                </p>
                <input
                  type="text"
                  className="Input"
                  placeholder="/path/to/destination"
                  value={this.state.s3Path}
                  onChange={(e) => this.handleFormChange("s3Path", e)}
                />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  &nbsp;
                </p>
                <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left">
                  <div
                    className={`BoxedCheckbox flex-auto flex alignItems--center ${
                      this.state.useIamAws ? "is-active" : ""
                    }`}
                  >
                    <input
                      type="checkbox"
                      className="u-cursor--pointer u-marginLeft--10"
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
                          Use IAM Instance Role
                        </p>
                      </div>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            {!useIamAws && (
              <div className="flex u-marginBottom--30">
                <div className="flex1 u-paddingRight--5">
                  <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
                  <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                    Access Key Secret
                  </p>
                  <input
                    type="password"
                    className="Input"
                    placeholder="access key"
                    value={this.state.s3KeySecret}
                    onChange={(e) => this.handleFormChange("s3KeySecret", e)}
                  />
                </div>
              </div>
            )}
          </>
        );

      case "azure":
        return (
          <>
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Path
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
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Client Secret
                </p>
                <input
                  type="password"
                  className="Input"
                  placeholder="Client Secret"
                  value={this.state.azureClientSecret}
                  onChange={(e) =>
                    this.handleFormChange("azureClientSecret", e)
                  }
                />
              </div>
            </div>

            <div className="flex-column u-marginBottom--30">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
                  onChange={this.handleAzureCloudNameChange}
                  isOptionSelected={(option) =>
                    option.value === selectedAzureCloudName
                  }
                />
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Path
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
            <div className="BoxedCheckbox-wrapper u-textAlign--left u-marginBottom--20">
              <div
                className={`BoxedCheckbox flex-auto flex alignItems--center u-width--half ${
                  this.state.gcsUseIam ? "is-active" : ""
                }`}
              >
                <input
                  type="checkbox"
                  className="u-cursor--pointer u-marginLeft--10"
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
              <div className="flex u-marginBottom--30">
                <div className="flex1 u-paddingRight--5">
                  <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
              <div className="flex u-marginBottom--30">
                <div className="flex1">
                  <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                    JSON File
                  </p>
                  <div className="gcs-editor">
                    <MonacoEditor
                      ref={(editor) => (this.monacoEditor = editor)}
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
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Path
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
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
                <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                  Access Key Secret
                </p>
                <input
                  type="password"
                  className="Input"
                  placeholder="access key"
                  value={this.state.s3CompatibleKeySecret}
                  onChange={(e) =>
                    this.handleFormChange("s3CompatibleKeySecret", e)
                  }
                />
              </div>
            </div>
            <div className="u-marginBottom--30">
              <div className="flex">
                <div className="flex1 u-paddingRight--5">
                  <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
                  <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
              {this.state.s3CompatibleFieldErrors.endpoint && (
                <div className="u-fontWeight--bold u-fontSize--small u-textColor--error u-marginBottom--10 u-marginTop--10">
                  {this.state.s3CompatibleFieldErrors.endpoint}
                </div>
              )}
            </div>
          </div>
        );

      case "internal":
        return null;

      case "nfs":
        return (
          <div className="flex u-marginBottom--30">
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
          <div className="flex u-marginBottom--30">
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
            {this.state.configuringFileSystemProvider && (
              <Loader className="u-marginRight--5" size="32" />
            )}
            <button
              type="button"
              className="btn blue primary u-marginRight--10"
              onClick={() => this.configureFileSystemProvider(false)}
              disabled={
                !this.state.tmpFileSystemHostPath ||
                this.state.configuringFileSystemProvider
              }
            >
              {this.state.configuringFileSystemProvider
                ? "Configuring"
                : "Configure"}
            </button>
            <button
              type="button"
              className="btn secondary"
              onClick={this.hideConfigureFileSystemProviderModal}
            >
              Cancel
            </button>
          </div>
          {this.state.configureFileSystemProviderErrorMsg && (
            <div className="flex u-fontWeight--bold u-fontSize--small u-textColor--error u-marginBottom--10 u-marginTop--10">
              {this.state.configureFileSystemProviderErrorMsg}
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
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
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
            {this.state.configuringFileSystemProvider && (
              <Loader className="u-marginRight--5" size="32" />
            )}
            <button
              type="button"
              className="btn blue primary u-marginRight--10"
              disabled={
                !this.state.tmpFileSystemNFSServer ||
                !this.state.tmpFileSystemNFSPath ||
                this.state.configuringFileSystemProvider
              }
              onClick={() => this.configureFileSystemProvider(false)}
            >
              {this.state.configuringFileSystemProvider
                ? "Configuring"
                : "Configure"}
            </button>
            <button
              type="button"
              className="btn secondary"
              onClick={this.hideConfigureFileSystemProviderModal}
            >
              Cancel
            </button>
          </div>
          {this.state.configureFileSystemProviderErrorMsg && (
            <div className="flex u-fontWeight--bold u-fontSize--small u-textColor--error u-marginBottom--10 u-marginTop--10">
              {this.state.configureFileSystemProviderErrorMsg}
            </div>
          )}
        </div>
      );
    }

    return null;
  };

  render() {
    const {
      snapshotSettings,
      updatingSettings,
      updateConfirm,
      updateErrorMsg,
      isKurlEnabled,
      checkForVeleroAndRestic,
    } = this.props;

    const availableDestinations = [];
    if (snapshotSettings?.veleroPlugins) {
      for (const veleroPlugin of snapshotSettings?.veleroPlugins) {
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
            label: "Other S3-Compatible Storage",
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
      (d) => d.value === this.state.selectedDestination.value
    );

    const showResetFileSystemWarningModal =
      this.state.showResetFileSystemWarningModal ||
      this.props.showResetFileSystemWarningModal;
    const resetFileSystemWarningMessage =
      this.state.resetFileSystemWarningMessage ||
      this.props.resetFileSystemWarningMessage;

    return (
      <div className="flex1 flex-column u-marginTop--40">
        <p className="u-fontSize--normal u-marginBottom--15 u-fontWeight--bold u-textColor--secondary">
          Snapshot settings
        </p>
        <div className="flex">
          <div className="flex flex-column">
            <div className="Info--wrapper flex flex-auto u-marginBottom--15">
              <span className="icon info-icon flex-auto u-marginTop--5" />
              <div className="flex flex-column u-marginLeft--5">
                <p className="u-fontSize--normal u-fontWeight--bold u-lineHeight--normal u-textColor--primary">
                  Configuration is shared
                </p>
                <span className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-textColor--bodyCopy">
                  Full (Instance) and Partial (Application) snapshots share
                  Velero configuration. Your storage destination will be used
                  for both.
                </span>
              </div>
            </div>
            <div className="flex flex-column u-marginRight--50">
              <form className="flex flex-column snapshot-form-wrapper">
                <p className="u-fontSize--normal u-marginBottom--20 u-fontWeight--bold u-textColor--secondary">
                  Storage
                </p>
                {updateErrorMsg && (
                  <div className="flex-auto u-fontWeight--bold u-fontSize--small u-textColor--error u-marginBottom--10">
                    {updateErrorMsg}
                  </div>
                )}
                <div className="flex flex-column u-marginBottom--20">
                  <div className="flex flex1 justifyContent--spaceBetween alignItems--center">
                    <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                      Destination
                    </p>
                    <span
                      className="replicated-link u-fontSize--normal flex justifyContent--flexEnd u-cursor--pointer"
                      onClick={this.props.toggleConfigureSnapshotsModal}
                    >
                      + Add a new storage destination
                    </span>
                  </div>
                  {!snapshotSettings?.isVeleroRunning &&
                    !checkForVeleroAndRestic &&
                    isKurlEnabled && (
                      <div className="flex-auto u-fontWeight--bold u-fontSize--small u-textColor--error u-marginBottom--10">
                        Please fix Velero so that the deployment is running. For
                        help troubleshooting this issue visit{" "}
                        <a
                          href="https://velero.io/docs/main/troubleshooting/"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="replicated-link u-marginLeft--5"
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
                          this.getDestinationLabel(
                            destination,
                            destination.label
                          )
                        }
                        getOptionValue={(destination) => destination.label}
                        value={selectedDestination}
                        onChange={this.handleDestinationChange}
                        isOptionSelected={(option) => {
                          option.value === selectedDestination;
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
                        className="AddCAButton replicated-link u-fontSize--normal"
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
                <span className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-textColor--bodyCopy u-marginTop--15">
                  All data in your snapshots will be deduplicated. Snapshots
                  makes use of Restic, a fast and secure backup technology with
                  native deduplication.
                </span>
              </form>
            </div>
          </div>
          <SnapshotSchedule
            apps={this.props.apps}
            isKurlEnabled={this.props.isKurlEnabled}
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

        {this.state.showConfigureFileSystemProviderNextStepsModal && (
          <Modal
            isOpen={this.state.showConfigureFileSystemProviderNextStepsModal}
            onRequestClose={this.hideConfigureFileSystemProviderNextStepsModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="File system next steps"
            ariaHideApp={false}
            className="Modal SmallSize"
          >
            <div className="Modal-body">
              <p className="u-fontSize--largest u-fontWeight--bold u-textColor--secondary u-marginBottom--10">
                Next steps
              </p>
              <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
                Run the following command for instructions on how to set up
                Velero:
              </p>
              <CodeSnippet
                language="bash"
                canCopy={true}
                onCopyText={
                  <span className="u-textColor--success">
                    Command has been copied to your clipboard
                  </span>
                }
              >
                {`kubectl kots velero print-fs-instructions --namespace ${this.state.configureFileSystemProviderNamespace}`}
              </CodeSnippet>
              <div className="u-marginTop--10 flex justifyContent--flexStart">
                <button
                  type="button"
                  className="btn blue primary"
                  onClick={this.hideConfigureFileSystemProviderNextStepsModal}
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
            onRequestClose={this.hideResetFileSystemWarningModal}
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
                  onClick={
                    this.state.showConfigureFileSystemProviderModal
                      ? () => this.configureFileSystemProvider(true)
                      : () => this.snapshotProviderFileSystem(true)
                  }
                >
                  Yes
                </button>
                <button
                  type="button"
                  className="btn secondary"
                  onClick={this.hideResetFileSystemWarningModal}
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

export default withRouter(SnapshotStorageDestination);
