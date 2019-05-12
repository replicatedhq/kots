import * as jaeger from "jaeger-client";
import * as _ from "lodash";
import * as randomstring from "randomstring";
import { ImageWatchItem } from "../generated/types";
import * as pg from "pg";

export class ImageWatchStore {
  constructor(private readonly pool: pg.Pool) {}

  async createBatch(ctx: jaeger.SpanContext, userId: string, unparsedInput: string): Promise<string> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = "insert into image_watch_batch (id, user_id, images_input, created_at) values ($1, $2, $3, $4)";
    const v = [id, userId, unparsedInput, new Date()];

    await this.pool.query(q, v);

    await Promise.all(
      _.split(unparsedInput, "\n").map(async (line: string) => {
        const imagesAndPullable = line.split(",");
        if (imagesAndPullable.length === 1) {
          await this.createImageWatch(null, id, imagesAndPullable[0]);
          return;
        }

        const images = imagesAndPullable[0].split(" ");
        const pullables = imagesAndPullable[1].split(" ");
        for (const i of Object.keys(images)) {
          await this.createImageWatch(null, id, images[i], pullables[i]);
        }
      }),
    );
    return id;
  }

  async createImageWatch(ctx: jaeger.SpanContext, batchId: string, imageName: string, dockerPullable?: string): Promise<ImageWatchItem> {
  const id = randomstring.generate({ capitalization: "lowercase" });

    const q = "insert into image_watch (id, batch_id, image_name, docker_pullable) values ($1, $2, $3, $4)";
    const v = [id, batchId, imageName, dockerPullable];

    await this.pool.query(q, v);

    return this.getImageWatchItem(null, id);
  }

  async getImageWatchItem(ctx: jaeger.SpanContext, id: string): Promise<ImageWatchItem> {
    const q = `select id, image_name, checked_at, is_private, versions_behind,
              detected_version, latest_version, compatible_version, path from image_watch where id = $1`;
    const v = [id];

    const { rows }: { rows: any[] } = await this.pool.query(q, v);

    return this.mapImageWatch(rows[0]);
  }

  async listImageWatchItemsInBatch(batchId: string): Promise<ImageWatchItem[]> {
    const q = `select id, image_name, checked_at, is_private, versions_behind,
              detected_version, latest_version, compatible_version, path from image_watch where batch_id = $1`;
    const v = [batchId];

    const result = await this.pool.query(q, v);
    const imageWatchItems: ImageWatchItem[] = [];
    for (const row of result.rows) {
      const result = this.mapImageWatch(row);
      imageWatchItems.push(result);
    }
    return imageWatchItems;
  }

  private mapImageWatch(row: any): ImageWatchItem {
    return {
      id: row.id,
      name: row.image_name,
      lastCheckedOn: row.checked_at,
      isPrivate: row.is_private,
      versionDetected: row.detected_version,
      latestVersion: row.latest_version,
      compatibleVersion: row.compatible_version,
      versionsBehind: row.versions_behind,
      path: row.path,
    };
  }
}
