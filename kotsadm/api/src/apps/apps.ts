import { KotsApp } from "../kots_app";
import { HelmChart } from "../helmchart";

export interface Apps {
  kotsApps?: Array<KotsApp>;
  pendingUnforks?: Array<HelmChart>;
}
