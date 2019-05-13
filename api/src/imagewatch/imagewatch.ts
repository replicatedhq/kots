
export interface ImageWatch {
  id: string;
  name: string;
  lastCheckedOn: string;
  isPrivate: boolean;
  versionDetected: string;
  latestVersion: string;
  compatibleVersion: string;
  versionsBehind: number;
  path: string;
}

