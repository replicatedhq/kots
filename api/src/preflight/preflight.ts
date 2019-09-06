export class PreflightResult {
  public appId: string;
  public appSlug: string;
  public clusterSlug: string;
  public clusterId: string;
  public result: string;
  public createdAt: string;

  public toSchema():any {
    return {
      ...this
    };
  }
};

export class PreflightSpec {
  public spec: string;

  public toSchema(): any {
    return {
      ...this
    };
  }
}
