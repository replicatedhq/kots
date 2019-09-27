import AWS from "aws-sdk";
import { Params } from "../server/params";

export function getS3(params: Params): AWS.S3 {
  const s3Params: AWS.S3.ClientConfiguration = {
    // signatureVersion: "v4",
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

export async function upload(params: Params, key: string, body: any, bucket: string): Promise<any> {
  const s3 = getS3(params);

  const uploadParams = {
    Body: body,
    Bucket: bucket,
    Key: key,
   };

   return s3.upload(uploadParams);
}

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

export async function signGetRequest(params: Params, bucket: string, key: string, expires?: number): Promise<any> {
  const s3 = getS3(params);

  return new Promise((resolve, reject) => {
    const params = {
      Bucket: bucket,
      Key: key,
      Expires: expires,
    };

    s3.getSignedUrl("getObject", params, (err: any, url: string) => {
      if (err) {
        reject(err);
        return;
      }

      if (process.env["S3_ENDPOINT"]) {
        url = url.replace(process.env["S3_ENDPOINT"]!, `${process.env["S3_ENDPOINT"]}${bucket}/`);
      }

      resolve(url);
    });
  });
}

export async function signPutRequest(params: Params, bucket: string, key: string, contentType: string, expires?: number): Promise<string> {
  const s3 = getS3(params);

  return new Promise((resolve, reject) => {
    const params = {
      Bucket: bucket,
      Key: key,
      ContentType: contentType,
      Expires: expires,
    };

    s3.getSignedUrl("putObject", params, (err, uploadUrl) => {
      if (err) {
        reject(err);
        return;
      }

      if (process.env["S3_ENDPOINT"]) {
        uploadUrl = uploadUrl.replace(process.env["S3_ENDPOINT"]!, `http://localhost:30456/${bucket}/`);
      }

      resolve(uploadUrl);
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

export async function getFileInfo(params: Params, requestParams: AWS.S3.Types.HeadObjectRequest): Promise<any> {
  if (params.s3BucketEndpoint && params.s3BucketEndpoint.trim() !== "") {
    requestParams.Key = `${params.shipOutputBucket.trim()}/${requestParams.Key}`;
  }

  return new Promise<any>((resolve, reject) => {
    const s3 = getS3(params);

    s3.headObject(requestParams, (err, info) => {
      if (err) {
        reject(err);
        return;
      }

      resolve(info);
    });
  });
}

export async function bucketExists(params: Params, bucketName: string): Promise<boolean> {
  return new Promise<boolean>(resolve => {
    const s3 = getS3(params);

    s3.headObject({ Bucket: bucketName, Key: "no_such_file.tar.gz" }, err => {
      if (err.code === "NotFound") {
        resolve(true);
        return;
      }

      resolve(false);
    });
  });
}
