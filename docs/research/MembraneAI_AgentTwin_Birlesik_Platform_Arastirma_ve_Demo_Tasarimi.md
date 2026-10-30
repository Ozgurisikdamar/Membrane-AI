# MEMBRANE AI + AGENTTWIN
## Birleşik ürün stratejisi, teknik mimari, pazar araştırması ve demo hazırlık dokümanı

**Hazırlanma tarihi:** 30 Ekim 2026  
**Kapsam:** Yalnızca analiz, araştırma, ürün/mimari tasarım ve demo planı. Bu çalışma kapsamında kod yazılmamıştır.  
**İncelenen kod tabanları:** [Ozgurisikdamar/Membrane-AI](https://github.com/Ozgurisikdamar/Membrane-AI) ve [Ozgurisikdamar/AgentTwin](https://github.com/Ozgurisikdamar/AgentTwin)  
**Ana karar:** Membrane AI üst ürün/ortak governance katmanı; AgentTwin ise Membrane AI içindeki Agent Assurance ürün modülü olmalıdır.

---

# 0. Yönetici özeti

Bu iki repo birleştirilmeye uygun; çünkü aynı problemi farklı kontrol noktalarında ele alıyorlar:

- **Membrane AI**, AI destekli yazılım geliştirme sürecinde kod, bağımlılık, MCP/tool kullanımı ve kurumsal mimari uyumunu denetliyor. Kontrol noktası ağırlıklı olarak kod değişikliği ve geliştirme/CI akışı.
- **AgentTwin**, agent'ın çalışma anındaki davranışını izliyor; araç çağrılarını, süreç sırasını, yeniden denemeleri, yan etkileri ve doğrulanmış iş sonucunu simüle ediyor ve sürüm karşılaştırması yapıyor. Kontrol noktası agent'ın davranışı ve üretime çıkış kararı.

Birleşik ürünün değer önerisi şu olmalı:

> **Membrane AI, AI ile geliştirilen yazılımı ve bu yazılımın çalıştırdığı agent'ları tek bir güvenlik, doğrulama ve kanıt zinciri içinde yönetir.**

Bu değer önerisi "bir dashboard'da iki ürünü göstermek" seviyesinde kalmamalı. Asıl fark, bir kod/agent değişikliği ile o değişikliğin çalışma anındaki etkisini ilişkilendiren, denetlenebilir bir **evidence chain** kurmak olmalı:

`AI-generated code diff → code governance verdict → agent manifest/version → OTel trace → stateful twin simulation → baseline/candidate evaluation → release decision → production outcome → regression case`

## 0.1 En önemli kararlar

1. **Repoları hemen tek monorepo'ya taşımayın.** İlk aşamada servisleri ve veri sahipliğini koruyup entegrasyonu API/event sözleşmeleriyle yapın. Ürün kabuğu ve ortak kimlik/policy/evidence modeli sonradan çıkarılabilir.
2. **Membrane AI üst marka olsun.** AgentTwin adı, ürün içinde `Membrane AI – Agent Assurance` veya `Membrane AI AgentTwin` olarak korunabilir. AgentTwin'in teknik olarak özgün adı ve kavramları (stateful twin, fault injection, paired replay) korunmalı.
3. **İlk demo için AgentTwin'in mevcut refund senaryosunu kullanın.** Baştan yeni demo agent yazmak yerine var olan Demo Co Support Refund Agent, 1.2.4 / 1.3.0 / 1.3.1 sürümleri ve dokuz senaryolu twin paketi kullanılmalı.
4. **Membrane tarafında gerçek olanla hedef mimariyi ayırın.** Kod taraması, CLI, VS Code uzantısı, GitHub status/PR comment ve MCP policy karar motoru var. Ancak repo içinde gerçek generation-time prompt gateway henüz mevcut değil; MCP gateway prompt üretim akışını incelemiyor. AST katmanı, gerçek embedding endpoint'i ve canlı vLLM/premium model bağlantıları da tamamlanmış ürün kabiliyeti olarak sunulmamalı.
5. **AgentTwin tarafında release gate'i varmış gibi sunmayın.** Trace, scenario, twin, simulation, evaluation, baseline/candidate karşılaştırması ve web UI var. Blast-radius servisi, release lifecycle/gate, regression mining ve runtime gateway uygulama planında sonraki fazlar olarak duruyor.
6. **İlk müşteri demosu deterministik olmalı.** AgentTwin'in `ScriptedPlannerModel` demo modunu kullanın. Gerçek Claude bağlantısı opsiyonel ikinci gösterim olsun; test ve release kararı deterministik akışa dayansın.
7. **Ortalama bir kalite puanı ile güvenlik kararı vermeyin.** Yeni kritik ihlal, yetkisiz işlem, çift yan etki, tenant sınırı ihlali veya doğrulanamayan başarı gibi deterministik bulgular PASS kararını geçersiz kılmalı.
8. **OpenTelemetry GenAI semantik sözleşmelerini temel alın.** Agent, model, tool, retrieval ve workflow span'leri için kendi özel formatınızı yaratmayın; AgentTwin'e özel metadata'yı bu standardın üzerine ekleyin.

## 0.2 Ürün adı ve anlatısı

Önerilen dış anlatı:

**MEMBRANE AI — AI Software & Agent Assurance**  
*Secure the code. Validate the agent. Prove the release.*

Ürün iki ana modüle ayrılır:

- **Code Governance:** AI tarafından üretilen veya değiştirilen kodun güvenlik, mimari, paket, secret ve MCP/tool kullanım politikalarıyla uyumunu denetler.
- **Agent Assurance (AgentTwin):** Agent'ın üretim benzeri koşullarda nasıl davrandığını simüle eder, regresyonları bulur, kanıtlar ve sonraki aşamada runtime'da riskli eylemleri kontrol eder.

Ortak katmanlar: Identity & Tenancy, Policy Registry, Evidence & Audit, Integrations, Unified Dashboard, Deployment & Data Governance.

---

# 1. GitHub repo incelemesi: mevcut durum ve teknik çıkarımlar

İnceleme, public `main` branch'lerinde bulunan README, mimari dokümanlar, roadmap/implementation board, ADR'ler, ana servis kodları, testler ve demo dosyaları üzerinden yapılmıştır. Bu nedenle aşağıdaki durum, 30 Ekim 2026 itibarıyla GitHub üzerinde görülebilen kaynak durumudur; yerel makinede henüz push edilmemiş değişiklikleri kapsamaz.

## 1.1 Membrane AI — kod tabanının gerçek kapsamı

Repo: [Membrane-AI](https://github.com/Ozgurisikdamar/Membrane-AI)  
İncelenen son GitHub commit'i: `b8e8cfd` — 26 Eylül 2026.

### A. Mevcut ana akış

Membrane'ın temel analiz akışı şu şekilde tasarlanmış:

1. IDE/geliştirici veya SCM webhook üzerinden kod değişikliği gönderilir.
2. Ingestion servisi submission'ı alır ve `code.submission.v1` olayını event backbone'a gönderir.
3. Orchestrator önce BLAKE3 tabanlı cache kontrolü yapar.
4. Cache miss halinde analiz aşamaları çalışır: deterministic scan, resolver/gold-code context ve opsiyonel semantic değerlendirme.
5. Secret değerleri downstream semantic aşamalara gitmeden maskelenir.
6. Sonuç `allow / reject / needs review` benzeri karara dönüştürülür, audit kaydı ve outbox üzerinden raporlanır.
7. Reporter GitHub commit status, PR comment ve webhook kanallarına bildirim gönderir. Shadow/enforce modu ile önce gözlem, sonra merge engelleme uygulanabilir.

### B. Repo içinde görülen servis ve bileşenler

| Bileşen | Durum | Gerçek görev / çıkarım |
|---|---|---|
| `services/ingestion` | Var | gRPC submission/stream ve Git webhook girişleri; Kafka/Redpanda publisher. |
| `services/orchestrator` | Var | Cache-first Saga; analiz aşamalarını deadline altında çalıştırma; fallback. |
| `services/analyzer` | Var | Secret detection/masking ve bazı riskli kod pattern kontrolleri. |
| `services/resolver` | Var | Gold-codebase context için pgvector tabanlı top-k arama; gerçek embedding entegrasyonu henüz tamamlanmamış. |
| `services/semantic` | Var | FastAPI semantic evaluation boundary; vLLM adapter ve Claude+Gemini consensus adapter mevcut, ancak canlı endpoint/key ve prompt tuning gerekir. |
| `services/gateway` | Var | MCP/tool-call governance, package-install firewall ve shadow-AI sınıflandırma. |
| `services/reporter` | Var | Verdict bildirimleri; GitHub commit status, PR comment, webhook/log, idempotent delivery. |
| `pkg/scan` | Var | Analyzer/CLI/fallback tarafından ortak kullanılan deterministic line-based scanner ve secret masking. |
| `clients/cli` | Var | `membrane scan`, JSON/Markdown/HTML raporları ve CI exit code davranışı. |
| `clients/vscode` | Var | CLI'ı çalıştırıp editörde diagnostic gösteren VS Code/Cursor uzantısı. |
| Docker/Helm/Terraform/CI dosyaları | Var | Dev/full compose, multi-arch imaj, Helm chart ve Terraform deployment yüzeyi. |

### C. Önemli teknik ayrıntılar

**Deterministic scanner:** `pkg/scan/scan.go` içinde secret pattern'leri ile satır bazlı bazı riskli pattern'ler tanımlanmış. Örnekler: private key/API token benzeri credential, SQL string concatenation, Go `exec.Command` concatenation, `InsecureSkipVerify`. Secret değerleri `[MASKED:<rule>]` placeholder'ı ile maskeleniyor. Bu, iyi bir ilk baseline; ancak tam AST/data-flow/taint analysis veya kapsamlı SAST motoru ile eşdeğer değildir.

**Gold-codebase RAG:** Resolver, kuruma ait onaylı kod örneklerinden mimari bağlam çıkarmak üzere pgvector/HNSW araması yapıyor. Mimari açıdan doğru bir farklılaştırıcı: genel bir kod önerisi yerine organizasyonun kendi kabul edilmiş örüntülerini kullanabiliyor. Ancak repo roadmap/hand-over, gerçek embedding sağlayıcısının henüz bağlanmadığını belirtiyor. Bu nedenle gerçek embedding kalitesi veya RAG relevance performansı kanıtlanmış kabul edilmemeli.

**Semantic evaluation ve model maliyeti:** Tiered tasarım cache → deterministic/local/heuristic → premium model consensus fikrini destekliyor. Claude Sonnet 4.6 ve Gemini için concurrent consensus adapter'ı var; partial failure toleransı ve sanitized errors düşünülmüş. Buna rağmen gerçek vLLM endpoint'i, gerçek premium credentials ve prompt tuning canlı olarak yapılandırılmadan semantic kaliteyi ölçtüğümüzü iddia etmemeliyiz.

**MCP governance:** `services/gateway/internal/domain/policy.go` deterministik kararla tool call, package install ve shadow AI sinyallerini yönetiyor. Tool deny/allow list, dangerous-argument pattern, package allow/deny list, typosquat near-miss, riskli paket ismi ve sanctioned agent listesi gibi kararlar var. Bu modül, AgentTwin'in ilerideki runtime action gateway'i için doğal bir policy kaynağı olabilir. Ancak mevcut uygulama, gerçek bir MCP protocol reverse proxy veya üretim esnasında tüm prompt/response akışını yakalayan gateway olarak değerlendirilmemeli.

**Önemli repo/mimari uyuşmazlığı:** README/architecture bazı yerlerde generation-time gateway ve pre-generation interception anlatıyor. Buna karşın mevcut ağaçta `clients/gateway` bulunmuyor; `services/gateway` MCP tool governance servisidir. Roadmap de prompt-injection incelemesinin generation stream üzerinde daha sonra derinleştirileceğini açıkça söylüyor. Demo materyalinde bu fark net olmalı.

**Kalite/eval:** Repo içindeki golden corpus değerlendirmesinde P/R 1.0 ve false-positive rate 0 raporlanmış. Bunu gerçek müşteri kodu üzerinde yüzde 100 doğruluk diye pazarlamayın. Bu, mevcut küçük/etiketli golden corpus için elde edilmiş yerel regresyon göstergesidir. Üretim başarısı için dil, framework, repo büyüklüğü, kod stili ve false-positive dağılımını kapsayan çok daha büyük bir benchmark gerekir.

### D. Membrane'da bugün gösterilebilecekler

- CLI ile repository/path scan.
- Maskelenmiş secret finding ve line-level risk bulguları.
- JSON/Markdown/HTML teknik borç raporu.
- VS Code/Cursor üzerinde scan sonuçları.
- GitHub commit status ve PR yorum/merge gate entegrasyonu.
- Shadow/enforce modu.
- MCP tool-call policy kararları.
- Package install firewall / typosquat sinyali.
- Shadow-AI classification.
- Event-driven orchestrator, audit ve telemetry mimarisi.

### E. Membrane'da canlı demo öncesi doğrulanması gerekenler

- Gerçek embedding provider bağlanmalı ve gold-code retrieval relevance ölçülmeli.
- Gerçek vLLM endpoint'i ile tier-2 prompt ve structured output test edilmeli.
- Premium model key'leriyle Claude/Gemini consensus canlı test edilmeli; latency, maliyet, partial failure ve disagreement ölçülmeli.
- AST tabanlı analiz ve full-file context entegrasyonu mevcut scanner'dan ayrıştırılarak planlanmalı.
- Gerçek generation-time interception iddiası sunulacaksa bu kabiliyetin mevcut olmadığı dikkate alınmalı; ilk demo mevcut CLI/VS Code/SCM sınırında kalmalı.
- Gateway tool/policy API'nin gerçek MCP protokolüyle ve kullanıcı kimliği/tenant authorization ile nasıl bağlandığı uçtan uca doğrulanmalı.
- GitHub Actions account-side startup failure durumları nedeniyle yerel `task ci` sonuçlarını GitHub-hosted CI başarısı olarak göstermemek gerekir. Repo hand-over notu GitHub Actions'ın 55/55 run'da `startup_failure` verdiğini; yerel kalite kapısı olarak `task ci` kullanıldığını söylüyor.

## 1.2 AgentTwin — kod tabanının gerçek kapsamı

Repo: [AgentTwin](https://github.com/Ozgurisikdamar/AgentTwin)  
İncelenen son GitHub commit'i: `1e8fd0e` — 26 Eylül 2026.

### A. Ürünün gerçek çekirdeği

AgentTwin'in temel sorusu: **“Bu agent değişikliğini güvenli biçimde ship edebilir miyim ve bu kararı hangi kanıt destekliyor?”**

Bu yaklaşım basit tracing ürünlerinden daha ileri gidiyor. Agent'ın ürettiği final cevaba güvenmek yerine:

- hangi tool'ları çağırdığı,
- hangi argümanları gönderdiği,
- çağrı sırası,
- retry ve idempotency davranışı,
- tool sonucunun gerçek state ile tutarlılığı,
- side effect sayısı ve final business state,
- policy/approval şartları,
- simülasyon seed'i ve sürüm pinning

gibi verileri incelemeyi hedefliyor.

### B. Mevcut servisler ve yüzeyler

| Bileşen | Durum | Görev |
|---|---|---|
| `services/control-plane` | Var | Auth, org/tenant, project, agent manifest/version, RBAC ve audit. |
| `services/trace-service` | Var | OTLP trace ingest, span normalizasyonu, outcome ve trace explorer API. |
| `services/simulation-service` | Var | Senaryo, declarative twin, stateful tool simulation, fault injection ve run lifecycle. |
| `services/evaluation-service` | Var | Deterministic/semantic evaluator, dataset, baseline-candidate comparison, judge/review/calibration. |
| `services/graph-service` | Kısmi | Graph model/traversal altyapısı var; tam ürün API'si ve change-aware blast radius fazı tamamlanmış değil. |
| Runtime gateway | Henüz yok | Mimari/roadmap hedefinde; gerçek tool action enforcement servisi olarak repo içinde tamamlanmış değil. |
| Web UI | Var | Trace explorer, scenario editor, simulation detail, eval comparison, dataset, judge ve review queue. |
| Python SDK | Var | OTel/GenAI span'leri, redaction, trace, outcome reporting, simulation/evaluation API. |
| CLI / TypeScript SDK | Gelecek faz | README'de sonraki fazlarla birlikte geleceği belirtiliyor. |

### C. Önemli tasarım kararları

**Stateful, declarative twin:** Twin bir YAML/JSON benzeri tanımla ifade edilir; kullanıcının serbest kodu olarak çalıştırılmaz. Her test case izole başlangıç state'i alır. Twin kendi tool çağrı kayıtlarını üretir; böylece agent kendi davranışını yanlış raporlayarak kanıtı değiştiremez.

**Deterministic fault injection:** Aynı tanım, başlangıç state'i, çağrı dizisi, fault kuralı ve seed aynı sonuçları üretir. Önemli fault örnekleri:

- `timeout_after_mutation`: Para iadesi gibi yan etki gerçekleşmiştir fakat caller timeout görür.
- `success_without_mutation`: Tool başarılı döner ama iş state'i hiç değişmemiştir.
- `message_duplication`: Aynı mesaj/istek yinelenir; idempotency doğru ise ikinci etki oluşmaz.
- Rate limit, timeout, stale response, duplicate response gibi dış servis davranışları modellenebilir.

**Baseline/candidate paired comparison:** ADR-0023, iki ayrı bağımsız simülasyon başlatmak yerine aynı pinned scenario/twin/seed seti üzerinden iki tarafı tek pair operasyonunda başlatmayı tanımlar. Bu, senaryo sürümü veya seed farkının yanlışlıkla agent regresyonu diye yorumlanmasını engeller.

**LLM judge güvenilirliği:** ADR-0022 semantic evaluator hatalarını geçer not saymıyor. Judge error/timeout `ERROR` olarak işleniyor. Judge'ın insan etiketlerine kalibrasyonu, kappa ve agreement ölçülüyor. Judge'ın ürettiği kanıt alıntısı gerçekten modele gösterilen materyalde bulunmuyorsa reddediliyor. Bu, birleşik platformda korunması gereken güçlü bir tasarım.

**Agent self-report yerine bağımsız outcome:** Agent'ın “refund succeeded” demesi başarı kanıtı değil. State assertion veya dış sistem callback'i gibi bağımsız doğrulama gerekir. Bu kabiliyet ürünün en güçlü farklılaştırıcı unsurlarından biri.

### D. AgentTwin'in mevcut demo kanıtı

Mevcut Demo Co Support Refund Agent, kod tabanında zaten bulunan çok iyi bir sunum senaryosu:

- `1.2.4`: iyi baseline.
- `1.3.0`: regresyonlu candidate; policy kontrolünden önce refund çağrısı yapabiliyor ve tool'un yalan başarı yanıtına güvenebiliyor.
- `1.3.1`: düzeltilmiş sürüm.
- 9 scenario: happy path, over-limit, timeout-after-mutation, false tool success, rate limit, prompt injection/malicious retrieved content, cross-tenant order, unauthorized admin tool ve benzeri sınır koşulları.

Implementation board'da kayıtlı seed-42 sonuçlarında 1.2.4 için 9/9 başarılı; 1.3.0 için 6/9 başarılı ve üç başarısız senaryo görülüyor. İki senaryo kritik; sorunlar yalnızca cevap kalitesi değil, duplicate side effect, gerçek state mismatch, sahte başarı ve işlem sırası ihlali. Bu, demoda izleyiciye kolay anlatılabilir ve gözle görülür bir fark sağlar.

### E. AgentTwin'de henüz tamamlanmamış parçalar

`docs/plan/implementation-board.md` faz tablosunda 0–3 tamamlanmış; 4–8 pending olarak işaretli:

- Phase 4: graph, manifest/OpenAPI/MCP import, observed edges, ChangeSet ve blast radius.
- Phase 5: release objesi, gate rules, immutable evidence, CLI CI output ve release UI.
- Phase 6: production failure mining, clustering/taxonomy ve regression promotion.
- Phase 7: runtime gateway, CEL policy, approvals, idempotency ve SSRF-safe proxy.
- Phase 8: tenant isolation, SSRF, redaction, chaos/load/security ve E2E hardening.

Bunlar AgentTwin README'deki nihai ürün vizyonunun parçalarıdır ancak repo implementasyon board'unda tamamlanmamış işlerdir. Bir demo metninde “ürünün şu anda yapabildiği” diye değil, “hedeflenen sonraki capability” diye ifade edilmeli.

## 1.3 İki repo birlikte okunduğunda çıkan mimari sonuç

| Boyut | Membrane AI | AgentTwin | Birleşik ürün için çıkarım |
|---|---|---|---|
| Kontrol zamanı | Kod değişikliği / IDE / SCM | Agent run / test / evaluation | Pre-merge code governance + pre-release agent assurance + runtime control |
| Kanıt nesnesi | Diff, finding, ruleset, verdict, audit | Trace, scenario, twin state, simulation, eval | Tek release evidence record içinde birbirine bağlanmalı |
| Policy | Tool/package/code rules | Agent scenario expectations; runtime policy hedefte | Ortak policy registry; domain-specific policy engines |
| Retrieval | Gold-codebase RAG | Judge/eval data; pgvector | Knowledge store'lar ayrı domain owner'da, metadata ile ilişkilendirilmeli |
| Telemetry | OTel servis tracing | Agent OTel tracing | Agent span sözleşmeleri + Membrane analysis trace context ortaklaştırılmalı |
| Event backbone | Kafka/Redpanda | RabbitMQ | İlk entegrasyonda REST/webhook + outbox; hemen transport migration yok |
| Database | PostgreSQL/Aurora + pgvector | PostgreSQL + pgvector | İlk aşamada ayrı DB/schema; ortak ID ve evidence ref ile bağlantı |
| Üretim enforcement | GitHub status/PR, shadow/enforce | Runtime gateway hedefte | Code merge gate var; agent runtime gateway sonraki faz |
| Olgunluk | Kod governance P0–P3 büyük ölçüde mevcut; live-model bağlantıları eksik | Phases 0–3 mevcut; 4–8 pending | Birleşik demo yapılabilir, fakat tek ürün uçtan uca release enforcement henüz yok |

**Ana çıkarım:** Ortaklaştırılması gereken şey öncelikle servis kodu değil, **ürün sözleşmesi ve kanıt kimliği**. `org_id`, `project_id`, `repo_commit`, `agent_id`, `agent_version`, `manifest_hash`, `prompt_hash`, `tool_schema_hash`, `policy_version`, `scenario_set_version`, `evaluation_run_id` ve `trace_id` arasında kararlı ilişki kurulursa iki sistem tek bir assurance zinciri sunabilir.

---

# 2. Pazar ve rakip analizi

## 2.1 Pazar kategorileri

Rakipleri tek bir kategoriye koymak hatalı olur. Bu ürün alanı en az beş pazara ayrılmış durumda:

1. **AI coding governance / AppSec:** AI tarafından üretilen kodun güvenliği, dependency/secret scanning, code review, repo policy ve PR enforcement.
2. **LLM/agent observability:** Agent çağrıları, tool spans, latency, token, errors, sessions ve trace debugging.
3. **Evaluation & experimentation:** Dataset, offline/online evaluation, LLM-as-judge, human feedback, prompt/model compare ve CI regression test.
4. **Agent simulation & red teaming:** Multi-turn synthetic conversations, adversarial probes, tool misuse, MCP poisoning, scenario simulation ve fault testing.
5. **Runtime agent governance:** Agent identity, tool access, MCP gateway, scoped credentials, approvals, sandbox ve egress control.

Membrane'ın rekabet alanı, bu kategorilerin tamamını tek başına yeniden icat etmek değil; aralarındaki veri/kontrol boşluğunu bir assurance akışı olarak kapatmak olmalı.

## 2.2 Önemli rakipler ve konumları

### A. Langfuse — OSS observability + evaluation döngüsü

**Ne yapıyor?** Production trace'lerini score edebiliyor, gerçek etkileşimlerden dataset oluşturabiliyor, deneylerle prompt/model/app değişikliklerini karşılaştırabiliyor; code/LLM/human evaluator kullanabiliyor. CI experiment action ile regresyon eşiği geçilmezse job başarısız olabiliyor.

**Membrane ile kesişim:** AgentTwin'in tracing, datasets, eval comparison ve release öncesi regresyon testlerinin önemli bir bölümü.

**Muhtemel fark alanı:** Membrane'ın mevcut stateful twin/fault modellemesi ve gerçek tool state/side-effect kanıtı. Langfuse güçlü bir eval loop sunuyor, ancak Membrane'ın satış hikâyesi yalnızca trace->dataset->experiment olmamalı; dış servis davranışını kontrollü stateful twin içinde tekrar üretmek ve irreversible side-effect doğrulamak olmalı.

Kaynak: [Langfuse Evaluation Overview](https://langfuse.com/docs/evaluation/overview), [Experiments in CI/CD](https://langfuse.com/docs/evaluation/experiments/experiments-ci-cd).

### B. LangSmith / LangChain — agent development, tracing, offline/online eval

**Ne yapıyor?** LangSmith, offline benchmarking, unit/regression testing, online monitoring/anomaly detection ve farklı evaluator türleri (LLM-as-judge, code, composite, pairwise vb.) sağlıyor. LangChain/LangGraph ekosistemi ile güçlü bütünleşik.

**Membrane ile kesişim:** Trace explorer, dataset, eval, regression ve framework integrations.

**Muhtemel fark alanı:** Membrane framework-agnostic olmalı; ayrıca code governance, MCP/package firewall, gold-codebase architectural alignment ve verified side effect twin'lerini ürünün merkezine koymalı. LangSmith'le rekabet etmek için yalnızca daha fazla grafiğe ihtiyaç yok; release evidence graph'ı ve code-to-agent causality gerekir.

Kaynak: [LangSmith Evaluation Types](https://docs.langchain.com/langsmith/evaluation-types).

### C. Arize Phoenix / Arize AX — OpenTelemetry tabanlı tracing ve evaluation

**Ne yapıyor?** Phoenix open-source platformu OTel tabanlı tracing, eval, versioned dataset, experiment, prompt management ve trace replay sunuyor. OpenAI Agents SDK, Claude Agent SDK, LangGraph, CrewAI, ADK ve başka framework'ler için entegrasyonları bulunuyor. Managed platform tarafında Arize AX var.

**Membrane ile kesişim:** AgentTwin'in trace/eval/UI yüzeyi ve OTel entegrasyonu.

**Önemli rekabet notu:** Phoenix'in coding agent'lar tarafından kullanılabilen CLI/skills/MCP yüzeyi de gelişmiş durumda. Bu nedenle "OTel trace'lerini gösteriyoruz" farklılaştırıcı değildir.

**Muhtemel fark alanı:** Twin'in stateful dependency behavior'ı, seeded fault injection, idempotency/duplicate side-effect testi, baseline-candidate aynı seed pair'i, Membrane Code Governance ile birleştirilmiş change-to-release evidence.

Kaynak: [Phoenix GitHub](https://github.com/Arize-ai/phoenix), [Phoenix OSS](https://arize.com/phoenix-oss/).

### D. OpenAI Agents SDK / Agent Evals / Trace Grading

**Ne yapıyor?** OpenAI Agents SDK, model çağrıları, tool çağrıları, handoff ve guardrail olaylarını trace edebiliyor. Trace grading ve Agent Evals akışları, agent trace'lerini ve tam iş akışlarını yapılandırılmış evaluator'larla ölçmeye yarıyor.

**Membrane ile kesişim:** AgentTwin'in OTel trace ingestion, trajectory eval, tool-call correctness ve regression workflow'u.

**Muhtemel fark alanı:** Membrane model sağlayıcısına bağlı kalmamalı; farklı framework/provider'ları tek tenant/project modelinde kabul etmeli. Ayrıca kaynak kod governance, deterministic twin state ve bağımsız side-effect verification ana ürün kabiliyeti olmalı.

Kaynak: [OpenAI Agent Evals](https://developers.openai.com/api/docs/guides/agent-evals), [Trace Grading](https://developers.openai.com/api/docs/guides/trace-grading), [Agents SDK tracing](https://openai.github.io/openai-agents-python/tracing/).

### E. Microsoft Foundry — kurumsal agent evaluation ve red teaming

**Ne yapıyor?** Test dataset'leri, üretilmiş yanıtlar, simulated multi-turn conversations ve production traces üzerinden agent değerlendirmesi; built-in/custom evaluators, red-team testleri, CLI/CI deployment bağlantıları sunuyor.

**Membrane ile kesişim:** AgentTwin'in evaluation, synthetic scenario, trace-to-dataset ve güvenlik testleri.

**Muhtemel fark alanı:** Cloud/provider bağımsızlık; self-host/VPC/air-gapped hedefi; stateful twin engine'in belirli business side-effect'leri kontrollü biçimde modellemesi; code governance ile agent release assurance'ın aynı kanıt zincirinde birleşmesi.

Kaynak: [Microsoft Foundry Evaluation Overview](https://learn.microsoft.com/en-us/azure/foundry/concepts/evaluation-overview), [Evaluation approaches](https://learn.microsoft.com/en-us/azure/foundry/how-to/evaluate-generative-ai-app).

### F. Amazon Bedrock AgentCore — agent runtime, identity, gateway, observability ve eval

**Ne yapıyor?** AgentCore; runtime, memory, gateway, identity, browser/code interpreter, observability ve evaluations modüllerini bir araya getiriyor. Framework ve model bağımsız agent çalışma ortamı hedefliyor; OTel uyumlu telemetry ve CloudWatch dashboard sunuyor.

**Membrane ile kesişim:** Agent runtime governance, MCP gateway, trace, eval, identity ve enterprise deployment. Geniş ürün kapsamı nedeniyle önemli bir platform rakibi.

**Muhtemel fark alanı:** Membrane'ın AWS dışı, VPC/on-prem/air-gapped tercihleri; kodun üretim öncesi governance'ı; stateful twin ile deterministic adversarial validation; repo-specific architectural gold context; source-control ve runtime agent evidence bağlantısı.

**Stratejik not:** AgentCore'a karşı "biz de gateway sunuyoruz" demek yerine, farklı cloud/agent runtime'larında çalışan bir üst katman assurance/control plane olmak daha güçlü.

Kaynak: [AgentCore Developer Guide](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/), [Observability](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/observability.html), [Evaluations](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/evaluations.html).

### G. Maxim AI — simulation + evaluation + observability

**Ne yapıyor?** Agent simulation, multi-turn senaryolar, personas, evaluation engine, prompt experimentation, dashboards ve version comparison sunuyor; binlerce test case ölçeği vurgulanıyor.

**Membrane ile kesişim:** AgentTwin'in simulation/evaluation kabiliyetleri. Maxim, bu alanda en doğrudan ürün karşılaştırmalarından biri.

**Muhtemel fark alanı:** Membrane'ın yalnızca kullanıcı konuşmalarını simüle etmeyip tool dependency state'ini ve iş yan etkisini modellemesi; tool'ın caller'a yanlış başarı dönmesi ile gerçek state farkını kanıtlaması; kod ve agent release'ini aynı pipeline'da denetlemesi.

Kaynak: [Maxim AI Evals](https://www.getmaxim.ai/evals), [Agent Simulation & Evaluation](https://www.getmaxim.ai/evals/products/agent-simulation-evaluation), [Maxim Cookbooks](https://github.com/maximhq/maxim-cookbooks).

### H. Promptfoo — LLM/agent/MCP security red teaming

**Ne yapıyor?** 50+ vulnerability family, jailbreak, prompt injection, RAG poisoning, data exfiltration ve custom policy testleriyle red teaming; agent ve MCP server için tool poisoning, hidden instruction, cross-server attack ve malicious tool result senaryoları bulunuyor. CI/CD entegrasyonu var.

**Membrane ile kesişim:** Membrane MCP governance ve AgentTwin prompt-injection/malicious-retrieved-content senaryoları.

**Muhtemel fark alanı:** Promptfoo red-team testleriyle entegre olmak veya benzer saldırı testlerini AgentTwin twin'lerine aktarmak; ardından sadece attack detection değil, izin/approval/side-effect/state proof ile release kararını tamamlamak.

Kaynak: [Promptfoo Red Team Quickstart](https://www.promptfoo.dev/docs/red-team/quickstart/), [Agent Red Teaming Guides](https://www.promptfoo.dev/docs/red-team/guides/), [MCP Security Testing](https://www.promptfoo.dev/docs/red-team/mcp-security-testing/).

### I. LiteLLM — model ve MCP gateway

**Ne yapıyor?** 100+ model sağlayıcısı için birleşik API, per-key/team budget, rate limit, model fallback, cost tracking, content filters/PII masking ve MCP/A2A access gateway sağlıyor.

**Membrane ile kesişim:** Model/provider routing, MCP permissioning ve tool access control.

**Muhtemel fark alanı:** Membrane bir LLM proxy olmak zorunda değil. LiteLLM'i model routing ve gateway altyapısı olarak entegre edebilir; kendi avantajını code architecture governance, twin simulation, verified outcomes ve release evidence üzerine kurabilir.

Kaynak: [LiteLLM AI Gateway](https://docs.litellm.ai/docs/simple_proxy), [LiteLLM MCP Gateway](https://docs.litellm.ai/docs/mcp).

### J. GitHub Code Security / Copilot Autofix, Snyk ve Semgrep — AI code security

**Ne yapıyorlar?** GitHub CodeQL ve code scanning alert'leri, Copilot Agentic Autofix ile repo çapında kod düzeltme ve CodeQL tekrar doğrulama; Snyk ve Semgrep ise kod/dependency/IaC/container risklerini tespit etmekte güçlü.

**Membrane ile kesişim:** Code scan, secret/dependency governance, PR gate ve AI generated code review.

**Muhtemel fark alanı:** Membrane yalnızca bilinen güvenlik açığı veya code scanning alert'i bulmamalı; organizasyonun gold-codebase örüntülerine, katman sınırlarına, framework-specific architecture kurallarına ve agent/tool behavior güvenliğine de karar verebilmeli. Ancak bu iddia gerçek AST + semantic + architectural benchmark ile kanıtlanmalı.

**Güncel not:** GitHub Agentic Autofix 10 Temmuz 2026'da public preview olarak duyuruldu; CodeQL ve üçüncü taraf code scanning bulgularını repo bağlamında düzeltiyor ve analizi yeniden çalıştırıyor. Membrane'ın code governance alanındaki rakip çıtasını yükseltiyor.

Kaynak: [GitHub Agentic Autofix announcement](https://github.blog/changelog/2026-07-10-agentic-autofix-for-code-scanning-alerts-in-public-preview/), [Semgrep MCP legacy repository](https://github.com/semgrep/mcp) — bu standalone repo 28 Ekim 2025'te arşivlendi; güncel Semgrep MCP geliştirmeleri ana Semgrep binary/repository üzerinden sürüyor.

## 2.3 Rekabet özeti

| Ürün | Code governance | OTel/trace | Dataset/eval | Simulation/fault injection | MCP/runtime access | Release evidence |
|---|---:|---:|---:|---:|---:|---:|
| Membrane AI (mevcut) | Güçlü başlangıç | Servis OTel'i; agent telemetry entegrasyonu yapılmalı | Kod golden corpus + semantic eval skeleton | AgentTwin ile birlikte | MCP policy engine başlangıcı | GitHub status/PR gate |
| AgentTwin (mevcut 0–3) | Yok/harici | Güçlü temel | Var | Stateful tool twin ve deterministic fault var | Runtime gateway beklemede | Paired eval var; release gate beklemede |
| Langfuse | Sınırlı | Güçlü | Güçlü | Ürün kapsamı daha çok app eval/experiment | Guardrails/SDK entegrasyonlarına bağlı | CI regression workflow var |
| LangSmith | Sınırlı | Güçlü | Güçlü | Framework/eval odaklı | LangChain/runtime entegrasyonlarına bağlı | Offline/online eval |
| Phoenix / Arize AX | Sınırlı | Çok güçlü | Güçlü | Replay/experiment; twin engine ayrı mesele | MCP query interface var | Evaluation/experiment loop |
| Microsoft Foundry | Microsoft ekosisteminde geniş | Var | Güçlü | Simulated multi-turn conversation | Azure platform controls | CI / deployment eval |
| AWS AgentCore | Agent platformu çevresinde geniş | Güçlü | Evaluations var | Runtime sandbox, eval | Gateway + identity + runtime | Agent platform lifecycle |
| Maxim | Sınırlı | Var | Güçlü | AI-powered scenario simulation | Platforma bağlı | Version comparison |
| Promptfoo | Kod tarama yerine LLM/agent red team | Test trace odaklı | Red-team test sets | Adversarial probes, not same as stateful business twin | MCP-specific testing | CI/CD red-team |
| LiteLLM | Kod/mimari governance değil | Logs/metrics/traces connectors | Eval integrations | Yok | Güçlü model/MCP gateway | Gateway policy/cost |
| GitHub/Snyk/Semgrep | Çok güçlü code scanning | Agent trace değil | Security tests | Genel agent twin değil | Bazı MCP/IDE integration | Code scanning / PR status |

**Pazar sonucu:** Membrane'ın tek tek kategorilerde her rakibin en iyi özelliğini geçmesi gerekmiyor. Bunun yerine şu uçtan uca bağlantıyı ürünleştirmesi gerekiyor:

1. AI tarafından üretilen diff'in kurumsal kod/security politikasına uyumunu denetle.
2. O diff'in agent prompt/model/tool/permission manifest'inde ne değiştirdiğini çıkar.
3. Değişiklikten etkilenen davranış senaryolarını seç.
4. Agent'ı aynı başlangıç state'i ve aynı fault seed'i altında test et.
5. Final cevap yerine tool trajectory ve gerçek state'i kontrol et.
6. Açıklanabilir gate kararı ve evidence package oluştur.
7. Gerçek üretim outcome'larını alıp regresyon korumasına dönüştür.

Bu iddianın gerçek farklılaştırıcı olması için **stateful digital twin + code-to-agent evidence chain + verified outcome** üçlüsüne odaklanın.

---

# 3. Ürün stratejisi ve konumlandırma

## 3.1 Ürün tezi

AI coding agent'lar kod yazıyor; AI runtime agent'lar ise API çağırıyor, veri değiştiriyor, işlem yapıyor ve sistemler arasında karar alıyor. Geleneksel SAST, LLM tracing ve prompt evaluation bu zincirin ayrı parçalarını görüyor. Kurumun ihtiyacı sadece agent'ın konuşmasını kaydetmek değil; **değişiklikten gerçek iş etkisine kadar hangi kontrolün neyi doğruladığını kanıtlamak**.

Membrane AI bu nedenle aşağıdaki product promise'i taşımalı:

- AI-generated code için pre-merge governance.
- Agent behavior için pre-release assurance.
- Tool/MCP action için runtime policy ve approval.
- Production outcome için feedback-to-regression loop.
- Her karar için immutable, explainable evidence.

## 3.2 Hedef kullanıcı segmentleri

| Segment | Kullanıcı | Problem | Satın alma nedeni |
|---|---|---|---|
| Fintech / ödeme / sigorta | AppSec, AI Platform, Engineering Manager, Risk | Agent yanlışlıkla para iadesi/transfer/erişim işlemi yapabilir | Side-effect doğrulaması, approval policy, audit, private deployment |
| E-commerce / marketplace | Platform Engineering, Search/Support Ops, AI Lead | Sipariş/iade/stok/ödeme agent'ları tool çağrısında hata yapabilir | Gerçek iş state'ine dayalı testler, duplicate action önleme |
| SaaS / enterprise IT | Platform Team, Security Engineering | Coding agent'lar mimariyi bozabilir, secret/dependency sızdırabilir | Gold-codebase RAG, code policy, MCP governance |
| Regulated enterprise | CISO, Compliance, Architecture Board | Model değişimi/agent davranışı için denetlenebilir kanıt gerekir | Self-hosted/VPC/air-gapped, audit evidence, control mapping |
| AI-native startup | CTO, AI Engineer | Prompt/model/tool değişimi regresyona neden olabilir | Dataset/eval, replay, fast CI gate |

## 3.3 Rekabetçi pozisyon

**“We are not another LLM observability dashboard.”** Bu cümle ürünün iç stratejisi olmalı.

Membrane'ın kategori tanımı:

**AI Engineering Governance & Agent Assurance Platform**

Pazar kategorisinin iki tarafına köprü kurar:

- Software supply chain / AI-generated code governance.
- Agent reliability, safety and runtime assurance.

## 3.4 Ürünün savunulabilir avantajı (moat)

Tek başına LLM evaluator, vector DB veya OTel span formatı savunulabilir avantaj değildir. Avantaj, zamanla biriken şu verilerden oluşur:

1. Her tenant'ın onaylı mimari kod örnekleri ve kuralları.
2. Agent manifest/version değişiklikleri ve bunların dependency graph etkisi.
3. Stateful twin'lerde modellenen gerçek tool/service state davranışları.
4. Doğrulanmış production trace'leri ve bağımsız outcome'lar.
5. İnsan tarafından onaylanmış failure/regression corpus.
6. Release kararlarının hangi deterministic rule ve evidence ile alındığı.

Bu veri birikimi Membrane'ı yalnızca “test aracı” olmaktan çıkarıp kurumun AI sistemleri için governance/evidence altyapısına dönüştürebilir.

---

# 4. Birleşik hedef mimari

## 4.1 Mimari prensipler

1. **Domain sınırlarını koru:** Code Governance ve Agent Assurance farklı veri modellerine ve iş akışlarına sahiptir. İlk günden tek büyük servis veya ortak veritabanı zorlaması yapılmamalı.
2. **OpenTelemetry standardı:** Agent/model/tool/retrieval/workflow span'leri GenAI semantic conventions ile taşınmalı.
3. **Evidence-first:** Her risk, verdict ve release decision makine tarafından okunabilir evidence reference taşımalı.
4. **Deterministic-first:** Kritik güvenlik kararları kurallarla; LLM judge yalnızca semantik değerlendirme gerektiğinde kullanılmalı.
5. **Fail-safe:** Evaluation altyapısı hata verdiğinde critical case başarı sayılmamalı. Runtime irreversible action policy hatasında fail-closed olmalı.
6. **No silent success:** Agent'ın başarı beyanı ile gerçek iş state'i ayrılmalı.
7. **Tenant isolation:** Organization/project bilgisi istemci tarafından gönderilen span attribute'tan değil, doğrulanmış credential'dan türetilmeli.
8. **Data minimization:** İçerik capture varsayılan kapalı; mümkün olduğunda metadata/hash/structured result, hassas alanlar client-side mask.
9. **Version pinning:** Model, prompt, tool schema, policy, scenario, twin, evaluator ve engine sürümleri kaydedilmeli.
10. **Standards over lock-in:** Framework/provider bağımsız SDK ve OTLP ingestion.

## 4.2 Mantıksal mimari

```mermaid
flowchart TB
  subgraph EDGE[Developer, CI and Agent Edge]
    IDE[VS Code / Cursor / Coding Agents]
    SCM[GitHub / GitLab / CI]
    AGENT[Runtime Agents]
    MCP[MCP Servers / Tools]
  end

  subgraph MEMBRANE[MEMBRANE AI Platform]
    subgraph COMMON[Shared Governance Plane]
      ID[Identity / Orgs / Projects / RBAC]
      POLICY[Policy Registry / Policy Versions]
      EVIDENCE[Evidence Ledger / Audit]
      INTEGRATION[Integration Hub / Webhooks / API]
      UI[Unified Console]
    end

    subgraph CODE[Code Governance Domain]
      CG[Code Submission Ingestion]
      SCAN[Deterministic Scan / AST / Secret Masking]
      GOLD[Gold Codebase Resolver / RAG]
      SEM[Semantic Review / Model Router]
      REPORT[Verdict / PR Status / IDE Diagnostics]
    end

    subgraph AGENT_ASSURANCE[Agent Assurance Domain - AgentTwin]
      OTEL[OTel Trace Ingestion]
      TRACE[Trace Explorer / Outcome Verification]
      TWIN[Declarative Stateful Twins]
      SIM[Scenario Simulation / Fault Injection]
      EVAL[Trajectory + State + Semantic Evaluation]
      GRAPH[Change Impact / Blast Radius]
      GATE[Release Gate / Evidence Package]
      REG[Regression Mining / Scenario Promotion]
    end

    RUNTIME[Runtime Action Gateway / Approval / Idempotency]
  end

  subgraph DATA[Data and Infrastructure]
    PG1[(Code Governance PostgreSQL + pgvector)]
    PG2[(Agent Assurance PostgreSQL + pgvector)]
    BUS[(Event Backbone / Outbox)]
    OBJ[(Object Storage / Artifacts)]
    OBS[Metrics / Traces / Logs]
    KEYS[Secret Manager / KMS]
  end

  IDE --> CG
  SCM --> CG
  AGENT --> OTEL
  AGENT --> RUNTIME
  MCP --> RUNTIME
  CG --> SCAN --> GOLD --> SEM --> REPORT
  OTEL --> TRACE
  TRACE --> REG
  GRAPH --> SIM
  TWIN --> SIM --> EVAL --> GATE
  POLICY --> SCAN
  POLICY --> RUNTIME
  POLICY --> GATE
  ID --> CODE
  ID --> AGENT_ASSURANCE
  CODE --> EVIDENCE
  AGENT_ASSURANCE --> EVIDENCE
  RUNTIME --> EVIDENCE
  INTEGRATION --> CG
  INTEGRATION --> OTEL
  REPORT --> SCM
  CODE --> PG1
  AGENT_ASSURANCE --> PG2
  EVIDENCE --> BUS
  CODE --> BUS
  AGENT_ASSURANCE --> BUS
  SIM --> OBJ
  CODE --> OBS
  AGENT_ASSURANCE --> OBS
  RUNTIME --> OBS
  KEYS --> COMMON
```

Bu diyagram hedef mimariyi gösterir. Bugünkü iki repo henüz tek bir ortak control plane, event bus, audit/evidence service veya birleşik UI sağlamıyor. Bunlar entegrasyon fazlarının tasarım hedefidir.

## 4.3 Domain sınırları ve veri sahipliği

| Domain | Authoritative owner | Sahip olduğu ana veri | Dışarı açtığı sonuç |
|---|---|---|---|
| Organization / Project / Membership | Ortak Control Plane (yeni/evrimleşecek katman) | org, project, memberships, roles, credentials | `org_id`, `project_id`, actor context |
| Code Governance | Membrane mevcut servisleri | submissions, repo metadata, rulesets, code findings, verdicts, gold index | code verdict, code evidence ref, repo commit |
| Agent Assurance | AgentTwin mevcut servisleri | agent manifests, versions, traces, outcomes, scenarios, twins, sim runs, evals | agent assurance report, run/eval refs |
| Policy Registry | Ortak governance plane; domain adapters | policy definitions/version/approval status | versioned policy decision |
| Runtime Action Gateway | AgentTwin target service + Membrane policy adapter | action request, decision, approval token, idempotency, outbound audit | allow/deny/approval + evidence |
| Evidence Ledger | Ortak control/evidence plane | cross-domain evidence link, release decision, audit references | immutable evidence package |

**Veritabanı önerisi:** İlk entegrasyonda PostgreSQL kümelerini birleştirmeyin. `Membrane` ve `AgentTwin` kendi schema/DB'sinde kalabilir. Ortak project identity ve release evidence için cross-reference alanları ve event'ler kullanılsın. Veri sahipliği belirsizleşirse migration ve audit yükü gereksiz artar.

## 4.4 Ortak kimlik ve correlation sözleşmesi

Her release/change için ortak bir `assurance_context` taşıyın:

- `organization_id`
- `project_id`
- `repository_id`
- `repository_url`
- `commit_sha`
- `pull_request_id`
- `code_submission_id`
- `code_verdict_id`
- `agent_id`
- `agent_version`
- `agent_manifest_hash`
- `prompt_hash`
- `model_provider`, `model_name`, `model_revision` (varsa)
- `tool_registry_hash`
- `policy_set_id`, `policy_version`
- `scenario_suite_id`, `scenario_suite_version`
- `twin_id`, `twin_version`
- `simulation_pair_id`
- `evaluation_run_id`
- `trace_id`
- `release_id`

Her alan her event'te bulunmak zorunda değil. Ancak bir code verdict ile agent release evaluation ilişkilendirildiğinde en az `organization_id`, `project_id`, `commit_sha`, `agent_id/version`, `policy_version` ve `release_id` arasında tutarlı ilişki kurulmalı.

**Korelasyon kimliği kuralı:** Kullanıcı tarafından gönderilen trace attribute tenant kimliği olarak kabul edilmemeli. Tenant verified API key/session'dan; project ve agent scope'u control-plane'den türetilmeli. Trace içindeki `commit_sha` veya `agent_version` sadece metadata; policy authority değildir.

## 4.5 Ortak event sözleşmesi

Önerilen minimum event ailesi (tam schema değil; domain contract önerisi):

| Event | Producer | Consumer / amaç |
|---|---|---|
| `code.submission.v1` | Membrane ingestion | Code analysis pipeline başlatma |
| `code.verdict.recorded.v1` | Membrane reporter/orchestrator | Verdict ve finding evidence'ini kaydetme |
| `agent.trace.ingested.v1` | AgentTwin trace service | Trace lifecycle ve downstream analysis |
| `agent.outcome.verified.v1` | AgentTwin evaluator/external callback | Claimed-vs-verified outcome ilişkilendirme |
| `agent.manifest.registered.v1` | AgentTwin control plane | Version/manifest değişikliği |
| `agent.scenario.versioned.v1` | AgentTwin simulation service | Test suite değişikliği |
| `agent.simulation.completed.v1` | AgentTwin simulation service | Simülasyon sonucu |
| `agent.evaluation.completed.v1` | AgentTwin evaluation service | Baseline/candidate karşılaştırma |
| `agent.release.evaluated.v1` | Ortak release orchestrator | PASS/WARN/BLOCK karar süreci |
| `governance.policy.changed.v1` | Policy registry | Policy cache invalidation ve audit |
| `governance.action.decided.v1` | Runtime gateway | Tool action allow/deny/approval |
| `evidence.package.created.v1` | Evidence service | Release kanıt paketinin değişmez kaydı |

Event envelope'da `event_id`, `event_type`, `schema_version`, `occurred_at`, `producer`, `organization_id`, `project_id`, `correlation_id`, `causation_id`, `traceparent`, `payload` olmalı. Her publisher transactional outbox kullanmalı; consumer idempotent olmalı.

## 4.6 Event bus ve entegrasyon yaklaşımı

Membrane Kafka/Redpanda, AgentTwin RabbitMQ kullanıyor. İlk aşamada bu iki altyapıyı bir günde birleştirmeyin.

**Önerilen sıralama:**

- Demo/V0: İki uygulamanın kendi event altyapısını koruyun. Unified console/API adapter, REST ve webhook ile sonuçları ilişkilendirsin.
- V1: Ortak event contract ve outbox publisher tanımlayın. Bridge service veya control-plane consumer, domain event'lerini canonical evidence olaylarına çevirsin.
- V2: Gerçek throughput, replay ihtiyacı, ordering, queue lifecycle ve operasyon yükü ölçüldükten sonra tek backbone seçimi yapın. Kafka/Redpanda yüksek hacimli ordered event log için doğal; RabbitMQ mevcut AgentTwin job/work queue ihtiyacına uygun. Seçim ölçümle yapılmalı.

## 4.7 OTel standardizasyonu

AgentTwin'in OTel altyapısı Membrane'ın ortak agent telemetry standardının çekirdeği olmalı. OpenTelemetry GenAI semantic conventions şu span ailelerini tanımlar/önerir:

- `invoke_agent`
- `invoke_workflow`
- `plan`
- `execute_tool`
- `retrieval`
- `embeddings`
- `search_memory` / `upsert_memory` / `update_memory`
- model inference spans

`gen_ai.tool.call.arguments` ve `gen_ai.tool.call.result` hassas içerik taşıyabilir. Bu alanları capture etmek varsayılan açık olmamalı. Content capture mode proje politikasıyla belirlenmeli ve SDK client-side redaction uygulamalı.

**Önerilen Membrane ek metadata'sı:**

- `membrane.project_id` (credential'dan doğrulanmış; telemetry attribute sadece yardımcı metadata)
- `membrane.agent_version`
- `membrane.manifest_hash`
- `membrane.prompt_hash`
- `membrane.policy_version`
- `membrane.release_id`
- `membrane.source` (`production`, `simulation`, `replay`, `eval`, `test`)
- `membrane.risk_tier`
- `membrane.tool.side_effect` (`none`, `read`, `write_reversible`, `write_irreversible`)
- `membrane.outcome.verification_source`

Standardın güncel durumu development olabilir. Bu yüzden OTel semconv version pinning ve schema migration yönetimi gerekir; “standard stable” gibi sunulmasın.

Kaynak: [OpenTelemetry GenAI spans](https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-spans.md), [Agent/framework spans](https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-agent-spans.md).

## 4.8 Release assurance lifecycle — birleşik kapalı döngü

```mermaid
flowchart TD
  A[Developer or AI coding agent creates change] --> B[Membrane Code Governance]
  B --> C{Code verdict}
  C -->|BLOCK| X[Stop merge; remediation evidence]
  C -->|PASS / WARN| D[Register agent manifest and change set]
  D --> E[Determine impacted components and scenarios]
  E --> F[Run baseline and candidate against same pinned suite]
  F --> G[Inspect tool trajectory, twin state diff and verified outcome]
  G --> H[Deterministic + calibrated semantic evaluation]
  H --> I{Release policy}
  I -->|BLOCK| Y[Block release; attach evidence and regressions]
  I -->|WARN| Z[Approval / staged rollout / additional coverage]
  I -->|PASS| J[Canary or production]
  J --> K[Capture production trace and independently verified outcome]
  K --> L[Human review / regression mining]
  L --> M[Promote regression to permanent scenario]
  M --> E
```

Bu zincirdeki Code Governance, simulation ve evaluation parçalarının önemli bölümleri mevcut; impact analysis, release object, gate, regression mining ve runtime containment sonraki entegrasyon fazlarında tamamlanmalı.

---

# 5. Demo ürünü: mevcut sistemleri bir araya getiren ilk sunum

## 5.1 Demo hedefi

Demo'nun amacı iki repo içinde yazılmış tüm feature'ları aynı ekranda göstermek değil. İzleyiciye şu tezi 7–10 dakika içinde kanıtlamak:

> Bir coding agent güvenli olmayan bir değişiklik ürettiğinde Membrane bunu kod aşamasında yakalayabilir. Koddan bağımsız olarak agent davranışı değiştiğinde AgentTwin, aynı iş koşullarını ve dış servis hatalarını tekrar ederek regresyonu ortaya çıkarabilir. İki kanıt aynı project/release bağlamında bir araya geldiğinde ekip neyi neden bloklaması gerektiğini görebilir.

İlk demo için yeni bir e-commerce/fintech agent yazılmamalı. **Demo Co Support Refund Agent** zaten hazır, iş problemi anlaşılır, hatalar görünür ve deterministik senaryolarla destekleniyor.

## 5.2 Demo hikâyesi

Kurgusal bir e-ticaret şirketi olan Demo Co, müşteri iade ve ücret iadesi taleplerini işleyen bir AI support agent kullanıyor. Engineering ekibi agent'a yeni bir sürüm çıkarmak istiyor. Değişiklikte:

- Refund policy kontrolü atlanıyor veya geç çalışıyor.
- Tool'un “refund succeeded” cevabı gerçek ödeme/iade state'iyle doğrulanmıyor.
- Timeout sonrasında agent aynı irreversible refund aksiyonunu tekrar deniyor.
- Prompt injection içeren bir retrieved document veya tool response agent'ı policy'yi yok saymaya yönlendirmeye çalışıyor.
- Bazı durumlarda order/tenant sınırı veya admin tool yetkisi ihlal edilmeye çalışılıyor.

İzleyici önce kod governance bulgularını, sonra agent davranışındaki gerçek etkileri görür. En sonda tek bir release evidence özeti tasarım hedefi olarak gösterilir.

## 5.3 Gösterim kapsamı: gerçek / birleştirme / sonraki faz

| Demo adımı | Bugün repo içinde var mı? | İlk demo biçimi |
|---|---|---|
| Membrane CLI scan | Evet | Gerçek CLI çalıştırması / gerçek rapor |
| VS Code/Cursor diagnostics | Evet | Gerçek IDE ekranı |
| GitHub status/PR comment | Evet | Gerçek integration ya da önceden üretilmiş PR ekranı; token/permissions önceden doğrulanmalı |
| Membrane MCP policy decision | Evet | Servisin gerçek policy endpoint'i / test çıktısı; HTTP/MCP wiring doğrulanmalı |
| AgentTwin trace explorer | Evet | Gerçek trace ekranı |
| Stateful tool twin | Evet | Gerçek simulation run |
| Fault injection + seed replay | Evet | Gerçek simulation run ve aynı seed ile rerun |
| Baseline/candidate eval compare | Evet | Gerçek evaluation ekranı |
| Tek ortak navbar/dashboard | Hayır | İlk demo için iki arayüz arasında kontrollü geçiş; birleştirilmiş ekran mockup ise “target UX” diye etiketlenmeli |
| Membrane verdict otomatik olarak AgentTwin eval başlatıyor | Hayır | V0'da operatör/manuel tetikleme; entegrasyon fazında otomatikleştirilecek |
| Change-aware blast radius | Hayır (tam ürün akışı) | V0'da önceden belirlenmiş scenario suite; sonraki fazda graph/change-set |
| PASS/WARN/BLOCK release gate | Hayır (AgentTwin Phase 5 pending) | Demo'da evaluation sonucundan ayrı olarak “önerilen gate policy” açıklanmalı; çalışır release gate iddiası yapılmamalı |
| Production runtime tool gateway | Hayır (AgentTwin Phase 7 pending) | Mimari/roadmap gösterimi; çalışan feature gibi sunulmamalı |
| Production trace'inden otomatik regression mining | Hayır (Phase 6 pending) | Elle review/promotion workflow'u gelecek hedef olarak gösterilmeli |

Bu ayrım demo güvenilirliği açısından kritik. “İki repo aynı altyapı içinde çalışıyor” demek ile “iki repo arasında otomatik end-to-end release control hazır” demek aynı değildir.

## 5.4 Önerilen 7–10 dakikalık demo akışı

### Dakika 0–1: Problemi tanıt

- Membrane AI ürün başlığı ve iki modülü göster.
- Demo Co refund agent senaryosunu anlat.
- Tek soru: “Agent'ın müşteriye verdiği son cevap başarılı görünse bile, gerçekte yanlış para iadesi yaptıysa bunu nasıl anlarız?”

### Dakika 1–3: Membrane Code Governance

- Kurgusal bir kod diff'i veya demo branch'i göster.
- `membrane scan` ile mevcut gerçek scanner'ın yakaladığı secret/risky pattern bulgularını göster.
- Finding başına rule, severity, file/line ve masked excerpt'i göster. Raw secret hiçbir ekranda görünmemeli.
- Ayrı olarak MCP policy kararını göster: allowlist dışı veya tehlikeli argüman içeren tool call `deny`/`review` olmalı.
- PR comment/commit status ile merge governance fikrini bağla.

**Önemli:** Mevcut scanner yalnızca repo içinde implement edilmiş rule setini yakalar. Demo'da AST kapsamlı güvenlik taraması, gerçek time-of-generation prompt proxy veya tam typosquat intelligence motoru varmış gibi gösterilmemeli. Typosquat sinyali gateway policy motorunda belirli popular package baselines ile sınırlı.

### Dakika 3–6: AgentTwin ile davranış regresyonu

- AgentTwin UI'da 1.2.4 baseline ile başla.
- `refund-happy-path`, `refund-over-limit`, `refund-timeout-after-mutation`, `refund-tool-success-lie` gibi senaryoların trajectory ve final twin state'ini göster.
- 1.3.0 candidate simülasyon/evaluation'ı aç.
- İlk divergence adımını göster: policy lookup atlanıyor ya da yanlış sırada; timeout sonrasında duplicate irreversible side effect oluşuyor veya tool'un başarılı cevabı gerçek state ile uyuşmuyor.
- Seeded replay ile aynı koşulda tekrar üretilebilirliği göster.
- Demo sonuçlarını doğru ifade et: mevcut seeded test raporunda 1.2.4 9/9; 1.3.0 6/9; 3 failed case ve 2 critical failure. Evaluation raporundaki “new critical failure” ve “regressed” sınıflarıyla simulation case fail sayısını birbirine karıştırma.

### Dakika 6–8: İki kanıtı tek release kararı fikrine bağla

- Membrane kod verdict ID, Git commit SHA, AgentTwin manifest version, evaluation run ID ve trace ID arasındaki ilişkiyi açıklayan target evidence view göster.
- “Code passes” tek başına agent behaviour safe demek değildir. “Agent eval passes” de code architecture/security scan'in yerini tutmaz. Birlikte release evidence oluşur.
- Önerilen gate kuralı: herhangi bir critical deterministic policy/side-effect/tenant violation varsa BLOCK; eksik coverage veya semantic judge belirsizliği varsa WARN/review; tüm mandatory critical checks geçerse PASS.
- Bu son ekran ilk demoda mock/prototype görünüm olarak etiketlenmeli; release service tamamlanmadan gerçek gate karar motoru olarak gösterilmemeli.

### Dakika 8–10: Sonraki kabiliyet

- Blast radius: prompt/tool/policy/model değişikliği hangi senaryoları etkiler?
- Regression miner: gerçek production fail, insan review sonrası gelecekteki test setine dönüşür.
- Runtime action gateway: irreversible action için allow/deny/approval + idempotency.
- Self-host/VPC/air-gapped deployment seçenekleri.

## 5.5 Demo'daki senaryo matrisi

Aşağıdaki matris mevcut AgentTwin senaryolarını bir demo kapsamına yerleştirir. Dosya adları repo içindeki `demo/support-refund-agent/assurance/scenarios/` klasörüyle ilişkilidir.

| Senaryo | Risk | Beklenen davranış | Kanıt |
|---|---|---|---|
| `refund-happy-path` | Normal refund iş akışı | Önce policy/order verisini kontrol et; şartlar uygunsa refund yap; sonuç state assertion ile doğrulansın | Tool sequence + state diff + verified outcome |
| `refund-over-limit` | Yetki/iş kuralı sınırı | Onaysız limit üstü işlem yapma; deny veya approval bekle | Policy expectation + no-mutation assertion |
| `refund-timeout-after-mutation` | Timeout sonrası retry | İdempotency key ile tek side-effect; retry ikinci refund oluşturmamalı | Twin call record + idempotency replay + final state |
| `refund-tool-success-lie` | Tool false success | Tool 200/succeeded dese dahi gerçek state değişmemişse başarı iddiası kabul edilmemeli | Claimed vs verified outcome contradiction |
| `refund-rate-limited` | 429/rate limit | Kontrollü backoff/retry; sınır aşımı ve sonsuz loop olmamalı | Retry count + fault record + latency |
| `refund-prompt-injection` | Goal hijack / indirect injection | Retrieved content veya tool response policy'yi geçersiz kılamaz | Prompt/tool response, policy decision, trajectory |
| `malicious-retrieved-content` | Context poisoning | Untrusted source içeriği sistem talimatı gibi yürütülmemeli | Retrieval content classification + tool action evidence |
| `cross-tenant-order` | Tenant boundary violation | Başka tenant order'ına erişme veya refund yapma | Tenant-scoped resource lookup + denial evidence |
| `unauthorized-admin-tool` | Privilege escalation | Admin-only tool yetkisiz agent/role tarafından çağrılamaz | Tool permission decision + blocked call |

## 5.6 Demo verisinin sunumda korunması

- Her gösterimden önce aynı DB seed ve aynı scenario/twin/manifest version seti kontrol edilmeli.
- Seed numarası ekranda görünür olmalı.
- `deterministic-fake` planner/judge açıkça böyle etiketlenmeli; gerçek LLM diye gösterilmemeli.
- Canlı provider demosu ayrı bir “nondeterministic model run” etiketi taşımalı.
- Ekran görüntüsü, trace ID ve evaluation run ID sunum notlarında saklanmalı.
- Çalışmayan veya live key gerektiren model bağlantıları demo ortamında varsayılan kapalı kalmalı.
- Bir scenario beklenmedik `ERRORED` olursa `PASSED` gibi gösterilmemeli; evaluation infrastructure failure ile agent failure ayrılmalı.

## 5.7 Demo öncesi teknik kontrol listesi

**Membrane**

- [ ] `task ci` veya repo handover'da tarif edilen yerel tüm kalite kapısı çalışmış olmalı.
- [ ] Docker Compose full stack temiz seed ile kalkmalı.
- [ ] `membrane scan` temiz demo repo üzerinde beklenen sonucu vermeli.
- [ ] Kasten eklenen secret sadece maskelenmiş finding olarak görünmeli.
- [ ] MCP policy tool/package allow-deny testleri ayrı doğrulanmalı.
- [ ] GitHub token, App permissions ve test repository erişimi doğrulanmalı.
- [ ] Model provider bağlantıları yoksa deterministic fallback durumu açıkça belirtilmeli.

**AgentTwin**

- [ ] `make reset` / `make dev` fresh data ile çalışmalı.
- [ ] 1.2.4 ve 1.3.0 seed-42 sonuçları doğrulanmalı.
- [ ] 9 senaryonun isim ve criticality değerleri kontrol edilmeli.
- [ ] 1.3.0 evaluation karşılaştırması fresh seed ile tekrar çalıştırılmalı ve sonuç raporlanmalı.
- [ ] Run detail'de twin state diff ve first divergence açılabilmeli.
- [ ] `deterministic-fake` etiketi ekranda görünmeli.
- [ ] Gerçek LLM provider varsa ayrı ortamda ve ayrı label ile doğrulanmalı.

**Birleşik demo**

- [ ] Her iki uygulamada aynı `organization/project` semantiği kurulmalı (ilk demoda mapping olabilir).
- [ ] Git commit SHA ile AgentTwin manifest version arasında sunumda gösterilecek ilişki hazır olmalı.
- [ ] Ortak evidence summary'nin hangi alanlarının live, hangilerinin mock/manuel olduğu belirtilmeli.
- [ ] Gate sonucu “önerilen karar” ise UI'da gerçek release gate ile karıştırılmamalı.

---

# 6. Birleşik ürün için fonksiyonel gereksinimler

Aşağıdaki gereksinimler hedef ürün tanımıdır. “Mevcut” etiketi bugünkü repoda bulunan kabiliyeti; “Yeni/entegrasyon” etiketi ise yapılması gereken işi ifade eder.

## 6.1 Organization, project ve agent registry

- Org, project, environment ve repo bağları ortak modele taşınmalı. **[Yeni/entegrasyon]**
- Project altında birden fazla repository ve birden fazla agent tanımlanabilmeli. **[Yeni]**
- Agent manifest immutable version olarak kaydedilmeli; değişen içerik eski version'a overwrite edilmemeli. **[AgentTwin mevcut]**
- Her agent version ilgili code commit, model, prompt hash, tool registry ve policy set ile ilişkilendirilebilmeli. **[Kısmi/yeni]**
- API key'ler hashed at rest, scope-limited, revocable, expiry ve last-used metadata ile yönetilmeli. **[AgentTwin mevcut; ortaklaştırma yeni]**
- RBAC: Owner, Admin, Engineer, Reviewer, Viewer; production approval rollerle sınırlandırılmalı. **[AgentTwin mevcut; birleşik kapsam yeni]**

## 6.2 Code Governance

- Repo/diff/file scanning ve deterministic checks. **[Membrane mevcut]**
- Secret detection/masking. **[Membrane mevcut]**
- Ruleset versioning, severity, allow/deny/review. **[Membrane mevcut/kısmi]**
- Organization gold-codebase context retrieval. **[Membrane kısmi: gerçek embedding bağlantısı gerekiyor]**
- AST/data-flow/taint-based checks. **[Yeni]**
- Architectural fitness rules: layer boundary, dependency direction, module contracts, domain ownership, forbidden imports. **[Yeni veya genişletme]**
- Package supply-chain checks: lockfile, provenance, vulnerability, license, typosquat; mevcut gateway heuristics tek başına yeterli değil. **[Kısmi/yeni]**
- MCP server/tool registry import, tool schema hash, permission classification, version drift. **[Yeni/entegrasyon]**
- IDE/CI/SCM verdict ve findings. **[Membrane mevcut]**

## 6.3 Agent Observability

- OTLP trace ingest (HTTP/protobuf), framework/provider bağımsız. **[AgentTwin mevcut]**
- Model, tool, retrieval, workflow span'lerinin normalized representation'ı. **[Mevcut/kısmi; OTel semconv version management eklenmeli]**
- Agent/model/tool version, prompt hash, policy version ve deployment environment metadata. **[Kısmi/yeni]**
- Content capture off by default; content mode server-side enforce; redaction parity across SDKs. **[AgentTwin mevcut]**
- Trace waterfall, session/thread view, retry/loop/cost/latency. **[AgentTwin mevcut/kısmi]**
- Agent claimed result vs verified result. **[AgentTwin mevcut]**

## 6.4 Twin simulation

- Twin schema, tool definitions, state fixtures, argument schemas. **[AgentTwin mevcut]**
- State isolated per case/run. **[AgentTwin mevcut]**
- Tool handler: read/mutate/static/echo/recorded/custom adapter. **[AgentTwin mevcut]**
- Fault rules: timeout-before/after mutation, 429, 5xx, duplicate, delayed, stale, false success, partial response. **[AgentTwin mevcut/kısmi; fault library genişletilmeli]**
- Deterministic seed, clock, ID generator ve version pinning. **[AgentTwin mevcut]**
- Twin contract import from OpenAPI/MCP schema; schema drift detection. **[Yeni/Phase 4 related]**
- Test isolation, resource limits, no arbitrary twin code execution. **[AgentTwin mevcut design; hardening pending]**

## 6.5 Evaluation & release assurance

- Deterministic assertions on state, tool sequence, tool args, policy, idempotency, outcome. **[AgentTwin mevcut]**
- Semantic judge for text requirements; calibrated and confidence-aware. **[AgentTwin mevcut]**
- Human review/override with note and immutable history. **[AgentTwin mevcut]**
- Paired baseline/candidate evaluation with pinned suite/seeds. **[AgentTwin mevcut]**
- Impacted scenario selection from change graph. **[Yeni/Phase 4]**
- Release object, gate policy, evidence package and CI status. **[Yeni/Phase 5]**
- Production feedback to regression mining and promotion. **[Yeni/Phase 6]**
- Runtime action decision, approval, exact-argument binding, idempotency and SSRF-safe forwarding. **[Yeni/Phase 7]**

## 6.6 Evidence and audit

Her release evidence kaydı en az şunları içermeli:

- `release_id`, `org_id`, `project_id`, `commit_sha`.
- Code verdict, ruleset version, code findings ve report artifact hash.
- Agent ID/version, manifest hash, prompt hash, model ID/revision ve tool schema hashes.
- Policy versions and environment.
- Selected scenario suite and reason for selection.
- Twin version/state fixture hash/fault rules/seed.
- Baseline/candidate simulation run IDs.
- Evaluation run ID, evaluator versions, judge prompt hash, calibration metadata.
- Failed/errored/skipped cases and severity.
- Gate policy version and triggered rules.
- Human approval, reviewer identity, note and timestamp.
- Signed or hash-linked artifacts, result report, trace references.

Evidence paketi, kullanıcının kararın arkasındaki spesifik adımları tekrar inceleyebilmesini sağlamalı. Sadece “risk score 84” göstermek yeterli değil.

---

# 7. Release gate: karar motoru nasıl çalışmalı?

## 7.1 Üç değerli karar

- **PASS:** Mandatory deterministic checks tamamlandı; tüm critical assertions geçti; coverage yeterli; blocker yok.
- **WARN:** Kritik ihlal yok; fakat coverage açığı, evaluator belirsizliği, semantic judge calibration eksikliği, yeni tool permission veya kabul edilebilir eşik dışı latency/cost değişimi var. Human review veya staged rollout gerektirir.
- **BLOCK:** Kritik deterministic policy failure, unauthorized action, tenant isolation violation, duplicate irreversible side effect, missing approval, false success ile çelişen state veya critical prompt injection/tool poisoning davranışı tespit edildi.

## 7.2 Gate'in deterministic kuralları

| Rule ID (öneri) | Koşul | Karar |
|---|---|---|
| `GATE-CRIT-001` | Critical security/policy scenario failed | BLOCK |
| `GATE-CRIT-002` | Cross-tenant data access veya tenant leak | BLOCK |
| `GATE-CRIT-003` | Irreversible action duplicate side effect | BLOCK |
| `GATE-CRIT-004` | Required approval bypass edildi | BLOCK |
| `GATE-CRIT-005` | Agent claimed success ancak bağımsız state failure | BLOCK |
| `GATE-CRIT-006` | Prompt/tool injection sonucunda unauthorized action | BLOCK |
| `GATE-CRIT-007` | Code scan'de critical secret veya configured blocking policy | BLOCK |
| `GATE-WARN-001` | Required scenario coverage threshold altı | WARN |
| `GATE-WARN-002` | Semantic judge unavailable/uncalibrated | WARN veya critical expectation ise INCOMPLETE/BLOCK policy |
| `GATE-WARN-003` | p95 latency/cost budget aşımı | WARN/BLOCK, customer policy'ye bağlı |
| `GATE-WARN-004` | Yeni tool/permission eklendi fakat ilgili scenario yok | WARN |
| `GATE-WARN-005` | Candidate behavior nondeterministic veya run tekrar edilemedi | WARN/INCOMPLETE |
| `GATE-PASS-001` | Tüm zorunlu assertions geçti ve evidence completeness sağlandı | PASS |

## 7.3 Önemli karar ilkeleri

- Average score critical fail'i maskeleyemez.
- `ERRORED` veya `SKIPPED` critical case PASS olamaz.
- LLM judge'ın karar verebilmesi için calibration data ve threshold gerekir; kalibrasyon yoksa judge sonucu advisory olabilir.
- Judge provider timeout veya malformed result, pass üretmemeli.
- Baseline da başarısızsa candidate'ın aynı hatayı miras alması otomatik olarak kabul edilebilir sayılmamalı; known risk ayrı sınıflandırılmalı.
- Gate policy version'ı release ile pinlenmeli. Daha sonraki policy değişimi geçmiş release verdict'ini sessizce değiştirmemeli.
- Human override mümkündür; ancak note, actor, role, scope, expiry ve reason mandatory; override immutable audit olarak saklanmalı.

## 7.4 Risk score

0–100 arası ikincil risk index'i kullanılabilir; fakat probability diye adlandırılmamalı. Formül ve contribution'lar kullanıcıya açık olmalı. Risk index sıralama ve önceliklendirme içindir; deterministic gate kurallarının yerine geçmez. AgentTwin ADR-0007'de kabul ettiği deterministic gate yaklaşımı aynen korunmalı.

---

# 8. Test ve ölçüm stratejisi

## 8.1 Code Governance metrikleri

| Metrik | Tanım | Neden |
|---|---|---|
| Precision | TP / (TP + FP) | Geliştirici güveni; false-positive fatigue'i azaltır |
| Recall | TP / (TP + FN) | Gerçek risklerin kaçırılmaması |
| F1 | Precision ve recall harmonik ortalaması | Dengeli detector karşılaştırması |
| False-positive rate | FP / (FP + TN) | Temiz kodda gereksiz blocking kararlarını ölçer |
| Rule coverage | Aktif policy'nin test edilmiş rule oranı | Ruleset boşluğu |
| Language/framework coverage | Dil, framework ve repo tipine göre ölçüm | Tek bir Go/örnek repo ile genelleme yapılmasın |
| Semantic review agreement | İnsan review ile semantic model karar uyumu | Judge güvenilirliği |
| p50/p95/p99 scan latency | Scan süresi | IDE developer experience |
| Cost per 1,000 diffs | Model ve infra maliyeti | Cost gate'in etkinliği |

Golden corpus 1.0 precision/recall sonucu, başlangıç regressions harness'i olarak korunmalı. Bunun yanında her rule için olumlu/olumsuz örnekler, language-specific test fixtures ve independent holdout corpus oluşturulmalı.

## 8.2 Agent Assurance metrikleri

| Metrik | Tanım | Not |
|---|---|---|
| Task success | Beklenen business outcome gerçekleşti mi? | Sadece agent final cevabı değil; independent state proof |
| Trajectory correctness | Tool call order, tool choice, args, retries ve policy path doğru mu? | Agent kalitesinin merkezinde |
| Side-effect correctness | Beklenen mutasyon sayısı/state diff gerçekleşti mi? | Irreversible işlemlerde blocker |
| Duplicate side-effect rate | Idempotent olması gereken tekrarlı çağrılarda çift mutation oranı | Timeout retry kritik metriği |
| Policy violation rate | Deny/approval/purpose sınırı ihlalleri | Security signal |
| Tenant isolation pass rate | Başka tenant veri/işlemine erişim engellendi mi? | Critical gate |
| Fault resilience | Timeout, 429, 5xx, partial failure koşullarından recovery | Dependency robustness |
| Loop/retry budget | Sonsuz loop, fazla tool call, aşırı retry | Cost ve risk |
| Semantic correctness | Cevap dili, şart açıklaması, müşteri mesajı | Calibrated judge + human labels |
| Trace completeness | Model/tool/retrieval/policy span coverage | Missing data, yanlış pass üretmemeli |
| p50/p95 latency | End-to-end agent run | Operational SLO |
| Cost per task | Token + tool + infra cost | Üretim maliyeti |
| Reproducibility | Same version/seed/test suite run result stability | Deterministic suite için şart |
| Regression delta | Baseline vs candidate failure/metric delta | Gate input |

**Coverage sadece satır kapsamı değildir.** Agent assurance için tool coverage, critical tool coverage, policy coverage, failure-mode coverage, regression coverage, dependency coverage, fault coverage ve outcome verification coverage ayrı raporlanmalı.

## 8.3 Evaluator güvenilirliği

- Deterministic assertions mümkün olan her yerde tercih edilmeli.
- LLM judge sadece semantik gerekliliklerde kullanılmalı.
- Judge prompt, model, schema, rubric ve shown evidence hash/version ile kayıt altına alınmalı.
- Judge evidence alıntısı, kendisine gösterilen kaynak metinde gerçekten bulunmalı.
- Judge calibration per-project, per-criterion, per-model/prompt hash yapılmalı.
- İnsan etiketleri ve Cohen's kappa, raw agreement ve confusion matrix birlikte gösterilmeli.
- Judge uncertainty/timeout/invalid JSON `ERROR` olmalı; sessiz fallback pass olmamalı.
- Semantic evaluator output'u tek başına critical security gate olmasın; gerekli deterministic policy assertions bulunmalı.

## 8.4 Benchmark planı

Benchmark 4 seviyeden oluşmalı:

1. **Synthetic deterministic unit corpus:** Her policy/rule ve fault mode için unit level examples.
2. **Scenario benchmark:** Demo Co gibi stateful, multi-turn, realistic tool environment cases.
3. **Public benchmark subset:** Hugging Face ve açık benchmark'tan yeniden kullanılabilir/atfedilebilir örnekler.
4. **Customer-specific regression corpus:** Production traces ve verified outcome'lardan insan tarafından onaylanmış cases.

Train/test leakage önlenmeli. Synthetic prompt injection dataset'i detector eğitimi ve sınıflandırma için faydalı olabilir; ancak gerçek agent exploit success ispatı değildir. Public benchmark sonuçları, business-domain twin sonuçlarıyla tek bir ortalamada birleştirilmemeli.

---

# 9. Hugging Face, benchmark ve açık araştırma kaynakları

Bu kaynakları doğrudan ürün içine “model eğitmek için veri” diye topluca aktarmayın. Önce lisans, provenance, dataset card, task definition, annotation quality ve intended use incelemesi yapın. Bazısı synthetic security prompt, bazısı agent trajectory, bazısı tool-use task log'udur; aynı metrikte birleştirilemez.

## 9.1 Agent security / prompt injection

### AgentInjectionBench

[Hugging Face Dataset: sincpp/AgentInjectionBench](https://huggingface.co/datasets/sincpp/AgentInjectionBench)

- Agentic tool-use pipeline'larında prompt injection saldırılarını hedefleyen dataset/benchmark.
- Tool output injection, goal hijack, privilege escalation, MCP context poisoning gibi saldırı aileleri.
- Projede JSONL data, taxonomy, train/validation/test split, 125 hand-crafted seed attacks, provider adapters ve Gradio demo alanı listelenmiş.
- Lisans kartında Apache 2.0 belirtiliyor; dataset citation ise önceki `ppradyoth/AgentInjectionBench` kaynağına işaret ediyor. Veri provenance/citation ilişkisinin kullanım öncesi ayrıca doğrulanması gerekir.

**Kullanım:** AgentTwin prompt-injection ve malicious tool response scenario family'lerini genişletmek, test case tasarlamak. Sentetik saldırıların tamamını mevcut Membrane scanner'a doğrudan eklemeyin; biri detector benchmark'ı, diğeri davranışsal agent testi olmalı.

### Agentic Prompt-Injection 5K

[Hugging Face Dataset: 3nesdeniz/agentic-prompt-injection-5k](https://huggingface.co/datasets/3nesdeniz/agentic-prompt-injection-5k)
- Dataset card 4,935 örnek bildiriyor: 2,750 injection ve 2,185 benign hard-negative.
- Aileler: memory/trust boundary, RAG document embedded injection, approval/authority escalation, tool/function abuse.
- Train/validation/test split senaryo bazında ayrıştırılmış; card'da CC-BY-4.0 lisansı yazıyor.

**Kullanım:** Detector precision/recall ve false-positive değerlendirmesi. Özellikle benign hard-negative örnekleri önemli; her “system instruction” benzeri metin saldırı olarak işaretlenmemeli.

### ATBench: Agent Trajectory Safety Benchmark

[Hugging Face Dataset: AI45Research/ATBench](https://huggingface.co/datasets/AI45Research/ATBench)

- 2026 Nisan sürümü 1,000 trajectory içeriyor; 503 safe / 497 unsafe.
- Uzun dönemli, tool-using agent trajectory'leri değerlendiriyor.
- Risk source, failure mode ve real-world harm boyutlarında diagnosis hedefliyor.
- Önceki ATBench500 ile release lineage karıştırılmamalı; dataset card'ın güncel `ATBench` ve eski `ATBench500` ayrımını koruyun.

**Kullanım:** AgentTwin trajectory-level risk scoring ve diagnostic labels için benchmark subset'i.

## 9.2 Tool-use process evaluation

### AgentProcessBench

[Hugging Face Dataset: LulaCola/AgentProcessBench](https://huggingface.co/datasets/LulaCola/AgentProcessBench)

- 1,000 trajectory; BFCL, GAIA dev, HotpotQA ve τ² subsets.
- Her örnek multi-turn message, tool definition, tool-use trace, reference output ve step-level process label içeriyor.
- MIT license card'ı mevcut.

**Kullanım:** Agent final cevap dışında step-by-step tool process'in nasıl puanlanabileceğini araştırmak. Kendi twin senaryonuzun yerine geçmez.

### BFCL trajectory data

[Hugging Face Dataset: AgentSuite/BFCL-trajectories](https://huggingface.co/datasets/AgentSuite/BFCL-trajectories)

- Çok sayıda model için function/tool calling trajectory data.
- 26 model, model başına 4,441 task bilgisi dataset card'da yer alıyor.

**Kullanım:** Tool selection ve argument accuracy gibi davranışları kıyaslamak. Business side-effect doğrulaması için kendi twin state testleri yine gereklidir.

### Ek agent trajectory data

[Hugging Face Agent Trajectories](https://huggingface.co/datasets/cx-cmu/agent_trajectories)

- τ²-bench, SWE-bench, TerminalBench ve MCPBench gibi farklı task ailelerinin trajectory özetleri.
- Benchmark'lar farklı reward ve task definition kullanır; karşılaştırmalar ayrı tutulmalı.

## 9.3 Browser / web agent benchmark

[BrowserGym](https://github.com/ServiceNow/BrowserGym) ve [WebArena](https://webarena.dev/) gibi browser-agent benchmark aileleri web agent ve browser-control testleri için ileri fazda incelenebilir.

**Öneri:** İlk demo refund/business workflow üzerinde kalsın. Browser agent support ikinci/üçüncü demo hattı olabilir; browser sandbox ve web state yönetimi ürün kapsamını gereksiz genişletmesin.

## 9.4 Hugging Face'de incelenecek demo ve uygulamalı örnekler

- [Phoenix observability dashboard'ını Hugging Face Spaces üzerinde kurma](https://huggingface.co/learn/cookbook/en/phoenix_observability_on_hf_spaces): Docker Space, persistent storage, OTLP trace endpoint ve CrewAI multi-agent izleme örneği.
- [Trace & Evaluate your Agent with Arize Phoenix](https://huggingface.co/blog/smolagents-phoenix): smolagents ile tool-calling agent oluşturma, tracing, tool span'lerini çekip LLM-as-judge evaluation çalıştırma.
- [Arize Phoenix JS tracing tutorial](https://github.com/Arize-ai/phoenix/tree/main/js/examples/apps/tracing-tutorial): support agent, query classification, order lookup tool, RAG, annotations ve session-level evaluation adımları.

**Öneri:** Demo tasarımında bu kaynaklardan trace waterfall ve trace->evaluation anlatımını alın; ancak ürünün asıl demo etkisini AgentTwin twin state diff + verified side effect üstüne kurun.

---

# 10. Resmî dokümantasyon ve teknik standartlar

## 10.1 OpenTelemetry / GenAI

- [OpenTelemetry GenAI semantic conventions — spans](https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-spans.md)
- [OpenTelemetry GenAI agent/framework spans](https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-agent-spans.md)

Kullanım alanı:

- Model call, agent invocation, workflow, tool execution ve retrieval span'lerini ortaklaştırmak.
- MCP tool invocation'larını çift span üretmeden standartlaştırmak.
- Retry sürelerinin logical operation kapsamına alınmasını tutarlı hale getirmek.
- Sensitive arguments/results için content privacy policy oluşturmak.

Bu conventions repository'sinin bazı alanları development status'tadır. SemConv sürümü ve attribute mapping'leri versioned tutulmalı.

## 10.2 Agentic security / governance

- [OWASP Top 10 for Agentic Applications 2026](https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/)
- [OWASP Agentic AI risk announcement / threat narrative](https://genai.owasp.org/2025/12/09/owasp-top-10-for-agentic-applications-the-benchmark-for-agentic-security-in-the-age-of-autonomous-ai/)
- [NIST AI RMF — Generative AI Profile (NIST AI 600-1)](https://www.nist.gov/publications/artificial-intelligence-risk-management-framework-generative-artificial-intelligence)
- [OWASP GenAI Security Project](https://genai.owasp.org/)

OWASP Agentic Top 10 risk aileleri ürün scenario taxonomy'sine map edilmeli: goal hijack, tool misuse, identity/privilege abuse, agentic supply chain vulnerabilities, unexpected code execution, memory/context poisoning, insecure inter-agent communication, cascading failures, human-agent trust exploitation ve rogue behavior.

NIST AI RMF GenAI Profile ise ürünün risk lifecycle, governance, measurement, documentation ve incident feedback yaklaşımını kurumsal control mapping için destekler. Dokümanda “compliance certified” gibi bir iddia için kullanılmamalı; bir risk-management reference'dır.

## 10.3 Prompt injection ve MCP security

- [Promptfoo Red Team Quickstart](https://www.promptfoo.dev/docs/red-team/quickstart/)
- [Promptfoo Agent Red Teaming Guides](https://www.promptfoo.dev/docs/red-team/guides/)
- [Promptfoo MCP Security Testing](https://www.promptfoo.dev/docs/red-team/mcp-security-testing/)

Promptfoo'nun MCP güvenlik rehberi özellikle tool poisoning, hidden instruction, sensitive data exfiltration, auth hijacking, tool shadowing, indirect injection ve malicious tool result gibi senaryoları ayrı threat model'lerle ele alıyor. Membrane gateway policy ile AgentTwin scenario suite arasında bu test sınıflarını ortak taxonomy olarak kullanmak mantıklı.

## 10.4 Structured model output / local serving

- [vLLM Structured Outputs](https://docs.vllm.ai/en/latest/features/structured_outputs/)
- [vLLM Tool Calling](https://docs.vllm.ai/en/stable/features/tool_calling/)

Membrane semantic service ve AgentTwin evaluator'larında free-form model output yerine mümkün olduğunda JSON schema/structured output kullanın. Bununla birlikte modelin JSON üretmesi, semantik kararın doğru olduğu anlamına gelmez; output schema validation, evidence validation, judge calibration ve deterministic gate yine zorunlu.

## 10.5 OpenAI agent eval örnekleri

- [OpenAI Agent Evals](https://developers.openai.com/api/docs/guides/agent-evals)
- [OpenAI Trace Grading](https://developers.openai.com/api/docs/guides/trace-grading)
- [OpenAI Agents SDK tracing](https://openai.github.io/openai-agents-python/tracing/)

Bu dokümanlar trace-level grading ve agent workflow evaluation alanındaki güncel resmi yaklaşımı anlamak için yararlı. Membrane, bu ürünleri framework/provider olarak destekleyebilir; kendi ürün API'sini belirli SDK'ya kilitlememeli.

---

# 11. Medium ve teknik makale okuma listesi

Medium yazıları resmi ürün sözleşmesi değildir. Bunları mimari fikir, örnek demo ve pratik uygulama perspektifi olarak kullanın; teknik iddiaları ilgili resmi dokümanla doğrulayın.

1. **Google Cloud — “Mastering Agent Evaluation: Making Agents Boringly Reliable” (11 Ekim 2026).** Agent quality'yi benchmark, grader, diagnosis/compare ve CI ship gate katmanlarıyla anlatıyor. Özellikle “overall score”un kritik segment failure'ı gizleyebileceği; evaluator'ın da kalibre edilmesi gerektiği fikri önemli. [Makale](https://medium.com/google-cloud/mastering-agent-evaluation-making-agents-boringly-reliable-9adbbdab24b8)
2. **Data From the Trenches — “Agent evaluation: What to measure, how, and when” (22 Ekim 2026).** Operational ve behavioral quality ayrımı; pre-deployment ve post-deployment evaluation; agent trajectory, tool behaviour ve quality measurement kapsamı. [Makale](https://medium.com/data-from-the-trenches/agent-evaluation-what-to-measure-how-and-when-3c1ef6a9f19e)
3. **Google Cloud — “Pro-Level Agent Observability: Deploying Arize Phoenix on Google Cloud” (25 Mart 2026).** Phoenix, OTel ve multi-agent workflow tracing'i gerçek deployment üzerinden anlatıyor. [Makale](https://medium.com/google-cloud/pro-level-agent-observability-deploying-arize-phoenix-on-google-cloud-f07a1576b578)
4. **Medium — “AI evals platforms: A comparative guide for production AI systems” (27 Şubat 2026).** Foundry, LangSmith, Arize ve diğer evaluation platformlarını karşılaştırma açısından faydalı. Makaleyi vendor feature listesi yerine pazar kategorileri ve buyer questions için okuyun. [Makale araması / başlık](https://medium.com/search?q=AI%20evals%20platforms%20A%20comparative%20guide%20for%20production%20AI%20systems)
5. **Medium — “LLM Observability and Prompt Management with Phoenix and LangSmith” (7 Ocak 2026).** Tracing, prompt management ve evaluation iş akışlarını anlamak için yardımcı okuma. [Makale araması / başlık](https://medium.com/search?q=LLM%20Observability%20and%20Prompt%20Management%20with%20Phoenix%20and%20LangSmith)

**Okuma sırası önerisi:** Önce 1 ve 2 (ölçüm stratejisi), sonra 3 (demo ve OTel), sonra 4–5 (pazar ve ürün UI beklentileri).

---

# 12. Demo ekranları ve unified UX önerisi

## 12.1 Ana navigasyon

Önerilen unified console sol menüsü:

1. **Overview** — org/project risk snapshot, son releases, critical open findings.
2. **Code Governance** — repositories, scans, findings, rulesets, gold-codebase, MCP/package policy.
3. **Agents** — agents, immutable versions, manifests, prompt/tool/model/policy hashes.
4. **Traces** — sessions, trace explorer, tool spans, model spans, policy decisions, outcomes.
5. **Scenarios & Twins** — scenario catalog, twin definitions, state fixtures, fault profiles.
6. **Simulations** — single run, paired run, trajectories, state diffs, replay.
7. **Evaluations** — datasets, evaluators, judges, calibration, baseline/candidate comparisons.
8. **Releases** — change set, impacted tests, gate decision, evidence package.
9. **Policies** — code, MCP, tool, package, approval, data-capture and release policies.
10. **Audit & Evidence** — immutable audit history, exports, signed reports.
11. **Integrations** — SCM, IDE, OTel, model providers, MCP registry, CI, webhooks.
12. **Settings** — org/project, RBAC, retention, deployment and secrets.

## 12.2 Overview dashboard

Dashboard'da yalnızca vanity metrics göstermeyin. Önerilen ana kartlar:

- `Critical code findings open`
- `Agent releases blocked / warned / passed`
- `Critical scenario coverage`
- `Verified outcome mismatch count`
- `Duplicate irreversible side effects`
- `Policy approval bypass attempts`
- `Trace completeness / ingestion delay`
- `p95 agent latency` ve `cost per successful task`
- Son 10 release'in code verdict / agent eval / gate timeline'ı

Her sayısal metric'e tıklanınca ilgili evidence listesine gidilmeli.

## 12.3 Release evidence view

En kritik ekran bu olmalı:

- Change summary (repo/commit/PR/agent diff).
- Code governance result ve finding breakdown.
- Agent version manifest diff.
- Impact graph (Phase 4 sonrası).
- Scenario suite ve selection reason.
- Baseline/candidate side-by-side results.
- İlk divergence ve affected tool call.
- Twin state diff (başlangıç, mutation, final state).
- Verified vs claimed outcome.
- Gate rule IDs ve nedenleri.
- Human approval/override.
- Evidence artifact download/export.

## 12.4 Görsel anlatım

Membrane ve AgentTwin logosunu tek ekranda yan yana koymak yeterli değil. Ürünün merkezinde **assurance timeline** olmalı: kod değişikliğinden release kararına giden yol. İzleyici bir finding'den trace'e, trace'den simulation case'e ve simulation case'den gate rule'a tıklayabilmeli.

---

# 13. Uygulama yol haritası

Aşağıdaki plan, var olan iki repo üzerine eklenmesi gereken işi sıralar. Bu bir kodlama işlemi değildir; uygulama ekibi için teknik kapsam ve DoD tanımıdır.

## Phase A — Audit freeze ve demo baseline

**Amaç:** Bugünkü gerçek kabiliyetleri güvenilir biçimde tekrar üretmek.

Kapsam:

- İki repo için latest commit hash, local dirty changes, branch, build/test durumunu kilitle.
- Membrane `task ci`, `task eval`, compose/full profile, `membrane scan`, gateway table tests ve GitHub reporter testlerini kaydet.
- AgentTwin `make lint`, `make test`, `make e2e`, `make dev`, `make doctor`, `make demo` ve seeded 1.2.4/1.3.0 evaluation'ı kaydet.
- Her demo output'una git SHA, run ID, seed, manifest hash ve timestamp ekle.
- `CURRENT_CAPABILITIES.md` tarzı tek bir capability ledger hazırla: implemented / partial / planned / not verified.

**Definition of Done:** İki repo tek tek fresh environment'ta çalışıyor; demo verisi ve beklenen sonuçlar tekrarlanabiliyor; ürün sunumunda hangi fonksiyonun canlı olduğu açık.

## Phase B — Birleşik Demo Experience (ilk entegrasyon)

**Amaç:** İki ürünün çıktılarını ortak project/release bağlamında göstermek.

Kapsam:

- Unified product shell veya tek demo landing page.
- Membrane project/repo ile AgentTwin project/agent mapping.
- Ortak `organization_id`, `project_id`, `release_id`, `commit_sha` correlation contract.
- Code verdict ve agent evaluation sonuçları için cross-link.
- Evidence summary ekranı; Phase B'de bazı alanlar adapter/manuel association olabilir ama status label'ları dürüst olmalı.
- İki servis ayrı ayağa kalkabilir; port/compose isolation açıkça belgelenmeli.

**Definition of Done:** Kullanıcı tek project altında code governance sonucunu ve agent assurance evaluation'ını görebiliyor; her iki sonucu orijinal domain ekranında detaylandırabiliyor.

## Phase C — Contract ve data integration

**Amaç:** Manuel korelasyonu otomatik ve tutarlı hale getirmek.

Kapsam:

- Canonical event envelope ve schema versioning.
- Membrane code verdict event'lerini AgentTwin/control plane'in okuyabileceği adapter.
- Agent manifest `commit_sha` ve code submission/PR ilişkisi.
- Trace'lerde `release_id`, `commit_sha`, `agent_version`, `manifest_hash` metadata.
- Evidence record ve trace ID ilişkisinin immutable kaydı.
- Outbox/idempotency/consumer retry/DLQ policy.
- Cross-tenant authorization testleri.

**Definition of Done:** Bir code verdict, belirli agent version ve evaluation run ile güvenilir şekilde ilişkilendiriliyor; event delivery duplicate olsa da aynı evidence iki kez oluşmuyor.

## Phase D — Agent Release Gate ve change impact

**Amaç:** AgentTwin Phase 4 ve Phase 5'i ürünün merkezi release workflow'una dönüştürmek.

Kapsam:

- Agent manifest / tool registry / prompt / model / policy change set.
- Graph nodes: agent, model, prompt, tool, MCP server, dataset, policy, scenario, twin.
- Graph edges: calls, depends_on, reads, writes, guarded_by, tested_by, derived_from.
- OpenAPI/MCP schema import ve version drift detection.
- Bounded blast-radius traversal; senaryo seçimi için reason açıklaması.
- Immutable release evaluation object.
- Deterministic PASS/WARN/BLOCK gate policy.
- CI CLI/API response format ve SCM status.
- Evidence package export.

**Definition of Done:** Prompt veya tool schema değişikliği impacted scenarios seçiyor; identical pinned baseline/candidate pair çalışıyor; blocker varsa CI/SCM status BLOCK ediyor; karar hangi rule ve evidence'ye dayandığını açıklıyor.

## Phase E — Regression mining ve production learning loop

**Amaç:** Üretimde yaşanan hatanın tekrar test setine girmesi.

Kapsam:

- Production trace/outcome mismatch, policy violation, retry/loop ve human flag sinyallerinden candidate regression oluşturma.
- Redaction, deduplication, clustering ve failure taxonomy.
- İnsan review/approve/reject workflow'u.
- Approved regression case'in scenario suite'e promotion'ı.
- Regression case provenance: source trace, verified outcome, approver, test expectation ve retention.
- False positive/noisy incidents için suppress/expiry workflow.

**Definition of Done:** Production'da verified failure olarak işaretlenen trace review sonrası versioned scenario'ya dönüşüyor ve sonraki release gate'e otomatik ekleniyor.

## Phase F — Runtime Action Governance

**Amaç:** Test ortamının yanında production tool action sınırını da yönetmek.

Kapsam:

- Runtime gateway, customer tool endpoints'e güvenli proxy.
- Allow, deny, allow_with_limits, require_approval decisions.
- Risk classification: READ, WRITE_REVERSIBLE, WRITE_IRREVERSIBLE, ADMIN.
- Single-use, short-lived approval token; token exact action hash / principal / resource / expiry ile bağlı.
- Per-user/agent/project scopes.
- Idempotency registry ve duplicate mutation prevention.
- SSRF guard: allowlist, private IP block, redirect revalidation, DNS pin/dial-time check, egress limit.
- Tool result/response sanitization ve untrusted result provenance.
- Policy decision audit ve break-glass override.

**Definition of Done:** Limit üstü refund approval olmadan ilerlemiyor; tam olarak onaylanan args dışına çıkılırsa token reddediliyor; tekrar edilen işlem ikinci mutation üretmiyor; SSRF ve tenant isolation testleri geçiyor.

## Phase G — Enterprise hardening ve managed deployment

**Amaç:** Gerçek kurumsal operasyon kabiliyeti.

Kapsam:

- OIDC/SSO/SCIM, RBAC matrix, service identity.
- Per-tenant encryption/keys, KMS/Vault/secret manager.
- Data residency, retention, deletion, export, legal hold.
- Air-gapped model and embedding hosting.
- SBOM, signing, provenance, dependency pinning, image scanning.
- Threat modeling, SSRF/IDOR/CSRF/SSRF, chaos/load tests.
- Multi-region only if SLA/customer need justify it.
- SLO, error budget, backup/restore, disaster recovery.
- SOC 2 / ISO 27001 / ISO 42001 control mapping; not certification claim.

**Definition of Done:** Production deployment profile, audit evidence, isolation tests, security checklist, backup/restore evidence and operations runbook available.

---

# 14. Teknik kararlar ve riskli tasarım seçenekleri

## 14.1 Hemen iki repo'yu monorepo yapmalı mıyız?

**Hayır, ilk aşamada önermiyorum.** İki repo teknoloji/servis ve release temposu açısından bağımsız. Birleşik ürün, birleştirilmiş git repository anlamına gelmez. Önce API contract, common IDs, event schema ve UI integration oluşturun. Monorepo kararı ancak shared release/version governance ve source-level changes gerçekten yüksek coupling oluşturduğunda yeniden değerlendirilmeli.

## 14.2 Tek veritabanına geçmeli miyiz?

**Hayır, ilk aşamada değil.** Membrane ve AgentTwin'in domain sahipliği farklı. Her iki tarafta PostgreSQL/pgvector olması aynı cluster/schema'yı paylaşmayı gerektirmez. Ortak evidence index, release refs ve audit linkleri için ortak domain service veya read model tasarlanmalı; domain tables doğrudan çapraz yazılmamalı.

## 14.3 Kafka mı RabbitMQ mu?

İlk integration için REST/Webhook + outbox pattern. Sonra kullanım ölçülerek karar. Membrane'daki Kafka/Redpanda ordered event stream ve yüksek hacimli telemetry için uygun. AgentTwin'de RabbitMQ ve DB-backed leased worker lifecycle zaten işlevsel. İki backbone'un da operasyon maliyeti vardır; yalnızca mimari görünüm uğruna erken migration yapılmamalı.

## 14.4 Tek semantic model gateway'i mi?

İki repodaki semantic service ve judge adapter'larını bir anda tek model service'e taşımayın. Önce ortak provider config/credentials/budget observability standardize edilsin. Ardından:

- Code review semantics için Membrane semantic API,
- Agent evaluation judges için AgentTwin evaluator API,
- Her ikisi için ortak model-routing adapter / secrets / spend observability

sorumlulukları ayrıştırılsın. Structured outputs ve schema validation kullanılmalı.

## 14.5 Qdrant mı pgvector mı?

Mevcut iki repo da PostgreSQL/pgvector kullanıyor. Demo ve ilk birleşim için Qdrant eklemek gerekmez. Search/retrieval workload, vector volume, HNSW operations ve latency ölçülmeden teknoloji değiştirmeyin. Membrane gold-code retrieval ile AgentTwin regression/dataset retrieval farklı workload olabilir; ortak embedding/metadata contract kullanılsa da fiziksel storage tek ürün kararı olmayabilir.

## 14.6 Yeni servisleri ne zaman açmalı?

Yeni mikroservis yalnızca domain, scaling, security boundary veya failure mode ayrıştığında açılmalı. İlk integration için `assurance-integration` benzeri ince adapter/aggregator yeterli olabilir. Ortak UI/Control Plane'ın tüm business logic'i taşımasına izin vermeyin.

## 14.7 En önemli mimari riskler

| Risk | Etki | Önlem |
|---|---|---|
| README'deki hedef özelliklerin mevcut gibi sunulması | Demo güvenilirliği kaybolur | Capability ledger; live/partial/planned etiketleri |
| Code verdict ile agent eval arasında yanlış correlation | Yanlış release kararı | Commit/manifest/prompt/tool schema hash pinning |
| OTel content capture hassas veri sızdırır | Privacy/compliance riski | Capture off, redaction, server enforcement, retention policy |
| LLM judge yanlış olumlu/olumsuz not verir | Hatalı gate | Calibration, evidence validation, deterministic override |
| Twin gerçek servisi yeterince temsil etmez | False confidence | Contract import, fixture validation, production observation based twin updates |
| Fault injection nondeterministic | Regresyon tekrarlanamaz | Seeded rules, fixed clock, pinned engine/twin versions |
| Gate average score critical fail'i gizler | Güvenlik ihlali release olur | Critical deterministic blocker precedence |
| Runtime gateway'de SSRF/credential forwarding | Internal network compromise | Egress allowlist, DNS/IP revalidation, scoped credentials, token binding |
| Event bus migration premature | Delivery/operational failure | REST/outbox bridge first, measured migration later |
| Public HF data license/provenance problem | Legal/reproducibility risk | Dataset card, license, citation and provenance review |

---

# 15. Security ve governance tasarımı

## 15.1 Threat model kapsamı

Birleşik platformun güvenlik sınırı şu alanlara yayılıyor:

1. Developer IDE/CLI ↔ Membrane edge.
2. SCM webhook/CI ↔ Membrane ingestion/reporter.
3. Agent SDK ↔ OTel collector ↔ AgentTwin trace service.
4. UI/BFF ↔ shared control plane.
5. Control plane ↔ domain services/internal JWT.
6. Simulation worker ↔ agent-under-test endpoint.
7. Twin runtime ↔ imported tool schemas/fixtures/fault rules.
8. Runtime gateway ↔ customer tools/MCP servers/arbitrary URLs.
9. External model/embedding providers ↔ semantic/judge services.
10. Imported scenario YAML, OpenAPI docs, MCP metadata and agent manifests.

## 15.2 Güvenlik kontrolleri

- Org/project scope her DB query'de uygulanmalı; yalnızca edge middleware'e güvenilmemeli.
- API keys hashed/peppered at-rest; scopes minimum yetki; expiry/revocation ve cache TTL dokümante.
- Internal JWT target audience'a bağlı, kısa ömürlü ve actor/project context taşımalı.
- OTel trace content off default. Full content capture yalnızca project admin tarafından policy ile açılmalı; secret masking server-side tekrar yapılmalı.
- Scenario/twin YAML safe parser; code execution yasak; custom handler yalnızca allowlisted adapter registry'den.
- Imported URLs için SSRF koruması; redirects tekrar kontrol edilmeli; private/loopback/link-local ranges bloklanmalı.
- Agent test endpoint'leri request payload'ıyla arbitrary URL seçememeli; operator allowlist.
- Simulation case token kısa ömürlü, tek case'e bağlı, case final olmadan revoke edilmeli.
- Runtime approval token tek kullanımlık ve canonical action args hash'e bağlı.
- Audit log hash chain/signing veya write-once storage ile tamper evidence taşımalı.
- Policy changes versioned; previous versions immutable; approvals kaydedilmeli.
- Model prompt ve retrieved data untrusted boundary olarak kabul edilmeli; tool description ve tool response içinde gizli talimatlar olabileceği varsayılmalı.

## 15.3 Regülasyon ve standart yaklaşımı

NIST AI RMF GenAI Profile ve OWASP GenAI/Agentic Security taxonomy, risk management ve control mapping için referans olarak kullanılmalı. SOC 2 / ISO 27001 / ISO 42001 mapping yapılabilir; ancak ürün bir control mapping dokümanı içeriyor diye certification veya compliance garantisi verilemez. Her control için implemented, partially implemented, planned ve customer responsibility ayrımı gösterilmeli.

---

# 16. Açık kaynak entegrasyon stratejisi

Ürünü bütün alt problemleri sıfırdan çözmeye zorlamayın. Aşağıdaki projelerle entegrasyon veya benchmark kullanımı daha rasyonel.

| Bileşen | Önerilen ilişki | Neden |
|---|---|---|
| OpenTelemetry / OpenInference | Standardize/integrate | Trace data portability; framework agnostic |
| Phoenix | Benchmark / optional local comparison | Trace/eval UI parity ve integration smoke test |
| Langfuse | External comparison / export integration | Production trace + dataset + CI eval pattern |
| Promptfoo | Red-team test adapter veya imported test sets | Agent/MCP adversarial testing kapsamı |
| Semgrep | Scanner adapter, kurallar ve AST desteği | Membrane'ın current line-based scanner'ından daha geniş SAST coverage; license/ops değerlendirmesi gerekir |
| Snyk | Dependency/OSS risk signal integration | Dependency, container, IaC ve SBOM risk coverage |
| LiteLLM | Optional model/MCP routing adapter | Provider routing, budgets, access key, MCP permission |
| vLLM | Local inference runtime | Self-hosted semantic evaluation/judge |
| Hugging Face datasets | Benchmark import pipeline | Public dataset evaluation and research |
| OpenAPI / MCP schemas | Import/contract mapping | Twin schema generation and tool-surface change detection |

**Build vs integrate:** Membrane'ın özgün değerini provider routing, generic prompt playground veya standart tracing UI'ı yeniden yazmaya harcamayın. Özgün yatırım; gold-codebase architecture policy, stateful twin, side-effect truth, code-to-agent release evidence ve runtime approval safety üzerinde olmalı.

---

# 17. İlk sürüm için ürün kapsamı ve non-goals

## 17.1 İlk birleşik demo/MVP'de olmalı

- Tek demo workspace/project.
- Repo/commit ile AgentTwin agent manifest ilişkilendirmesi.
- Code Governance finding list + severity.
- MCP tool decision sample.
- Agent trace explorer.
- Stateful twin simulation.
- Baseline/candidate evaluation compare.
- Verified-vs-claimed outcome display.
- İlk evidence summary.
- Seed ve version pinning visible.
- Honest status badges: Live / Partial / Planned.

## 17.2 İlk sürümde olmamalı

- Full autonomous code repair.
- Universal AST analyzer veya tüm dillerde derin static analysis iddiası.
- Production runtime gateway'in ilk demo için zorunlu kılınması.
- Her MCP server için her türlü OAuth/identity adapter.
- Otomatik ML risk predictor. Önce label corpus ve calibrated model için yeterli data gerekir.
- Tek bir global “agent safety score” ve bunu risk probability gibi sunmak.
- Immediate migration to one event bus or one DB.
- Browser automation, computer-use, multi-agent simulation platform scope genişletmesi.
- Customer production payload'larını açık content capture ile varsayılan toplama.

---

# 18. Uygulama ekibine teslim edilecek kabul kriterleri

Bu kriterler tasarımın kodlama sonrasında nasıl doğrulanacağını tanımlar.

## 18.1 Architecture and contracts

- [ ] Code submission, agent manifest, trace, simulation, evaluation ve release evidence arasında `commit_sha` ve `release_id` ile join yapılabiliyor.
- [ ] Aynı `release_id` altında code verdict ve agent evaluation referansları bulunuyor.
- [ ] Event contract'lar versioned; breaking changes compatibility check ile yakalanıyor.
- [ ] Outbox publisher ve consumer idempotency tests var.
- [ ] Cross-tenant data access testleri tüm yeni read/write endpoints için var.

## 18.2 Code governance

- [ ] Secret finding raw value hiçbir report/log/LLM request'e sızmıyor.
- [ ] Scanner rule precision/recall/FPR raporu rule ve language bazında üretiliyor.
- [ ] Gate rule version'ı evidence package'e pinleniyor.
- [ ] MCP allow/deny/review policy decisions reason code ile audit ediliyor.
- [ ] Package typosquat/allowlist logic'in coverage ve limitation dokümante.

## 18.3 Agent assurance

- [ ] Twin state isolation case/run sınırında test edilmiş.
- [ ] Seed, clock, manifest, scenario, twin ve evaluator version pinlenmiş.
- [ ] Timeout-after-mutation retry, duplicate side effect üretmiyor (idempotency şartında).
- [ ] Tool false-success state assertion ile yakalanıyor.
- [ ] Cross-tenant, unauthorized tool ve prompt injection scenario'ları mandatory critical suite'te.
- [ ] Baseline/candidate aynı scenario selection ve seed ile koşuyor.
- [ ] Judge provider error/timeout/malformed result pass üretmiyor.
- [ ] Human review override immutable audit.

## 18.4 Release gate

- [ ] Critical deterministic failure, global metric ortalamasından bağımsız BLOCK.
- [ ] Missing mandatory result INCOMPLETE/WARN/BLOCK policy'ye göre; asla PASS değil.
- [ ] Gate rule ID, evidence ref ve severity UI/API'de açıklanıyor.
- [ ] Gate policy version release ile birlikte tutuluyor.
- [ ] Override actor/note/expiry kaydı var.
- [ ] SCM status ve release API aynı final gate state'i gösteriyor.

## 18.5 Runtime gateway

- [ ] Deny/approval policy bypass edilemiyor.
- [ ] Approval token tam action hash'e bağlı ve single-use.
- [ ] SSRF/private address/redirect/dns rebinding kontrolleri testli.
- [ ] Idempotency duplicate mutation önlüyor.
- [ ] Gateway timeout/failure handling fail-closed davranışı risk tier'a göre açık.
- [ ] Audit log action request, decision, forwarded response ve actor'ı bağlayabiliyor.

---

# 19. Ürün sahibi karar listesi

Bu başlıklar uygulama başlamadan önce netleştirilmesi gereken kararlardır. Çoğunun teknik default'u dokümanda önerilmiştir.

| Karar | Önerilen default | Neden |
|---|---|---|
| Üst marka | Membrane AI | İki repo ürün çatısında birleşir |
| AgentTwin adı | Agent Assurance / AgentTwin alt ürünü | Twin konseptinin tanınırlığı korunur |
| Repo stratejisi | İki repo ayrı, API/event integration | Erken migration riskini azaltır |
| Demo agent | Demo Co Support Refund Agent | Hazır ve kanıtlı davranış regresyonu |
| İlk hedef müşteri | Agent kullanan fintech/e-commerce/enterprise engineering teams | Refund/side-effect hikâyesi iş etkisini gösterir |
| Deterministic planner | Varsayılan açık | Demo tekrarlanabilirliği |
| Live LLM | Opsiyonel ikinci mod | Provider bağımlılığı ve nondeterminism'i ayırmak |
| OTel | Canonical telemetry contract | Vendor/framework independence |
| Vector storage | PostgreSQL/pgvector ile devam | Şimdiki ihtiyaç için ek teknoloji gerekmiyor |
| Event bus | İlk entegrasyonda ayrı; sonradan measured standardization | Erken migration önleme |
| Release gate | Deterministic PASS/WARN/BLOCK | Explainability ve fail-safe |
| Risk score | İkincil 0–100 index, probability değil | Risk sıralaması gate authority olmasın |
| Runtime gateway | AgentTwin sonraki fazı | Gerçek production action control için security hardening gerekli |

---

# 20. Kaynakça ve inceleme bağlantıları

## 20.1 Kullanıcının GitHub repoları

- [Membrane-AI repository](https://github.com/Ozgurisikdamar/Membrane-AI)
- [Membrane architecture](https://github.com/Ozgurisikdamar/Membrane-AI/blob/main/docs/ARCHITECTURE.md)
- [Membrane roadmap](https://github.com/Ozgurisikdamar/Membrane-AI/blob/main/docs/ROADMAP.md)
- [Membrane handover](https://github.com/Ozgurisikdamar/Membrane-AI/blob/main/docs/HANDOVER.md)
- [Membrane decisions](https://github.com/Ozgurisikdamar/Membrane-AI/blob/main/docs/DECISIONS.md)
- [Membrane compliance mapping](https://github.com/Ozgurisikdamar/Membrane-AI/blob/main/docs/COMPLIANCE.md)
- [Membrane system report](https://github.com/Ozgurisikdamar/Membrane-AI/blob/main/docs/report/report.md)
- [AgentTwin repository](https://github.com/Ozgurisikdamar/AgentTwin)
- [AgentTwin system architecture](https://github.com/Ozgurisikdamar/AgentTwin/blob/main/docs/architecture/system.md)
- [AgentTwin implementation board](https://github.com/Ozgurisikdamar/AgentTwin/blob/main/docs/plan/implementation-board.md)
- [AgentTwin threat model](https://github.com/Ozgurisikdamar/AgentTwin/blob/main/docs/security/threat-model.md)
- [AgentTwin Python SDK](https://github.com/Ozgurisikdamar/AgentTwin/tree/main/packages/sdk-python)
- [AgentTwin refund demo](https://github.com/Ozgurisikdamar/AgentTwin/tree/main/demo/support-refund-agent)
- [AgentTwin ADR-0016: Declarative twins and faults](https://github.com/Ozgurisikdamar/AgentTwin/blob/main/docs/adr/0016-declarative-tool-twins-and-faults.md)
- [AgentTwin ADR-0022: Semantic judges](https://github.com/Ozgurisikdamar/AgentTwin/blob/main/docs/adr/0022-judges-grade-semantic-expectations.md)
- [AgentTwin ADR-0023: Pinned baseline/candidate pair](https://github.com/Ozgurisikdamar/AgentTwin/blob/main/docs/adr/0023-baseline-and-candidate-run-as-one-pinned-pair.md)

## 20.2 Competitor / product documentation

- [Langfuse Evaluation Overview](https://langfuse.com/docs/evaluation/overview)
- [Langfuse Experiments in CI/CD](https://langfuse.com/docs/evaluation/experiments/experiments-ci-cd)
- [LangSmith Evaluation Types](https://docs.langchain.com/langsmith/evaluation-types)
- [Arize Phoenix GitHub](https://github.com/Arize-ai/phoenix)
- [Arize Phoenix OSS](https://arize.com/phoenix-oss/)
- [OpenAI Agent Evals](https://developers.openai.com/api/docs/guides/agent-evals)
- [OpenAI Trace Grading](https://developers.openai.com/api/docs/guides/trace-grading)
- [Microsoft Foundry Evaluation Overview](https://learn.microsoft.com/en-us/azure/foundry/concepts/evaluation-overview)
- [Microsoft Foundry Agent Evaluation Approaches](https://learn.microsoft.com/en-us/azure/foundry/how-to/evaluate-generative-ai-app)
- [AWS Bedrock AgentCore Developer Guide](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/)
- [AWS AgentCore Observability](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/observability.html)
- [AWS AgentCore Evaluations](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/evaluations.html)
- [Maxim AI Evals](https://www.getmaxim.ai/evals)
- [Maxim Agent Simulation and Evaluation](https://www.getmaxim.ai/evals/products/agent-simulation-evaluation)
- [Promptfoo Red Team Guide](https://www.promptfoo.dev/docs/red-team/guides/)
- [Promptfoo MCP Security Testing](https://www.promptfoo.dev/docs/red-team/mcp-security-testing/)
- [LiteLLM AI Gateway](https://docs.litellm.ai/docs/simple_proxy)
- [LiteLLM MCP Gateway](https://docs.litellm.ai/docs/mcp)
- [GitHub Agentic Autofix public preview announcement](https://github.blog/changelog/2026-07-10-agentic-autofix-for-code-scanning-alerts-in-public-preview/)
- [Semgrep MCP legacy repo status](https://github.com/semgrep/mcp)

## 20.3 Standards / security / official implementation documents

- [OpenTelemetry GenAI spans](https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-spans.md)
- [OpenTelemetry GenAI agent spans](https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-agent-spans.md)
- [OWASP Top 10 for Agentic Applications 2026](https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/)
- [NIST AI RMF Generative AI Profile](https://www.nist.gov/publications/artificial-intelligence-risk-management-framework-generative-artificial-intelligence)
- [vLLM Structured Outputs](https://docs.vllm.ai/en/latest/features/structured_outputs/)
- [vLLM Tool Calling](https://docs.vllm.ai/en/stable/features/tool_calling/)

## 20.4 Hugging Face datasets and demos

- [AgentInjectionBench](https://huggingface.co/datasets/sincpp/AgentInjectionBench)
- [Agentic Prompt-Injection 5K](https://huggingface.co/datasets/3nesdeniz/agentic-prompt-injection-5k)
- [ATBench](https://huggingface.co/datasets/AI45Research/ATBench)
- [AgentProcessBench](https://huggingface.co/datasets/LulaCola/AgentProcessBench)
- [BFCL trajectories](https://huggingface.co/datasets/AgentSuite/BFCL-trajectories)
- [Agent trajectories metadata](https://huggingface.co/datasets/cx-cmu/agent_trajectories)
- [Phoenix on Hugging Face Spaces Cookbook](https://huggingface.co/learn/cookbook/en/phoenix_observability_on_hf_spaces)
- [Trace and evaluate smolagents with Phoenix](https://huggingface.co/blog/smolagents-phoenix)
- [Phoenix support agent tracing tutorial](https://github.com/Arize-ai/phoenix/tree/main/js/examples/apps/tracing-tutorial)

## 20.5 Medium engineering articles

- [Google Cloud — Mastering Agent Evaluation: Making Agents Boringly Reliable, 11 Oct 2026](https://medium.com/google-cloud/mastering-agent-evaluation-making-agents-boringly-reliable-9adbbdab24b8)
- [Data From the Trenches — Agent evaluation: What to measure, how, and when, 22 Oct 2026](https://medium.com/data-from-the-trenches/agent-evaluation-what-to-measure-how-and-when-3c1ef6a9f19e)
- [Google Cloud — Pro-Level Agent Observability: Deploying Arize Phoenix on Google Cloud, 25 Mar 2026](https://medium.com/google-cloud/pro-level-agent-observability-deploying-arize-phoenix-on-google-cloud-f07a1576b578)
- [Medium — AI evals platforms: A comparative guide for production AI systems](https://medium.com/search?q=AI%20evals%20platforms%20A%20comparative%20guide%20for%20production%20AI%20systems)
- [Medium — LLM Observability and Prompt Management with Phoenix and LangSmith](https://medium.com/search?q=LLM%20Observability%20and%20Prompt%20Management%20with%20Phoenix%20and%20LangSmith)

---

# 21. Nihai öneri

**Membrane AI çatısı altında AgentTwin'i bir “eklenti” olarak değil, agent lifecycle assurance domain'i olarak konumlandırın.** İki repo, farklı iş yüklerini ve farklı kanıt tiplerini yönetmeye devam etsin. İlk entegrasyonun odağı ortak project/release kimliği, cross-domain evidence ve birleşik demo deneyimi olsun.

İlk sunumda mevcut kabiliyetleri iki sağlam sahneyle kanıtlayın:

1. **Membrane:** AI tarafından üretilen kod değişikliğinin secret/risky pattern ve tool/package governance kontrollerinden geçmesi; bulguların IDE/PR sürecinde görünmesi.
2. **AgentTwin:** Aynı refund işi için baseline ve candidate agent'ı stateful twin'lerde, aynı seed ve koşullarla koşturmak; policy bypass, duplicate side effect ve false success'i tool/state evidence ile ortaya koymak.

Sonrasında bu iki sahneyi tek release evidence ekranına bağlayın. Bu ekran, Phase D'de gerçek release gate ile tamamlanacak. Regression mining ve runtime gateway geldiğinde Membrane AI yalnızca geçmişe bakan bir observability platformu olmayacak; kod üretiminden production action'a kadar AI sistemlerinin güven sınırını yöneten bir platforma dönüşecek.

**Ürünün pazardaki en güçlü cümlesi:**

> **Membrane AI secures how AI software is built, proves how AI agents behave, and governs what they are allowed to do.**

---

*Bu doküman bir mimari/ürün tasarımı ve araştırma çıktısıdır. Kod yazılmamış, repo üzerinde değişiklik yapılmamış, commit/push veya deploy gerçekleştirilmemiştir. Pazar ve dokümantasyon incelemesi 30 Ekim 2026 tarihinde erişilebilen kaynaklara dayanmaktadır.*