import Analytics from "analytics-node";
import { Params } from "../server/params";

export function getAnalytics(params: Params){
  return new Analytics(params.segmentioAnalyticsKey, { flushAt: 1 });
};


export function identifyUser(params: Params, userId: string, username: string) {
  const analytics = getAnalytics(params);
  analytics.identify({
    userId: userId,
    traits: {
      username: username
    }
  });
};

export function trackUserClusterCreated(params: Params, userId: string, event: string, properties?: string) {
  const analytics = getAnalytics(params);
  analytics.track({
    userId: userId,
    event: event,
    properties: {
      owner: properties
    },
  });
};

export function trackUserSCMLeads(params: Params, anonymousId: string, event: string, email: string, deploymentType: string, scmProvider: string) {
  const analytics = getAnalytics(params);
  analytics.track({
    anonymousId: anonymousId,
    event: event,
    properties: {
      email: email,
      deploymentType: deploymentType,
      scmProvider: scmProvider
    },
  });
};