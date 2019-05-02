const path = require("path");
const Pact = require("@pact-foundation/pact").Pact;

global.__basedir = __dirname;

global.port = 3333;
global.provider = new Pact({
  cors: true,
  port: global.port,
  log: path.resolve(process.cwd(), "logs", "pact-api.log"),
  loglevel: "debug",
  dir: path.resolve(process.cwd(), "pacts"),
  spec: 2,
  pactfileWriteMode: "merge",
  consumer: "ship-cluster-ui",
  provider: "ship-cluster-api",
  host: "127.0.0.1"
});
