"""Create digest-bound SPDX documents for a verified legacy KOTS OCI overlay."""

import argparse
from datetime import datetime, timezone
import gzip
import hashlib
import io
import json
from pathlib import Path
import re
import subprocess
import tarfile
import tempfile
from urllib.parse import quote


TOOL = "Tool: Replicated KOTS compatibility rebuild"
OCI_LAYER = "application/vnd.oci.image.layer.v1.tar+gzip"


def digest(data):
    return "sha256:" + hashlib.sha256(data).hexdigest()


def hex_digest(value):
    if not isinstance(value, str) or not re.fullmatch(r"sha256:[a-f0-9]{64}", value):
        raise ValueError(f"Invalid SHA-256 digest {value!r}")
    return value[7:]


def must(condition, message):
    if not condition:
        raise ValueError(message)


def single(items, description):
    items = list(items)
    must(len(items) == 1, f"Expected one {description}, got {len(items)}")
    return items[0]


def verified_blob(layout, value, size=None):
    data = (layout / "blobs" / "sha256" / hex_digest(value)).read_bytes()
    must(digest(data) == value, f"OCI blob does not match {value}")
    must(size is None or len(data) == size, f"Wrong size for OCI blob {value}")
    return data


def verified_descriptor(layout, descriptor):
    return verified_blob(layout, descriptor["digest"], descriptor["size"])


def is_linux_architecture(descriptor, architecture):
    platform = descriptor.get("platform", {})
    return platform.get("os") == "linux" and platform.get("architecture") == architecture


def go_build_info(data):
    with tempfile.NamedTemporaryFile() as path:
        path.write(data)
        path.flush()
        metadata = subprocess.check_output(["go", "version", "-m", path.name], text=True)
    version, *settings = metadata.splitlines()
    return {"toolchain": version.split(": ", 1)[1], "settings": settings}


def verify_overlay(layout, receipt):
    candidate = json.loads(verified_blob(layout, receipt["candidate_index"]))
    amd64 = single((d for d in candidate["manifests"] if is_linux_architecture(d, "amd64")), "linux/amd64 image")
    arm64 = single((d for d in candidate["manifests"] if is_linux_architecture(d, "arm64")), "linux/arm64 image")
    must(amd64["digest"] == receipt["candidate_amd64"], "Candidate index points at the wrong amd64 digest")
    must([d for d in candidate["manifests"] if d is not amd64] == receipt["unchanged_platforms"],
         "Candidate changed other platforms")
    manifest = json.loads(verified_descriptor(layout, amd64))
    config = json.loads(verified_descriptor(layout, manifest["config"]))
    overlay = manifest["layers"][-1]
    must(overlay["digest"] == receipt["overlay_layer"] and overlay["mediaType"] == OCI_LAYER,
         "Candidate ended with an unexpected layer")
    layer = gzip.decompress(verified_descriptor(layout, overlay))
    must(config["rootfs"]["diff_ids"][-1] == digest(layer) == receipt["overlay_diff_id"],
         "Candidate overlay diff ID does not match")
    with tarfile.open(fileobj=io.BytesIO(layer), mode="r:") as archive:
        member = single(archive.getmembers(), "overlay file")
        must(member.isfile() and member.mode & 0o111, "Overlay file is not executable")
        path = "/" + member.name.lstrip("/")
        must(path in ("/usr/bin/kustomize", "/usr/local/bin/kustomize"), "Overlay must replace Kustomize")
        must(path == receipt["overlay_path"], "Unexpected overlay path")
        executable = archive.extractfile(member).read()
    must(digest(executable) == receipt["source"]["binary_sha256"], "Overlay executable digest does not match")
    info = go_build_info(executable)
    must(info == receipt["replacement_go"], "Overlay Go build metadata does not match")
    must("\tbuild\tGOAMD64=v1" in info["settings"], "Overlay file does not target baseline amd64")
    must("\tbuild\tGOAMD64=v2" in receipt["original_go"]["settings"], "Original Go target was not v2")
    return arm64["digest"], overlay["digest"]


