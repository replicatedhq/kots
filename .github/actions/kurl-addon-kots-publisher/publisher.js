import semver from 'semver';

export const appendVersion = (kotsAddonVersions, version) => {
  kotsAddonVersions = kotsAddonVersions.filter(el => el.version !== version.version);
  kotsAddonVersions.unshift(version);
  return kotsAddonVersions.sort((a, b) => semver.compare(a.version, b.version)).reverse();
};
