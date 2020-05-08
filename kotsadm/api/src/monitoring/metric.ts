export interface MetricChart {
  title: string;
  tickFormat: string;
  tickTemplate: string;
  series: Series[];
}

export interface Metric {
  name: string;
  value: string;
}

export interface Series {
  legendTemplate: string;
  metric: Metric[];
  data: ValuePair[];
}

export interface ValuePair {
  timestamp: number;
  value: number;
}
