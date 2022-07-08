export default function fetchWithTimeout(url, options, timeout = undefined) {
  const controller = new AbortController();
  const { signal } = controller;

  if (timeout) {
    setTimeout(() => controller.abort(), timeout);
  }

  return fetch(url, { ...options, signal });
}
