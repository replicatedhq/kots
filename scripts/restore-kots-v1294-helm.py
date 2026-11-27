import copy
import gzip
from hashlib import file_digest, sha256
import io
import json
from pathlib import Path
import shutil
import sys
import tarfile


ARCHIVE = {
    "amd64": "dbb4c8fc8e19d159d1a63dda8db655f9ffa4aac1b9a6b188b34a40957119b286",
    "arm64": "bfb14953295d5324d47ab55f3dfba6da28d46c848978c8fbf412d4271bdc29f1",
}
BINARY = {
    "amd64": "1f7ed083dbc200a10fdfe04df94c21530140fdc955de5a1daed2694150f77b17",
    "arm64": "2d776c1ff5653d95010e318d0db08a591cf7f183f4cc75647078de458960b2b9",
}
OLD_INDEX = "sha256:04d5faa4ad52e929f026a0d53fa9d073dd9ef2b822e37011896cdb46a02c1e11"
REF = "org.opencontainers.image.ref.name"
BINARY_PATH = "usr/bin/helm"
LINK_PATH = "usr/local/bin/helm"


def digest(data):
    return "sha256:" + sha256(data).hexdigest()


def blob(layout, blob_digest):
    return layout / "blobs" / "sha256" / blob_digest.removeprefix("sha256:")


def load(layout, descriptor):
    data = blob(layout, descriptor["digest"]).read_bytes()
    if len(data) != descriptor["size"] or digest(data) != descriptor["digest"]:
        raise ValueError(f"OCI object did not match {descriptor['digest']}")
    return json.loads(data)


def store(layout, data, descriptor):
    checksum = digest(data)
    blob(layout, checksum).write_bytes(data)
    return {**descriptor, "digest": checksum, "size": len(data)}


def encode(document):
    return json.dumps(document, ensure_ascii=False, separators=(",", ":")).encode()


def release_helm(directory, arch):
    data = (directory / f"helm-v3.20.0-linux-{arch}.tar.gz").read_bytes()
    if sha256(data).hexdigest() != ARCHIVE[arch]:
        raise ValueError(f"Upstream archive checksum failed for {arch}")
    with tarfile.open(fileobj=io.BytesIO(data)) as archive:
        executable = archive.getmember(f"linux-{arch}/helm")
        if not executable.isfile():
            raise ValueError(f"Upstream Helm is not a regular file for {arch}")
        content = archive.extractfile(executable).read()
    if sha256(content).hexdigest() != BINARY[arch]:
        raise ValueError(f"Helm differs from the original KOTS release for {arch}")
    return content


def assert_missing_helm(layout, manifest):
    parents = {"usr", "usr/bin", "usr/local", "usr/local/bin"}
    for descriptor in manifest["layers"]:
        path = blob(layout, descriptor["digest"])
        with path.open("rb") as stream:
            checksum = "sha256:" + file_digest(stream, "sha256").hexdigest()
            if path.stat().st_size != descriptor["size"] or checksum != descriptor["digest"]:
                raise ValueError(f"Original layer did not match {descriptor['digest']}")
        with tarfile.open(path, "r|*") as layer:
            for member in layer:
                name = member.name.removeprefix("./").strip("/")
                if name in (BINARY_PATH, LINK_PATH) or (name in parents and not member.isdir()):
                    raise ValueError(f"Unexpected original Helm path {name}")


def make_layer(binary):
    content = io.BytesIO()
    with tarfile.open(fileobj=content, mode="w", format=tarfile.USTAR_FORMAT) as layer:
        executable = tarfile.TarInfo(BINARY_PATH)
        executable.mode = 0o755
        executable.size = len(binary)
        layer.addfile(executable, io.BytesIO(binary))
        link = tarfile.TarInfo(LINK_PATH)
        link.type = tarfile.SYMTYPE
        link.mode = 0o777
        link.linkname = f"/{BINARY_PATH}"
        layer.addfile(link)
    raw = content.getvalue()
    compressed = io.BytesIO()
    with gzip.GzipFile(fileobj=compressed, mode="wb", filename="", compresslevel=6, mtime=0) as writer:
        writer.write(raw)
    return digest(raw), compressed.getvalue()


def restore(source, assets, destination):
    root = json.loads((source / "index.json").read_bytes())
    original, = root["manifests"]
    if original["digest"] != OLD_INDEX:
        raise ValueError("The predecessor is not the known KOTS v1.129.4 image")
    index = load(source, original)
    platforms = [item["platform"] for item in index["manifests"]]
    if sorted((p["os"], p["architecture"]) for p in platforms) != [("linux", "amd64"), ("linux", "arm64")]:
        raise ValueError("Unexpected predecessor platforms")

    shutil.copytree(source, destination)
    updated = []
    results = {}
    for descriptor in index["manifests"]:
        arch = descriptor["platform"]["architecture"]
        manifest = load(source, descriptor)
        config = load(source, manifest["config"])
        if (config["os"], config["architecture"]) != ("linux", arch):
            raise ValueError(f"Predecessor config platform mismatch for {arch}")
        assert_missing_helm(source, manifest)
        diff_id, compressed = make_layer(release_helm(assets, arch))
        layer = store(destination, compressed, {"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip"})
        replacement = copy.deepcopy(config)
        replacement["rootfs"]["diff_ids"].append(diff_id)
        replacement.setdefault("history", []).append({"created_by": "Restore the original Helm 3.20.0 binary"})
        new_config = store(destination, encode(replacement), manifest["config"])
        new = store(destination, encode({**manifest, "layers": [*manifest["layers"], layer], "config": new_config}), descriptor)
        updated.append(new)
        results[arch] = {"manifest": new["digest"], "config": new_config["digest"], "helm_sha256": BINARY[arch]}

    candidate = store(destination, encode({**index, "manifests": updated}), original)
    candidate["annotations"] = {**original.get("annotations", {}), REF: "candidate"}
    (destination / "index.json").write_bytes(encode({**root, "manifests": [candidate]}))
    print(json.dumps({"index": candidate["digest"], "platforms": results}, indent=2))


if __name__ == "__main__":
    source, assets, destination = map(Path, sys.argv[1:])
    restore(source, assets, destination)
