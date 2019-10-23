import pg from "pg";
import { Params } from "../server/params";

export class KurlStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

}
