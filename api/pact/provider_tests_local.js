const { Verifier } = require("@pact-foundation/pact");
const packageJson = require("../package.json");
const path = require("path");
const S3rver = require("s3rver");
const os = require("os");

const s3rver = new S3rver({
  port: 14569,
  hostname: "localhost",
  silent: false,
  directory: os.tmpdir(),
  configureBuckets: [{
    name: "ship-pacts",
  }]
});

let opts = {
  providerBaseUrl: "http://localhost:3000",
  provider: "ship-cluster-api",
  pactUrls: [
    path.resolve(process.cwd(), "pacts", "ship-cluster-ui-ship-cluster-api.json"),
  ],
  publishVerificationResult: process.env["PUBLISH_PACT_VERIFICATION"] === "true",
  providerVersion: packageJson.version,
};

console.log("Starting s3 server");
s3rver.run().then(() => {
  console.log("Starting pact verifier");
  new Verifier().verifyProvider(opts).then(() => {
    console.log("Stopping s3 server");
    s3rver.close();
  })
  .catch((err) => {
    console.log("Stopping s3 server");
    s3rver.close();
    console.error(err);
    process.exit(1);
  });
});
