import React, { Component } from "react";
import Select from "react-select";
import { withRouter } from "react-router-dom"
import MonacoEditor from "react-monaco-editor";
import find from "lodash/find";

import ConfigureSnapshots from "./ConfigureSnapshots";

import "../../scss/components/shared/SnapshotForm.scss";

import SnapshotSchedule from "./SnapshotSchedule";

const DESTINATIONS = [
  {
    value: "aws",
    label: "Amazon S3",
  },
  {
    value: "azure",
    label: "Azure Blob Storage",
  },
  {
    value: "gcp",
    label: "Google Cloud Storage",
  },
  {
    value: "other",
    label: "Other S3-Compatible Storage",
  },
  {
    value: "internal",
    label: "Internal Storage (Default)",
  }
];

const AZURE_CLOUD_NAMES = [
  {
    value: "AzurePublicCloud",
    label: "Public",
  },
  {
    value: "AzureUSGovernmentCloud",
    label: "US Government",
  },
  {
    value: "AzureChinaCloud",
    label: "China",
  },
  {
    value: "AzureGermanCloud",
    label: "German",
  }
];

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
  };

  componentDidMount() {
    if (this.props.snapshotSettings) {
      this.setFields();
    }
  }

  componentDidUpdate(lastProps) {
    if (this.props.snapshotSettings !== lastProps.snapshotSettings && this.props.snapshotSettings) {
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
      s3CompatibleEndpoint
    } = this.state;

    const { snapshotSettings } = this.props;

    if (provider === "aws") {
      return (
        snapshotSettings?.store?.aws?.region !== s3Region || snapshotSettings?.store?.aws?.accessKeyID !== s3KeyId ||
        snapshotSettings?.store?.aws?.secretAccessKey !== s3KeySecret || snapshotSettings?.store?.aws?.useInstanceRole !== useIamAws
      )
    }
    if (provider === "gcp") {
      return (snapshotSettings?.store?.gcp?.useInstanceRole !== gcsUseIam || snapshotSettings?.store?.gcp?.serviceAccount !== gcsServiceAccount ||
        snapshotSettings?.store?.gcp?.jsonFile !== gcsJsonFile
      )
    }
    if (provider === "azure") {
      return (
        snapshotSettings?.store?.azure?.resourceGroup !== azureResourceGroupName || snapshotSettings?.store?.azure?.storageAccount !== azureStorageAccountId ||
        snapshotSettings?.store?.azure?.subscriptionId !== azureSubscriptionId || snapshotSettings?.store?.azure?.tenantId !== azureTenantId ||
        snapshotSettings?.store?.azure?.clientId !== azureClientId || snapshotSettings?.store?.azure?.clientSecret !== azureClientSecret ||
        snapshotSettings?.store?.azure?.cloudName !== selectedAzureCloudName.value
      )
    }
    if (provider === "other") {
      return (
        snapshotSettings?.store?.other?.region !== s3CompatibleRegion || snapshotSettings?.store?.other?.accessKeyID !== s3CompatibleKeyId ||
        snapshotSettings?.store?.other?.secretAccessKey !== s3CompatibleKeySecret || snapshotSettings?.store?.other?.endpoint !== s3CompatibleEndpoint
      )
    }
  }

  getCurrentProviderStores = (provider) => {
    const hasChanges = this.checkForStoreChanges(provider);
    if (hasChanges) {
      switch (provider) {
        case "aws":
          return {
            aws: {
              region: this.state.s3Region,
              accessKeyID: !this.state.useIamAws ? this.state.s3KeyId : "",
              secretAccessKey: !this.state.useIamAws ? this.state.s3KeySecret : "",
              useInstanceRole: this.state.useIamAws
            }
          }
        case "azure":
          return {
            azure: {
              resourceGroup: this.state.azureResourceGroupName,
              storageAccount: this.state.azureStorageAccountId,
              subscriptionId: this.state.azureSubscriptionId,
              tenantId: this.state.azureTenantId,
              clientId: this.state.azureClientId,
              clientSecret: this.state.azureClientSecret,
              cloudName: this.state.selectedAzureCloudName.value
            }
          }
        case "gcp":
          return {
            gcp: {
              serviceAccount: this.state.gcsUseIam ? this.state.gcsServiceAccount : "",
              jsonFile: !this.state.gcsUseIam ? this.state.gcsJsonFile : "",
              useInstanceRole: this.state.gcsUseIam
            }
          }
        case "other":
          return {
            other: {
              region: this.state.s3CompatibleRegion,
              accessKeyID: this.state.s3CompatibleKeyId,
              secretAccessKey: this.state.s3CompatibleKeySecret,
              endpoint: this.state.s3CompatibleEndpoint
            }
          }
      }
    }
  }

  setFields = () => {
    const { snapshotSettings } = this.props;
    if (!snapshotSettings) return;
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
        s3KeySecret: store?.aws?.secretAccessKey || ""
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
        selectedAzureCloudName: find(AZURE_CLOUD_NAMES, ["value", store?.azure?.cloudName])
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
        gcsUseIam: store?.gcp?.useInstanceRole
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
        s3CompatibleRegion: store?.other?.region
      });
    }

    if (store?.internal) {
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "internal"])
      });
    }

    // if nothing exists yet, we've determined default state is good
    this.setState({
      determiningDestination: false,
      selectedDestination: find(DESTINATIONS, ["value", "aws"]),
    });
  }

  handleFormChange = (field, e) => {
    let nextState = {};
    if (field === "useIamAws" || field === "gcsUseIam") {
      nextState[field] = e.target.checked;
    } else {
      nextState[field] = e.target.value;
    }
    this.setState(nextState);
  }

  handleDestinationChange = (retentionUnit) => {
    this.setState({ selectedDestination: retentionUnit });
  }

  handleAzureCloudNameChange = (azureCloudName) => {
    this.setState({ selectedAzureCloudName: azureCloudName });
  }

  onGcsEditorChange = (value) => {
    this.setState({ gcsJsonFile: value });
  }

  onSubmit = async (e) => {
    e.preventDefault();
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
        await this.snapshotProviderS3Compatible();
        break;
      case "internal":
        await this.snapshotProviderInternal();
        break;
    }
  }

  getProviderPayload = (provider, bucket, path) => {
    return Object.assign({
      provider,
      bucket,
      path
    }, this.getCurrentProviderStores(provider));
  }

  snapshotProviderAWS = async () => {
    const payload = this.getProviderPayload("aws", this.state.s3bucket, this.state.s3Path);
    this.props.updateSettings(payload);
  }

  snapshotProviderAzure = async () => {
    const payload = this.getProviderPayload("azure", this.state.azureBucket, this.state.azurePath);
    this.props.updateSettings(payload);
  }

  snapshotProviderGoogle = async () => {
    const payload = this.getProviderPayload("gcp", this.state.gcsBucket, this.state.gcsPath);
    this.props.updateSettings(payload);
  }

  snapshotProviderS3Compatible = async () => {
    const payload = this.getProviderPayload("other", this.state.s3CompatibleBucket, this.state.s3CompatiblePath);
    this.props.updateSettings(payload);
  }

  snapshotProviderInternal = async () => {
    const payload = { internal: true };
    this.props.updateSettings(payload);
  }

  renderIcons = (destination) => {
    if (destination) {
      return <span className={`icon snapshotDestination--${destination.value}`} />;
    } else {
      return;
    }
  }

  getDestinationLabel = (destination, label) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span style={{ fontSize: 18, marginRight: "10px", minWidth: 16, textAlign: "center" }}>{this.renderIcons(destination)}</span>
        <span style={{ fontSize: 14, lineHeight: "16px" }}>{label}</span>
      </div>
    );
  }

  renderDestinationFields = () => {
    const { selectedDestination, useIamAws, gcsUseIam } = this.state;
    const selectedAzureCloudName = AZURE_CLOUD_NAMES.find((cn) => {
      return cn.value === this.state.selectedAzureCloudName.value;
    });
    switch (selectedDestination.value) {
      case "aws":
        return (
          <div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Bucket</p>
                <input type="text" className="Input" placeholder="Bucket name" value={this.state.s3bucket} onChange={(e) => { this.handleFormChange("s3bucket", e) }} />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Region</p>
                <input type="text" className="Input" placeholder="Bucket region" value={this.state.s3Region} onChange={(e) => { this.handleFormChange("s3Region", e) }} />
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Path</p>
                <input type="text" className="Input" placeholder="/path/to/destination" value={this.state.s3Path} onChange={(e) => { this.handleFormChange("s3Path", e) }} />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">&nbsp;</p>
                <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left">
                  <div className={`BoxedCheckbox flex-auto flex alignItems--center ${this.state.useIamAws ? "is-active" : ""}`}>
                    <input
                      type="checkbox"
                      className="u-cursor--pointer u-marginLeft--10"
                      id="useIamAws"
                      checked={this.state.useIamAws}
                      onChange={(e) => { this.handleFormChange("useIamAws", e) }}
                    />
                    <label htmlFor="useIamAws" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                      <div className="flex1">
                        <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Use IAM Instance Role</p>
                      </div>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            {!useIamAws &&
              <div className="flex u-marginBottom--30">
                <div className="flex1 u-paddingRight--5">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Access Key ID</p>
                  <input type="text" className="Input" placeholder="key ID" value={this.state.s3KeyId} onChange={(e) => { this.handleFormChange("s3KeyId", e) }} />
                </div>
                <div className="flex1 u-paddingLeft--5">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Access Key Secret</p>
                  <input type="password" className="Input" placeholder="access key" value={this.state.s3KeySecret} onChange={(e) => { this.handleFormChange("s3KeySecret", e) }} />
                </div>
              </div>
            }
          </div>
        )

      case "azure":
        return (
          <div>
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Bucket</p>
              <input type="text" className="Input" placeholder="Bucket name" value={this.state.azureBucket} onChange={(e) => { this.handleFormChange("azureBucket", e) }} />
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Path</p>
                <input type="text" className="Input" placeholder="/path/to/destination" value={this.state.azurePath} onChange={(e) => { this.handleFormChange("azurePath", e) }} />
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Subscription ID</p>
                <input type="text" className="Input" placeholder="Subscription ID" value={this.state.azureSubscriptionId} onChange={(e) => { this.handleFormChange("azureSubscriptionId", e) }} />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Tenant ID</p>
                <input type="text" className="Input" placeholder="Tenant ID" value={this.state.azureTenantId} onChange={(e) => { this.handleFormChange("azureTenantId", e) }} />
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Client ID</p>
                <input type="text" className="Input" placeholder="Client ID" value={this.state.azureClientId} onChange={(e) => { this.handleFormChange("azureClientId", e) }} />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Client Secret</p>
                <input type="password" className="Input" placeholder="Client Secret" value={this.state.azureClientSecret} onChange={(e) => { this.handleFormChange("azureClientSecret", e) }} />
              </div>
            </div>

            <div className="flex-column u-marginBottom--30">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Cloud Name</p>
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
                  isOptionSelected={(option) => { option.value === selectedAzureCloudName }}
                />
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Resource Group Name</p>
                <input type="text" className="Input" placeholder="Resource Group Name" value={this.state.azureResourceGroupName} onChange={(e) => { this.handleFormChange("azureResourceGroupName", e) }} />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Storage Account ID</p>
                <input type="text" className="Input" placeholder="Storage Account ID" value={this.state.azureStorageAccountId} onChange={(e) => { this.handleFormChange("azureStorageAccountId", e) }} />
              </div>
            </div>
          </div>
        )

      case "gcp":
        return (
          <div>
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Bucket</p>
              <input type="text" className="Input" placeholder="Bucket name" value={this.state.gcsBucket} onChange={(e) => { this.handleFormChange("gcsBucket", e) }} />
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Path</p>
                <input type="text" className="Input" placeholder="/path/to/destination" value={this.state.gcsPath} onChange={(e) => { this.handleFormChange("gcsPath", e) }} />
              </div>
            </div>
            <div className="BoxedCheckbox-wrapper u-textAlign--left u-marginBottom--20">
              <div className={`BoxedCheckbox flex-auto flex alignItems--center u-width--half ${this.state.gcsUseIam ? "is-active" : ""}`}>
                <input
                  type="checkbox"
                  className="u-cursor--pointer u-marginLeft--10"
                  id="gcsUseIam"
                  checked={this.state.gcsUseIam}
                  onChange={(e) => { this.handleFormChange("gcsUseIam", e) }}
                />
                <label htmlFor="gcsUseIam" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                  <div className="flex1">
                    <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Use IAM Instance Role</p>
                  </div>
                </label>
              </div>
            </div>

            {gcsUseIam &&
              <div className="flex u-marginBottom--30">
                <div className="flex1 u-paddingRight--5">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Service Account</p>
                  <input type="text" className="Input" placeholder="" value={this.state.gcsServiceAccount} onChange={(e) => { this.handleFormChange("gcsServiceAccount", e) }} />
                </div>
              </div>
            }

            {!gcsUseIam &&
              <div className="flex u-marginBottom--30">
                <div className="flex1">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">JSON File</p>
                  <div className="gcs-editor">
                    <MonacoEditor
                      ref={(editor) => { this.monacoEditor = editor }}
                      language="json"
                      value={this.state.gcsJsonFile}
                      height="420"
                      width="100%"
                      onChange={this.onGcsEditorChange}
                      options={{
                        contextmenu: false,
                        minimap: {
                          enabled: false
                        },
                        scrollBeyondLastLine: false,
                      }}
                    />
                  </div>
                </div>
              </div>
            }
          </div>
        )

      case "other":
        return (
          <div>
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Bucket</p>
              <input type="text" className="Input" placeholder="Bucket name" value={this.state.s3CompatibleBucket} onChange={(e) => { this.handleFormChange("s3CompatibleBucket", e) }} />
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Path</p>
                <input type="text" className="Input" placeholder="/path/to/destination" value={this.state.s3CompatiblePath} onChange={(e) => { this.handleFormChange("s3CompatiblePath", e) }} />
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Access Key ID</p>
                <input type="text" className="Input" placeholder="key ID" value={this.state.s3CompatibleKeyId} onChange={(e) => { this.handleFormChange("s3CompatibleKeyId", e) }} />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Access Key Secret</p>
                <input type="password" className="Input" placeholder="access key" value={this.state.s3CompatibleKeySecret} onChange={(e) => { this.handleFormChange("s3CompatibleKeySecret", e) }} />
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Endpoint</p>
                <input type="text" className="Input" placeholder="endpoint" value={this.state.s3CompatibleEndpoint} onChange={(e) => { this.handleFormChange("s3CompatibleEndpoint", e) }} />
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Region</p>
                <input type="text" className="Input" placeholder="/path/to/destination" value={this.state.s3CompatibleRegion} onChange={(e) => { this.handleFormChange("s3CompatibleRegion", e) }} />
              </div>
            </div>
          </div>
        )

      case "internal":
        return (
          null
        )

      default:
        return (
          <div>No snapshot destination is selected</div>
        )
    }
  }

  render() {
    const { snapshotSettings, updatingSettings, updateConfirm, updateErrorMsg } = this.props;

    const availableDestinations = [];
    if (snapshotSettings?.veleroPlugins) {
      for (const veleroPlugin of snapshotSettings?.veleroPlugins) {
        switch (veleroPlugin) {
          case "velero-plugin-for-gcp":
            availableDestinations.push({
              value: "gcp",
              label: "Google Cloud Storage",
            });
            break;
          case "velero-plugin-for-aws":
            availableDestinations.push({
              value: "aws",
              label: "Amazon S3",
            });
            availableDestinations.push({
              value: "other",
              label: "Other S3-Compatible Storage",
            });
            if (snapshotSettings.isKurl) {
              availableDestinations.push({
                value: "internal",
                label: "Internal Storage (Default)",
              });
            }
            break;
          case "velero-plugin-for-azure":
            availableDestinations.push({
              value: "azure",
              label: "Azure Blob Storage",
            });
            break;
        }
      }
    }

    const selectedDestination = availableDestinations.find((d) => {
      return d.value === this.state.selectedDestination.value;
    });


    return (
      <div className="flex1 flex-column u-marginTop--40">
        <p className="u-fontSize--normal u-marginBottom--15 u-fontWeight--bold u-color--tundora">Snapshot settings</p>
        <div className="flex">
          <div className="flex flex-column">
            <div className="Info--wrapper flex flex-auto u-marginBottom--15">
              <span className="icon info-icon flex-auto u-marginTop--5" />
              <div className="flex flex-column u-marginLeft--5">
                <p className="u-fontSize--normal u-fontWeight--bold u-lineHeight--normal u-color--tuna"> Configuration is shared </p>
                <span className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-color--dustyGray">
                  Full (Instance) and Partial (Application) snapshots share Velero configuration. Your storage destination will be used for both.
            </span>
              </div>
            </div>
            <div className="flex flex-column u-marginRight--50">
              <form className="flex flex-column snapshot-form-wrapper">
                <p className="u-fontSize--normal u-marginBottom--20 u-fontWeight--bold u-color--tundora">Storage</p>
                {updateErrorMsg &&
                  <div className="flex u-fontWeight--bold u-fontSize--small u-color--red u-marginBottom--10">{updateErrorMsg}</div>}
                <div className="flex flex-column u-marginBottom--20">
                  <div className="flex flex1 justifyContent--spaceBetween alignItems--center">
                    <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Destination</p>
                    <span className="replicated-link u-fontSize--normal flex justifyContent--flexEnd u-cursor--pointer" onClick={() => this.props.toggleConfigureModal(this.props.history)}> + Add a new storage destination </span>
                  </div>
                  {!snapshotSettings?.isVeleroRunning &&
                    <div className="flex u-fontWeight--bold u-fontSize--small u-color--red u-marginBottom--10"> Please fix Velero so that the deployment is running. <a href="https://kots.io/kotsadm/snapshots/velero-troubleshooting/" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">View docs</a>  </div>}
                  <div className="flex1">
                    {availableDestinations.length > 1 ?
                      <Select
                        className="replicated-select-container"
                        classNamePrefix="replicated-select"
                        placeholder="Select unit"
                        options={availableDestinations}
                        isSearchable={false}
                        getOptionLabel={(destination) => this.getDestinationLabel(destination, destination.label)}
                        getOptionValue={(destination) => destination.label}
                        value={selectedDestination}
                        onChange={this.handleDestinationChange}
                        isOptionSelected={(option) => { option.value === selectedDestination }}
                      />
                      :
                      availableDestinations.length === 1 ?
                        <div className="u-color--tuna u-fontWeight--medium flex alignItems--center">
                          {this.getDestinationLabel(availableDestinations[0], availableDestinations[0].label)}
                        </div>
                        :
                        null
                    }
                  </div>
                </div>
                {!this.state.determiningDestination &&
                  <div>
                    {this.renderDestinationFields()}
                    <div className="flex">
                      <button className="btn primary blue" disabled={updatingSettings} onClick={this.onSubmit}>{updatingSettings ? "Updating" : "Update storage settings"}</button>
                      {updateConfirm &&
                        <div className="u-marginLeft--10 flex alignItems--center">
                          <span className="icon checkmark-icon" />
                          <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-color--chateauGreen">Settings updated</span>
                        </div>
                      }
                    </div>
                  </div>
                }
                <span className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-color--dustyGray u-marginTop--15"> All data in your snapshots will be deduplicated. To learn more about how,
                <a href="https://kots.io/kotsadm/snapshots/restic-deduplication/" target="_blank" rel="noopener noreferrer" className="replicated-link"> check out the Restic docs</a>. </span>
              </form>
            </div>
          </div>
          <SnapshotSchedule 
            isVeleroRunning={snapshotSettings?.isVeleroRunning} 
            isKurlEnabled={this.props.isKurlEnabled} 
            toggleSnapshotsRBACModal={this.props.toggleSnapshotsRBACModal}
            apps={this.props.apps} />
        </div>
        {this.props.configureSnapshotsModal &&
          <ConfigureSnapshots
            snapshotSettings={this.props.snapshotSettings}
            fetchSnapshotSettings={this.props.fetchSnapshotSettings}
            renderNotVeleroMessage={this.props.renderNotVeleroMessage}
            hideCheckVeleroButton={this.props.hideCheckVeleroButton}
            configureSnapshotsModal={this.props.configureSnapshotsModal}
            toggleConfigureModal={this.props.toggleConfigureModal}
          />}
      </div>
    );
  }
}

export default withRouter(SnapshotStorageDestination);