def original_file_hashes(layout, manifest, paths):
    actual = {}
    for layer in manifest["layers"]:
        with tarfile.open(fileobj=io.BytesIO(verified_descriptor(layout, layer)), mode="r:*") as archive:
            for member in archive:
                path = member.name.removeprefix("./")
                if "/.wh." in "/" + path:
                    parent = path.rpartition("/")[0]
                    if any(name.startswith(parent + "/") or not parent for name in paths):
                        raise ValueError(f"Historical executable lookup needs whiteout handling for {path}")
                if path in paths:
                    must(member.isfile(), f"Expected a historical executable file at {path}")
                    with archive.extractfile(member) as file:
                        actual[path] = "sha256:" + hashlib.file_digest(file, "sha256").hexdigest()
    return actual


def compare_original_binaries(layout, receipt):
    original = json.loads(verified_blob(layout, receipt["original_amd64"]))
    expected = {path: record["sha256"] for path, record in receipt["verified_legacy_binary_sha256"].items()}
    expected["usr/bin/kustomize"] = receipt["kustomize_standalone_source"]["original_binary_sha256"]
    must(original_file_hashes(layout, original, expected) == expected,
         "Compared executable differs from the pinned historical image")


def verify_companion_overlay(layout, original_layout, receipt):
    candidate = json.loads(verified_blob(layout, receipt["candidate_index"]))
    amd64 = single((d for d in candidate["manifests"] if is_linux_architecture(d, "amd64")), "linux/amd64 image")
    arm64 = single((d for d in candidate["manifests"] if is_linux_architecture(d, "arm64")), "linux/arm64 image")
    must(amd64["digest"] == receipt["candidate_amd64"], "Candidate index points at the wrong amd64 digest")
    must([arm64["digest"]] == receipt["arm64"], "Companion arm64 digest differs from its receipt")

    original = json.loads(verified_blob(original_layout, receipt["original_amd64"]))
    manifest = json.loads(verified_descriptor(layout, amd64))
    must(manifest["layers"][:-1] == original["layers"], "Companion overlay changed historical layers")
    overlay = manifest["layers"][-1]
    must(overlay["digest"] == receipt["compressed_layer"] and overlay["mediaType"] == OCI_LAYER,
         "Candidate ended with an unexpected companion layer")
    original_config = json.loads(verified_descriptor(original_layout, original["config"]))
    config = json.loads(verified_descriptor(layout, manifest["config"]))
    layer = gzip.decompress(verified_descriptor(layout, overlay))
    must(config["config"] == original_config["config"], "Companion runtime config changed")
    must(config["rootfs"]["diff_ids"] == [*original_config["rootfs"]["diff_ids"], digest(layer)],
         "Candidate companion diff ID does not match")

    files = receipt["files"]
    must(isinstance(files, dict) and bool(files), "Companion receipt must identify the replaced files")
    expected = {path: "sha256:" + hex_digest(record["old"]) for path, record in files.items()}
    must(original_file_hashes(original_layout, original, files) == expected,
         "Compared companion file differs from the pinned historical image")
    seen = set()
    with tarfile.open(fileobj=io.BytesIO(layer), mode="r:") as archive:
        for member in archive:
            path = member.name.removeprefix("./")
            must(path in files and path not in seen and member.isfile(), f"Unexpected companion overlay file {path}")
            with archive.extractfile(member) as file:
                actual = "sha256:" + hashlib.file_digest(file, "sha256").hexdigest()
            must(actual == files[path]["new"], f"Companion overlay digest differs for {path}")
            seen.add(path)
    must(seen == set(files), "The companion overlay is missing a verified file")
    return arm64["digest"], overlay["digest"]


