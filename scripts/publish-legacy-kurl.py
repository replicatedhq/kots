"""Stage, promote, finalize, or roll back one verified historical kURL add-on."""

from argparse import ArgumentParser
from base64 import b64encode
from copy import deepcopy
from hashlib import file_digest, sha256
from pathlib import Path
from tempfile import TemporaryDirectory
from urllib.request import Request, urlopen
import json
import os
import re
import subprocess
import time


ORIGIN = "kots-kurl-addons-production-1658439274"
MIRROR = "kurl-sh"
VERSIONS = "versions.json"
REGISTRY = "external/addon-registry.json"
MEDIA = "application/x-tar"
STAGE_PREFIX = "test/"


def public_url(bucket, key):
    return f"https://{bucket}.s3.amazonaws.com/{key}"


def public_open(url, method="GET"):
    return urlopen(Request(url, method=method, headers={"Cache-Control": "no-cache"}), timeout=45)


def sha_file(path):
    with path.open("rb") as reader:
        return file_digest(reader, "sha256").hexdigest()


def checked_file(path, wanted):
    actual = {"sha256": sha_file(path), "size": path.stat().st_size}
    if actual != {"sha256": wanted["sha256"], "size": wanted["size"]}:
        raise ValueError(f"SHA-256 or size mismatch for {path}")


def object_head(url):
    with public_open(url, "HEAD") as reply:
        result = {
            "etag": reply.headers["ETag"],
            "version_id": reply.headers.get("x-amz-version-id"),
            "size": int(reply.headers["Content-Length"]),
            "metadata": {},
        }
        for header in ("Content-Type", "Cache-Control", "Content-Encoding",
                       "Content-Disposition", "Content-Language", "Expires"):
            value = reply.headers.get(header)
            if value:
                result["metadata"][header.lower()] = value
        return result


def verify_public_bytes(url, wanted, dest=None):
    with public_open(url) as stream:
        actual_size = int(stream.headers["Content-Length"])
        if dest is None:
            actual_sha = file_digest(stream, "sha256").hexdigest()
        else:
            digest = sha256()
            with dest.open("xb") as output:
                for block in iter(lambda: stream.read(4 * 1024 * 1024), b""):
                    digest.update(block)
                    output.write(block)
            actual_sha = digest.hexdigest()
    if actual_sha != wanted["sha256"] or actual_size != wanted["size"]:
        raise ValueError(f"Published bytes do not match plan for {url}")


def live_json(bucket, key):
    with public_open(public_url(bucket, key)) as response:
        return json.load(response), response.headers["ETag"]


def selected(rows, version):
    matches = [row for row in rows if row["version"] == version]
    if len(matches) != 1:
        raise ValueError(f"Expected one existing version entry for {version}")
    return matches[0]


def marker(plan, version):
    tail = re.fullmatch(r"1\.(\d+)\.(\d+)", version)
    if not tail or not (1, 108, 4) <= tuple(map(int, version.split("."))) <= (1, 134, 4):
        raise ValueError("Version is outside the historical backfill")
    if plan["version"] != version:
        raise ValueError("Candidate receipt version does not match")
    return f"kotsadm-{version}.tar.gz"


