// tslint:disable no-http-string

import { lookupParams } from "../util/params";

export class Params {
  private static instance: Params;

  readonly postgresUri: string;
  readonly apiEncryptionKey: string;
  readonly sessionKey: string;
  readonly shipOutputBucket: string;
  readonly sigsciRpcAddress: string;
  readonly shipApiEndpoint: string;
  readonly objectStoreInDatabase: string;
  readonly s3Endpoint: string;
  readonly s3AccessKeyId: string;
  readonly s3SecretAccessKey: string;
  readonly s3BucketEndpoint: string;
  readonly s3Region: string;
  readonly s3SkipEnsureBucket: boolean;
  readonly apiAdvertiseEndpoint: string;
  readonly graphqlPremEndpoint: string;
  readonly enableKurl: boolean;
  readonly prometheusAddress: string;
  readonly storageBaseURI: string;
  readonly storageBaseURIPlainHttp: boolean;

  constructor({
    postgresUri,
    apiEncryptionKey,
    sessionKey,
    shipOutputBucket,
    sigsciRpcAddress,
    shipApiEndpoint,
    objectStoreInDatabase,
    s3Endpoint,
    s3AccessKeyId,
    s3SecretAccessKey,
    s3BucketEndpoint,
    s3Region,
    s3SkipEnsureBucket,
    apiAdvertiseEndpoint,
    graphqlPremEndpoint,
    enableKurl,
    prometheusAddress,
    storageBaseURI,
    storageBaseURIPlainHttp,
  }) {
    this.postgresUri = postgresUri;
    this.apiEncryptionKey = apiEncryptionKey;
    this.sessionKey = sessionKey;
    this.shipOutputBucket = shipOutputBucket;
    this.sigsciRpcAddress = sigsciRpcAddress;
    this.shipApiEndpoint = shipApiEndpoint;
    this.objectStoreInDatabase = objectStoreInDatabase;
    this.s3Endpoint = s3Endpoint;
    this.s3AccessKeyId = s3AccessKeyId;
    this.s3SecretAccessKey = s3SecretAccessKey;
    this.s3BucketEndpoint = s3BucketEndpoint;
    this.s3Region = s3Region;
    this.s3SkipEnsureBucket = s3SkipEnsureBucket;
    this.apiAdvertiseEndpoint = apiAdvertiseEndpoint;
    this.graphqlPremEndpoint = graphqlPremEndpoint;
    this.enableKurl = enableKurl;
    this.prometheusAddress = prometheusAddress;
    this.storageBaseURI = storageBaseURI;
    this.storageBaseURIPlainHttp = storageBaseURIPlainHttp;
  }

  public static async getParams(): Promise<Params> {
    if (Params.instance) {
      return Params.instance;
    }

    const params = await this.loadParams();
    Params.instance = new Params({
      postgresUri: params["POSTGRES_URI"],
      apiEncryptionKey: params["API_ENCRYPTION_KEY"],
      sessionKey: params["SESSION_KEY"],
      shipOutputBucket: params["S3_BUCKET_NAME"],
      sigsciRpcAddress: params["SIGSCI_RPC_ADDRESS"],
      shipApiEndpoint: process.env["SHIP_API_ENDPOINT"],
      objectStoreInDatabase: process.env["OBJECT_STORE_IN_DATABASE"],
      s3Endpoint: process.env["S3_ENDPOINT"],
      s3AccessKeyId: params["S3_ACCESS_KEY_ID"],
      s3SecretAccessKey: params["S3_SECRET_ACCESS_KEY"],
      s3BucketEndpoint: params["S3_BUCKET_ENDPOINT"],
      s3Region: process.env["S3_REGION"] || "us-east-1",
      s3SkipEnsureBucket: process.env["S3_SKIP_ENSURE_BUCKET"] === "1",
      apiAdvertiseEndpoint: process.env["SHIP_API_ADVERTISE_ENDPOINT"],
      graphqlPremEndpoint: params["GRAPHQL_PREM_ENDPOINT"],
      enableKurl: process.env["ENABLE_KURL"] === "1",
      prometheusAddress: process.env["PROMETHEUS_ADDRESS"],
      storageBaseURI: process.env["STORAGE_BASEURI"],
      storageBaseURIPlainHttp: JSON.parse(process.env["STORAGE_BASEURI_PLAINHTTP"] || "false"),
    });
    return Params.instance;
  }

  private static async loadParams(): Promise<{ [key:string]: string; }> {
    const paramLookup = [
      "POSTGRES_URI",
      "API_ENCRYPTION_KEY",
      "SESSION_KEY",
      "S3_BUCKET_NAME",
      "SIGSCI_RPC_ADDRESS",
      "SHIP_API_ENDPOINT",
      "OBJECT_STORE_IN_DATABASE",
      "S3_ENDPOINT",
      "S3_ACCESS_KEY_ID",
      "S3_SECRET_ACCESS_KEY",
      "S3_BUCKET_ENDPOINT",
      "SHIP_API_ADVERTISE_ENDPOINT",
      "GRAPHQL_PREM_ENDPOINT",
      "ENABLE_KURL",
      "STORAGE_BASEURI",
      "STORAGE_BASEURI_PLAINHTTP",
    ];
    return await lookupParams(paramLookup);
  }
}
