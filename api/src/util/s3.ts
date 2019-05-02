import * as AWS from "aws-sdk";
import { Params } from "../server/params";

export function getS3(params: Params): AWS.S3 {
  const s3Params: AWS.S3.ClientConfiguration = {
    signatureVersion: "v4",
  };

  if (params.s3Endpoint && params.s3Endpoint.trim() !== "") {
    s3Params.endpoint = params.s3Endpoint.trim();
  }

  if (params.s3AccessKeyId && params.s3AccessKeyId.trim() !== "" && params.s3SecretAccessKey && params.s3SecretAccessKey.trim() !== "") {
    s3Params.accessKeyId = params.s3AccessKeyId.trim();
    s3Params.secretAccessKey = params.s3SecretAccessKey.trim();
  }

  if (params.s3BucketEndpoint && params.s3BucketEndpoint.trim() !== "") {
    s3Params.s3BucketEndpoint = true;
  }

  return new AWS.S3(s3Params);
};

export async function putObject(params: Params, filepath: string, body: any, bucket: string): Promise<boolean> {
  const s3 = getS3(params);

  const putObjectParams = {
    Body: body,
    Bucket: bucket,
    Key: filepath,
   };

   return new Promise<boolean>((resolve, reject) => {
     s3.putObject(putObjectParams, (err, data) => {
       if (err) {
         reject(err);
       }

       resolve(true);
    });
   });
}

export async function signGetRequest(params: any): Promise<any> {
  const s3 = new AWS.S3({signatureVersion: 'v4'});

  return new Promise((resolve, reject) => {
    s3.getSignedUrl("getObject", params, (err: any, url: string) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(url);
    });
  });
}

// checks if a file exists in S3
export async function checkExists(params: Params, requestParams: AWS.S3.Types.HeadObjectRequest): Promise<boolean> {
  if (params.s3BucketEndpoint && params.s3BucketEndpoint.trim() !== "") {
    requestParams.Key = `${params.shipOutputBucket.trim()}/${requestParams.Key}`;
  }

  return new Promise<boolean>(resolve => {
    const s3 = getS3(params);

    s3.headObject(requestParams, err => {
      if (err) {
        resolve(false);
        return;
      }

      resolve(true);
    });
  });
}
