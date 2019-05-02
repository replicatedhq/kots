/* tslint:disable */
import { GraphQLResolveInfo } from "graphql";

export type Resolver<Result, Parent = any, Context = any, Args = any> = (
  parent: Parent,
  args: Args,
  context: Context,
  info: GraphQLResolveInfo,
) => Promise<Result> | Result;

export type SubscriptionResolver<Result, Parent = any, Context = any, Args = any> = {
  subscribe<R = Result, P = Parent>(parent: P, args: Args, context: Context, info: GraphQLResolveInfo): AsyncIterator<R | Result>;
  resolve?<R = Result, P = Parent>(parent: P, args: Args, context: Context, info: GraphQLResolveInfo): R | Result | Promise<R | Result>;
};

export interface Query {
  healthz: Healthz;
  installationOrganizations?: GetInstallationsResponse | null;
  orgRepos?: GetForOrgResponse | null;
  repoBranches?: (GetBranchesResponseItem | null)[] | null;
  githubUser?: GithubUser | null;
  userFeatures?: (Feature | null)[] | null;
  orgMembers?: (GetMembersResponseItem | null)[] | null;
  listClusters?: (ClusterItem | null)[] | null;
  listWatches?: (WatchItem | null)[] | null;
  searchWatches?: (WatchItem | null)[] | null;
  getWatch?: WatchItem | null;
  watchContributors?: (ContributorItem | null)[] | null;
  getWatchVersion?: VersionItemDetail | null;
  listPendingWatchVersions?: (VersionItem | null)[] | null;
  listPastWatchVersions?: (VersionItem | null)[] | null;
  getCurrentWatchVersion?: VersionItem | null;
  validateUpstreamURL: boolean;
  listNotifications?: (Notification | null)[] | null;
  getNotification?: Notification | null;
  pullRequestHistory?: (PullRequestHistoryItem | null)[] | null;
  imageWatchItems?: (ImageWatchItem | null)[] | null;
  getGitHubInstallationId: string;
}

export interface Healthz {
  version?: string | null;
}

export interface GetInstallationsResponse {
  totalCount?: number | null;
  installations?: (GetInstallationsResponseItem | null)[] | null;
}

export interface GetInstallationsResponseItem {
  login?: string | null;
  id?: number | null;
  url?: string | null;
  avatar_url?: string | null;
}

export interface GetForOrgResponse {
  totalCount?: number | null;
  repos?: (GetForOrgResponseItem | null)[] | null;
}

export interface GetForOrgResponseItem {
  id?: number | null;
  node_id?: string | null;
  url?: string | null;
  repository_url?: string | null;
  html_url?: string | null;
  title?: string | null;
  body?: string | null;
  created_at?: string | null;
  updated_at?: string | null;
  name?: string | null;
  full_name?: string | null;
  default_branch?: string | null;
}

export interface GetBranchesResponseItem {
  name?: string | null;
}

export interface GithubUser {
  login?: string | null;
  avatar_url?: string | null;
  id?: number | null;
  email?: string | null;
}

export interface Feature {
  id?: string | null;
}

export interface GetMembersResponseItem {
  id?: number | null;
  login?: string | null;
  avatar_url?: string | null;
}

export interface ClusterItem {
  id?: string | null;
  title?: string | null;
  slug?: string | null;
  lastUpdated?: string | null;
  createdOn?: string | null;
  gitOpsRef?: GitOpsRef | null;
  shipOpsRef?: ShipOpsRef | null;
  watchCounts?: WatchCounts | null;
  totalApplicationCount?: number | null;
  enabled?: boolean | null;
}

export interface GitOpsRef {
  owner: string;
  repo: string;
  branch?: string | null;
  path?: string | null;
}

export interface ShipOpsRef {
  token: string;
}

export interface WatchCounts {
  pending?: number | null;
  past?: number | null;
}

export interface WatchItem {
  id?: string | null;
  stateJSON?: string | null;
  watchName?: string | null;
  slug?: string | null;
  watchIcon?: string | null;
  lastUpdated?: string | null;
  createdOn?: string | null;
  contributors?: (ContributorItem | null)[] | null;
  notifications?: (Notification | null)[] | null;
  features?: (Feature | null)[] | null;
  cluster?: ClusterItem | null;
  watches?: (WatchItem | null)[] | null;
  currentVersion?: VersionItem | null;
  pendingVersions?: (VersionItem | null)[] | null;
  pastVersions?: (VersionItem | null)[] | null;
  parentWatch?: WatchItem | null;
}

export interface ContributorItem {
  id?: string | null;
  createdAt?: string | null;
  githubId?: number | null;
  login?: string | null;
  avatar_url?: string | null;
}

export interface Notification {
  id?: string | null;
  webhook?: WebhookNotification | null;
  email?: EmailNotification | null;
  pullRequest?: PullRequestNotification | null;
  createdOn?: string | null;
  updatedOn?: string | null;
  triggeredOn?: string | null;
  enabled?: number | null;
  isDefault?: boolean | null;
  pending?: boolean | null;
}

export interface WebhookNotification {
  uri?: string | null;
}

export interface EmailNotification {
  recipientAddress?: string | null;
}

export interface PullRequestNotification {
  org: string;
  repo: string;
  branch?: string | null;
  rootPath?: string | null;
}

export interface VersionItem {
  title: string;
  status: string;
  createdOn: string;
  sequence?: number | null;
  pullrequestNumber?: number | null;
}

export interface VersionItemDetail {
  title?: string | null;
  status?: string | null;
  createdOn?: string | null;
  sequence?: number | null;
  pullrequestNumber?: number | null;
  rendered?: string | null;
}