def verify_companion_rebase(layout, original_layout, receipt):
    candidate = json.loads(verified_blob(layout, receipt["candidate_index"]))
    amd64 = single((d for d in candidate["manifests"] if is_linux_architecture(d, "amd64")), "linux/amd64 image")
    arm64 = single((d for d in candidate["manifests"] if is_linux_architecture(d, "arm64")), "linux/arm64 image")
    must(amd64["digest"] == receipt["candidate_amd64"], "Candidate index points at the wrong amd64 digest")
    must([d for d in candidate["manifests"] if d is not amd64] == receipt["unchanged_platforms"],
         "Candidate changed other platforms")
    manifest = json.loads(verified_descriptor(layout, amd64))
    config = json.loads(verified_descriptor(layout, manifest["config"]))
    original = json.loads(verified_blob(original_layout, receipt["original_amd64"]))
    original_config = json.loads(verified_descriptor(original_layout, original["config"]))
    must(config["config"] == original_config["config"] == receipt["unchanged_runtime_config"],
         "Companion configuration differs from the original")
    layers = manifest["layers"]
    expected_layers = [receipt["new_layer"], *receipt.get("additional_layers", [])]
    must([layer["digest"] for layer in layers] == expected_layers, "Unexpected companion layers")

    binaries = receipt.get("application_binaries", receipt.get("preserved_binaries"))
    rebuilt = receipt.get("rebuilt_binaries", {})
    must(binaries and rebuilt.keys() <= binaries.keys(), "Missing companion executable records")
    expected_original = {
        path: rebuilt.get(path, {}).get("original_binary_sha256", record["sha256"])
        for path, record in binaries.items()
    }
    must(original_file_hashes(original_layout, original, binaries) == expected_original,
         "Companion executable differs from the historical image")
    resolved = {}
    diff_ids = []
    for layer in layers:
        must(layer["mediaType"] == OCI_LAYER, "Unexpected companion layer format")
        data = gzip.decompress(verified_descriptor(layout, layer))
        diff_ids.append(digest(data))
        with tarfile.open(fileobj=io.BytesIO(data), mode="r:") as archive:
            seen = set()
            for member in archive:
                path = member.name.removeprefix("./")
                must(not path.rsplit("/", 1)[-1].startswith(".wh."), "Unexpected companion whiteout")
                if path not in binaries:
                    continue
                must(path not in seen and member.isfile() and member.mode & 0o111,
                     f"Invalid companion executable {path}")
                seen.add(path)
                resolved[path] = archive.extractfile(member).read()
    must(config["rootfs"]["diff_ids"] == diff_ids, "Companion diff IDs do not match")
    must(resolved.keys() == binaries.keys(), "Companion executable missing from candidate")
    for path, executable in resolved.items():
        info = go_build_info(executable)
        expected = binaries[path]
        must(digest(executable) == expected["sha256"] and info["toolchain"] == expected["go"],
             f"Unexpected executable digest or Go version for {path}")
        must("\tbuild\tGOAMD64=v1" in info["settings"], f"{path} does not target baseline amd64")
    return arm64["digest"], layers[-1]["digest"]


def verify_rebase(layout, receipt):
    candidate = json.loads(verified_blob(layout, receipt["candidate_index"]))
    amd64 = single((d for d in candidate["manifests"] if is_linux_architecture(d, "amd64")), "linux/amd64 image")
    arm64 = single((d for d in candidate["manifests"] if is_linux_architecture(d, "arm64")), "linux/arm64 image")
    must(amd64["digest"] == receipt["candidate_amd64"], "Candidate index points at the wrong amd64 digest")
    must([d for d in candidate["manifests"] if d is not amd64] == receipt["unchanged_platforms"],
         "Candidate changed other platforms")
    manifest = json.loads(verified_descriptor(layout, amd64))
    config = json.loads(verified_descriptor(layout, manifest["config"]))
    must(config["config"] == receipt["runtime_config"], "Candidate runtime config differs from the verified receipt")
    layer = single(manifest["layers"], "rebased filesystem layer")
    must(layer["digest"] == receipt["new_layer"] and layer["mediaType"] == OCI_LAYER,
         "Candidate has an unexpected filesystem layer")
    data = gzip.decompress(verified_descriptor(layout, layer))
    must(config["rootfs"]["diff_ids"] == [digest(data)] == [receipt["new_diff_id"]],
         "Candidate filesystem diff ID does not match")
    expected = receipt["verified_legacy_binary_sha256"]
    required = {"kots", "kotsadm", "usr/bin/kustomize"}
    must(required <= expected.keys(), "Missing KOTS executable evidence")
    must(any(Path(name).name.startswith("helm") for name in expected), "Missing Helm executable evidence")
    must(any(Path(name).name.startswith("kubectl") for name in expected), "Missing kubectl executable evidence")
    seen = set()
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:") as archive:
        for member in archive:
            path = member.name.removeprefix("./")
            if path not in expected:
                continue
            must(path not in seen and member.isfile() and member.mode & 0o111, f"Invalid executable entry {path}")
            payload = archive.extractfile(member).read()
            must(digest(payload) == expected[path]["sha256"], f"Rebased executable digest changed for {path}")
            build_info = go_build_info(payload)
            must(build_info["toolchain"] == expected[path]["go"], f"Wrong Go toolchain for {path}")
            must("\tbuild\tGOAMD64=v1" in build_info["settings"], f"{path} does not target baseline amd64")
            seen.add(path)
    must(seen == set(expected), "Rebased filesystem is missing a verified executable")
    must(expected["usr/bin/kustomize"]["sha256"] == receipt["kustomize_standalone_source"]["binary_sha256"],
         "Kustomize differs from its verified source build")
    return arm64["digest"], layer["digest"]


