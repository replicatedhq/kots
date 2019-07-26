import util from "util";
import { getPostgresPool } from "../util/persistence/db";
import { Params } from "../server/params";

export const name = "migrate-downstream-cluster-users";
export const describe = "DB Migration to ensure downstream cluster contributors are in user_watch cluster"
export const builder = {

};

export const handler = async argv => {
  main(argv).catch(err => {
    console.log(`Faiuled with error ${util.inspect(err)}`);
    process.exit(1);
  })
}

async function main(argv): Promise<any> {
  process.on("SIGTERM", function onSigterm() {
    process.exit();
  });

  console.log("Migrating downstream watches to include contributors in user_watch table");

  const pool = await getPostgresPool();
  // const params = await Params.getParams();

  const downstreamWatches = await pool.query(
    `SELECT watch.id, watch.parent_watch_id FROM watch WHERE watch.parent_watch_id IS NOT NULL`
  );

  for (const downstream of downstreamWatches.rows) {
    const { id: downstream_id, parent_watch_id } = downstream;

    const parent_watch_contributors = await pool.query(
      `SELECT ship_user.id as contributor_id
      FROM user_watch
        JOIN ship_user ON ship_user.id = user_watch.user_id
      WHERE user_watch.watch_id = $1`
      [parent_watch_id]
    );

    for (const contributor of parent_watch_contributors.rows) {
      const { id: contributor_id } = contributor;

      await pool.query(
        `INSERT INTO user_watch (user_watch.user_id, user_watch.watch_id) VALUES ($1, $2)`,
        [contributor_id, downstream_id]
      );
    }

  }

  process.exit(0);

}
