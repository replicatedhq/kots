import * as AWS from "aws-sdk";
import {describe, it} from "mocha";
import {expect} from "chai";
import { S3Signer } from "./s3";

describe("S3Signer", () => {
  const s3Signer = new S3Signer();
  it("parses a url", async () => {
    const params = await s3Signer.parse("https://customer-avatars.s3.amazonaws/some-long-key");
    expect(params.Key).to.deep.equal("some-long-key");
    expect(params.Bucket).to.deep.equal("customer-avatars");
  });
});