export interface PullRequestHistoryItem {
  title: string;
  status: string;
  createdOn: string;
  number?: number | null;
  uri?: string | null;
  sequence?: number | null;
  sourceBranch?: string | null;
}

export interface ImageWatchItem {
  id?: string | null;
  name?: string | null;
  lastCheckedOn?: string | null;
  isPrivate?: boolean | null;
  versionDetected?: string | null;
  latestVersion?: string | null;
  compatibleVersion?: string | null;
  versionsBehind?: number | null;
  path?: string | null;
}

export interface Mutation {
  ping?: string | null;
  createGithubNonce: string;
  createGithubAuthToken?: AccessToken | null;
  trackScmLead?: string | null;
  refreshGithubTokenMetadata?: string | null;
  logout?: string | null;
  createShipOpsCluster?: ClusterItem | null;
  createGitOpsCluster?: ClusterItem | null;
  updateCluster?: ClusterItem | null;
  deleteCluster?: boolean | null;
  createWatch?: WatchItem | null;
  updateWatch?: WatchItem | null;
  deleteWatch?: boolean | null;
  updateStateJSON?: WatchItem | null;
  deployWatchVersion?: boolean | null;
  saveWatchContributors?: (ContributorItem | null)[] | null;
  createNotification?: Notification | null;
  updateNotification?: Notification | null;
  enableNotification?: Notification | null;
  deleteNotification?: boolean | null;
  createInitSession: InitSession;
  createUnforkSession: UnforkSession;
  createUpdateSession: UpdateSession;
  uploadImageWatchBatch?: string | null;
  createFirstPullRequest?: number | null;
  updatePullRequestHistory?: (PullRequestHistoryItem | null)[] | null;
}

export interface AccessToken {
  access_token: string;
}

export interface InitSession {
  id?: string | null;
  upstreamUri?: string | null;
  createdOn?: string | null;
  finishedOn?: string | null;
  result?: string | null;
}

export interface UnforkSession {
  id?: string | null;
  upstreamUri?: string | null;
  forkUri?: string | null;
  createdOn?: string | null;
  finishedOn?: string | null;
  result?: string | null;
}

export interface UpdateSession {
  id?: string | null;
  watchId?: string | null;
  userId?: string | null;
  createdOn?: string | null;
  finishedOn?: string | null;
  result?: string | null;
}

export interface GitHubIntegration {
  installApp?: string | null;
  installations?: (GitHubInstallation | null)[] | null;
}

export interface GitHubInstallation {
  id?: string | null;
  name: string;
  repos?: (GitHubRepo | null)[] | null;
  accountLogin?: string | null;
  createdAt?: string | null;
}

export interface GitHubRepo {
  name: string;
  fullName: string;
}

export interface GitHubRef {
  owner: string;
  repoFullName: string;
  branch: string;
  path: string;
}

export interface GitHubFile {
  isLoggedIn: boolean;
  fileContents?: string | null;
}

export interface StateMetadata {
  name?: string | null;
  icon?: string | null;
  version?: string | null;
}

export interface PendingPr {
  pullrequest_history_id: string;
  org?: string | null;
  repo?: string | null;
  branch?: string | null;
  root_path?: string | null;
  created_at?: string | null;
  github_installation_id?: number | null;
  pullrequest_number?: number | null;
  watch_id?: string | null;
}

export interface GitOpsRefInput {
  owner: string;
  repo: string;
  branch?: string | null;
}

export interface ContributorItemInput {
  githubId?: number | null;
  login?: string | null;
  avatar_url?: string | null;
}

export interface WebhookNotificationInput {
  uri?: string | null;
}

export interface EmailNotificationInput {
  recipientAddress?: string | null;
}

export interface PullRequestNotificationInput {
  org: string;
  repo: string;
  branch?: string | null;
  rootPath?: string | null;
  pullRequestId?: string | null;
}

