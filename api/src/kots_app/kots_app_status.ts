
export enum State {
  Ready = "ready",
  Degraded = "degraded",
  Unavailable = "unavailable",
  Missing = "missing",
}

export interface ResourceState {
  kind: string;
  name: string;
  namespace: string;
  state: State;
}

export class KotsAppStatus {
  appId: string;
  updatedAt: Date;
  resourceStates: ResourceState[];

  private getState(): State {
    if (!this.resourceStates) {
      return State.Missing;
    }
    let max = State.Ready;
    this.resourceStates.forEach(resourceState => {
      max = maxState(max, resourceState.state);
    });
    return max;
  };

  public toSchema(): any {
    return {
      appId: this.appId,
      updatedAt: this.updatedAt,
      resourceStates: this.resourceStates,
      state: () => this.getState(),
    };
  }
}

function maxState(a: State, b: State): State {
  if (a === State.Missing || b === State.Missing) {
    return State.Missing;
  }
  if (a === State.Unavailable || b === State.Unavailable) {
    return State.Unavailable;
  }
  if (a === State.Degraded || b === State.Degraded) {
    return State.Degraded;
  }
  if (a === State.Ready || b === State.Ready) {
    return State.Ready;
  }
  return State.Missing
}
