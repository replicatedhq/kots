import React, { Component } from "react";
import Select from "react-select";
import { graphql, compose, withApollo } from "react-apollo";
import { Link, withRouter } from "react-router-dom"
import MonacoEditor from "react-monaco-editor";
import find from "lodash/find";
import { snapshotSettings } from "../../queries/SnapshotQueries";
import { snapshotProviderAWS, snapshotProviderS3Compatible, snapshotProviderAzure, snapshotProviderGoogle } from "../../mutations/SnapshotMutations";
import "../../scss/components/shared/SnapshotForm.scss";

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
    value: "google",
    label: "Google Cloud Storage",
  },
  {
    value: "s3compatible",
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

class AppSnapshotSettings extends Component {
  state = {
    determiningDestination: true,
    selectedDestination: {},
    updatingSettings: false,
    s3bucket: "",
    s3Region: "",
    s3Path: "",
    useIam: false,
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

    s3CompatibleBucket: "",
    s3CompatiblePath: "",
    s3CompatibleKeyId: "",
    s3CompatibleKeySecret: "",
    s3CompatibleEndpoint: "",
    s3CompatibleRegion: ""
  };

  setFields = () => {
    const { snapshotSettings } = this.props;
    if (!snapshotSettings.snapshotConfig) return;
    const { store } = snapshotSettings.snapshotConfig;

    if (store?.s3AWS) {
      const useIam = !!store.s3AWS.accessKeyID.length || !!store.s3AWS.accessKeySecret.length;
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "aws"]),
        s3bucket: store.bucket,
        s3Region: store.s3AWS.region,
        s3Path: store.path,
        useIam,
        s3KeyId: store.s3AWS.accessKeyID || "",
        s3KeySecret: store.s3AWS.accessKeySecret || ""
      });
    }

    if (store?.azure) {
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "azure"]),
        azureBucket: store.bucket,
        azurePath: store.path,
        azureSubscriptionId: store.azure.subscriptionID,
        azureTenantId: store.azure.tenantID,
        azureClientId: store.azure.clientID,
        azureClientSecret: store.azure.clientSecret,
        azureResourceGroupName: store.azure.resourceGroup,
        azureStorageAccountId: store.azure.storageAccount,
        selectedAzureCloudName: find(AZURE_CLOUD_NAMES, ["value", store.azure.cloudName])
      });
    }

    if (store?.google) {
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "google"]),
        gcsBucket: store.bucket,
        gcsPath: store.path,
        gcsServiceAccount: store.google.serviceAccount
      });
    }

    if (store?.s3Compatible) {
      return this.setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "s3compatible"]),
        s3CompatibleBucket: store.bucket,
        s3CompatiblePath: store.path,
        s3CompatibleKeyId: store.s3Compatible.accessKeyID,
        s3CompatibleKeySecret: store.s3Compatible.accessKeySecret,
        s3CompatibleEndpoint: store.s3Compatible.endpoint,
        s3CompatibleRegion: store.s3Compatible.region
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
    if (field === "useIam") {
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

  onSubmit = (e) => {
    e.preventDefault();
    switch (this.state.selectedDestination.value) {
    case "aws":
      this.snapshotProviderAWS();
      break;
    case "azure":
      this.snapshotProviderAzure();
      break;
    case "google":
      this.snapshotProviderGoogle();
      break;
    case "s3compatible":
      this.snapshotProviderS3Compatible();
      break;
    }
  }

  snapshotProviderAWS = () => {
    this.setState({ updatingSettings: true });
    this.props.snapshotProviderAWS(
      this.state.s3bucket,
      this.state.s3Path,
      this.state.s3Region,
      this.state.useIam && this.state.s3KeyId,
      this.state.useIam && this.state.s3KeySecret,
    ).then(() => {
      this.setState({ updatingSettings: false });
    })
    .catch(err => {
        console.log(err);
        err.graphQLErrors.map(({ msg }) => {
          this.setState({
            message: msg,
            messageType: "error",
            updatingSettings: false
          });
        });
      });
  }

  snapshotProviderAzure = () => {
    this.setState({ updatingSettings: true });
    this.props.snapshotProviderAzure(
      this.state.azureBucket,
      this.state.azurePath,
      this.state.azureTenantId,
      this.state.azureResourceGroupName,
      this.state.azureStorageAccountId,
      this.state.azureSubscriptionId,
      this.state.azureClientId,
      this.state.azureClientSecret,
      this.state.selectedAzureCloudName.value,
    ).then(() => {
      this.setState({ updatingSettings: false });
    })
    .catch(err => {
        console.log(err);
        err.graphQLErrors.map(({ msg }) => {
          this.setState({
            message: msg,
            messageType: "error",
            updatingSettings: false
          });
        });
      });
  }

  snapshotProviderGoogle = () => {
    this.setState({ updatingSettings: true });
    this.props.snapshotProviderGoogle(
      this.state.gcsBucket,
      this.state.gcsPath,
      this.state.gcsServiceAccount
    ).then(() => {
      this.setState({ updatingSettings: false })
    })
    .catch(err => {
        console.log(err);
        err.graphQLErrors.map(({ msg }) => {
          this.setState({
            message: msg,
            messageType: "error",
            updatingSettings: false
          });
        });
      });
  }

  snapshotProviderS3Compatible = () => {
    this.setState({ updatingSettings: true });
    this.props.snapshotProviderS3Compatible(
      this.state.s3CompatibleBucket,
      this.state.s3CompatiblePath,
      this.state.s3CompatibleRegion,
      this.state.s3CompatibleEndpoint,
      this.state.s3CompatibleKeyId,
      this.state.s3CompatibleKeySecret,
    ).then(() => {
      this.setState({ updatingSettings: false });
    })
    .catch(err => {
        console.log(err);
        err.graphQLErrors.map(({ msg }) => {
          this.setState({
            message: msg,
            messageType: "error",
            updatingSettings: false
          });
        });
      });
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
    const { selectedDestination, useIam } = this.state;
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
                <input type="text" className="Input" placeholder="Bucket name" value={this.state.s3bucket} onChange={(e) => { this.handleFormChange("s3bucket", e) }}/>
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Region</p>
                <input type="text" className="Input" placeholder="Bucket region" value={this.state.s3Region} onChange={(e) => { this.handleFormChange("s3Region", e) }}/>
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Path</p>
                <input type="text" className="Input" placeholder="/path/to/destination" value={this.state.s3Path} onChange={(e) => { this.handleFormChange("s3Path", e) }}/>
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">&nbsp;</p>
                <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left">
                  <div className={`BoxedCheckbox flex-auto flex alignItems--center ${this.state.useIam ? "is-active" : ""}`}>
                    <input
                      type="checkbox"
                      className="u-cursor--pointer u-marginLeft--10"
                      id="useIam"
                      checked={this.state.useIam}
                      onChange={(e) => { this.handleFormChange("useIam", e) }}
                    />
                    <label htmlFor="useIam" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                      <div className="flex1">
                        <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Use IAM Role</p>
                      </div>
                    </label>
                  </div>
                </div>
              </div>
            </div>

            {useIam &&
              <div className="flex u-marginBottom--30">
                <div className="flex1 u-paddingRight--5">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Access Key ID</p>
                  <input type="text" className="Input" placeholder="key ID" value={this.state.s3KeyId} onChange={(e) => { this.handleFormChange("s3KeyId", e) }}/>
                </div>
                <div className="flex1 u-paddingLeft--5">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Access Key Secret</p>
                  <input type="text" className="Input" placeholder="access key" value={this.state.s3KeySecret} onChange={(e) => { this.handleFormChange("s3KeySecret", e) }}/>
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
              <input type="text" className="Input" placeholder="Bucket name" value={this.state.azureBucket} onChange={(e) => { this.handleFormChange("azureBucket", e) }}/>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Path</p>
                <input type="text" className="Input" placeholder="/path/to/destination" value={this.state.azurePath} onChange={(e) => { this.handleFormChange("azurePath", e) }}/>
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Subscription ID</p>
                <input type="text" className="Input" placeholder="Subscription ID" value={this.state.azureSubscriptionId} onChange={(e) => { this.handleFormChange("azureSubscriptionId", e) }}/>
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Tenant ID</p>
                <input type="text" className="Input" placeholder="Tenant ID" value={this.state.azureTenantId} onChange={(e) => { this.handleFormChange("azureTenantId", e) }}/>
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Client ID</p>
                <input type="text" className="Input" placeholder="Client ID" value={this.state.azureClientId} onChange={(e) => { this.handleFormChange("azureClientId", e) }}/>
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Client Secret</p>
                <input type="text" className="Input" placeholder="Client Secret" value={this.state.azureClientSecret} onChange={(e) => { this.handleFormChange("azureClientSecret", e) }}/>
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
                <input type="text" className="Input" placeholder="Resource Group Name" value={this.state.azureResourceGroupName} onChange={(e) => { this.handleFormChange("azureResourceGroupName", e) }}/>
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Storage Account ID</p>
                <input type="text" className="Input" placeholder="Storage Account ID" value={this.state.azureStorageAccountId} onChange={(e) => { this.handleFormChange("azureStorageAccountId", e) }}/>
              </div>
            </div>
          </div>
        )

      case "google":
        return (
          <div>
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Bucket</p>
              <input type="text" className="Input" placeholder="Bucket name" value={this.state.gcsBucket} onChange={(e) => { this.handleFormChange("gcsBucket", e) }}/>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Path</p>
                <input type="text" className="Input" placeholder="/path/to/destination" value={this.state.gcsPath} onChange={(e) => { this.handleFormChange("gcsPath", e) }}/>
              </div>
            </div>
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
          </div>
        )

      case "s3compatible":
        return (
          <div>
            <div className="flex1 u-paddingRight--5">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Bucket</p>
              <input type="text" className="Input" placeholder="Bucket name" value={this.state.s3CompatibleBucket} onChange={(e) => { this.handleFormChange("s3CompatibleBucket", e) }}/>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Path</p>
                <input type="text" className="Input" placeholder="/path/to/destination" value={this.state.s3CompatiblePath} onChange={(e) => { this.handleFormChange("s3CompatiblePath", e) }}/>
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Access Key ID</p>
                <input type="text" className="Input" placeholder="key ID" value={this.state.s3CompatibleKeyId} onChange={(e) => { this.handleFormChange("s3CompatibleKeyId", e) }}/>
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Access Key Secret</p>
                <input type="text" className="Input" placeholder="access key" value={this.state.s3CompatibleKeySecret} onChange={(e) => { this.handleFormChange("s3CompatibleKeySecret", e) }}/>
              </div>
            </div>
            <div className="flex u-marginBottom--30">
              <div className="flex1 u-paddingRight--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Endpoint</p>
                <input type="text" className="Input" placeholder="endpoint" value={this.state.s3CompatibleEndpoint} onChange={(e) => { this.handleFormChange("s3CompatibleEndpoint", e) }}/>
              </div>
              <div className="flex1 u-paddingLeft--5">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Region</p>
                <input type="text" className="Input" placeholder="/path/to/destination" value={this.state.s3CompatibleRegion} onChange={(e) => { this.handleFormChange("s3CompatibleRegion", e) }}/>
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

  componentDidUpdate = (lastProps) => {
    if (this.props.snapshotSettings.snapshotConfig && this.props.snapshotSettings.snapshotConfig !== lastProps.snapshotSettings.snapshotConfig) {
      this.setFields();
    }
  }

  componentDidMount = () => {
    if (this.props.snapshotSettings.snapshotConfig) {
      this.setFields();
    }
  }

  render() {
    const { updatingSettings } = this.state;
    const { app } = this.props;
    const selectedDestination = DESTINATIONS.find((d) => {
      return d.value === this.state.selectedDestination.value;
    });
    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <div className="snapshot-form-wrapper">
          <p className="u-marginBottom--30 u-fontSize--small u-color--tundora u-fontWeight--medium">
            <Link to={`/app/${app?.slug}/snapshots`} className="replicated-link">Snapshots</Link>
            <span className="u-color--dustyGray"> > </span>
            Settings
          </p>
          <form>
            <div className="flex1 u-marginBottom--30">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Deduplication</p>
              <p className="u-fontSize--small u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginBottom--10">All data in your snapshots will be deduplicated. To learn more about how, <a className="replicated-link u-fontSize--small">check out our docs</a>.</p>
            </div>
            <div className="flex-column u-marginBottom--20">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Destination</p>
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
                  <div className="u-marginRight--10">
                    <Link to={`/app/${app?.slug}/snapshots`} className="btn secondary">Cancel</Link>
                  </div>
                  <div>
                    <button className="btn primary blue" disabled={updatingSettings} onClick={this.onSubmit}>{updatingSettings ? "Updating" : "Update settings"}</button>
                  </div>
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
  graphql(snapshotSettings, {
    name: "snapshotSettings",
    options: ({ match }) => {
      const slug = match.params.slug;
      return {
        variables: { slug },
        fetchPolicy: "no-cache"
      }
    }
  }),
  graphql(snapshotProviderAWS, {
    props: ({ mutate }) => ({
      snapshotProviderAWS: (bucket, prefix, region, accessKeyID, accessKeySecret) => mutate({ variables: { bucket, prefix, region, accessKeyID, accessKeySecret } })
    })
  }),
  graphql(snapshotProviderS3Compatible, {
    props: ({ mutate }) => ({
      snapshotProviderS3Compatible: (bucket, prefix, region, endpoint, accessKeyID, accessKeySecret) => mutate({ variables: { bucket, prefix, region, endpoint, accessKeyID, accessKeySecret } })
    })
  }),
  graphql(snapshotProviderGoogle, {
    props: ({ mutate }) => ({
      snapshotProviderGoogle: (bucket, prefix, serviceAccount) => mutate({ variables: { bucket, prefix, serviceAccount } } )
    })
  }),
  graphql(snapshotProviderAzure, {
    props: ({ mutate }) => ({
      snapshotProviderAzure: (bucket, prefix, tenantID, resourceGroup, storageAccount, subscriptionID, clientID, clientSecret, cloudName) => mutate({ variables: { bucket, prefix, tenantID, resourceGroup, storageAccount, subscriptionID, clientID, clientSecret, cloudName } })
    })
  })

)(AppSnapshotSettings);
