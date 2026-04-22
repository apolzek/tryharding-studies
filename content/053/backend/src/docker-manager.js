import Docker from "dockerode";
import tar from "tar-stream";
import { randomUUID } from "node:crypto";
import { Readable } from "node:stream";
import fs from "node:fs";
import path from "node:path";

const docker = new Docker({ socketPath: "/var/run/docker.sock" });

const IMAGE = process.env.CHALLENGE_IMAGE || "sre-challenge-base:latest";
const PORT_MIN = Number(process.env.CHALLENGE_PORT_MIN || 9100);
const PORT_MAX = Number(process.env.CHALLENGE_PORT_MAX || 9199);
const NETWORK = process.env.CHALLENGE_NETWORK || "sre-challenges";
const BASE_CTX = "/workspace/challenge-base";

// Ports currently in use by active challenge containers.
const usedPorts = new Set();

export function allocatePort() {
  for (let p = PORT_MIN; p <= PORT_MAX; p++) {
    if (!usedPorts.has(p)) {
      usedPorts.add(p);
      return p;
    }
  }
  throw new Error("no free ports in challenge range");
}

export function releasePort(p) {
  usedPorts.delete(p);
}

export async function ensureBaseImage() {
  try {
    await docker.getImage(IMAGE).inspect();
    console.log(`[docker] image ${IMAGE} already present`);
    return;
  } catch {
    // not found → build
  }

  if (!fs.existsSync(BASE_CTX)) {
    console.warn(`[docker] build context ${BASE_CTX} not mounted; skipping build`);
    return;
  }
  console.log(`[docker] building ${IMAGE} from ${BASE_CTX}`);
  const files = fs.readdirSync(BASE_CTX);
  const stream = await docker.buildImage(
    { context: BASE_CTX, src: files },
    { t: IMAGE }
  );
  await new Promise((resolve, reject) => {
    docker.modem.followProgress(
      stream,
      (err, out) => (err ? reject(err) : resolve(out)),
      (ev) => {
        if (ev.stream) process.stdout.write(ev.stream);
      }
    );
  });
  console.log(`[docker] built ${IMAGE}`);
}

// Build a tar stream with the challenge scripts so we can `putArchive`
// them into the container before start. Keeps Dockerfile generic.
function buildChallengeTar({ setup, verify, title, objective }) {
  const pack = tar.pack();
  pack.entry({ name: "challenge/setup.sh", mode: 0o755 }, setup || "");
  pack.entry({ name: "challenge/verify.sh", mode: 0o755 }, verify || "");
  pack.entry(
    { name: "challenge/meta.env", mode: 0o644 },
    `CHALLENGE_TITLE=${JSON.stringify(title)}\nCHALLENGE_OBJECTIVE=${JSON.stringify(objective)}\n`
  );
  pack.finalize();
  return pack;
}

async function ensureNetwork() {
  try {
    await docker.getNetwork(NETWORK).inspect();
  } catch {
    await docker.createNetwork({ Name: NETWORK, Driver: "bridge" });
  }
}

export async function startChallengeContainer(challenge, port) {
  await ensureNetwork();
  const name = `sre-${challenge.slug}-${randomUUID().slice(0, 8)}`;

  const container = await docker.createContainer({
    Image: IMAGE,
    name,
    Env: [
      `CHALLENGE_TITLE=${challenge.title}`,
      `CHALLENGE_OBJECTIVE=${challenge.objective}`,
    ],
    Labels: {
      "sre-challenge": "true",
      "sre-challenge-slug": challenge.slug,
    },
    ExposedPorts: { "7681/tcp": {} },
    HostConfig: {
      PortBindings: { "7681/tcp": [{ HostPort: String(port) }] },
      RestartPolicy: { Name: "no" },
      // memory caps to keep things sane
      Memory: 512 * 1024 * 1024,
      PidsLimit: 500,
      Privileged: !!challenge.privileged,
      NetworkMode: NETWORK,
      AutoRemove: false,
    },
  });

  const pack = buildChallengeTar({
    setup: challenge.setup_script,
    verify: challenge.verify_script,
    title: challenge.title,
    objective: challenge.objective,
  });
  await container.putArchive(pack, { path: "/" });

  await container.start();
  return { id: container.id, name };
}

export async function execVerify(containerId) {
  const container = docker.getContainer(containerId);
  const exec = await container.exec({
    Cmd: ["bash", "/challenge/verify.sh"],
    AttachStdout: true,
    AttachStderr: true,
    User: "root",
  });
  const stream = await exec.start({ hijack: true, stdin: false });
  const chunks = [];
  await new Promise((resolve) => {
    stream.on("data", (c) => chunks.push(c));
    stream.on("end", resolve);
    stream.on("close", resolve);
  });
  const info = await exec.inspect();
  // Docker multiplexes stdout/stderr; strip the 8-byte frames for readability.
  const output = Buffer.concat(chunks)
    .toString("utf8")
    .replace(/[\x00-\x08\x0B-\x1F]/g, "");
  return { exitCode: info.ExitCode ?? -1, output };
}

export async function stopChallengeContainer(containerId) {
  try {
    const c = docker.getContainer(containerId);
    await c.stop({ t: 2 }).catch(() => {});
    await c.remove({ force: true }).catch(() => {});
  } catch {
    // already gone
  }
}

// Clean up any orphans on boot + periodically. A session is "orphaned" if
// the container exited but we still track the port.
export async function listChallengeContainers() {
  return docker.listContainers({
    all: true,
    filters: { label: ["sre-challenge=true"] },
  });
}
