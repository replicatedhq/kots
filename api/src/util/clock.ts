import moment from "moment";

// helps with testability
export interface Clock {
  now(): moment.Moment;
}

export class DefaultClock implements Clock {
  now() {
    return moment();
  }
}
