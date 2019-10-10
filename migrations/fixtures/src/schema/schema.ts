import * as yaml from "js-yaml";
import * as fs from "fs";
import * as escape from "pg-escape";

export class Schema {
  private parsedDoc: any;

  public parse(filename: string) {
    this.parsedDoc = yaml.safeLoad(fs.readFileSync(filename, "utf-8"));
  }

  public generateFixtures(): string[] {
    let statements: string[] = [];
    console.log("Generating users...");
    if (this.parsedDoc.users) {
      for (const user of this.parsedDoc.users) {
        statements = statements.concat(this.generateUserFixture(user));
      }
    }

    console.log("Generating clusters...");
    if (this.parsedDoc.clusters) {
      for (const cluster of this.parsedDoc.clusters) {
        statements = statements.concat(this.generateClusterFixture(cluster));
      }
    }

    console.log("Generating watches...");
    if (this.parsedDoc.watches) {
      for (const watch of this.parsedDoc.watches) {
        statements = statements.concat(this.generateWatchFixture(watch));
      }
    }

    if (this.parsedDoc.imageBatches) {
      console.log("Generating image batches...");
      for (const imageBatch of this.parsedDoc.imageBatches) {
        statements = statements.concat(this.generateImageWatchFixture(imageBatch));
      }
    }

    if (this.parsedDoc.apps) {
      console.log("Generating Kots Apps...");
      for (const app of this.parsedDoc.apps) {
        statements = statements.concat(this.generateKotsAppFixture(app))
      }
    }

    return statements;
  }

  public generateUserFixture(user: any): string[] {
    const statements: string[] = [
      escape(`insert into ship_user (id, created_at) values (%L, %L)`, user.id, user.created_at),
    ];

    if (user.type === "github") {
      statements.push(escape(
        `insert into github_user (user_id, github_id, username, avatar_url, email) values (%L, ${user.github.github_id}, %L, %L, %L)`,
        user.id, user.github.username, user.github.avatar_url, user.github.email)
      );
    }

    const metadata = {};
    metadata[user.username] = user.github_id;

    for (const session of user.sessions) {
      statements.push(
        escape(`insert into session (id, user_id, metadata, expire_at) values (%L, %L, %L, %L)`, session, user.id, JSON.stringify(metadata), new Date().toISOString())
      );
    }

    return statements;
  }

  public generateClusterFixture(cluster: any): string[] {
    const statements: string[] = [
      escape(`insert into cluster (id, title, slug, created_at, updated_at, token, cluster_type, is_all_users) values (%L, %L, %L, %L, %L, %L, %L, ${false})`,
        cluster.id, cluster.title, cluster.slug, cluster.created_at, cluster.updated_at, cluster.token, cluster.cluster_type),
    ];

    if (cluster.github) {
      statements.push(
        escape(`insert into cluster_github (cluster_id, owner, repo, branch, installation_id) values (%L, %L, %L, %L, %L::integer)`,
          cluster.id, cluster.github.owner, cluster.github.repo, cluster.github.branch, ''+cluster.github.installation_id),
      );
    }

    for (const user of cluster.users) {
      statements.push(
        escape(`insert into user_cluster (user_id, cluster_id) values (%L, %L)`, user, cluster.id)
      );
    }

    return statements;
  }

  public generateWatchFixture(watch: any): string[] {
    const statements: string[] = [];

    const currentSequenceEscapeSequence = watch.current_sequence === null ? "%L" : "%L::integer";
    const currentSequenceValue = watch.current_sequence === null ? "NULL" : ''+watch.current_sequence;

    statements.push(
      escape(`insert into watch (id, current_state, title, icon_uri, created_at, updated_at, slug, parent_watch_id, current_sequence, metadata) values (%L, %L, %L, %L, %L, %L, %L, %L, ${currentSequenceEscapeSequence}, %L)`,
        watch.id, watch.current_state, watch.title, watch.icon_uri, watch.created_at, watch.updated_at, watch.slug, watch.parent_watch_id, currentSequenceValue, watch.metadata)
    );

    if (watch.cluster) {
      statements.push(
        escape(`insert into watch_cluster (watch_id, cluster_id) values (%L, %L)`, watch.id, watch.cluster),
      );
    }
    for (const user of watch.users) {
      statements.push(
        escape(`insert into user_watch (user_id, watch_id) values (%L, %L)`, user, watch.id)
      );
    }

    if (watch.downstream_tokens) {
      for (const downstreamToken of watch.downstream_tokens) {
        statements.push(
          escape(`insert into watch_downstream_token (watch_id, token) values (%L, %L)`, watch.id, downstreamToken)
        );
      }
    }

    if (watch.versions) {
      for (const version of watch.versions) {
        const pullRequestNumberEscapeSequence = version.pullrequest_number === null ? "%L" : "%L::integer";
        const pullRequestNumberValue = version.pullrequest_number === null ? null : ''+version.pullrequest_number;

        statements.push(
          escape(`insert into watch_version (watch_id, created_at, version_label, status, source_branch, sequence, pullrequest_number) values (%L, %L, %L, %L, %L, %L::integer, ${pullRequestNumberEscapeSequence})`,
            watch.id, version.created_at, version.version_label, version.status, version.source_branch, ''+version.sequence, pullRequestNumberValue),
          escape(`insert into ship_output_files (watch_id, created_at, sequence, filepath) values (%L, %L, %L, %L)`,
            watch.id, version.created_at, ''+version.sequence, `${watch.id}/${version.sequence}.tar.gz`),
          escape(`insert into object_store (filepath, encoded_block) values (%L, %L)`,
            `${watch.id}/${version.sequence}.tar.gz`, version.output),
        );
      }
    }

    return statements;
  }

