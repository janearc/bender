#!/usr/bin/env python3
# freeze-audit -- the freeze exit condition, as a number.
#
# Walks the live cluster and checks three properties of every workload it
# finds. It carries no inventory and no spec of its own, so a cluster that
# changes daily does not invalidate it: whatever is running today is what gets
# checked. The properties, per workload:
#
#   sha       every image tag resolves. a first-party image, a bare local
#             name, must carry a commit sha, not a mutable tag like :dev or
#             :latest and not a version string. a third-party image cannot
#             resolve to a commit in this estate, so its bar is a pinned
#             version tag. containers are judged one by one, so a sidecar
#             cannot launder the pod it rides in.
#   manifest  a yaml document under a git repository in ~/mesh/dev declares
#             this kind and name. a workload with no manifest was made by hand.
#   health    the service reports its own deployed sha at /health, reached by
#             name through traefik, and that sha matches the running image.
#             asked of first-party workloads that have a route; a workload
#             nothing routes to is not serving, so there is nothing to ask,
#             and third-party images do not know their commit.
#
# A workload is accounted for when every check that applies to it holds. The
# tool prints a table, then one number: how many workloads cannot be accounted
# for. The freeze ends when that number is zero.
#
#   freeze-audit.py [--context=k3d-whoops] [--all-namespaces]
#
# kube-system is skipped by default because k3d creates it, not repository
# code; --all-namespaces includes it. The exit code is the unaccounted count,
# capped at 100, so this runs unchanged as an integration test: layer it into
# any suite and gate on zero.
#
# Read-only against the cluster. The only kubectl verb used is get.

import json
import random
import re
import subprocess
import os
import sys
import time
import urllib.request
from concurrent.futures import ThreadPoolExecutor

# the estate root and the kubectl context are the operator's, not this
# tool's. they were literals here, which made the tool run on exactly one
# machine; an audit that silently reads the wrong tree is worse than one
# that refuses.
DEV = os.environ.get("CASSOWARY_ROOT", "")
CONTEXT = os.environ.get("CASSOWARY_CONTEXT", "")
KINDS = "deployments,statefulsets,daemonsets"
SKIP_NAMESPACES = {"kube-system"}

# a commit sha tag: hex, at least seven characters
SHA_TAG = re.compile(r"^[0-9a-f]{7,40}$")

# tags that can move under a running pod
MUTABLE_TAGS = {"latest", "dev", "main", "master", ""}

# a hex string anywhere in a health payload, candidate for the deployed sha
HEX = re.compile(r"\b[0-9a-f]{7,40}\b")


# run a command with exponential backoff and jitter, returning stdout or None.
# the cluster api is a network resource like any other and gets the same
# courtesy as one
def run_with_backoff(argv, attempts=3):
    for i in range(attempts):
        try:
            r = subprocess.run(argv, capture_output=True, text=True, timeout=30)
            if r.returncode == 0:
                return r.stdout
        except (subprocess.TimeoutExpired, OSError):
            pass
        if i < attempts - 1:
            time.sleep((0.5 * (2 ** i)) + random.uniform(0, 0.3))
    return None


# fetch one kubectl get as parsed json, or exit loudly if the cluster is gone.
# a freeze audit that cannot see the cluster has nothing to count
def kubectl_get(context, *args):
    out = run_with_backoff(["kubectl", "--context", context, "get", *args, "-o", "json"])
    if out is None:
        print(f"cannot reach context {context} with kubectl get", file=sys.stderr)
        sys.exit(100)
    return json.loads(out)


# split an image reference into (name, tag). a digest reference keeps the
# digest as its tag so it classifies as pinned rather than mutable
def image_parts(image):
    if "@" in image:
        name, digest = image.split("@", 1)
        return name, digest
    last = image.rsplit("/", 1)[-1]
    if ":" in last:
        name = image[: len(image) - len(last)] + last.split(":", 1)[0]
        return name, last.split(":", 1)[1]
    return image, ""


