import { expect } from "chai";
import { describe, it } from "mocha";
import { InjectorService } from "ts-express-decorators";
import { DecoratedMutations, DecoratedQueries, Mutation, Query } from "./decorators";

function runBefore(fn: () => any) {
  return (target: any, key: string, descriptor: PropertyDescriptor | undefined) => {
    if (descriptor === undefined) {
      descriptor = Object.getOwnPropertyDescriptor(target, key);
    }
    if (descriptor === undefined) {
      return descriptor;
    }

    const originalMethod = descriptor.value;

    // this needs to be a non-arrow function or we'll get the wrong `this`
    function overrideMethod() {
      const args = arguments;
      const self = this;
      fn();
      return originalMethod.apply(self, args);
    }

    descriptor.value = overrideMethod;
    return descriptor;
  };
}

describe("@Mutation()", () => {
  class FakeClass {
    constructor(private readonly someField: string) {}

    @Mutation("vendor", "entitlements")
    fakeMethod(root: any, params: any, context: any, meta: any): string {
      return this.someField;
    }
  }

  InjectorService.service(FakeClass).set(FakeClass, new FakeClass("lol"));

  it("puts methods in the DecoratedMutations Registry", () => {
    const fake = null as any;
    const field = DecoratedMutations().fakeMethod(fake, fake, fake, fake);
    expect(field).to.equal("lol");
  });

  it("filters by product", () => {
    const fake = null as any;
    const method = DecoratedMutations("troubleshoot").fakeMethod;
    expect(method).to.equal(undefined);
  });
});

describe("@Query()", () => {
  class FakeClass {
    constructor(private readonly someField: string) {}

    @Query("vendor", "entitlements")
    fakeMethod(root: any, params: any, context: any, meta: any): string {
      return this.someField;
    }
  }

  InjectorService.service(FakeClass).set(FakeClass, new FakeClass("lol"));

  it("puts methods in the Query Registry", () => {
    const fake = null as any;
    const field = DecoratedQueries().fakeMethod(fake, fake, fake, fake);
    expect(field).to.equal("lol");
  });

  it("filters by product", () => {
    const fake = null as any;
    const method = DecoratedQueries("troubleshoot").fakeMethod;
    expect(method).to.equal(undefined);
  });
});