export interface GitHubRefInput {
  owner: string;
  repoFullName: string;
  branch: string;
  path: string;
}
export interface InstallationOrganizationsQueryArgs {
  page?: number | null;
}
export interface OrgReposQueryArgs {
  org: string;
  page?: number | null;
}
export interface RepoBranchesQueryArgs {
  owner: string;
  repo: string;
  page?: number | null;
}
export interface OrgMembersQueryArgs {
  org: string;
  page?: number | null;
}
export interface SearchWatchesQueryArgs {
  watchName: string;
}
export interface GetWatchQueryArgs {
  slug?: string | null;
  id?: string | null;
}
export interface WatchContributorsQueryArgs {
  id: string;
}
export interface GetWatchVersionQueryArgs {
  id: string;
  sequence?: number | null;
}
export interface ListPendingWatchVersionsQueryArgs {
  watchId: string;
}
export interface ListPastWatchVersionsQueryArgs {
  watchId: string;
}
export interface GetCurrentWatchVersionQueryArgs {
  watchId: string;
}
export interface ValidateUpstreamUrlQueryArgs {
  upstream: string;
}
export interface ListNotificationsQueryArgs {
  watchId: string;
}
export interface GetNotificationQueryArgs {
  notificationId: string;
}
export interface PullRequestHistoryQueryArgs {
  notificationId: string;
}
export interface ImageWatchItemsQueryArgs {
  batchId: string;
}
export interface CreateGithubAuthTokenMutationArgs {
  state: string;
  code: string;
}
export interface TrackScmLeadMutationArgs {
  deploymentPreference: string;
  emailAddress: string;
  scmProvider: string;
}
export interface CreateShipOpsClusterMutationArgs {
  title: string;
}
export interface CreateGitOpsClusterMutationArgs {
  title: string;
  installationId?: number | null;
  gitOpsRef?: GitOpsRefInput | null;
}
export interface UpdateClusterMutationArgs {
  clusterId: string;
  clusterName: string;
  gitOpsRef?: GitOpsRefInput | null;
}
export interface DeleteClusterMutationArgs {
  clusterId: string;
}
export interface CreateWatchMutationArgs {
  stateJSON: string;
  owner: string;
  clusterID?: string | null;
  githubPath?: string | null;
}
export interface UpdateWatchMutationArgs {
  watchId: string;
  watchName?: string | null;
  iconUri?: string | null;
}
export interface DeleteWatchMutationArgs {
  watchId: string;
  childWatchIds?: (string | null)[] | null;
}
export interface UpdateStateJsonMutationArgs {
  slug: string;
  stateJSON: string;
}
export interface DeployWatchVersionMutationArgs {
  watchId: string;
  sequence?: number | null;
}
export interface SaveWatchContributorsMutationArgs {
  id: string;
  contributors: (ContributorItemInput | null)[];
}
export interface CreateNotificationMutationArgs {
  watchId: string;
  webhook?: WebhookNotificationInput | null;
  email?: EmailNotificationInput | null;
}
export interface UpdateNotificationMutationArgs {
  watchId: string;
  notificationId: string;
  webhook?: WebhookNotificationInput | null;
  email?: EmailNotificationInput | null;
}
export interface EnableNotificationMutationArgs {
  watchId: string;
  notificationId: string;
  enabled: number;
}
export interface DeleteNotificationMutationArgs {
  id: string;
  isPending?: boolean | null;
}
export interface CreateInitSessionMutationArgs {
  upstreamUri: string;
  clusterID?: string | null;
  githubPath?: string | null;
}
export interface CreateUnforkSessionMutationArgs {
  upstreamUri: string;
  forkUri: string;
}
export interface CreateUpdateSessionMutationArgs {
  watchId: string;
}
export interface UploadImageWatchBatchMutationArgs {
  imageList: string;
}
export interface CreateFirstPullRequestMutationArgs {
  watchId: string;
  notificationId?: string | null;
  pullRequest?: PullRequestNotificationInput | null;
}
export interface UpdatePullRequestHistoryMutationArgs {
  notificationId: string;
}

export namespace QueryResolvers {
  export interface Resolvers<Context = any> {
    healthz?: HealthzResolver<Healthz, any, Context>;
    installationOrganizations?: InstallationOrganizationsResolver<GetInstallationsResponse | null, any, Context>;
    orgRepos?: OrgReposResolver<GetForOrgResponse | null, any, Context>;
    repoBranches?: RepoBranchesResolver<(GetBranchesResponseItem | null)[] | null, any, Context>;
    githubUser?: GithubUserResolver<GithubUser | null, any, Context>;
    userFeatures?: UserFeaturesResolver<(Feature | null)[] | null, any, Context>;
    orgMembers?: OrgMembersResolver<(GetMembersResponseItem | null)[] | null, any, Context>;
    listClusters?: ListClustersResolver<(ClusterItem | null)[] | null, any, Context>;
    listWatches?: ListWatchesResolver<(WatchItem | null)[] | null, any, Context>;
    searchWatches?: SearchWatchesResolver<(WatchItem | null)[] | null, any, Context>;
    getWatch?: GetWatchResolver<WatchItem | null, any, Context>;
    watchContributors?: WatchContributorsResolver<(ContributorItem | null)[] | null, any, Context>;
    getWatchVersion?: GetWatchVersionResolver<VersionItemDetail | null, any, Context>;
    listPendingWatchVersions?: ListPendingWatchVersionsResolver<(VersionItem | null)[] | null, any, Context>;
    listPastWatchVersions?: ListPastWatchVersionsResolver<(VersionItem | null)[] | null, any, Context>;
    getCurrentWatchVersion?: GetCurrentWatchVersionResolver<VersionItem | null, any, Context>;
    validateUpstreamURL?: ValidateUpstreamUrlResolver<boolean, any, Context>;
    listNotifications?: ListNotificationsResolver<(Notification | null)[] | null, any, Context>;
    getNotification?: GetNotificationResolver<Notification | null, any, Context>;
    pullRequestHistory?: PullRequestHistoryResolver<(PullRequestHistoryItem | null)[] | null, any, Context>;
    imageWatchItems?: ImageWatchItemsResolver<(ImageWatchItem | null)[] | null, any, Context>;
    getGitHubInstallationId?: GetGitHubInstallationIdResolver<string, any, Context>;
  }

  export type HealthzResolver<R = Healthz, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type InstallationOrganizationsResolver<R = GetInstallationsResponse | null, Parent = any, Context = any> = Resolver<
    R,
    Parent,
    Context,
    InstallationOrganizationsArgs
  >;
  export interface InstallationOrganizationsArgs {
    page?: number | null;
  }

  export type OrgReposResolver<R = GetForOrgResponse | null, Parent = any, Context = any> = Resolver<R, Parent, Context, OrgReposArgs>;
  export interface OrgReposArgs {
    org: string;
    page?: number | null;
  }