# a first-party image is a bare local name; anything with a registry or an
# organisation prefix is third-party
def first_party(image):
    name, _ = image_parts(image)
    return "/" not in name


# classify one workload's images, each judged by its own party. 'sha' when
# every tag passes its bar, 'pin' when the pass leans on third-party version
# tags, 'mut' for a mutable tag anywhere, 'ver' for a first-party image on a
# version string, which names a build but not a commit
def sha_status(images):
    worst = "sha"
    for img in images:
        _, tag = image_parts(img)
        if SHA_TAG.match(tag) or tag.startswith("sha256:"):
            continue
        if tag.lower() in MUTABLE_TAGS:
            return "mut"
        if first_party(img):
            worst = "ver"
        elif worst == "sha":
            worst = "pin"
    return worst


# every git repository directly under ~/mesh/dev. external/ holds third-party
# checkouts and has no top-level .git, so it excludes itself
def dev_repos():
    import os
    out = []
    for entry in sorted(os.listdir(DEV)):
        d = os.path.join(DEV, entry)
        if os.path.isdir(os.path.join(d, ".git")):
            out.append(d)
    return out


# collect (kind, name) pairs declared by every yaml document in the dev repos.
# a light parser on purpose: split multi-document files on ---, take the kind
# line and the first two-space name line, which is metadata.name in practice
def declared_workloads():
    import os
    found = set()
    kind_re = re.compile(r"^kind:\s*(Deployment|StatefulSet|DaemonSet)\s*$", re.M)
    name_re = re.compile(r"^  name:\s*([\w.-]+)\s*$", re.M)
    for repo in dev_repos():
        for root, dirs, files in os.walk(repo):
            dirs[:] = [d for d in dirs if not d.startswith(".") and d != "node_modules"]
            for f in files:
                if not f.endswith((".yaml", ".yml")):
                    continue
                try:
                    text = open(os.path.join(root, f), errors="replace").read()
                except OSError:
                    continue
                for doc in re.split(r"^---\s*$", text, flags=re.M):
                    k = kind_re.search(doc)
                    n = name_re.search(doc)
                    if k and n:
                        found.add((k.group(1), n.group(1)))
    return found


# map service name -> hosts from traefik ingressroutes. the estate rule is
# services by name through the .test tld, so the routes are the address book
def service_hosts(context):
    hosts = {}
    out = run_with_backoff(["kubectl", "--context", context, "get",
                            "ingressroutes.traefik.io", "-A", "-o", "json"])
    if out is None:
        return hosts
    for item in json.loads(out).get("items", []):
        for route in item.get("spec", {}).get("routes", []):
            names = re.findall(r"Host\(`([^`]+)`\)", route.get("match", ""))
            for svc in route.get("services", []):
                hosts.setdefault(svc.get("name", ""), []).extend(names)
    return hosts


# map workload -> hosts by matching each service selector against the
# workload's pod template labels, then following the service into the routes
def workload_hosts(context, workloads):
    services = kubectl_get(context, "services", "-A")
    by_route = service_hosts(context)
    out = {}
    for w in workloads:
        labels = w["labels"]
        hosts = []
        for svc in services.get("items", []):
            sel = svc.get("spec", {}).get("selector") or {}
            if sel and all(labels.get(k) == v for k, v in sel.items()):
                hosts.extend(by_route.get(svc["metadata"]["name"], []))
        out[w["key"]] = hosts
    return out


# fetch one url with exponential backoff and jitter, returning the body or None
def fetch_with_backoff(url, attempts=3):
    for i in range(attempts):
        try:
            with urllib.request.urlopen(url, timeout=4) as r:
                if r.status == 200:
                    return r.read().decode("utf-8", "replace")
        except OSError:
            pass
        if i < attempts - 1:
            time.sleep((0.5 * (2 ** i)) + random.uniform(0, 0.3))
    return None