def read_spdx(path):
    doc = json.loads(path.read_bytes())
    must(doc["spdxVersion"] == "SPDX-2.3" and doc["SPDXID"] == "SPDXRef-DOCUMENT",
         f"Unexpected SPDX format in {path}")
    return doc


def image_package(doc, subject_digest):
    subject = hex_digest(subject_digest)
    result = single(
        (p for p in doc["packages"]
         if any(c["algorithm"] == "SHA256" and c["checksumValue"] == subject for c in p.get("checksums", []))),
        f"SPDX image package for {subject_digest}",
    )
    must(any(r["referenceType"] == "purl" and r["referenceLocator"].startswith("pkg:oci/")
             and subject in r["referenceLocator"] for r in result.get("externalRefs", [])),
         f"SPDX image package {subject_digest} has no matching OCI purl")
    return result


def verify_subject(doc, value):
    subject = image_package(doc, value)
    must(doc["documentDescribes"] == [subject["SPDXID"]], f"SPDX document describes the wrong image for {value}")
    return subject["SPDXID"]


def rewrite_digests(value, replacements):
    if isinstance(value, str):
        for old, new in replacements.items():
            value = value.replace(hex_digest(old), hex_digest(new))
        return value
    if isinstance(value, list):
        return [rewrite_digests(element, replacements) for element in value]
    if isinstance(value, dict):
        return {key: rewrite_digests(element, replacements) for key, element in value.items()}
    return value


def revise_document(doc, subject, created, component="kotsadm"):
    doc["name"] = f"sbom-{subject}"
    doc["documentNamespace"] = f"https://github.com/replicatedhq/kots/spdxdocs/{component}-{hex_digest(subject)}"
    doc["creationInfo"]["created"] = created
    doc["creationInfo"]["creators"].append(TOOL)


def bind_overlay(doc, receipt, overlay_digest, created):
    source = receipt["source"]
    toolchain = receipt["replacement_go"]["toolchain"]
    binary = source["binary_sha256"]
    package = single(
        (p for p in doc["packages"]
         if re.fullmatch(r"kustomize(?:-[0-9]+\.[0-9]+)?", p["name"]) and p.get("versionInfo") == source["package_version"]),
        "SPDX Kustomize package",
    )
    layer_id = f"SPDXRef-Package-sha256-{hex_digest(overlay_digest)}"
    must(all(p["SPDXID"] != layer_id for p in doc["packages"]), "Original SPDX already includes the candidate overlay")
    subject_id = image_package(doc, receipt["candidate_amd64"])["SPDXID"]
    provenance = f" It uses upstream source {source.get('source_revision') or source.get('source_tag') or source.get('revision')}."
    if source.get("source_sha256"):
        provenance += f" The source archive digest is {source['source_sha256']}."
    if source.get("recipe_revision"):
        provenance += f" The original package uses recipe revision {source['recipe_revision']}."
    comment = (f"This layer supplies {receipt['overlay_path']} with digest {binary}. "
               f"The executable was rebuilt with {toolchain} and GOAMD64=v1." + provenance)
    doc["packages"].append({
        "SPDXID": layer_id,
        "name": overlay_digest,
        "versionInfo": overlay_digest,
        "downloadLocation": "NOASSERTION",
        "filesAnalyzed": False,
        "checksums": [{"algorithm": "SHA256", "checksumValue": hex_digest(overlay_digest)}],
        "externalRefs": [{
            "referenceCategory": "PACKAGE-MANAGER",
            "referenceType": "purl",
            "referenceLocator": f"pkg:oci/kotsadm@{quote(overlay_digest, safe='')}?arch=amd64&mediaType={quote(OCI_LAYER,safe='')}&os=linux",
        }],
        "annotations": [{
            "annotator": TOOL,
            "annotationDate": created,
            "annotationType": "OTHER",
            "comment": comment,
        }],
    })
    doc["relationships"].extend([
        {"spdxElementId": subject_id, "relationshipType": "CONTAINS", "relatedSpdxElement": layer_id},
        {"spdxElementId": layer_id, "relationshipType": "OTHER", "relatedSpdxElement": package["SPDXID"],
         "comment": "The layer supplies a baseline x86-64 rebuild of the executable from this package."},
    ])


