# skip.md — repos not tested and why

The bookmarks yielded ~280 unique GitHub repos. For this POC batch I only
covered tools that are:

- Docker-native (official image or a trivial `docker-compose.yml`),
- runnable end-to-end without manual secrets or external accounts,
- finishable in a few minutes on a 24-core Linux box with no GPU.

Everything below was intentionally left out. Categories, not a full list.

---

## 1. Awesome-lists / cheatsheets / curricula / books

No runnable code — they're study material.

- `Helmi/awesome-crypto`, `e2b-dev/awesome-ai-agents`,
  `anderspitman/awesome-tunneling`, `rothgar/awesome-tuis`,
  `ansible-community/awesome-ansible`, `dastergon/awesome-sre`,
  `ivbeg/awesome-status-pages`
- `bregman-arie/devops-exercises`, `bregman-arie/devops-resources`,
  `rohitg00/devops-interview-questions`
- `imthenachoman/How-To-Secure-A-Linux-Server`,
  `trimstray/nginx-admins-handbook`,
  `tanprathan/MobileApp-Pentest-Cheatsheet`,
  `jassics/security-study-plan`
- `mikeroyal/Apache-Kafka-Guide`, `MilovanTomasevic/Design-Patterns`,
  `sirupsen/napkin-math`, `Lissy93/networking-toolbox`
- `kubernauts/practical-kubernetes-problems`, `SadServers/sadservers`,
  `pavan-kumar-99/medium-manifests`, `jonataaraujo/CKA-2026`,
  `techiescamp/cks-certification-guide`, `cncf/curriculum`,
  `Rmarieta/AWS-SAP-Notes`
- `public-apis/public-apis`, `devsecops/bootcamp`,
  `tharlesson-platform/plano_estudos_sre`,
  `lydtechconsulting/monitoring-demo`

**Why skipped:** read-only content, not software to test.

## 2. Bookmark URLs that aren't projects

- Gists: `danielkec/4381b4c9af...`, `glaucia86/c16186da46d8...`
- Org/search pages: `orgs/nochaosio`, `orgs/grafana-community`,
  `https://github.com/`, `https://codeberg.org/`
- Specific issues / tree / blob paths (e.g. `VictoriaMetrics/VictoriaMetrics/issues/889`,
  `open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/prometheusreceiver`)

**Why skipped:** no single repo to boot.

## 3. Giant emulators / VMs / OSes

- `dockur/macos`, `dockur/windows` — image pull is tens of GB and requires
  nested virtualization; wipes a laptop's disk.
- `FEX-Emu/FEX`, `leaningtech/webvm`, `TibixDev/winboat` — desktop VM /
  emulation layers, not quick to smoke-test.

**Why skipped:** disk/CPU budget, and a reboot-adjacent blast radius.

## 4. Heavy ML / needs a GPU

- `openai/whisper`, `suno-ai/bark`, `microsoft/VibeVoice`,
  `rsxdalv/TTS-WebUI`, `CorentinJ/Real-Time-Voice-Cloning`,
  `google-research/timesfm`, `facebookresearch/co-tracker`,
  `roboflow/supervision`, `SanshruthR/CCTV_YOLO`, `mudler/LocalAI`,
  `nomic-ai/gpt4all`, `exo-explore/exo`, `langflow-ai/openrag`,
  `0xSojalSec/airllm`, `coaidev/coai`

**Why skipped:** need a GPU, tens of GB of model weights, and a real workload
to be a meaningful test.

## 5. Agent / Claude-Code study material (not standalone tools)

- `alejandrobalderas/claude-code-from-source`,
  `affaan-m/everything-claude-code`,
  `mukul975/Anthropic-Cybersecurity-Skills`,
  `ColeMurray/claude-code-otel`,
  `drona23/claude-token-efficient`,
  `jarrodwatts/claude-hud`,
  `thedotmack/claude-mem`,
  `peteromallet/dataclaw`,
  `nullclaw/nullclaw`, `krillclaw/KrillClaw`, `openclaw/openclaw`,
  `angristan/noclaw`, `pixlcore/xyops`

**Why skipped:** they're configs/skills/research artifacts — better reviewed
inside an actual Claude Code session, not booted.

## 6. Desktop GUIs / Android / hardware

- `Abdenasser/neohtop`, `AndnixSH/APKToolGUI`,
  `amir1376/ab-download-manager`, `mountain-loop/yaak`,
  `antares-sql/antares`, `tiny-craft/tiny-rdm`,
  `WerWolv/ImHex`, `casualsnek/cassowary`, `rxi/lite`, `SpartanJ/ecode`
- `MewPurPur/GodSVG`, `Lxtharia/minegrub-theme`,
  `JaKooLit/Arch-Hyprland`, `Hyde-project/hyde`, `prasanthrangan/hyprdots`
- `apolzek/DigiSpark-Scripts`, `luisbraganca/rubber-ducky-library-for-arduino`,
  `sipeed/NanoCluster`, `cabelo/libzupt`

**Why skipped:** need an X/Wayland session, Android, or physical hardware
(rubber ducky, Sipeed cluster, DigiSpark).

## 7. TUIs / shell tooling that needs a real TTY

