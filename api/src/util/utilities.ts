export function decodeBase64(data: string): string {
  if (!data) {
    return "";
  }
  const buffer = new Buffer(data, 'base64');
  return buffer.toString("ascii");
}

export function getPreflightResultState(preflightResults): string {
  const results = preflightResults.results;
  let resultState = "pass";
  for (const check of results) {
    if (check.isWarn) {
      resultState = "warn";
    } else if (check.isFail) {
      return "fail";
    }
  }
  return resultState;
}
