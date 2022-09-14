export default function fetchWithTimeout(
  url: string,
  options:
    | RequestInit
    | {
        headers: {
          Authorization: string | null;
          "Content-Type": string;
        };
      },
  timeout?: number
) {
  const controller = new AbortController();
  const { signal } = controller;

  if (timeout) {
    setTimeout(() => controller.abort(), timeout);
  }

  return fetch(url, { ...options, signal });
}
