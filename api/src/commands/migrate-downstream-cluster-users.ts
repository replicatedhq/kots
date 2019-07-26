import util from "util";
import { getPostgresPool } from "../util/persistence/db";

export const name = "migrate-downstream-cluster-users";
export const describe = "DB Migration to ensure downstream cluster contributors are in user_watch cluster"
export const builder = {

};

export const handler = async argv => {
  main(argv).catch(err => {
    console.log(`Failed with error ${util.inspect(err)}`);
    process.exit(1);
  });
}

async function main(argv): Promise<any> {
  process.on("SIGTERM", () => {
    process.exit();
  });

  console.log("Migrating downstream watches to include contributors in user_watch table");

  const pool = await getPostgresPool();
  console.log("Getting downstream watches... ");
  const downstreamWatches = await pool.query(
    `SELECT watch.id, watch.parent_watch_id FROM watch WHERE watch.parent_watch_id IS NOT NULL`
  );


  for (const downstream of downstreamWatches.rows) {
    const { id: downstream_id, parent_watch_id } = downstream;

    // If the downstream_id is a pact test fixture, skip it.
    if (downstream_id.includes('-')) { continue; }
    console.log(`Getting contributors for ${downstream_id}, Parent: ${parent_watch_id}`);
    const parent_watch_contributors = await pool.query(
      `SELECT ship_user.id as contributor_id
      FROM user_watch
        JOIN ship_user ON ship_user.id = user_watch.user_id
      WHERE user_watch.watch_id = $1`,
      [parent_watch_id]
    );

    for (const contributor of parent_watch_contributors.rows) {
      const { contributor_id } = contributor;
      console.log(`Adding contributor ${contributor_id} to downstream ${downstream_id}`);
      try {
        await pool.query(
          `INSERT INTO user_watch (user_id, watch_id) VALUES ($1, $2)`,
          [contributor_id, downstream_id]
        );
      } catch (error) {
        console.log("duplicate key found. Skipping entry...");
      }
    }

  }
  console.log("Finished!");
  process.exit(0);

}
