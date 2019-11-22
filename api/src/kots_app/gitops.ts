import { KotsAppStore } from "./kots_app_store";

export async function sendInitialGitCommitsForAppDownstream(kotsAppStore: KotsAppStore, appId: string, clusterId: string): Promise<any> {
  const gitOpsCreds = await kotsAppStore.getGitOpsCreds(appId, clusterId);

  const currentVersion = await kotsAppStore.getCurrentVersion(appId, clusterId);
  if (currentVersion) {
    await createGitCommit(gitOpsCreds);
  }

  const pendingVersions = await kotsAppStore.listPendingVersions(appId, clusterId);
  for (const pendingVersion of pendingVersions) {
    await createGitCommit(gitOpsCreds);
  }
}

export async function createGitCommit(gitOpsCreds: any): Promise<any> {

}