# does /health on any of the workload's hosts report the deployed sha. the
# reported sha must match the image tag, prefix-wise in either direction,
# because a health endpoint that reports a sha the image does not carry is
# reporting somebody else's deployment
def health_reports_sha(hosts, image_shas):
    if not hosts:
        return "-"
    for host in hosts:
        body = fetch_with_backoff(f"http://{host}/health")
        if body is None:
            continue
        reported = set(HEX.findall(body.lower()))
        for r in reported:
            for s in image_shas:
                if r.startswith(s) or s.startswith(r):
                    return "ok"
        return "nosha"
    return "down"


# flatten one workload item from the cluster into what the checks need
def workload_of(item):
    md = item["metadata"]
    tmpl = item["spec"]["template"]
    images = [c["image"] for c in tmpl["spec"].get("containers", [])]
    return {
        "key": f"{md['namespace']}/{item['kind']}/{md['name']}",
        "ns": md["namespace"],
        "kind": item["kind"],
        "name": md["name"],
        "images": images,
        "labels": tmpl["metadata"].get("labels") or {},
    }


# the audit: gather, check, print the table, print the number
def main():
    context = CONTEXT
    include_all = "--all-namespaces" in sys.argv
    for a in sys.argv[1:]:
        if a.startswith("--context="):
            context = a.split("=", 1)[1]

    # refuse rather than traceback: an audit with no estate to read and no
    # cluster to ask is a configuration mistake, and saying so is the whole
    # of the fix.
    if not DEV:
        sys.stderr.write(
            "freeze-audit: set CASSOWARY_ROOT to the tree holding the "
            "service repositories\n")
        return 2
    if not context:
        sys.stderr.write(
            "freeze-audit: set CASSOWARY_CONTEXT or pass "
            "--context=NAME\n")
        return 2

    raw = kubectl_get(context, KINDS, "-A")
    workloads = [workload_of(i) for i in raw.get("items", [])]
    skipped = 0
    if not include_all:
        skipped = sum(1 for w in workloads if w["ns"] in SKIP_NAMESPACES)
        workloads = [w for w in workloads if w["ns"] not in SKIP_NAMESPACES]

    declared = declared_workloads()
    hosts = workload_hosts(context, workloads)

    # health checks go through a small pool so a hung service does not
    # serialise the whole audit behind its timeouts
    def probe(w):
        if not any(first_party(i) for i in w["images"]):
            return "-"
        shas = [image_parts(i)[1] for i in w["images"] if SHA_TAG.match(image_parts(i)[1])]
        return health_reports_sha(hosts.get(w["key"], []), shas or ["_"])

    with ThreadPoolExecutor(max_workers=8) as pool:
        health = dict(zip((w["key"] for w in workloads), pool.map(probe, workloads)))

    hdr = f"{'workload':<28}{'kind':<13}{'party':<7}{'sha':<6}{'manifest':<10}{'health':<8}{'accounted':<9}"
    print(hdr)
    print("-" * len(hdr))
    unaccounted = 0
    for w in sorted(workloads, key=lambda w: (w["ns"], w["name"])):
        fp = sum(1 for i in w["images"] if first_party(i))
        party = "first" if fp == len(w["images"]) else ("third" if fp == 0 else "mixed")
        sha = sha_status(w["images"])
        manifest = "ok" if (w["kind"], w["name"]) in declared else "none"
        h = health[w["key"]]
        ok = sha in ("sha", "pin") and manifest == "ok" and h in ("ok", "-")
        unaccounted += 0 if ok else 1
        print(f"{w['name']:<28}{w['kind']:<13}{party:<7}{sha:<6}{manifest:<10}{h:<8}"
              f"{'yes' if ok else 'no':<9}")

    print()
    if skipped:
        print(f"{skipped} kube-system workloads skipped; --all-namespaces includes them")
    print(f"{unaccounted} workloads cannot be accounted for")
    return min(unaccounted, 100)


if __name__ == "__main__":
    sys.exit(main())
