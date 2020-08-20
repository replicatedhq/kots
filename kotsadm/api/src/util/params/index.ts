import * as _ from "lodash";
import * as AWS from "aws-sdk";

let ssmClient: AWS.SSM;

export async function lookupParams(paramLookup: string[]): Promise<{ [key: string]: string; }> {
  return getParamsFromEnv(paramLookup);
}

function getParamsFromEnv(paramLookup: string[]): { [key: string]: string; } {
  const params: { [key: string]: string; } = {};
  for (const envName of paramLookup) {
    params[envName] = process.env[envName] ? process.env[envName]! : "";
  }
  return params;
}
