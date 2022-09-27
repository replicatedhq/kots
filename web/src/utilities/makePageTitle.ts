function makePageTitle({
  appSlug,
  pageName,
}: {
  appSlug?: string;
  pageName: string;
}): string {
  if (appSlug) {
    return `${pageName} | ${appSlug} | Admin Console`;
  }

  return `${pageName} | Admin Console`;
}

export { makePageTitle };
