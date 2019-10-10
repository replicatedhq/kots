export function decodeBase64(data: string): string {
  const buffer = new Buffer(data, 'base64');
  return buffer.toString("ascii");
}
