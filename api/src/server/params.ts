// tslint:disable no-http-string

import { ParamLookup, lookupParams } from "../util/params";

export class Params {
  private static instance: Params;

  readonly postgresUri: string;
  readonly githubClientId: string;
  readonly githubPrivateKeyFile: string;
  readonly githubPrivateKeyContents: string;
  readonly githubClientSecret: string;
  readonly githubIntegrationID: string;
  readonly githubAppInstallURL: string;
  readonly bugsnagKey: string;
  readonly sessionKey: string;
  readonly shipInitBaseURL: string;
  readonly shipUpdateBaseURL: string;
  readonly shipEditBaseURL: string;
  readonly shipWatchBaseURL: string;
  readonly shipOutputBucket: string;
  readonly airgapBucket: string;
  readonly sigsciRpcAddress: string;
  readonly shipApiEndpoint: string;
  readonly skipDeployToWorker: string;
  readonly objectStoreInDatabase: string;
  readonly s3Endpoint: string;
  readonly s3AccessKeyId: string;
  readonly s3SecretAccessKey: string;
  readonly s3BucketEndpoint: string;
  readonly apiAdvertiseEndpoint: string;
  readonly graphqlPremEndpoint: string;
  readonly segmentioAnalyticsKey: string;
  readonly enableShip: boolean;
  readonly enableKots: boolean;

  constructor({
    postgresUri,
    githubAppInstallURL,
    githubClientId,
    githubPrivateKeyFile,
    githubPrivateKeyContents,
    githubClientSecret,
    githubIntegrationID,
    bugsnagKey,
    sessionKey,
    shipInitBaseURL,
    shipUpdateBaseURL,
    shipEditBaseURL,
    shipWatchBaseURL,
    shipOutputBucket,
    airgapBucket,
    sigsciRpcAddress,
    shipApiEndpoint,
    skipDeployToWorker,
    objectStoreInDatabase,
    s3Endpoint,
    s3AccessKeyId,
    s3SecretAccessKey,
    s3BucketEndpoint,
    apiAdvertiseEndpoint,
    graphqlPremEndpoint,
    segmentioAnalyticsKey,
    enableShip,
    enableKots,
  }) {
    this.postgresUri = postgresUri;
    this.githubAppInstallURL = githubAppInstallURL;
    this.githubClientId = githubClientId;
    this.githubPrivateKeyFile = githubPrivateKeyFile;
    this.githubPrivateKeyContents = githubPrivateKeyContents;
    this.githubClientSecret = githubClientSecret;
    this.githubIntegrationID = githubIntegrationID;
    this.bugsnagKey = bugsnagKey;
    this.sessionKey = sessionKey;
    this.shipInitBaseURL = shipInitBaseURL;
    this.shipUpdateBaseURL = shipUpdateBaseURL;
    this.shipEditBaseURL = shipEditBaseURL;
    this.shipWatchBaseURL = shipWatchBaseURL;
    this.shipOutputBucket = shipOutputBucket;
    this.airgapBucket = airgapBucket;
    this.sigsciRpcAddress = sigsciRpcAddress;
    this.shipApiEndpoint = shipApiEndpoint;
    this.skipDeployToWorker = skipDeployToWorker;
    this.objectStoreInDatabase = objectStoreInDatabase;
    this.s3Endpoint = s3Endpoint;
    this.s3AccessKeyId = s3AccessKeyId;
    this.s3SecretAccessKey = s3SecretAccessKey;
    this.s3BucketEndpoint = s3BucketEndpoint;
    this.apiAdvertiseEndpoint = apiAdvertiseEndpoint;
    this.graphqlPremEndpoint = graphqlPremEndpoint;
    this.segmentioAnalyticsKey = segmentioAnalyticsKey;
    this.enableShip = enableShip;
    this.enableKots = enableKots;
  }

