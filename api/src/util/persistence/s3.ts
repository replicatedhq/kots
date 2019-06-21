import * as AWS from "aws-sdk";

let s3Client: AWS.S3;
export default function s3(): AWS.S3 {
  if (!s3Client) {
    let params = {};

    if (process.env["S3_ENDPOINT"]) {
      params = {
        endpoint: new AWS.Endpoint(process.env["S3_ENDPOINT"]!),
        s3ForcePathStyle: true,
        signatureVersion: "v4",
      };
    }

    s3Client = new AWS.S3(params);
  }

  return s3Client;
}


export async function signPutRequest(bucket: string, key: string, contentType: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const params = {
      Bucket: bucket,
      Key: key,
      ContentType: contentType,
    };

    s3().getSignedUrl("putObject", params, (err, uploadUrl) => {
      if (err) {
        reject(err);
        return;
      }

      if (process.env["S3_ENDPOINT"]) {
        uploadUrl = uploadUrl.replace(process.env["S3_ENDPOINT"]!, "http://localhost:4569");
      }

      resolve(uploadUrl);
    });
  });
}

export async function signGetRequest(bucket: string, key: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const params = {
      Bucket: bucket,
      Key: key,
    };

    s3().getSignedUrl("getObject", params, (err, signedUrl) => {
      if (err) {
        reject(err);
        return;
      }

      resolve(signedUrl);
    });
  });
}
