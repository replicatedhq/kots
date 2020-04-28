import React, { Component } from "react";
import Select from "react-select";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom"
import MonacoEditor from "react-monaco-editor";
import Helmet from "react-helmet";
import find from "lodash/find";

import Loader from "../shared/Loader";
import "../../scss/components/shared/SnapshotForm.scss";
import { Utilities } from "../../utilities/utilities";

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

class Snapshots extends Component {
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
    gcsUseIam: false,

    s3CompatibleBucket: "",
    s3CompatiblePath: "",
    s3CompatibleKeyId: "",
    s3CompatibleKeySecret: "",
    s3CompatibleEndpoint: "",
    s3CompatibleRegion: "",
    hideCheckVeleroButton: false,
    updateConfirm: false,

    snapshotSettings: null,
    isLoadingSnapshotSettings: true,
    snapshotSettingsErr: false,
    snapshotSettingsErrMsg: "",
    updateErrorMsg: ""
  };

  fetchSnapshotSettings = (isCheckForVelero) => {
    this.setState({
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
      hideCheckVeleroButton: isCheckForVelero ? true : false
    });

    fetch(`${window.env.API_ENDPOINT}/snapshots/settings`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(res => res.json())
      .then(result => {
        this.setState({
          snapshotSettings: result,
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: false,
          snapshotSettingsErrMsg: "",
        })
        if (!result.isVeleroRunning) {
          setTimeout(() => {
            this.setState({ hideCheckVeleroButton: false });
          }, 5000);
        } else {
          this.setState({ hideCheckVeleroButton: false });
        }
      })
      .catch(err => {
        console.log(err);
        this.setState({
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: true,
          snapshotSettingsErrMsg: err,
        })
      })
  }

  componentDidMount = () => {
    this.fetchSnapshotSettings();
  }

  componentDidUpdate(lastProps, lastState) {
    if (this.state.snapshotSettings !== lastState.snapshotSettings && this.state.snapshotSettings) {
      this.setFields();
    }
  }

  checkForStoreChanges = (provider) => {
    const {
      snapshotSettings,
      s3Region,
      s3KeyId,
      s3KeySecret,
      useIamAws,
      gcsUseIam,
      gcsServiceAccount,
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

    if (provider === "aws") {
      return (
        snapshotSettings?.store?.aws?.region !== s3Region || snapshotSettings?.store?.aws?.accessKeyID !== s3KeyId ||
        snapshotSettings?.store?.aws?.secretAccessKey !== s3KeySecret || snapshotSettings?.store?.aws?.useInstanceRole !== useIamAws
      )
    }
    if (provider === "gcp") {
      return (snapshotSettings?.store?.gcp?.useInstanceRole !== gcsUseIam || snapshotSettings?.store?.gcp?.serviceAccount !== gcsServiceAccount)
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
              serviceAccount: !this.state.gcsUseIam ? this.state.gcsServiceAccount : "",
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

  updateSettings = (provider, bucket, path) => {
    this.setState({ updatingSettings: true, updateErrorMsg: "" });

    const payload = Object.assign({
      provider,
      bucket,
      path
    }, this.getCurrentProviderStores(provider));

    fetch(`${window.env.API_ENDPOINT}/snapshots/settings`, {
      method: "PUT",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload)
    })
      .then(async (res) => {

        const settingsResponse = await res.json();
        if (!res.ok) {
          this.setState({
            updatingSettings: false,
            updateErrorMsg: settingsResponse.error
          })
          return;
        }

        if (settingsResponse.success) {
          this.setState({
            snapshotSettings: settingsResponse,
            updatingSettings: false,
            updateConfirm: true,
            updateErrorMsg: ""
          });
          setTimeout(() => {
            this.setState({ updateConfirm: false })
          }, 3000);
        } else {
          this.setState({
            updatingSettings: false,
            updateErrorMsg: settingsResponse.error
          })
        }
      })
      .catch((err) => {
        console.error(err);
        this.setState({
          updatingSettings: false
        });
      });
  }

  setFields = () => {
    const { snapshotSettings } = this.state;
    if (!snapshotSettings) return;
    const { store } = snapshotSettings;


    if (store?.provider === "aws") {
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

    if (store?.provider === "azure") {
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

    if (store?.provider === "gcp") {
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "gcp"]),
        gcsBucket: store.bucket,
        gcsPath: store.path,
        gcsServiceAccount: store?.gcp?.serviceAccount || "",
        gcsUseIam: store?.gcp?.useInstanceRole
      });
    }

    if (store?.provider === "other") {
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
    this.setState({ gcsServiceAccount: value });
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
    }
  }

  snapshotProviderAWS = async () => {
    this.updateSettings("aws", this.state.s3bucket, this.state.s3Path)
  }

  snapshotProviderAzure = async () => {
    this.updateSettings("azure", this.state.azureBucket, this.state.azurePath);
  }

  snapshotProviderGoogle = async () => {
    this.updateSettings("gcp", this.state.gcsBucket, this.state.gcsPath);
  }

  snapshotProviderS3Compatible = async () => {
    this.updateSettings("other", this.state.s3CompatibleBucket, this.state.s3CompatiblePath);
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
        <span style={{ fontSize: 18, marginRight: "10px" }}>{this.renderIcons(destination)}</span>
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
            {!gcsUseIam &&
              <div className="flex u-marginBottom--30">
                <div className="flex1">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Service Account</p>
                  <div className="gcs-editor">
                    <MonacoEditor
                      ref={(editor) => { this.monacoEditor = editor }}
                      language="json"
                      value={this.state.gcsServiceAccount}
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

      default:
        return (
          <div>No snapshot destination is selected</div>
        )
    }
  }

  renderNotVeleroMessage = () => {
    return <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">Not able to find Velero</p>
  }


  render() {
    const { updatingSettings, hideCheckVeleroButton, updateConfirm, isLoadingSnapshotSettings, updateErrorMsg } = this.state;

    const selectedDestination = DESTINATIONS.find((d) => {
      return d.value === this.state.selectedDestination.value;
    });


    if (isLoadingSnapshotSettings) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    if (!this.state.snapshotSettings || this.state.snapshotSettings.veleroVersion === "") {
      return (
        <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 justifyContent--center alignItems--center">
          <div className="flex-column u-textAlign--center AppSnapshotsEmptyState--wrapper">
            <p className="u-fontSize--largest u-fontWeight--bold u-color--tundora u-marginBottom--10">Configure snapshots</p>
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">
              In order to configure and use Snapshots (backup and restore), please install <a href="https://velero.io/" target="_blank" rel="noopener noreferrer" className="replicated-link">Velero</a> to the cluster. Once Velero is installed, click the button below and the Admin Console will verify the installation and begin configuring Snapshots.
            </p>
            <div className="flex justifyContent--center u-marginTop--40">
              <p className="u-color--tundora u-fontSize--large u-fontWeight--bold">To install Velero</p>
            </div>
            <div className="flex1 flex-column u-marginBottom--50 u-paddingLeft--20">
              <p className="u-fontSize--small flex alignItems--center u-fontWeight--medium u-color--dustyGray u-marginTop--20"><span className="circleNumberGray u-marginRight--10"> 1 </span>Install the CLI on your machine by <a href="https://velero.io/docs/v1.3.2/basic-install/#install-the-cli" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">following these instructions</a> </p>
              <p className="u-fontSize--small flex alignItems--center u-fontWeight--medium u-color--dustyGray u-marginTop--20"><span className="circleNumberGray u-marginRight--10"> 2 </span> Install the Restic integration on your machince by <a href="https://velero.io/docs/v1.3.2/restic/" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">following these instructions</a> </p>
              <div className="flex flex1 u-marginTop--20">
                <div className="flex">
                  <span className="circleNumberGray u-marginRight--10"> 3 </span>
                </div>
                <div className="flex flex-column">
                  <p className="u-fontSize--small flex alignItems--center u-fontWeight--medium u-color--dustyGray"> Run the commands from the instructions for your cloud provider </p>
                  <div className="flex flex1 u-marginTop--15">
                    <a href="https://github.com/vmware-tanzu/velero-plugin-for-aws#setup" target="_blank" rel="noopener noreferrer" className="snapshotOptions"> <span className="icon awsIcon u-cursor--pointer" /></a>
                    <a href="https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure#setup" target="_blank" rel="noopener noreferrer" className="snapshotOptions"> <span className="icon azureIcon u-cursor--pointer" /></a>
                    <a href="https://github.com/vmware-tanzu/velero-plugin-for-gcp#setup" target="_blank" rel="noopener noreferrer" className="snapshotOptions"> <span className="icon googleCloudIcon u-cursor--pointer" /></a>
                    <a href="https://velero.io/docs/v1.3.2/supported-providers/" target="_blank" rel="noopener noreferrer" className="snapshotOptions"> <span className="icon cloudIcon u-cursor--pointer" /> Other </a>
                  </div>
                </div>
              </div>
              <p className="u-fontSize--small flex alignItems--center u-fontWeight--medium u-color--dustyGray u-marginTop--20"><span className="circleNumberGray u-marginRight--10"> 4 </span> With all providers, you must install using the  <span className="inline-code u-marginLeft--5 u-marginRight--5"> --use-restic </span>  flag for snapshots to work. </p>
            </div>
            <div className="u-textAlign--center">
              {!hideCheckVeleroButton ?
                <button className="btn primary blue" onClick={() => this.fetchSnapshotSettings(true)}>Check for Velero</button>
                : this.renderNotVeleroMessage()
              }
            </div>
          </div>
        </div>
      )
    }

    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>Snapshots</title>
        </Helmet>
        <div className="snapshot-form-wrapper">
          <div className="flex flex-column justifyContent--center alignItems--center u-marginBottom--20">
            <p className="u-fontSize--largest u-marginBottom--20 u-fontWeight--bold u-color--tundora">Snapshots</p>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-fontWeight--medium">Snapshots are a way to back up and restore the application and application data. The Admin Console uses <a href="https://velero.io/" target="_blank" rel="noopener noreferrer" className="replicated-link">Velero</a> to enable Snapshots. On this page, you can configure how the Admin Console will use Velero to perform backups and restores.</p>
          </div>
          <form className="flex flex-column">
            <div className={`${this.state.snapshotSettings.isVeleroRunning ? "u-display--none" : "flex flex1 u-marginBottom--30 flex justifyContent--center alignItems--center"}`}>
              <div className="flex u-marginRight--20">
                <span className="icon redWarningIcon" />
              </div>
              <div className="flex flex-column">
                <p className="u-color--chestnut u-fontSize--larger u-fontWeight--bold"> Velero is not running </p>
                <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-fontWeight--medium u-marginTop--10">
                  Velero has been detected, but it's not running successfully. Snapshots will not work until Velero is running reliably.
                  <a href="https://velero.io/docs/master/troubleshooting/" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">Get help</a>
                </p>
              </div>
            </div>
            <div className={`${this.state.snapshotSettings.veleroVersion !== "" && this.state.snapshotSettings.resticVersion === "" ? "flex flex1 u-marginBottom--30 flex justifyContent--center alignItems--center" : "u-display--none"}`}>
              <div className="flex u-marginRight--20">
                <span className="icon redWarningIcon" />
              </div>
              <div className="flex flex-column">
                <p className="u-color--chestnut u-fontSize--larger u-fontWeight--bold"> Restic integration not found </p>
                <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-fontWeight--medium u-marginTop--10">
                  The Admin Console requires the Velero restic integration to use Snapshots, but it was not found. Please install the Velero restic integration to continue.
                  <a href="https://velero.io/" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">Get help</a>
                </p>
              </div>
            </div>
            <div className={`${this.state.snapshotSettings.veleroVersion !== "" && this.state.snapshotSettings.resticVersion !== "" && !this.state.snapshotSettings.isResticRunning ? "flex flex1 u-marginBottom--30 flex justifyContent--center alignItems--center" : "u-display--none"}`}>
              <div className="flex u-marginRight--20">
                <span className="icon redWarningIcon" />
              </div>
              <div className="flex flex-column">
                <p className="u-color--chestnut u-fontSize--larger u-fontWeight--bold"> Restic is not working </p>
                <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-fontWeight--medium u-marginTop--10">
                  Velero and the restic integration have been detected, but restic is not running successfully. Snapshots will not work until Restic is running reliably.
                  <a href="https://velero.io/docs/master/restic/#troubleshooting" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">Get help</a>
                </p>
              </div>
            </div>
            <div className="flex1 u-marginBottom--30">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Deduplication</p>
              <p className="u-fontSize--small u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginBottom--10">All data in your snapshots will be deduplicated. To learn more about how, <a className="replicated-link u-fontSize--small">check out our docs</a>.</p>
            </div>
            {updateErrorMsg &&
              <div className="flex u-fontWeight--bold u-fontSize--small u-color--red u-marginBottom--10">{updateErrorMsg}</div>}
            <div className="flex flex-column u-marginBottom--20">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Destination</p>
              {!this.state.snapshotSettings.isVeleroRunning &&
                <div className="flex u-fontWeight--bold u-fontSize--small u-color--red u-marginBottom--10"> Please fix Velero so that the deployment is running. <a href="https://velero.io/docs/master/troubleshooting/" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">View docs</a>  </div>}
              <div className="flex1">
                <Select
                  className="replicated-select-container"
                  classNamePrefix="replicated-select"
                  placeholder="Select unit"
                  options={DESTINATIONS}
                  isSearchable={false}
                  getOptionLabel={(destination) => this.getDestinationLabel(destination, destination.label)}
                  getOptionValue={(destination) => destination.label}
                  value={selectedDestination}
                  onChange={this.handleDestinationChange}
                  isOptionSelected={(option) => { option.value === selectedDestination }}
                />
              </div>
            </div>
            {!this.state.determiningDestination &&
              <div>
                {this.renderDestinationFields()}
                <div className="flex u-marginBottom--30">
                  <button className="btn primary blue" disabled={updatingSettings} onClick={this.onSubmit}>{updatingSettings ? "Updating" : "Update settings"}</button>
                  {updateConfirm &&
                    <div className="u-marginLeft--10 flex alignItems--center">
                      <span className="icon checkmark-icon" />
                      <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-color--chateauGreen">Settings updated</span>
                    </div>
                  }
                </div>
              </div>
            }
          </form>
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
)(Snapshots);