These would run but the POC harness here can only do non-interactive smoke
tests; for these, a smoke test is no more meaningful than a `docker pull`.

- `ynqa/jnv`, `darrenburns/posting`, `alemidev/scope-tui`,
  `fedexist/grafatui`, `Canop/dysk`, `rgwood/systemctl-tui`,
  `erikjuhani/basalt`, `atuinsh/atuin`, `ayn2op/discordo`,
  `yorukot/superfile`, `Dyneteq/reconya`, `bensadeh/tailspin`,
  `fujiapple852/trippy`, `hatoo/oha`, `tsl0922/ttyd`

## 8. Offensive / RAT / dual-use requiring explicit engagement context

- `1N3/Sn1per`, `Cryakl/Ultimate-RAT-Collection`, `Neo23x0/yarGen-Go`,
  `matrixleons/evilwaf`, `Ragnt/AngryOxide`, `sepinf-inc/IPED`,
  `collaborator-ai/collab-public`, `lijiejie/EasyPen`,
  `gitlab.com/bigbodycobain/Shadowbroker`

**Why skipped:** these are offensive tooling / leaked-exploit archives.
Running them needs a stated authorized engagement or isolated lab, both
outside this POC.

## 9. Big platforms / full stacks (would need a real deploy)

Not a bad fit technically, but each of these is a multi-service stack with
SSO, persistence, or multi-tenant config — too heavy for a one-shot POC:

- `SigNoz/signoz`, `coroot/coroot`, `perses/perses`,
  `VictoriaMetrics/VictoriaMetrics`, `VictoriaMetrics/VictoriaLogs`,
  `VictoriaMetrics/prometheus-benchmark`
- `jumpserver/jumpserver`, `Infisical/infisical`, `flagsmith/flagsmith`,
  `openobserve/openobserve`, `localstack/localstack`, `ministackorg/ministack`,
  `raghavyuva/nixopus`, `aws-samples/aws2tf`, `safing/portmaster`,
  `konstructio/kubefirst`, `aenix-io/cozystack`, `azukaar/Cosmos-Server`,
  `IceWhaleTech/CasaOS`, `syncthing/syncthing`, `dokku/dokku`,
  `hoppscotch/hoppscotch`, `n8n-io/n8n`, `jumpserver/jumpserver`
- `pocketbase/pocketbase`, `PostgREST/postgrest`, `neo4j/neo4j`,
  `kuzudb/kuzu`, `rustfs/rustfs`, `AutoMQ/automq`, `rosedblabs/rosedb`,
  `kubeshark/kubeshark`, `kubernetes-sigs/kubespray`,
  `kubernetes-sigs/cluster-capacity`, `IBM/charts`
- `flyteorg/flyte`, `apify/crawlee`, `puppeteer/puppeteer`
- `Shopify/toxiproxy`, `SigNoz/signoz`

**Why skipped:** worth a dedicated POC each.

## 10. Needs accounts, cloud creds, or specific SaaS

- `guerzon/vaultwarden`, `Infisical/infisical`, `tellerops/teller`
- `kavehtehrani/cloudflare-speed-cli`, `dockur/*`
- `flagsmith/flagsmith`, `Flagsmith/flagsmith`

**Why skipped:** POC would just dead-end at a login screen.

## 11. Build-from-source non-Docker projects

Still probably testable, but each is a full toolchain install (Rust/Zig/Java)
and the goal here was "safe, Docker-only POCs".

- `dioxuslabs/dioxus`, `pyrra-dev/pyrra`, `securego/gosec`,
  `VictoriaMetrics/prometheus-benchmark`, `Hakky54/certificate-ripper`,
  `baldimario/cq`, `orhun/ratzilla`, `avinassh/s3-log`,
  `maravondra/mq_communication`, `wiliamvj/golang-sqlc`,
  `indrayyana/go-fiber-boilerplate`, `lspraciano/fastapiAPITemplate`,
  `renggadiansa/golang-API`
- `Feleys/simple-golang-realtime-chat-room`, `calvinmclean/babyapi`,
  `sammwyy/MikuMikuBeam` (DDoS stress tool — also dual-use),
  `cesarferreira/rip`, `denizgursoy/inpu`

## 12. Candidates worth a second-round POC

Perfectly testable, just didn't fit today's batch. If you want a round 2,
start here:

- `dstotijn/hetty` — HTTP pentest proxy (tiny, Docker-friendly)
- `opengrep/opengrep` — fork of semgrep, scan this very repo
- `projectdiscovery/katana` — crawler, one-liner Docker
- `sbom-tool/sbom-tools` — SBOM diff
- `atuinsh/atuin` — shell history sync (needs server + client)
- `Shopify/toxiproxy` — network-fault injector, trivial Docker
- `groundcover-com/murre` — K8s container metrics, needs a kind cluster
- `undistro/marvin` — K8s CEL-based auditor, also needs a kind cluster
- `tarampampam/webhook-tester` — already tested ✓
- `fatedier/frp`, `xjasonlyu/tun2socks`, `go-gost/gost` — tunneling stack
- `clidey/whodb` — universal DB browser
- `perses/perses`, `dotdc/grafana-dashboards-kubernetes` — observability
  dashboards (need a Prometheus/Grafana stack standing behind them)
