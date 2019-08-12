import { Watch } from "../watch";
// import { KotsApps } from "../kots";
import { HelmChart } from "../helmchart";

export interface Apps {
  watches?: Array<Watch>;
  kotsApps?: string;
  pendingUnforks?: Array<HelmChart>,
}
