const { retry } = require('ts-retry');
const { execSync } = require("child_process");

retry(
  () => {
    const getAppPodCommand = `kubectl get pod -l app=example,component=nginx -n qakotsregression | grep example-nginx`;
    console.log(getAppPodCommand, "\n");
    execSync(getAppPodCommand, {stdio: 'inherit'});
  },
  { delay: 1000, maxTry: 10 }
).then(() => {
  console.log("Pod found");
});