  export type RepoBranchesResolver<R = (GetBranchesResponseItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context, RepoBranchesArgs>;
  export interface RepoBranchesArgs {
    owner: string;
    repo: string;
    page?: number | null;
  }

  export type GithubUserResolver<R = GithubUser | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type UserFeaturesResolver<R = (Feature | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type OrgMembersResolver<R = (GetMembersResponseItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context, OrgMembersArgs>;
  export interface OrgMembersArgs {
    org: string;
    page?: number | null;
  }

  export type ListClustersResolver<R = (ClusterItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ListWatchesResolver<R = (WatchItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type SearchWatchesResolver<R = (WatchItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context, SearchWatchesArgs>;
  export interface SearchWatchesArgs {
    watchName: string;
  }

  export type GetWatchResolver<R = WatchItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context, GetWatchArgs>;
  export interface GetWatchArgs {
    slug?: string | null;
    id?: string | null;
  }

  export type WatchContributorsResolver<R = (ContributorItem | null)[] | null, Parent = any, Context = any> = Resolver<
    R,
    Parent,
    Context,
    WatchContributorsArgs
  >;
  export interface WatchContributorsArgs {
    id: string;
  }

  export type GetWatchVersionResolver<R = VersionItemDetail | null, Parent = any, Context = any> = Resolver<R, Parent, Context, GetWatchVersionArgs>;
  export interface GetWatchVersionArgs {
    id: string;
    sequence?: number | null;
  }

  export type ListPendingWatchVersionsResolver<R = (VersionItem | null)[] | null, Parent = any, Context = any> = Resolver<
    R,
    Parent,
    Context,
    ListPendingWatchVersionsArgs
  >;
  export interface ListPendingWatchVersionsArgs {
    watchId: string;
  }

  export type ListPastWatchVersionsResolver<R = (VersionItem | null)[] | null, Parent = any, Context = any> = Resolver<
    R,
    Parent,
    Context,
    ListPastWatchVersionsArgs
  >;
  export interface ListPastWatchVersionsArgs {
    watchId: string;
  }

  export type GetCurrentWatchVersionResolver<R = VersionItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context, GetCurrentWatchVersionArgs>;
  export interface GetCurrentWatchVersionArgs {
    watchId: string;
  }

  export type ValidateUpstreamUrlResolver<R = boolean, Parent = any, Context = any> = Resolver<R, Parent, Context, ValidateUpstreamUrlArgs>;
  export interface ValidateUpstreamUrlArgs {
    upstream: string;
  }

  export type ListNotificationsResolver<R = (Notification | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context, ListNotificationsArgs>;
  export interface ListNotificationsArgs {
    watchId: string;
  }

  export type GetNotificationResolver<R = Notification | null, Parent = any, Context = any> = Resolver<R, Parent, Context, GetNotificationArgs>;
  export interface GetNotificationArgs {
    notificationId: string;
  }

  export type PullRequestHistoryResolver<R = (PullRequestHistoryItem | null)[] | null, Parent = any, Context = any> = Resolver<
    R,
    Parent,
    Context,
    PullRequestHistoryArgs
  >;
  export interface PullRequestHistoryArgs {
    notificationId: string;
  }

  export type ImageWatchItemsResolver<R = (ImageWatchItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context, ImageWatchItemsArgs>;
  export interface ImageWatchItemsArgs {
    batchId: string;
  }

  export type GetGitHubInstallationIdResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace HealthzResolvers {
  export interface Resolvers<Context = any> {
    version?: VersionResolver<string | null, any, Context>;
  }

  export type VersionResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GetInstallationsResponseResolvers {
  export interface Resolvers<Context = any> {
    totalCount?: TotalCountResolver<number | null, any, Context>;
    installations?: InstallationsResolver<(GetInstallationsResponseItem | null)[] | null, any, Context>;
  }

  export type TotalCountResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type InstallationsResolver<R = (GetInstallationsResponseItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GetInstallationsResponseItemResolvers {
  export interface Resolvers<Context = any> {
    login?: LoginResolver<string | null, any, Context>;
    id?: IdResolver<number | null, any, Context>;
    url?: UrlResolver<string | null, any, Context>;
    avatar_url?: AvatarUrlResolver<string | null, any, Context>;
  }

  export type LoginResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type IdResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type UrlResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type AvatarUrlResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GetForOrgResponseResolvers {
  export interface Resolvers<Context = any> {
    totalCount?: TotalCountResolver<number | null, any, Context>;
    repos?: ReposResolver<(GetForOrgResponseItem | null)[] | null, any, Context>;
  }

  export type TotalCountResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ReposResolver<R = (GetForOrgResponseItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GetForOrgResponseItemResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<number | null, any, Context>;
    node_id?: NodeIdResolver<string | null, any, Context>;
    url?: UrlResolver<string | null, any, Context>;
    repository_url?: RepositoryUrlResolver<string | null, any, Context>;
    html_url?: HtmlUrlResolver<string | null, any, Context>;
    title?: TitleResolver<string | null, any, Context>;
    body?: BodyResolver<string | null, any, Context>;
    created_at?: CreatedAtResolver<string | null, any, Context>;
    updated_at?: UpdatedAtResolver<string | null, any, Context>;
    name?: NameResolver<string | null, any, Context>;
    full_name?: FullNameResolver<string | null, any, Context>;
    default_branch?: DefaultBranchResolver<string | null, any, Context>;
  }

  export type IdResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type NodeIdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type UrlResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type RepositoryUrlResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type HtmlUrlResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type TitleResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type BodyResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedAtResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type UpdatedAtResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type NameResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type FullNameResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type DefaultBranchResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GetBranchesResponseItemResolvers {
  export interface Resolvers<Context = any> {
    name?: NameResolver<string | null, any, Context>;
  }

  export type NameResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GithubUserResolvers {
  export interface Resolvers<Context = any> {
    login?: LoginResolver<string | null, any, Context>;
    avatar_url?: AvatarUrlResolver<string | null, any, Context>;
    id?: IdResolver<number | null, any, Context>;
    email?: EmailResolver<string | null, any, Context>;
  }

  export type LoginResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type AvatarUrlResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type IdResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type EmailResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace FeatureResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<string | null, any, Context>;
  }

  export type IdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GetMembersResponseItemResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<number | null, any, Context>;
    login?: LoginResolver<string | null, any, Context>;
    avatar_url?: AvatarUrlResolver<string | null, any, Context>;
  }

  export type IdResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type LoginResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type AvatarUrlResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace ClusterItemResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<string | null, any, Context>;
    title?: TitleResolver<string | null, any, Context>;
    slug?: SlugResolver<string | null, any, Context>;
    lastUpdated?: LastUpdatedResolver<string | null, any, Context>;
    createdOn?: CreatedOnResolver<string | null, any, Context>;
    gitOpsRef?: GitOpsRefResolver<GitOpsRef | null, any, Context>;
    shipOpsRef?: ShipOpsRefResolver<ShipOpsRef | null, any, Context>;
    watchCounts?: WatchCountsResolver<WatchCounts | null, any, Context>;
    totalApplicationCount?: TotalApplicationCountResolver<number | null, any, Context>;
    enabled?: EnabledResolver<boolean | null, any, Context>;
  }

  export type IdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type TitleResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type SlugResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type LastUpdatedResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type GitOpsRefResolver<R = GitOpsRef | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ShipOpsRefResolver<R = ShipOpsRef | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type WatchCountsResolver<R = WatchCounts | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type TotalApplicationCountResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type EnabledResolver<R = boolean | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GitOpsRefResolvers {
  export interface Resolvers<Context = any> {
    owner?: OwnerResolver<string, any, Context>;
    repo?: RepoResolver<string, any, Context>;
    branch?: BranchResolver<string | null, any, Context>;
    path?: PathResolver<string | null, any, Context>;
  }

  export type OwnerResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type RepoResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type BranchResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PathResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace ShipOpsRefResolvers {
  export interface Resolvers<Context = any> {
    token?: TokenResolver<string, any, Context>;
  }

  export type TokenResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace WatchCountsResolvers {
  export interface Resolvers<Context = any> {
    pending?: PendingResolver<number | null, any, Context>;
    past?: PastResolver<number | null, any, Context>;
  }

  export type PendingResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PastResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace WatchItemResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<string | null, any, Context>;
    stateJSON?: StateJsonResolver<string | null, any, Context>;
    watchName?: WatchNameResolver<string | null, any, Context>;
    slug?: SlugResolver<string | null, any, Context>;
    watchIcon?: WatchIconResolver<string | null, any, Context>;
    lastUpdated?: LastUpdatedResolver<string | null, any, Context>;
    createdOn?: CreatedOnResolver<string | null, any, Context>;
    contributors?: ContributorsResolver<(ContributorItem | null)[] | null, any, Context>;
    notifications?: NotificationsResolver<(Notification | null)[] | null, any, Context>;
    features?: FeaturesResolver<(Feature | null)[] | null, any, Context>;
    cluster?: ClusterResolver<ClusterItem | null, any, Context>;
    watches?: WatchesResolver<(WatchItem | null)[] | null, any, Context>;
    currentVersion?: CurrentVersionResolver<VersionItem | null, any, Context>;
    pendingVersions?: PendingVersionsResolver<(VersionItem | null)[] | null, any, Context>;
    pastVersions?: PastVersionsResolver<(VersionItem | null)[] | null, any, Context>;
    parentWatch?: ParentWatchResolver<WatchItem | null, any, Context>;
  }

  export type IdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type StateJsonResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type WatchNameResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type SlugResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type WatchIconResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type LastUpdatedResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ContributorsResolver<R = (ContributorItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type NotificationsResolver<R = (Notification | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type FeaturesResolver<R = (Feature | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ClusterResolver<R = ClusterItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type WatchesResolver<R = (WatchItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CurrentVersionResolver<R = VersionItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PendingVersionsResolver<R = (VersionItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PastVersionsResolver<R = (VersionItem | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ParentWatchResolver<R = WatchItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace ContributorItemResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<string | null, any, Context>;
    createdAt?: CreatedAtResolver<string | null, any, Context>;
    githubId?: GithubIdResolver<number | null, any, Context>;
    login?: LoginResolver<string | null, any, Context>;
    avatar_url?: AvatarUrlResolver<string | null, any, Context>;
  }

  export type IdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedAtResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type GithubIdResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type LoginResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type AvatarUrlResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace NotificationResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<string | null, any, Context>;
    webhook?: WebhookResolver<WebhookNotification | null, any, Context>;
    email?: EmailResolver<EmailNotification | null, any, Context>;
    pullRequest?: PullRequestResolver<PullRequestNotification | null, any, Context>;
    createdOn?: CreatedOnResolver<string | null, any, Context>;
    updatedOn?: UpdatedOnResolver<string | null, any, Context>;
    triggeredOn?: TriggeredOnResolver<string | null, any, Context>;
    enabled?: EnabledResolver<number | null, any, Context>;
    isDefault?: IsDefaultResolver<boolean | null, any, Context>;
    pending?: PendingResolver<boolean | null, any, Context>;
  }

  export type IdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type WebhookResolver<R = WebhookNotification | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type EmailResolver<R = EmailNotification | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PullRequestResolver<R = PullRequestNotification | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type UpdatedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type TriggeredOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type EnabledResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type IsDefaultResolver<R = boolean | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PendingResolver<R = boolean | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace WebhookNotificationResolvers {
  export interface Resolvers<Context = any> {
    uri?: UriResolver<string | null, any, Context>;
  }

  export type UriResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace EmailNotificationResolvers {
  export interface Resolvers<Context = any> {
    recipientAddress?: RecipientAddressResolver<string | null, any, Context>;
  }

  export type RecipientAddressResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace PullRequestNotificationResolvers {
  export interface Resolvers<Context = any> {
    org?: OrgResolver<string, any, Context>;
    repo?: RepoResolver<string, any, Context>;
    branch?: BranchResolver<string | null, any, Context>;
    rootPath?: RootPathResolver<string | null, any, Context>;
  }

  export type OrgResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type RepoResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type BranchResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type RootPathResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace VersionItemResolvers {
  export interface Resolvers<Context = any> {
    title?: TitleResolver<string, any, Context>;
    status?: StatusResolver<string, any, Context>;
    createdOn?: CreatedOnResolver<string, any, Context>;
    sequence?: SequenceResolver<number | null, any, Context>;
    pullrequestNumber?: PullrequestNumberResolver<number | null, any, Context>;
  }

  export type TitleResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type StatusResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedOnResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type SequenceResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PullrequestNumberResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace VersionItemDetailResolvers {
  export interface Resolvers<Context = any> {
    title?: TitleResolver<string | null, any, Context>;
    status?: StatusResolver<string | null, any, Context>;
    createdOn?: CreatedOnResolver<string | null, any, Context>;
    sequence?: SequenceResolver<number | null, any, Context>;
    pullrequestNumber?: PullrequestNumberResolver<number | null, any, Context>;
    rendered?: RenderedResolver<string | null, any, Context>;
  }

  export type TitleResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type StatusResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type SequenceResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PullrequestNumberResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type RenderedResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace PullRequestHistoryItemResolvers {
  export interface Resolvers<Context = any> {
    title?: TitleResolver<string, any, Context>;
    status?: StatusResolver<string, any, Context>;
    createdOn?: CreatedOnResolver<string, any, Context>;
    number?: NumberResolver<number | null, any, Context>;
    uri?: UriResolver<string | null, any, Context>;
    sequence?: SequenceResolver<number | null, any, Context>;
    sourceBranch?: SourceBranchResolver<string | null, any, Context>;
  }

  export type TitleResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type StatusResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedOnResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type NumberResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type UriResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type SequenceResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type SourceBranchResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace ImageWatchItemResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<string | null, any, Context>;
    name?: NameResolver<string | null, any, Context>;
    lastCheckedOn?: LastCheckedOnResolver<string | null, any, Context>;
    isPrivate?: IsPrivateResolver<boolean | null, any, Context>;
    versionDetected?: VersionDetectedResolver<string | null, any, Context>;
    latestVersion?: LatestVersionResolver<string | null, any, Context>;
    compatibleVersion?: CompatibleVersionResolver<string | null, any, Context>;
    versionsBehind?: VersionsBehindResolver<number | null, any, Context>;
    path?: PathResolver<string | null, any, Context>;
  }

  export type IdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type NameResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type LastCheckedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type IsPrivateResolver<R = boolean | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type VersionDetectedResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type LatestVersionResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CompatibleVersionResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type VersionsBehindResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PathResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace MutationResolvers {
  export interface Resolvers<Context = any> {
    ping?: PingResolver<string | null, any, Context>;
    createGithubNonce?: CreateGithubNonceResolver<string, any, Context>;
    createGithubAuthToken?: CreateGithubAuthTokenResolver<AccessToken | null, any, Context>;
    trackScmLead?: TrackScmLeadResolver<string | null, any, Context>;
    refreshGithubTokenMetadata?: RefreshGithubTokenMetadataResolver<string | null, any, Context>;
    logout?: LogoutResolver<string | null, any, Context>;
    createShipOpsCluster?: CreateShipOpsClusterResolver<ClusterItem | null, any, Context>;
    createGitOpsCluster?: CreateGitOpsClusterResolver<ClusterItem | null, any, Context>;
    updateCluster?: UpdateClusterResolver<ClusterItem | null, any, Context>;
    deleteCluster?: DeleteClusterResolver<boolean | null, any, Context>;
    createWatch?: CreateWatchResolver<WatchItem | null, any, Context>;
    updateWatch?: UpdateWatchResolver<WatchItem | null, any, Context>;
    deleteWatch?: DeleteWatchResolver<boolean | null, any, Context>;
    updateStateJSON?: UpdateStateJsonResolver<WatchItem | null, any, Context>;
    deployWatchVersion?: DeployWatchVersionResolver<boolean | null, any, Context>;
    saveWatchContributors?: SaveWatchContributorsResolver<(ContributorItem | null)[] | null, any, Context>;
    createNotification?: CreateNotificationResolver<Notification | null, any, Context>;
    updateNotification?: UpdateNotificationResolver<Notification | null, any, Context>;
    enableNotification?: EnableNotificationResolver<Notification | null, any, Context>;
    deleteNotification?: DeleteNotificationResolver<boolean | null, any, Context>;
    createInitSession?: CreateInitSessionResolver<InitSession, any, Context>;
    createUnforkSession?: CreateUnforkSessionResolver<UnforkSession, any, Context>;
    createUpdateSession?: CreateUpdateSessionResolver<UpdateSession, any, Context>;
    uploadImageWatchBatch?: UploadImageWatchBatchResolver<string | null, any, Context>;
    createFirstPullRequest?: CreateFirstPullRequestResolver<number | null, any, Context>;
    updatePullRequestHistory?: UpdatePullRequestHistoryResolver<(PullRequestHistoryItem | null)[] | null, any, Context>;
  }

  export type PingResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreateGithubNonceResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreateGithubAuthTokenResolver<R = AccessToken | null, Parent = any, Context = any> = Resolver<R, Parent, Context, CreateGithubAuthTokenArgs>;
  export interface CreateGithubAuthTokenArgs {
    state: string;
    code: string;
  }

  export type TrackScmLeadResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context, TrackScmLeadArgs>;
  export interface TrackScmLeadArgs {
    deploymentPreference: string;
    emailAddress: string;
    scmProvider: string;
  }

  export type RefreshGithubTokenMetadataResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type LogoutResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreateShipOpsClusterResolver<R = ClusterItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context, CreateShipOpsClusterArgs>;
  export interface CreateShipOpsClusterArgs {
    title: string;
  }

  export type CreateGitOpsClusterResolver<R = ClusterItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context, CreateGitOpsClusterArgs>;
  export interface CreateGitOpsClusterArgs {
    title: string;
    installationId?: number | null;
    gitOpsRef?: GitOpsRefInput | null;
  }

  export type UpdateClusterResolver<R = ClusterItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context, UpdateClusterArgs>;
  export interface UpdateClusterArgs {
    clusterId: string;
    clusterName: string;
    gitOpsRef?: GitOpsRefInput | null;
  }

  export type DeleteClusterResolver<R = boolean | null, Parent = any, Context = any> = Resolver<R, Parent, Context, DeleteClusterArgs>;
  export interface DeleteClusterArgs {
    clusterId: string;
  }

  export type CreateWatchResolver<R = WatchItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context, CreateWatchArgs>;
  export interface CreateWatchArgs {
    stateJSON: string;
    owner: string;
    clusterID?: string | null;
    githubPath?: string | null;
  }

  export type UpdateWatchResolver<R = WatchItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context, UpdateWatchArgs>;
  export interface UpdateWatchArgs {
    watchId: string;
    watchName?: string | null;
    iconUri?: string | null;
  }

  export type DeleteWatchResolver<R = boolean | null, Parent = any, Context = any> = Resolver<R, Parent, Context, DeleteWatchArgs>;
  export interface DeleteWatchArgs {
    watchId: string;
    childWatchIds?: (string | null)[] | null;
  }

  export type UpdateStateJsonResolver<R = WatchItem | null, Parent = any, Context = any> = Resolver<R, Parent, Context, UpdateStateJsonArgs>;
  export interface UpdateStateJsonArgs {
    slug: string;
    stateJSON: string;
  }

  export type DeployWatchVersionResolver<R = boolean | null, Parent = any, Context = any> = Resolver<R, Parent, Context, DeployWatchVersionArgs>;
  export interface DeployWatchVersionArgs {
    watchId: string;
    sequence?: number | null;
  }

  export type SaveWatchContributorsResolver<R = (ContributorItem | null)[] | null, Parent = any, Context = any> = Resolver<
    R,
    Parent,
    Context,
    SaveWatchContributorsArgs
  >;
  export interface SaveWatchContributorsArgs {
    id: string;
    contributors: (ContributorItemInput | null)[];
  }

  export type CreateNotificationResolver<R = Notification | null, Parent = any, Context = any> = Resolver<R, Parent, Context, CreateNotificationArgs>;
  export interface CreateNotificationArgs {
    watchId: string;
    webhook?: WebhookNotificationInput | null;
    email?: EmailNotificationInput | null;
  }

  export type UpdateNotificationResolver<R = Notification | null, Parent = any, Context = any> = Resolver<R, Parent, Context, UpdateNotificationArgs>;
  export interface UpdateNotificationArgs {
    watchId: string;
    notificationId: string;
    webhook?: WebhookNotificationInput | null;
    email?: EmailNotificationInput | null;
  }

  export type EnableNotificationResolver<R = Notification | null, Parent = any, Context = any> = Resolver<R, Parent, Context, EnableNotificationArgs>;
  export interface EnableNotificationArgs {
    watchId: string;
    notificationId: string;
    enabled: number;
  }

  export type DeleteNotificationResolver<R = boolean | null, Parent = any, Context = any> = Resolver<R, Parent, Context, DeleteNotificationArgs>;
  export interface DeleteNotificationArgs {
    id: string;
    isPending?: boolean | null;
  }

  export type CreateInitSessionResolver<R = InitSession, Parent = any, Context = any> = Resolver<R, Parent, Context, CreateInitSessionArgs>;
  export interface CreateInitSessionArgs {
    upstreamUri: string;
    clusterID?: string | null;
    githubPath?: string | null;
  }

  export type CreateUnforkSessionResolver<R = UnforkSession, Parent = any, Context = any> = Resolver<R, Parent, Context, CreateUnforkSessionArgs>;
  export interface CreateUnforkSessionArgs {
    upstreamUri: string;
    forkUri: string;
  }

  export type CreateUpdateSessionResolver<R = UpdateSession, Parent = any, Context = any> = Resolver<R, Parent, Context, CreateUpdateSessionArgs>;
  export interface CreateUpdateSessionArgs {
    watchId: string;
  }

  export type UploadImageWatchBatchResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context, UploadImageWatchBatchArgs>;
  export interface UploadImageWatchBatchArgs {
    imageList: string;
  }

  export type CreateFirstPullRequestResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context, CreateFirstPullRequestArgs>;
  export interface CreateFirstPullRequestArgs {
    watchId: string;
    notificationId?: string | null;
    pullRequest?: PullRequestNotificationInput | null;
  }

  export type UpdatePullRequestHistoryResolver<R = (PullRequestHistoryItem | null)[] | null, Parent = any, Context = any> = Resolver<
    R,
    Parent,
    Context,
    UpdatePullRequestHistoryArgs
  >;
  export interface UpdatePullRequestHistoryArgs {
    notificationId: string;
  }
}

export namespace AccessTokenResolvers {
  export interface Resolvers<Context = any> {
    access_token?: AccessTokenResolver<string, any, Context>;
  }

  export type AccessTokenResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace InitSessionResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<string | null, any, Context>;
    upstreamUri?: UpstreamUriResolver<string | null, any, Context>;
    createdOn?: CreatedOnResolver<string | null, any, Context>;
    finishedOn?: FinishedOnResolver<string | null, any, Context>;
    result?: ResultResolver<string | null, any, Context>;
  }

  export type IdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type UpstreamUriResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type FinishedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ResultResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace UnforkSessionResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<string | null, any, Context>;
    upstreamUri?: UpstreamUriResolver<string | null, any, Context>;
    forkUri?: ForkUriResolver<string | null, any, Context>;
    createdOn?: CreatedOnResolver<string | null, any, Context>;
    finishedOn?: FinishedOnResolver<string | null, any, Context>;
    result?: ResultResolver<string | null, any, Context>;
  }

  export type IdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type UpstreamUriResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ForkUriResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type FinishedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ResultResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace UpdateSessionResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<string | null, any, Context>;
    watchId?: WatchIdResolver<string | null, any, Context>;
    userId?: UserIdResolver<string | null, any, Context>;
    createdOn?: CreatedOnResolver<string | null, any, Context>;
    finishedOn?: FinishedOnResolver<string | null, any, Context>;
    result?: ResultResolver<string | null, any, Context>;
  }

  export type IdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type WatchIdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type UserIdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type FinishedOnResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ResultResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GitHubIntegrationResolvers {
  export interface Resolvers<Context = any> {
    installApp?: InstallAppResolver<string | null, any, Context>;
    installations?: InstallationsResolver<(GitHubInstallation | null)[] | null, any, Context>;
  }

  export type InstallAppResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type InstallationsResolver<R = (GitHubInstallation | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GitHubInstallationResolvers {
  export interface Resolvers<Context = any> {
    id?: IdResolver<string | null, any, Context>;
    name?: NameResolver<string, any, Context>;
    repos?: ReposResolver<(GitHubRepo | null)[] | null, any, Context>;
    accountLogin?: AccountLoginResolver<string | null, any, Context>;
    createdAt?: CreatedAtResolver<string | null, any, Context>;
  }

  export type IdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type NameResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type ReposResolver<R = (GitHubRepo | null)[] | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type AccountLoginResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedAtResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GitHubRepoResolvers {
  export interface Resolvers<Context = any> {
    name?: NameResolver<string, any, Context>;
    fullName?: FullNameResolver<string, any, Context>;
  }

  export type NameResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type FullNameResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GitHubRefResolvers {
  export interface Resolvers<Context = any> {
    owner?: OwnerResolver<string, any, Context>;
    repoFullName?: RepoFullNameResolver<string, any, Context>;
    branch?: BranchResolver<string, any, Context>;
    path?: PathResolver<string, any, Context>;
  }

  export type OwnerResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type RepoFullNameResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type BranchResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PathResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace GitHubFileResolvers {
  export interface Resolvers<Context = any> {
    isLoggedIn?: IsLoggedInResolver<boolean, any, Context>;
    fileContents?: FileContentsResolver<string | null, any, Context>;
  }

  export type IsLoggedInResolver<R = boolean, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type FileContentsResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace StateMetadataResolvers {
  export interface Resolvers<Context = any> {
    name?: NameResolver<string | null, any, Context>;
    icon?: IconResolver<string | null, any, Context>;
    version?: VersionResolver<string | null, any, Context>;
  }

  export type NameResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type IconResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type VersionResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}

export namespace PendingPrResolvers {
  export interface Resolvers<Context = any> {
    pullrequest_history_id?: PullrequestHistoryIdResolver<string, any, Context>;
    org?: OrgResolver<string | null, any, Context>;
    repo?: RepoResolver<string | null, any, Context>;
    branch?: BranchResolver<string | null, any, Context>;
    root_path?: RootPathResolver<string | null, any, Context>;
    created_at?: CreatedAtResolver<string | null, any, Context>;
    github_installation_id?: GithubInstallationIdResolver<number | null, any, Context>;
    pullrequest_number?: PullrequestNumberResolver<number | null, any, Context>;
    watch_id?: WatchIdResolver<string | null, any, Context>;
  }

  export type PullrequestHistoryIdResolver<R = string, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type OrgResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type RepoResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type BranchResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type RootPathResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type CreatedAtResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type GithubInstallationIdResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type PullrequestNumberResolver<R = number | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
  export type WatchIdResolver<R = string | null, Parent = any, Context = any> = Resolver<R, Parent, Context>;
}
