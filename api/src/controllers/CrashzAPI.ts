import { Controller, Get } from "@tsed/common";

@Controller("/crashz")
export class CrashzAPI {
  @Get("/")
  async crashIntentionally() {
    throw new Error("Crashz!");
  }
}
