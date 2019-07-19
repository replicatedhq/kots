// tslint:disable no-http-string

import { param } from "../util/params";

export class Params {
  private static instance: Params;

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

  constructor({
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
  }) {
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
  }

  static async getParams(): Promise<Params> {
    if (Params.instance) {
      return Params.instance;
    }

    Params.instance = new Params({
      githubAppInstallURL: await param("GITHUB_APP_INSTALL_URL", "/shipcloud/github/app_install_url", false),
      githubClientId: await param("GITHUB_CLIENT_ID", "/shipcloud/github/app_client_id", false),
      githubClientSecret: await param("GITHUB_CLIENT_SECRET", "/shipcloud/github/app_client_secret", true),
      githubIntegrationID: await param("GITHUB_INTEGRATION_ID", "/shipcloud/github/app_integration_id", false),
      githubPrivateKeyFile: (await param("GITHUB_PRIVATE_KEY_FILE", "/shipcloud/github/app_private_key_file", false)) || "/keys/github/private-key.pem",
      githubPrivateKeyContents: await param("GITHUB_PRIVATE_KEY_CONTENTS", "/shipcloud/github/app_private_key", true),
      shipInitBaseURL: (await param("INIT_SERVER_URI", "/shipcloud/initserver/baseURL", false)) || "http://init-server:3000",
      shipWatchBaseURL: (await param("WATCH_SERVER_URI", "/shipcloud/watchserver/baseURL", false)) || "http://watch-server:3000",
      shipUpdateBaseURL: (await param("UPDATE_SERVER_URI", "/shipcloud/updateserver/baseURL", false)) || "http://update-server:3000",
      shipEditBaseURL: (await param("EDIT_BASE_URI", "/shipcloud/editserver/baseURL", false)) || "http://edit-server:3000",
      bugsnagKey: await param("BUGSNAG_KEY", "/shipcloud/bugsnag/key", false),
      sessionKey: await param("SESSION_KEY", "/shipcloud/session/key", true),
      shipOutputBucket: await param("S3_BUCKET_NAME", "/shipcloud/s3/ship_output_bucket", false),
      sigsciRpcAddress: await param("SIGSCI_RPC_ADDRESS", "/shipcloud/sigsci_rpc_address", false),
      shipApiEndpoint: process.env["SHIP_API_ENDPOINT"],
      skipDeployToWorker: process.env["SKIP_DEPLOY_TO_WORKER"],
      objectStoreInDatabase: process.env["OBJECT_STORE_IN_DATABASE"],
      s3Endpoint: process.env["S3_ENDPOINT"],
      s3AccessKeyId: await param("S3_ACCESS_KEY_ID", "/shipcloud/s3/access_key_id", false),
      s3SecretAccessKey: await param("S3_SECRET_ACCESS_KEY", "/shipcloud/s3/secret_access_key", true),
      s3BucketEndpoint: await param("S3_BUCKET_ENDPOINT", "/shipcloud/s3/bucket_endpoint", false),
      apiAdvertiseEndpoint: process.env["SHIP_API_ADVERTISE_ENDPOINT"],
      graphqlPremEndpoint: await param("GRAPHQL_PREM_ENDPOINT", "/graphql/prem_endpoint", false),
    });

    return Params.instance;
  }
}
