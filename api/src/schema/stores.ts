import { SessionStore } from "../session/session_store";
import { UserStore } from "../user";
import { ClusterStore } from "../cluster";
import { WatchStore } from "../watch/watch_store";
import { NotificationStore } from "../notification/store";
import { UpdateStore } from "../update/store";
import { UnforkStore } from "../unfork/unfork_store";
import { InitStore } from "../init/init_store";
import { ImageWatchStore } from "../imagewatch/store";
import { FeatureStore } from "../feature/feature_store";

export interface Stores {
  sessionStore: SessionStore;
  userStore: UserStore;
  clsuterStore: ClusterStore;
  watchStore: WatchStore,
  notificationStore: NotificationStore,
  updateStore: UpdateStore,
  unforkStore: UnforkStore,
  initStore: InitStore,
  imageWatchStore: ImageWatchStore,
  featureStore: FeatureStore,
}
