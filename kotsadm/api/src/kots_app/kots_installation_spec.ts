export interface InstallationSpec {
  updateCursor: string;
  channelName: string;
  versionLabel: string;
  releaseNotes?: string;
  encryptionKey: string;
  knownImages?: InstallationImage[];
  yamlErrors?: InstallationYAMLError[];
}

export interface InstallationImage {
  image: string;
  isPrivate: boolean;
}

export interface InstallationYAMLError {
  path: string;
  error?: string;
}
