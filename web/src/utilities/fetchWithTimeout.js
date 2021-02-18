export default function (url, options, timeout = undefined) {
  const controller = new AbortController();
  const signal = controller.signal;

  if (timeout) {
    setTimeout(() => controller.abort(), timeout);
  }

  return fetch(url, { ...options, signal });
}