  public static async getParams(): Promise<Params> {
    if (Params.instance) {
      return Params.instance;
    }

    const params = await this.loadParams();
    Params.instance = new Params({
      postgresUri: params["POSTGRES_URI"],
      githubAppInstallURL: params["GITHUB_APP_INSTALL_URL"],
      githubClientId: params["GITHUB_CLIENT_ID"],
      githubClientSecret: params["GITHUB_CLIENT_SECRET"],
      githubIntegrationID: params["GITHUB_INTEGRATION_ID"],
      githubPrivateKeyFile: params["GITHUB_PRIVATE_KEY_FILE"],
      githubPrivateKeyContents: params["GITHUB_PRIVATE_KEY_CONTENTS"],
      shipInitBaseURL: params["INIT_SERVER_URI"] || "http://init-server:3000",
      shipWatchBaseURL: params["WATCH_SERVER_URI"] || "http://watch-server:3000",
      shipUpdateBaseURL: params["UPDATE_SERVER_URI"] || "http://update-server:3000",
      shipEditBaseURL: params["EDIT_BASE_URI"] || "http://edit-server:3000",
      bugsnagKey: params["BUGSNAG_KEY"],
      sessionKey: params["SESSION_KEY"],
      shipOutputBucket: params["S3_BUCKET_NAME"],
      airgapBucket: params["AIRGAP_BUNDLE_S3_BUCKET"],
      sigsciRpcAddress: params["SIGSCI_RPC_ADDRESS"],
      shipApiEndpoint: process.env["SHIP_API_ENDPOINT"],
      skipDeployToWorker: process.env["SKIP_DEPLOY_TO_WORKER"],
      objectStoreInDatabase: process.env["OBJECT_STORE_IN_DATABASE"],
      s3Endpoint: process.env["S3_ENDPOINT"],
      s3AccessKeyId: params["S3_ACCESS_KEY_ID"],
      s3SecretAccessKey: params["S3_SECRET_ACCESS_KEY"],
      s3BucketEndpoint: params["S3_BUCKET_ENDPOINT"],
      apiAdvertiseEndpoint: process.env["SHIP_API_ADVERTISE_ENDPOINT"],
      graphqlPremEndpoint: params["GRAPHQL_PREM_ENDPOINT"],
      segmentioAnalyticsKey: params["SEGMENTIO_ANALYTICS_WRITE_KEY"],
      enableShip: process.env["ENABLE_SHIP"] === "1",
      enableKots: process.env["ENABLE_KOTS"] === "1",
    });
    return Params.instance;
  }

  private static async loadParams(): Promise<{ [key:string]: string; }> {
    const paramLookup: ParamLookup = {
      POSTGRES_URI: "/shipcloud/postgres/uri",
      GITHUB_APP_INSTALL_URL: "/shipcloud/github/app_install_url",
      GITHUB_CLIENT_ID: "/shipcloud/github/app_client_id",
      GITHUB_CLIENT_SECRET: "/shipcloud/github/app_client_secret",
      GITHUB_INTEGRATION_ID: "/shipcloud/github/app_integration_id",
      GITHUB_PRIVATE_KEY_FILE: "/shipcloud/github/app_private_key_file",
      GITHUB_PRIVATE_KEY_CONTENTS: "/shipcloud/github/app_private_key",
      INIT_SERVER_URI: "/shipcloud/initserver/baseURL",
      WATCH_SERVER_URI: "/shipcloud/watchserver/baseURL",
      UPDATE_SERVER_URI: "/shipcloud/updateserver/baseURL",
      EDIT_BASE_URI: "/shipcloud/editserver/baseURL",
      BUGSNAG_KEY: "/shipcloud/bugsnag/key",
      SESSION_KEY: "/shipcloud/session/key",
      S3_BUCKET_NAME: "/shipcloud/s3/ship_output_bucket",
      AIRGAP_BUNDLE_S3_BUCKET: "/shipcloud/airgap_bucket_name",
      SIGSCI_RPC_ADDRESS: "/shipcloud/sigsci_rpc_address",
      SHIP_API_ENDPOINT: "",
      SKIP_DEPLOY_TO_WORKER: "",
      OBJECT_STORE_IN_DATABASE: "",
      S3_ENDPOINT: "",
      S3_ACCESS_KEY_ID: "/shipcloud/s3/access_key_id",
      S3_SECRET_ACCESS_KEY: "/shipcloud/s3/secret_access_key",
      S3_BUCKET_ENDPOINT: "/shipcloud/s3/bucket_endpoint",
      SHIP_API_ADVERTISE_ENDPOINT: "",
      GRAPHQL_PREM_ENDPOINT: "/graphql/prem_endpoint",
      SEGMENTIO_ANALYTICS_WRITE_KEY: "/shipcloud/segmentio/analytics_write_key",
      ENABLE_SHIP: "",
      ENABLE_KOTS: "",
    }
    return await lookupParams(paramLookup);
  }
}