  public generateImageWatchFixture(imageBatch: any): string[] {
    const statements: string[] = [];

    statements.push(
      escape(`insert into image_watch_batch (id, user_id, images_input, created_at) values (%L, %L, %L, %L)`, imageBatch.id, imageBatch.user_id, imageBatch.images_input, imageBatch.created_at)
    );

    if (imageBatch.batch_watches) {
      for (const batchWatch of imageBatch.batch_watches) {
        statements.push(
          escape(`insert into image_watch (id, batch_id, image_name, checked_at, is_private, versions_behind, detected_version, latest_version, compatible_version, check_error, docker_pullable, path, started_processing_at) values (%L, %L, %L, %L, ${batchWatch.is_private}, ${batchWatch.versions_behind}, %L, %L, %L, %L, %L, %L, %L)`,
          batchWatch.id, batchWatch.batch_id, batchWatch.image_name, batchWatch.checked_at, batchWatch.detected_version, batchWatch.latest_version, batchWatch.compatible_version, batchWatch.check_error, batchWatch.docker_pullable, batchWatch.path, batchWatch.started_processing_at)
        )
      }
    }

    return statements;
  }

  public generateKotsAppFixture(app: any): string[] {
    const statements: string[] = [];

    const yamlLicense = yaml.safeDump(app.license);
    statements.push(
      escape(`INSERT INTO app (
        id,
        name,
        icon_uri,
        created_at,
        updated_at,
        slug,
        current_sequence,
        last_update_check_at,
        is_all_users,
        upstream_uri,
        license,
        registry_hostname,
        registry_username,
        registry_password,
        namespace,
        last_registry_sync,
        install_state
        ) VALUES (
          %L, %L, %L, %L, %L, %L, ${app.current_sequence}, %L, %L, %L, %L, %L, %L, %L, %L, %L, %L)`,
        app.id,
        app.name,
        app.icon_uri,
        app.created_at,
        app.updated_at,
        app.slug,
        app.last_update_check_at,
        app.is_all_users === "true" ? "true" : "false",
        app.upstream_uri,
        yamlLicense,
        app.registry_hostname,
        app.registry_username,
        app.registry_password,
        app.namespace,
        app.last_registry_sync,
        app.install_state
      )
    );
    if (app.users) {
      for (const user of app.users) {
        statements.push(
          escape(`INSERT INTO user_app (user_id, app_id) VALUES (%L, %L)`,
            user, app.id
          )
        );
      }
    }

    if (app.downstreams) {
      for (const downstream of app.downstreams) {
        // NOTE: id is used for both id and name
        statements.push(
          escape(`INSERT INTO app_downstream (app_id, cluster_id, downstream_name, current_sequence) VALUES (%L, %L, %L, ${downstream.sequence || app.current_sequence})`,
            app.id, downstream.id, downstream.id
          )
        );
      }
    }

    if (app.downstream_versions) {
      for (const downstreamVersion of app.downstream_versions) {
        statements.push(
          escape(`INSERT INTO app_downstream_version (
            app_id,
            cluster_id,
            sequence,
            parent_sequence,
            created_at,
            applied_at,
            version_label,
            status,
            preflight_result,
            preflight_result_created_at
          ) VALUES (
            %L, %L, ${downstreamVersion.sequence}, ${app.current_sequence}, %L, %L, %L, %L, %L, %L
          )`,
          app.id,
          downstreamVersion.cluster_id,
          downstreamVersion.created_at,
          downstreamVersion.applied_at,
          downstreamVersion.version_label,
          downstreamVersion.status,
          downstreamVersion.preflight_result,
          downstreamVersion.preflight_result_created_at
          )
        )
      }
    }
    if (app.versions) {
      for (const version of app.versions) {
        let appSpec = null;
        if (version.kots_app_spec) {
          appSpec = yaml.safeDump(version.kots_app_spec);
        }

        let preflightSpec = null;
        if (version.preflight_spec) {
          preflightSpec = yaml.safeDump(version.preflight_spec);
        }

        let supportbundleSpec = null;
        if (version.supportbundle_spec) {
          supportbundleSpec = yaml.safeDump(version.supportbundle_spec);
        }

        statements.push(
          escape(`INSERT INTO app_version (
            app_id,
            sequence,
            update_cursor,
            created_at,
            version_label,
            supportbundle_spec,
            preflight_spec,
            release_notes,
            kots_app_spec
          ) VALUES (
            %L, ${version.sequence}, ${version.update_cursor}, %L, %L, %L, %L, %L, %L
            )`,
            app.id,
            version.created_at,
            version.version_label,
            supportbundleSpec,
            preflightSpec,
            version.release_notes,
            appSpec
          )
        )
      }
    }
    return statements;
  }
}
