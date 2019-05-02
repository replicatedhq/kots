import * as AWS from "aws-sdk";
import * as monkit from "monkit";

let ssmClient: AWS.SSM;
const cache: { [key: string]: string } = {};

export async function param(envName: string, ssmName: string, encrypted = false): Promise<string | undefined> {
  if (!process.env.USE_EC2_PARAMETERS) {
    return process.env[envName];
  }

  if (cache[ssmName]) {
    monkit
      .getRegistry()
      .meter("SSM.cache.hits")
      .mark();

    return cache[ssmName];
  }

  monkit
    .getRegistry()
    .meter("SSM.cache.misses")
    .mark();
  if (!ssmClient) {
    ssmClient = new AWS.SSM({
      apiVersion: "2014-11-06",
    });
  }
  const params = {
    Names: [ssmName],
    WithDecryption: encrypted,
  };
  const result = await ssmClient.getParameters(params).promise();
  if (!result.Parameters || result.Parameters.length === 0) {
    // tslint:disable-next-line:no-console
    console.error(`Parameter ${ssmName} was not found in SSM`);

    return "";
  }
  cache[ssmName] = result.Parameters[0].Value!;

  return cache[ssmName];
}
