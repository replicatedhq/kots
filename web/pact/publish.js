let publisher = require("@pact-foundation/pact-node");
let path = require("path");

let opts = {
    providerBaseUrl: "http://localhost:8000",
    pactFilesOrDirs: [path.resolve(process.cwd(), "pacts")],
    pactBroker: "https://replicated-pact-broker.herokuapp.com/",
    pactBrokerUsername: process.env.PACT_BROKER_USERNAME,
    pactBrokerPassword: process.env.PACT_BROKER_PASSWORD,
    consumerVersion: require("../package.json").version,
};

publisher.publishPacts(opts).then(() => console.log("Pacts successfully published"));