def create_plan(args):
    receipt = json.loads(args.receipt.read_text())
    version = receipt["version"]
    filename = marker(receipt, version)
    candidate = {"sha256": receipt["candidate_addon_sha256"], "size": receipt["candidate_addon_size"]}
    checked_file(args.candidate, candidate)
    log = args.test_log.read_text()
    expected_pass = f"KOTS {version} passed full offline kURL kubeadm installation and live-pod kustomize render on baseline x86-64."
    expected_seal = f"Verified candidate SHA-256 {candidate['sha256']} for KOTS {version}."
    if expected_pass not in log or expected_seal not in log:
        raise ValueError(f"Expected completed successful offline test for {version}")
    mirror_key = "external/" + filename
    mirror_head = object_head(public_url(MIRROR, mirror_key))
    if not mirror_head["etag"] or not mirror_head["version_id"] or mirror_head["version_id"] == "null":
        raise ValueError("Expected exact original mirror ETag and S3 VersionId")
    original = {**mirror_head, "sha256": receipt["original_addon_sha256"]}
    checked_file(args.original, original)

    mirror, registry_etag = live_json(MIRROR, REGISTRY)
    source, versions_etag = live_json(ORIGIN, VERSIONS)
    old_mirror_row = selected(mirror["kotsadm"], version)
    old_origin_row = selected(source, version)
    if old_mirror_row["sha256Sum"] != original["sha256"] or old_mirror_row["origin"] != old_origin_row["url"]:
        raise ValueError("Live kURL metadata disagrees with the verified original")
    verify_public_bytes(public_url(MIRROR, mirror_key), original)

    sha = candidate["sha256"]
    origin_key = f"{STAGE_PREFIX}kotsadm-{version}-baseline-{sha}.tar.gz"
    origin_url = public_url(ORIGIN, origin_key)
    plan = {
        "version": version,
        "mirror_key": mirror_key,
        "origin_key": origin_key,
        "origin_url": origin_url,
        "original": original,
        "candidate": candidate,
        "test_log_sha256": sha_file(args.test_log),
        "candidate_build_receipt_sha256": sha_file(args.receipt),
        "expected_live_registry_etag_at_plan": registry_etag,
        "expected_live_versions_etag_at_plan": versions_etag,
        "old_mirror_row": old_mirror_row,
        "old_origin_row": old_origin_row,
        "new_mirror_row": {**old_mirror_row, "origin": origin_url, "sha256Sum": sha},
        "new_origin_row": {**old_origin_row, "url": origin_url},
    }
    return plan


def validate_plan(plan):
    filename = marker(plan, plan["version"])
    sha = plan["candidate"]["sha256"]
    if not all(re.fullmatch("[a-f0-9]{64}", data["sha256"]) and 0 < data["size"] < 5 * 1024**3
               for data in (plan["candidate"], plan["original"])):
        raise ValueError("Invalid package seal")
    origin_key = f"{STAGE_PREFIX}kotsadm-{plan['version']}-baseline-{sha}.tar.gz"
    origin_url = public_url(ORIGIN, origin_key)
    if plan["mirror_key"] != "external/" + filename or plan["origin_key"] != origin_key or plan["origin_url"] != origin_url:
        raise ValueError("Plan targets an unexpected object")
    if (plan["old_mirror_row"]["version"] != plan["version"]
            or plan["old_origin_row"]["version"] != plan["version"]
            or plan["new_mirror_row"] != {**plan["old_mirror_row"], "origin": origin_url, "sha256Sum": sha}
            or plan["new_origin_row"] != {**plan["old_origin_row"], "url": origin_url}
            or plan["old_mirror_row"]["sha256Sum"] != plan["original"]["sha256"]
            or plan["old_mirror_row"]["origin"] != plan["old_origin_row"]["url"]):
        raise ValueError("Plan contains an unrelated registry row edit")


def put(bucket, key, file, metadata, *, if_match=None, if_none_match=None):
    flags = ["aws", "s3api", "put-object", "--no-cli-pager", "--output", "json",
             "--bucket", bucket, "--key", key, "--body", str(file), "--checksum-algorithm", "SHA256",
             "--checksum-sha256", b64encode(bytes.fromhex(sha_file(file))).decode()]
    if if_match:
        flags.extend(["--if-match", if_match])
    if if_none_match:
        flags.extend(["--if-none-match", if_none_match])
    for name, value in metadata.items():
        flags.extend(["--" + name, value])
    result = subprocess.run(flags, capture_output=True, text=True, check=False)
    if result.returncode:
        errors = result.stderr.strip()
        if "PreconditionFailed" in errors or "ConditionalRequestConflict" in errors:
            return None
        raise RuntimeError(f"Guarded S3 upload failed: {errors}")
    return json.loads(result.stdout)


