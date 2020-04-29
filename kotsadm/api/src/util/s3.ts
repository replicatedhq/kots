import AWS from "aws-sdk";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import * as Minio from "minio";
import * as util from "util";
import url from "url";

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

export async function ensureBucket(params: Params): Promise<boolean> {
  const parsedEndpoint = url.parse(params.s3Endpoint);
  const minioClient = new Minio.Client({
    endPoint: parsedEndpoint.hostname,
    port: parseInt(parsedEndpoint.port || "0"),
    useSSL: false,
    accessKey: params.s3AccessKeyId,
    secretKey: params.s3SecretAccessKey,
  });

  return new Promise((resolve, reject) => {
    minioClient.makeBucket(params.shipOutputBucket, params.s3Region, function(err) {
      if (err) {
        if (err.toString().indexOf("Your previous request to create the named bucket succeeded and you already own it") !== -1) {
          resolve(true);
          return;
        }
        console.log(err);
        reject(err);
        return;
      }

      resolve(true);
    });
  })
}

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
      // minio returns not found
      if (err.code === "NotFound") {
        resolve(true);
        return;
      }

      // Amazon S3 returns BadRequest
      if (err.code === "BadRequest" && params.s3Endpoint.indexOf("amazonaws.com") !== -1) {
        resolve(true);
        return;

      }

      const {code, message, statusCode} = err;
      const {s3Endpoint, s3Region, s3BucketEndpoint, s3SkipEnsureBucket} = params;
      logger.debug({msg: "failed to check if bucket exsists",
        code, message, statusCode, s3Endpoint, s3Region, s3BucketEndpoint, s3SkipEnsureBucket});

      resolve(false);
    });
  });
}