def generate(args):
    receipt = json.loads(args.overlay_receipt.read_bytes())
    old_index, new_index = receipt["original_index"], receipt["candidate_index"]
    old_amd64, new_amd64 = receipt["original_amd64"], receipt["candidate_amd64"]
    must(old_index != new_index and old_amd64 != new_amd64, "Expected distinct original and candidate OCI digests")
    is_rebase = "baseline_userland_index" in receipt
    if args.companion:
        must(args.original_oci_layout, "A companion overlay needs the pinned original amd64 OCI layout")
        must(not args.original_amd64_spdx, "A rebuilt companion needs a fresh amd64 SBOM")
        verify_companion = verify_companion_rebase if is_rebase else verify_companion_overlay
        unchanged_arm64, overlay_digest = verify_companion(
            args.candidate_oci_layout, args.original_oci_layout, receipt)
    elif is_rebase:
        must(not args.original_amd64_spdx, "A rebased filesystem needs a fresh amd64 SBOM")
        must(args.original_oci_layout, "A rebased filesystem needs the pinned original amd64 OCI layout")
        compare_original_binaries(args.original_oci_layout, receipt)
        unchanged_arm64, overlay_digest = verify_rebase(args.candidate_oci_layout, receipt)
    else:
        unchanged_arm64, overlay_digest = verify_overlay(args.candidate_oci_layout, receipt)

    created = datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")
    replacements = {old_index: new_index, old_amd64: new_amd64}
    documents = {}
    if args.original_index_spdx:
        index = read_spdx(args.original_index_spdx)
        original_index_id = verify_subject(index, old_index)
        for child in (old_amd64, unchanged_arm64):
            child_id = image_package(index, child)["SPDXID"]
            must(any({r["spdxElementId"], r["relatedSpdxElement"]} == {original_index_id, child_id}
                     for r in index["relationships"]), f"Original index SPDX has no relationship to {child}")
        candidate = rewrite_digests(index, replacements)
        verify_subject(candidate, new_index)
        must(image_package(candidate, unchanged_arm64) == image_package(index, unchanged_arm64),
             "arm64 inventory changed")
        revise_document(candidate, new_index, created, args.companion or "kotsadm")
        documents["index"] = candidate

    if args.original_amd64_spdx:
        amd64 = read_spdx(args.original_amd64_spdx)
        verify_subject(amd64, old_amd64)
        candidate = rewrite_digests(amd64, replacements)
        verify_subject(candidate, new_amd64)
        revise_document(candidate, new_amd64, created)
        bind_overlay(candidate, receipt, overlay_digest, created)
        documents["amd64"] = candidate

    args.output.mkdir(parents=True, exist_ok=True)
    for arch, doc in documents.items():
        filename = args.output / f"{arch}.spdx.json"
        with filename.open("x") as output:
            json.dump(doc, output, indent=2)
            output.write("\n")
        print(f"{digest(filename.read_bytes())}  {filename}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--original-index-spdx", type=Path)
    parser.add_argument("--original-amd64-spdx", type=Path)
    parser.add_argument("--original-oci-layout", type=Path)
    parser.add_argument(
        "--companion", choices=("dex", "rqlite", "minio", "local-volume-provider", "kotsadm-migrations"))
    parser.add_argument("overlay_receipt", type=Path)
    parser.add_argument("candidate_oci_layout", type=Path)
    parser.add_argument("output", type=Path)
    generate(parser.parse_args())
