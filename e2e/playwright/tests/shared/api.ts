const { execSync } = require("child_process");

export async function downloadAirgapBundle(
  customerID: string,
  channelSequence: number,
  portalBase64Password: string,
  destPath: string
) {
  // get airgap bundle download url
  const getBundleURLCommand = `curl -XGET "https://api.replicated.com/market/v3/airgap/images/url?customer_id=${customerID}&channel_sequence=${channelSequence}" -H 'Authorization: Basic ${portalBase64Password}'`;
  console.log(getBundleURLCommand, "\n");
  let output = execSync(getBundleURLCommand).toString();
  const bundleUrl = JSON.parse(output).url

  // download airgap bundle
  const downloadCommand = `curl '${bundleUrl}' -o ${destPath}`;
  execSync(downloadCommand, {stdio: 'inherit'});
}
