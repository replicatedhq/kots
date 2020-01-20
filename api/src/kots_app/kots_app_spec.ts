export interface ApplicationSpec {
  title: string;
  icon?: string;
  ports?: ApplicationPort[];
  releaseNotes?: string;
  allowRollback?: boolean;
  allowSnapshots?: boolean;
  statusInformers?: string[];
  graphs?: MetricGraph[];
  kubectlVersion?: string;
}

export interface ApplicationPort {
  serviceName: string;
  servicePort: number;
  localPort?: number;
  applicationUrl?: string;
}

export interface MetricGraph {
  title: string;
  query?: string;
  legend?: string;
  queries?: MetricQuery[];
  durationSeconds?: number;
  yAxisFormat?: AxisFormat;
  yAxisTemplate?: string;
}

export interface MetricQuery {
  query: string;
  legend?: string;
}

// this lib is dope
// https://github.com/grafana/grafana/blob/009d58c4a228b89046fdae02aa82cf5ff05e5e69/packages/grafana-ui/src/utils/valueFormats/categories.ts
export enum AxisFormat {
  Bytes = "bytes",
  Short = "short",
  Long = "long",
}
