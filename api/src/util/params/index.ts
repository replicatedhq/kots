import * as _ from "lodash";
import * as AWS from "aws-sdk";

let ssmClient: AWS.SSM;

export interface ParamLookup { [key: string]: string; };

export async function lookupParams(paramLookup: ParamLookup): Promise<{ [key: string]: string; }> {
  if (process.env["USE_EC2_PARAMETERS"]) {
    try {
      return await getParamsFromSsm(paramLookup)
    } catch(err) {
      // tslint:disable-next-line:no-console
      console.error(err);
      throw err;
    }
  }
  return getParamsFromEnv(paramLookup);
}

async function getParamsFromSsm(paramLookup: ParamLookup): Promise<{ [key: string]: string; }> {
  if (!ssmClient) {
    ssmClient = new AWS.SSM({
      apiVersion: "2014-11-06",
    });
  }

  const params: { [key: string]: string; } = {};
  const lookup: string[] = [];
  const reverseLookup: { [key: string]: string; } = {};

  for (const key in paramLookup) {
    const envName = key;
    const ssmName = paramLookup[key];
		if (ssmName) {
			lookup.push(ssmName);
			reverseLookup[ssmName] = envName;
		} else {
			params[envName] = process.env[envName] ? process.env[envName]! : "";
		}
	}

  await Promise.all(_.chunk(lookup, 10).map(async (chunk) => {
    const getParametersOptions = {
      Names: chunk,
      WithDecryption: true,
    };
    const result = await ssmClient.getParameters(getParametersOptions).promise();
    if (result.InvalidParameters) {
      result.InvalidParameters.forEach(paramName => {
        // tslint:disable-next-line:no-console
        console.error(`Parameter ${paramName} invalid`);
      });
    }
    if (result.Parameters) {
      result.Parameters.forEach(param => {
        params[reverseLookup[param.Name!]] = param.Value!;
      });
    }
  }));

  return params;
}

function getParamsFromEnv(paramLookup: ParamLookup): { [key: string]: string; } {
  const params: { [key: string]: string; } = {};
  for (const key in paramLookup) {
    const envName = key;
    const ssmName = paramLookup[key];
    params[envName] = process.env[envName] ? process.env[envName]! : "";
  }
  return params;
}