def update_row(bucket, key, section, version, old_row, new_row):
    for attempt in range(5):
        document, etag = live_json(bucket, key)
        rows = document[section] if section else document
        live = selected(rows, version)
        if live == new_row:
            return
        if live != old_row:
            raise ValueError(f"Refusing to replace changed {key} row for {version}")
        updated = deepcopy(document)
        editable = updated[section] if section else updated
        editable[editable.index(live)] = new_row
        with TemporaryDirectory() as tmp:
            body = Path(tmp) / "updated.json"
            body.write_text(json.dumps(updated, separators=(",", ":"), ensure_ascii=False) + "\n")
            result = put(bucket, key, body, {"content-type": "application/json"}, if_match=etag)
        if result is not None:
            check, _ = live_json(bucket, key)
            if selected(check[section] if section else check, version) != new_row:
                raise ValueError(f"Verification failed after updating {key}")
            return
        time.sleep(2 + attempt)
    raise RuntimeError(f"{key} kept changing. Retry after checking the active writers.")


def stage(plan, candidate):
    checked_file(candidate, plan["candidate"])
    result = put(ORIGIN, plan["origin_key"], candidate, {"content-type": MEDIA}, if_none_match="*")
    verify_public_bytes(plan["origin_url"], plan["candidate"])
    print(json.dumps({"origin_url": plan["origin_url"], "s3": result, "verified_sha256": plan["candidate"]["sha256"]}))


def expect_mirror(plan, wanted):
    url = public_url(MIRROR, plan["mirror_key"])
    verify_public_bytes(url, wanted)
    return object_head(url)


def cdn_url(plan):
    return "https://s3.kurl.sh/" + plan["mirror_key"]


def cloudflare_purger(plan):
    zone = os.environ.get("CF_KURL_ZONE", "")
    email = os.environ.get("CF_EMAIL")
    token = os.environ.get("CF_API_KEY")
    if not (re.fullmatch(r"[a-f0-9]{32}", zone) and email and token):
        raise ValueError("The kURL Cloudflare purge credentials are unavailable")
    request = Request(
        f"https://api.cloudflare.com/client/v4/zones/{zone}/purge_cache",
        method="POST",
        headers={"X-Auth-Email": email, "Authorization": f"Bearer {token}", "Content-Type": "application/json"},
        data=json.dumps({"files": [cdn_url(plan)]}).encode(),
    )

    def purge():
        with urlopen(request, timeout=30) as response:
            success = json.load(response).get("success")
        if success is not True:
            raise RuntimeError("Cloudflare did not accept the targeted kURL archive purge")

    return purge


def cdn_verified(plan, wanted):
    direct = object_head(public_url(MIRROR, plan["mirror_key"]))
    cdn = cdn_url(plan)
    for attempt in range(6):
        lead = object_head(cdn)
        if lead["etag"] == direct["etag"]:
            verify_public_bytes(cdn, wanted)
            return
        time.sleep(3 + attempt)
    raise RuntimeError("Origin has expected bytes, but s3.kurl.sh still serves an old ETag. Check the CDN before finalizing.")


def promote_mirror(plan):
    purge = cloudflare_purger(plan)
    old, new = plan["original"], plan["candidate"]
    live = object_head(public_url(MIRROR, plan["mirror_key"]))
    mirror_index, _ = live_json(MIRROR, REGISTRY)
    live_row = selected(mirror_index["kotsadm"], plan["version"])
    if live_row not in (plan["old_mirror_row"], plan["new_mirror_row"]):
        raise ValueError("Unexpected mirror registry row")
    with TemporaryDirectory() as tmp:
        candidate = Path(tmp) / "candidate.tar.gz"
        verify_public_bytes(plan["origin_url"], new, candidate)
        if (live["etag"], live["version_id"]) == (old["etag"], old["version_id"]):
            if live_row != plan["old_mirror_row"]:
                raise ValueError("Publish payload and registry are inconsistent")
            result = put(MIRROR, plan["mirror_key"], candidate, old["metadata"], if_match=old["etag"])
            if result is None:
                raise RuntimeError("The mirror changed while uploading. Recheck live bytes and safely retry.")
        else:
            # A retry must already be serving the exact candidate, including a complete GitHub-tested payload.
            expect_mirror(plan, new)
    current = expect_mirror(plan, new)
    if current["version_id"] in (None, "null", old["version_id"]):
        raise ValueError("Expected a different S3 VersionId after replacing the package")
    purge()
    cdn_verified(plan, new)
    update_row(MIRROR, REGISTRY, "kotsadm", plan["version"], plan["old_mirror_row"], plan["new_mirror_row"])
    print(json.dumps({"mirror_key": plan["mirror_key"], "mirror_s3": current, "verified_sha256": new["sha256"]}))


