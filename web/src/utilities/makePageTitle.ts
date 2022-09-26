function makePageTitle({
  appName,
  pageName,
}: {
  appName?: string;
  pageName: string;
}): string {
  if (appName) {
    return `${appName} | ${pageName} | Admin Console`;
  }

  return `${pageName} | Admin Console`;
}

export { makePageTitle };
