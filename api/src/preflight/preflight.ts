export class PreflightResult {
  public watchId: string;
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

export class KotsPreflightResult {
  public result: string;
  public updatedAt: string;
  public clusterId: string;

  public toSchema(): any {
    return {
      ...this
    };
  }
}