def finalize(plan):
    verify_public_bytes(plan["origin_url"], plan["candidate"])
    expect_mirror(plan, plan["candidate"])
    mirror_index, _ = live_json(MIRROR, REGISTRY)
    if selected(mirror_index["kotsadm"], plan["version"]) != plan["new_mirror_row"]:
        raise ValueError("Mirror registry is not yet promoted")
    cdn_verified(plan, plan["candidate"])
    update_row(ORIGIN, VERSIONS, None, plan["version"], plan["old_origin_row"], plan["new_origin_row"])
    print("Verified mirror and CDN and finalized the KOTS origin versions row.")


def restore_mirror(plan, original):
    purge = cloudflare_purger(plan)
    old, new = plan["original"], plan["candidate"]
    checked_file(original, old)
    url = public_url(MIRROR, plan["mirror_key"])
    mirror_data, _ = live_json(MIRROR, REGISTRY)
    row = selected(mirror_data["kotsadm"], plan["version"])
    if row not in (plan["old_mirror_row"], plan["new_mirror_row"]):
        raise ValueError("Unexpected mirror registry row during rollback")
    try:
        expect_mirror(plan, old)
    except ValueError:
        current = expect_mirror(plan, new)
        result = put(MIRROR, plan["mirror_key"], original, old["metadata"], if_match=current["etag"])
        if result is None:
            raise RuntimeError("The mirror changed during rollback. Verify live bytes and retry.")
    expect_mirror(plan, old)
    purge()
    cdn_verified(plan, old)
    update_row(MIRROR, REGISTRY, "kotsadm", plan["version"], plan["new_mirror_row"], plan["old_mirror_row"])
    print("Original mirror bytes and registry entry are live.")


def restore_origin_pointer(plan):
    expect_mirror(plan, plan["original"])
    mirror_data, _ = live_json(MIRROR, REGISTRY)
    if selected(mirror_data["kotsadm"], plan["version"]) != plan["old_mirror_row"]:
        raise ValueError("The original mirror entry has not yet been restored")
    cdn_verified(plan, plan["original"])
    update_row(ORIGIN, VERSIONS, None, plan["version"], plan["new_origin_row"], plan["old_origin_row"])
    print("Original KOTS origin versions row restored.")


def main():
    parser = ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    for operation in ("plan", "check-plan", "stage", "promote-mirror", "finalize", "restore-mirror", "restore-origin-pointer"):
        command = commands.add_parser(operation)
        command.add_argument("--plan", type=Path, required=True)
        if operation in ("plan", "stage"):
            command.add_argument("--candidate", type=Path, required=True)
        if operation in ("plan", "restore-mirror"):
            command.add_argument("--original", type=Path, required=True)
        if operation == "plan":
            command.add_argument("--receipt", type=Path, required=True)
            command.add_argument("--test-log", type=Path, required=True)
    args = parser.parse_args()
    if args.command == "plan":
        plan = create_plan(args)
        with args.plan.open("x") as output:
            json.dump(plan, output, indent=2)
            output.write("\n")
        print(f"Prepared read-only publication plan for {plan['version']}.")
        return
    plan = json.loads(args.plan.read_text())
    validate_plan(plan)
    if args.command == "check-plan":
        print(f"Checked sealed kURL plan for {plan['version']}.")
    elif args.command == "stage":
        stage(plan, args.candidate)
    elif args.command == "promote-mirror":
        promote_mirror(plan)
    elif args.command == "finalize":
        finalize(plan)
    elif args.command == "restore-mirror":
        restore_mirror(plan, args.original)
    elif args.command == "restore-origin-pointer":
        restore_origin_pointer(plan)


if __name__ == "__main__":
    main()
