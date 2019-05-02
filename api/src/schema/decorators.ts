import { InjectorService } from "ts-express-decorators";
import { ResolveFunc } from "./types";

export type Label = string;

export interface ResolverGroup {
  [key: string]: ResolveFunc;
}

export interface ResolverRegistry {
  [key: string]: ResolverGroup;
}

const mutationRegistry: ResolverRegistry = {};
const queryRegistry: ResolverRegistry = {};

export function Query(...labels: Label[]) {
  return (target: {}, key: string, descriptor: PropertyDescriptor | undefined) => {
    let desc = descriptor;
    if (descriptor === undefined) {
      desc = Object.getOwnPropertyDescriptor(target, key);
    }
    if (descriptor === undefined) {
      return descriptor;
    }

    const originalMethod = descriptor.value;

    // this needs to be a non-arrow function or we'll get the wrong `this`
    function resolvedMethod() {
      const args = arguments;
      const self = InjectorService.get(target.constructor);

      return originalMethod.apply(self, args);
    }

    for (const label of labels) {
      queryRegistry[label] = queryRegistry[label] || {};
      queryRegistry[label][key] = resolvedMethod;
    }

    return descriptor;
  };
}

export function Mutation(...labels: Label[]) {
  return (target: {}, key: string, inDescriptor: PropertyDescriptor | undefined) => {
    let descriptor = inDescriptor;
    if (descriptor === undefined) {
      descriptor = Object.getOwnPropertyDescriptor(target, key);
    }
    if (descriptor === undefined) {
      return descriptor;
    }

    const originalMethod = descriptor.value;

    // this needs to be a non-arrow function or we'll get the wrong `this`
    function resolvedMethod() {
      const args = arguments;
      const self = InjectorService.get(target.constructor);

      return originalMethod.apply(self, args);
    }

    for (const label of labels) {
      mutationRegistry[label] = mutationRegistry[label] || {};
      mutationRegistry[label][key] = resolvedMethod;
    }

    return descriptor;
  };
}

export function DecoratedMutations(label?: Label): ResolverGroup {
  return filterOrFlatten(mutationRegistry, label);
}

export function DecoratedQueries(label?: Label): ResolverGroup {
  return filterOrFlatten(queryRegistry, label);
}

function filterOrFlatten(registry: ResolverRegistry, label?: Label): ResolverGroup {
  const empty: ResolverGroup = {};

  if (label) {
    if (registry[label]) {
      return registry[label];
    }

    return empty;
  }

  return flatten(registry);
}

function flatten(registry: ResolverRegistry): ResolverGroup {
  const all: ResolverGroup = {};
  for (const label of Object.keys(registry)) {
    for (const method of Object.keys(registry[label])) {
      all[method] = registry[label][method];
    }
  }

  return all;
}
