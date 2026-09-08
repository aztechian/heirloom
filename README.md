# Heirloom

> **Working name.** "Heirloom" is a placeholder until we settle on something better.

Heirloom turns an inherited pile of scanned photographs and documents into a searchable, annotated family archive — by doing the parts a machine is good at automatically, and asking living relatives for the parts only they know.

## The problem

When a family estate is digitized, the result is thousands of files with names like `scan_0417.jpg`. The images survive; the knowledge does not. Who is in the photo, when it was taken, where, and why it mattered lived in someone's memory, and that memory is finite. Meanwhile the people who *do* still remember are typically not the kind of users who will create an account, learn an interface, and methodically enter data into a genealogy tool.

Existing options fail at one end or the other. Genealogy platforms — Ancestry, FamilySearch, Gramps — are excellent at modeling family trees but treat photos as attachments, and the hosted ones want your collection on their servers. Self-hosted photo managers — Immich, PhotoPrism — do local face clustering well but have no concept of a person who died in 1961, a marriage, a source, or a claim that might be wrong. Neither is built for the actual workflow: an archivist with a scanner and a closing window of time to ask elderly relatives questions.

The distinguishing idea here is not photo management. It is **provenance** — treating every fact as a claim by someone, with evidence and a confirmation state, so the archive stays honest about what it actually knows. See "Build versus adopt" under open decisions, because whether that layer belongs on top of an existing photo pipeline is an unsettled and consequential question.

## Principles

These are the commitments that should survive contact with implementation. When a design decision is unclear, resolve it in favor of these.

**Nothing is true until a person says so.** Automated inference produces proposals. The distinction between "the model thinks 1952" and "Aunt Ruth confirmed 1952" must be visible everywhere in the UI and preserved in the data. An archive that quietly launders guesses into facts is worse than no archive, because the errors become permanent and uncorrectable.

**Measurement and interpretation are different subsystems.** Face detection and embedding are measurement: given the same image and model version, the same vectors. Language-model work is interpretation: non-reproducible, sometimes wrong, always a draft. Keeping these separate is what lets the archive be reprocessed without becoming untrustworthy.

**Human decisions are never overwritten by machine output.** Reprocessing may add proposals and may regroup machine clusters. It may not alter or discard a confirmation, a rejection, or an identification made by a person.

**Contributor friction is the binding constraint.** The scarce resource in this project is not compute or storage — it is the attention of elderly relatives. Every additional click, login, or explanation between a relative and answering a question costs real data that cannot be recovered later.

**Self-hostable with minimal infrastructure.** A person should be able to run their family's archive without a cloud account, a container orchestrator, or an ops background.

**The archive outlives the application.** All data must be exportable to open, documented formats. If this project is abandoned, no one's family history should be trapped inside it.

**Prefer asking a human over inferring, where a human is available and the question is cheap.** The model's best use is on questions no one alive can answer, and on volume no one alive has time for. This principle governs *how proposals are presented*, not whether inference runs — inference is what discovers which questions are worth asking.

## Who this is for

The primary user is a **family archivist**: one motivated person, moderately technical, who has custody of a collection and is doing the work. They run the instance, ingest material, review machine proposals, and decide what to ask whom.

The secondary users are **contributing relatives**, who arrive via a link, answer questions, and leave. They are not expected to understand the data model, and may be using a phone with a large font size. Their experience must work with zero onboarding.

A third audience is **other archivists self-hosting their own instance** — initially two colleagues with their own family collections. This is why configurability and collection scoping are structural from the start even though multi-tenant *operation* is a later phase: the code must never assume anything specific about one family's data.

## Scope

**Ships eventually, in roughly this order:** photo and document ingest with local face clustering; link-based relative contribution; cross-asset interpretation; hardened multi-tenant self-hosting. The Roadmap section below is the authoritative sequencing — this section defines the boundaries of the project, not the order of work.

**In scope**

- Scanned photographs and document scans amenable to OCR (letters, certificates, clippings that fit a scanner bed)
- Local face detection, embedding, and clustering
- People, relationships, places, and events, sufficient to describe who appears where and when
- Link-based relative contribution without accounts
- A configurable language-model backend, from a local model to a hosted API
- Collection scoping suitable for several unrelated families on one instance
- Export of all data in open formats

**Deferred, and acknowledged as real**

- Audio and video, including tape and film transfers. These exist in the collection and matter, but transcoding, storage volume, and time-based annotation are a distinct problem. The asset model should not make them impossible to add.
- Oversized or awkward physical items — full newspapers, framed diplomas — that cannot be flatbed scanned. Likely photographed rather than scanned, with different quality, lighting, and cropping needs.
- Handwriting recognition on photo backs, album captions, and inscriptions. Probably the highest-value unexploited source in most collections, since pencil on the reverse of a print often states exactly who and when. Requires two-sided capture and pairing; strong candidate for the first phase after the core is working.
- Narrative and print output: books, printed timelines, public galleries.
- Automated matching against external genealogy databases.

**Out of scope**

- Being a general-purpose photo manager. Heirloom will inevitably ingest, dedupe, thumbnail, and browse images, because it must. The line is that it does not aim to be where you keep your current photos, has no interest in albums-as-organization for its own sake, and adds no feature whose purpose is not ultimately to establish or record a fact about the past.
- Being a hosted commercial service.
- Being the authoritative genealogy system of record. Heirloom describes what the *media* shows and interoperates with genealogy tools rather than replacing them.

## Data model

The core idea is that the archive stores **assertions**, not fields.

An **asset** is an ingested source file, identified by content hash, with its original path, filename, and technical metadata. Source assets are immutable and are never modified in place. Metadata is published to sidecar files (XMP) rather than written into the original bytes, so that the content hash stays a stable identity. Optional embedding of metadata into *exported copies* is a separate, explicit operation.

**Derivatives** — thumbnails, web-sized renditions, deskewed or cropped versions, OCR text layers — are generated artifacts belonging to an asset. They are regenerable and are not themselves subjects of assertions. This matters more than it sounds: source scans are often 50–200 MB TIFFs, and the relative-facing UI is a phone.

A **subject** is anything an assertion can be about: an asset, a person, a place, an event, or an ordered pair of people (which is how relationships such as parentage or marriage are expressed).

An **assertion** is a claim about a subject. It carries the claim itself, the **author** (a named human contributor, an unverified token holder, or a specific model and version), a **basis** — the evidence, such as an EXIF field, an OCR span, a face match, or a relative's memory — a **polarity**, since "this face is *not* Aunt Ruth" is as essential as the positive claim, a timestamp, and a **status** of proposed, confirmed, rejected, or superseded. Machine assertions additionally carry a model-reported score; human assertions do not carry a numeric confidence, because a number attached to a person's memory is false precision. Where a human wants to hedge, they say so in words, and the archive records that.

**Face detections** are per-image measurements: a bounding box and an embedding. **Clusters** are a machine grouping of detections and are *disposable* — they may be regrouped on any reprocessing run. Therefore **identity is asserted against detections, not clusters.** Naming a cluster is a convenience that fans out into per-detection assertions, which is what makes those identifications survive re-clustering. Splitting and merging in the UI edits the assertions, not the clusters.

A **resolved view** sits over the assertion store and answers "what do we currently believe?" for search, timelines, and display. Resolution follows a stated precedence — a human confirmation outranks a machine proposal, a more recent human decision outranks an older one, and an explicit adjudication by the archivist outranks both. When relatives disagree irreconcilably, the resolved view reports the conflict rather than silently picking, and the archivist's adjudication is itself a recorded assertion with an author.

**Superseding** keeps the review queue usable. When reprocessing generates a proposal that replaces an earlier one from the same model family for the same subject, the earlier one is marked superseded rather than left to accumulate. Proposals contradicted by a human decision are closed, not re-offered.

Every subject and assertion carries a **collection** (the isolation boundary) and a **sensitivity** flag, which is what makes "exclude this from anything shared" enforceable rather than aspirational.

## Sharing and security

Relative contribution works through **scoped capability tokens**. A token is not a login; it is a narrow grant.

Because these links will be forwarded, pasted into group chats, and left in inboxes for years, the design assumes every token eventually leaks. Each token therefore grants access to a specific minimal slice — a set of questions, a group of detections to name — rather than the archive as a whole. Tokens expire, are individually revocable, never confer deletion, and never expose material outside their grant or flagged sensitive.

Authorship through a token is **unverified by construction.** A forwarded link means the person answering may not be the person invited, so contributions are recorded as "asserted by the holder of the token issued to Ruth," not as Ruth. A contributor may optionally name themselves. Presenting unverified provenance as verified would violate the project's central principle, so the data model distinguishes the two and the UI must too.

The archivist's own access is a separate concern: session-based operator authentication, scoped to the collections they own. This is the actual multi-tenant isolation risk and needs designing before a second family shares an instance.

Both media reads and token-scoped access proxy through the application rather than relying on presigned storage URLs, so that behavior is identical on object storage and on a plain filesystem, and so revocation is immediate.

This is the security-critical surface of the project and deserves its own threat model before the feature ships.

## Privacy and consent

The archive contains images of living people, including children, who have not consented to anything. Two positions, and one unresolved obligation.

**Face recognition runs inside the deployment.** Biometric templates do not leave the instance. This is a reason for local recognition libraries over a hosted vision service, alongside cost.

**The language-model backend is configurable, and what it implies must be stated plainly.** Pointed at a local model, no content leaves the machine. Pointed at a hosted API, content is sent to a third party. The software cannot make this choice for every family, but it can refuse to be vague: the active configuration must be visible to the archivist and disclosed on any page where a relative is asked to contribute.

**Local processing is not the same as legal compliance.** Under GDPR Article 9 and statutes such as Illinois BIPA and Texas CUBI, obligations attach to *generating and retaining* biometric data, not to transmitting it. Keeping embeddings on-premises reduces exposure; it does not by itself discharge consent or retention duties. A deletion and redaction path — for a relative who asks to be removed, and for the biometric templates specifically — is a requirement, not a nicety, and its absence from the current design is a known gap.

## Scale and target environment

Characterized rather than assumed, because several design questions collapse once the numbers are on the table.

**The collection is small.** Roughly 2,000 scans in hand, with 10,000 as the high end of the support target. At four faces per image that is 8,000 face detections expected and 40,000 at the ceiling. Fewer than 200 distinct individuals to identify. Scans are 600 dpi JPEGs averaging 1.3 MiB, so source material is about 2.5 GiB now and under 13 GiB at the ceiling — call it 40 GiB with derivatives.

**Compute is not a constraint anywhere in this range.** Brute-force pairwise comparison of every face embedding is 32 million comparisons at the expected size and 800 million at the ceiling — roughly 33 and 820 GFLOP respectively, which is seconds to a couple of minutes on a CPU. All embeddings fit in 82 MB of RAM at the ceiling. Nothing here justifies a vector index, an analytical query engine, or a GPU.

**Human review is the constraint, by two orders of magnitude.** Confirming 8,000 detections individually at three seconds each is about seven hours of one person's attention; at the ceiling it is thirty-three. This is the real budget, and it is what makes cluster-level naming with fan-out to per-detection assertions the highest-leverage thing in the interface rather than a convenience. The design target is confirming roughly 200 clusters and correcting the mistakes, not visiting every detection.

**The deployment target is a Kubernetes home lab** on older x86_64 server hardware. The deliverable is therefore a container image, and the single-binary framing is a property of that image rather than the distribution story. Supported build targets are **linux/amd64 and linux/arm64**, each built on a runner of its own architecture; there is no darwin or 32-bit arm target. That narrowness is deliberate and is what keeps cgo affordable — see the open decisions.

**Language-model access goes through a LiteLLM gateway**, which is OpenAI-compatible and already fronts an NVIDIA 3060. This is a good split: GPU scheduling for inference is the gateway's problem, not the application's, and it validates the adapter interface against a real deployment rather than a hypothetical one.

**GPU acceleration for face recognition is optional and probably unnecessary.** Detection and embedding over 2,000 images is a one-time batch measured in minutes on CPU. If it is ever built, note that it is not a runtime toggle: CUDA-linked inference means either a container image carrying multi-gigabyte CUDA libraries or a second image variant, gated on node feature labels. That cost is not worth paying for a job that finishes over lunch.

## Proposed architecture

A **single Go binary** serves the API and an embedded pre-built React frontend. Shipped as a container image for the Kubernetes target; the self-contained binary is what keeps that image small and dependency-free, and keeps a plain `docker run` viable for another archivist without a cluster.

**Model access sits behind one adapter interface**, whose primary implementation speaks the OpenAI-compatible chat API — covering local Ollama through most hosted providers. Note that some providers, AWS Bedrock among them, are not OpenAI-compatible natively and need a translating proxy. The commitment is to the interface, not to the wire format as a universal assumption.

**Face recognition is a distinct capability**, using purpose-built detection and embedding models with vector similarity, not an LLM. It is the least settled part of the design; see below.

**Storage sits behind an interface** with S3 and plain-filesystem implementations, so self-hosting requires no cloud account. Uploads and reads are proxied through the application, which keeps behavior identical across backends and makes revocation immediate. Presigned direct-to-S3 transfer is a planned per-deployment configuration option — an optional capability on that interface, not a second API shape.

**The HTTP contract is generated from an OpenAPI document.** `api/openapi.yaml` is the source of truth; Go models and strict server interfaces are generated from it and are not committed, so a handler that drifts from the spec fails the build. Two limits are worth stating rather than discovering: file-upload handlers have no typed request struct and so are not compiler-checked, and the generator emits no validation, meaning every `pattern`, bound, and enum in the document is a handler obligation rather than generated code. Paths are versioned under `/api/v1/` because outside self-hosters will depend on them. See [`api/README.md`](api/README.md).

**Heavy work runs as durable background jobs.** Ingest, derivative generation, OCR, embedding, and clustering must be resumable across restarts and must process incrementally, because the collection grows continuously as relatives contribute.

## Open decisions

Unresolved and deliberately recorded rather than papered over. Each should become a decision record before the relevant code is written.

**Build versus adopt.** Immich and PhotoPrism already solve local ingest, derivative generation, face detection, embedding, and clustering — which is to say, they already solve the piece identified below as this project's largest technical risk. The case for building fresh is that the assertion and provenance model is the actual product and is hard to retrofit onto a schema that assumes facts are fields; the case against is spending months rebuilding a solved pipeline. A serious evaluation of layering Heirloom's assertion store over an existing pipeline's API should happen before Go code is written, because it is the single decision with the largest effect on schedule.

**cgo — no longer a significant constraint.** `CGO_ENABLED=0` was being defended as though it were the distribution story, but the artifact is a container image, so it never was. What it genuinely buys is a small image and a binary with no runtime library dependencies. Those are worth keeping while they are free, and no longer worth refusing a required capability over.

**Supported build targets are linux/amd64 and linux/arm64.** That decision is what makes cgo cheap, and it is the load-bearing half of this entry. Cross-compiling cgo is where the pain lives — cross-toolchains, or QEMU emulation in CI — so both architectures build on runners of their own architecture instead. No cross-toolchain, no emulation, and enabling cgo becomes a one-line change rather than a CI rework. Deciding the architecture list first is what turned a schedule risk into a footnote; deciding it after picking an OCR engine would have been the expensive order.

`CGO_ENABLED=0` therefore stays as the default *because nothing needs it yet*, not as a constraint on what may be adopted. The database does not force the question: `modernc.org/sqlite` is a pure-Go driver, so SQLite is settled independently of cgo. OCR (no engine chosen) and face recognition may force it, and now they are free to.

**Face recognition implementation.** The strongest models are C++ or Python. Candidate paths are ONNX inference through a Go binding with a shared-library dependency, a sidecar vision worker, or pure-Go inference, currently immature. The scale numbers above take performance out of the decision entirely — CPU inference over 2,000 images is a batch job that finishes over lunch — so this should be chosen on build complexity and model quality alone. Prototype in Phase 0.

**System of record — settled: SQLite.** DuckDB was the initial thought, but it is an analytical engine optimized for scans over columns, whereas this workload is small point reads and writes against a normalized graph. SQLite fits better on shape, maturity, and migration tooling. Note that the concurrency argument frequently made here is bogus in both directions: SQLite is also single-writer, and the real write volume is a handful of relatives typing occasionally, which neither engine finds difficult.

DuckDB as an optional analytical companion is now **rejected**, not deferred. At 10,000 assets and 40,000 detections, the queries it would serve — "every asset between 1948 and 1955 with no confirmed location" — are a full table scan of a few tens of megabytes that SQLite answers immediately. A second engine would be carried for no measurable benefit.

The more interesting argument for DuckDB was not query speed but deployment topology: it reads and writes over S3 as well as a local filesystem, so choosing S3 might have meant needing no local disk at all — no PersistentVolume to declare on Kubernetes. That property turns out not to survive contact. DuckDB's ACID guarantees apply to a *local* database file; attaching a database over HTTP is read-only, and DuckLake, which does support S3-backed writes, needs a separate catalog database to coordinate them. Every path back to writable storage reintroduces the thing the choice was meant to avoid. So the PV-free property was never actually on offer, and DuckDB loses on the axis it was chosen for.

**The database is an external concern, with two dialects.** `internal/store` defines the boundary: SQLite is the default and the only implemented dialect, and Postgres is present as a recognized-but-unimplemented one. This is not fence-sitting. SQLite's cost is that the application pod holds durable state, which on Kubernetes means a PersistentVolume, `replicas: 1`, and `strategy: Recreate`. Postgres does not eliminate that storage — it *delegates* it to an operator that already knows how to back it up, which is the right trade for someone who runs Postgres anyway and the wrong one for someone who does not.

Postgres exists in the schema and in CI before it exists in the application because retrofitting a dialect is expensive and keeping a seam open is cheap. A year of SQLite-only development accumulates `INSERT OR IGNORE`, `strftime`, and `?` placeholders that all have to be found by hand later. The Postgres migrations therefore run against a real Postgres on every commit — an untested second dialect is just a comment. `store.Open` still refuses the dialect, because a driver was never the blocker; the query layer is. What has to be true before it can be turned on is enumerated in [`internal/store/README.md`](internal/store/README.md).

**Persistence uses `database/sql` and hand-written SQL, not an ORM** — With two dialects the differences should be visible in a diff rather than resolved invisibly by a driver abstraction, and the assertion store's precedence-resolution queries are exactly where an ORM stops helping. The cost is boilerplate and manual `Scan` targets, and it is accepted.

**Whatever is in the database must be reconstructible from object storage.** Source bytes are content-addressed, sidecars sit next to them, and face embeddings are stored as plain objects rather than as rows or index structures — 40,000 embeddings is 82 MB, so there is nothing to be gained from a vector store, and a great deal to be gained from being able to recover the archive by looking at the bucket. The database is a fast index over durable objects, not the only copy of anything. Human-authored assertions are the sole exception: they are not derived from the pixels, so they must be exported to sidecars too or they are genuinely at risk.

**Accepted input formats.** JPEG is the only format the existing scans use, so it is the only one the first phase must handle. TIFF, HEIC, and multi-page PDF each imply real work in the derivative pipeline and can wait for a file that needs them. RAW is out.

One archival concern belongs here rather than in a decision record, because it is one-way. At 600 dpi a 4×6 print is 8.6 megapixels, so 1.3 MiB works out to about 1.26 bits per pixel — and 0.87 for a 5×7. Visually lossless JPEG is usually 1.5–2.5. The current scans are therefore compressed harder than an archival master normally would be. It does not affect face recognition, where a face in a four-person group shot is still around 190 pixels wide against the ~112 an embedding model needs. But JPEG artifacts are permanent, and rescanning 2,000 physical items is not a thing anyone does twice. If the scanner's quality setting can be raised for whatever remains unscanned, that is close to free and worth doing before the next batch.

**Genealogy interoperability.** Whether to support GEDCOM import and export, and how much of it, affects the person and relationship model directly. Deciding after that model is built will be expensive, so this needs an answer during the first phase rather than after it.

**How relatives are actually reached.** Someone has to send the link, over some channel, with some reminder, and the system has to avoid asking five relatives the same question or asking one relative the same question twice. This is the core workflow of the second phase and is currently undesigned.

**When machine output is good enough to show a human.** The project's central principle is distrust of machine output, so there needs to be a bar and a metric — proposal acceptance rate, cluster purity, false-merge rate — rather than a vibe. Without it there is no way to tell whether a model change helped. The scale numbers give this teeth: with roughly seven hours of review budget against 8,000 detections, clustering quality translates directly into hours of a person's life. A false merge is worse than a missed one, because splitting a wrongly-merged cluster is manual work while an unmerged cluster is just another confirmation.

**Whether a local model can do the interpretation work.** Partly answered: a LiteLLM gateway fronting a 3060 is the primary configuration, so "local model" is the default path rather than a hypothetical. What remains unestablished is whether a model that fits in 12 GB of VRAM can do Phase 3's cross-asset reasoning usefully — grouping assets into events, inferring date ranges from evidence across images. That is a capability question, not a throughput one, and it needs a prototype on real photos rather than an estimate. The hosted path still needs a per-estate cost figure before it is offered as an alternative.

**Not yet addressed at all:** license, backup and restore format, and the migration path for the assertion store across schema changes.

## Roadmap

**Phase 0 — De-risk.** Decide build versus adopt. Prototype face detection and embedding, and choose the OCR engine — both now free to use cgo, so choose on model quality and build complexity rather than on toolchain purity. The collection is characterized, the system of record is settled, and the build targets are fixed; what remains here is not user-visible and all of it is load-bearing.

**Phase 1 — Working archive for one archivist.** Ingest photos and OCR-able scans, dedupe, derive thumbnails, extract existing metadata, detect and cluster faces, name people, and review proposals. Single collection, single operator, running end to end on the real pile of files. The assertion model, collection scoping, and pluggable model adapter are all built here, because retrofitting them is the expensive path — but multi-tenant operation is not yet exercised.

**Phase 2 — Relatives contribute.** Scoped share tokens, a zero-onboarding contribution interface built for a phone and an older user, distribution and reminder workflow, conflict surfacing, and question selection driven initially by simple gap enumeration — missing dates, unnamed detections — rather than by model reasoning.

**Phase 3 — Interpretation at scale.** Cross-asset reasoning: grouping assets into events, proposing date ranges from evidence, building timelines, drafting descriptions, and prioritizing which gaps are worth a person's time. This is where question selection gets smart; Phase 2 deliberately does not depend on it.

**Phase 4 — Multi-tenant self-hosting.** Operator authentication, hardened collection isolation, deletion and redaction, configuration, backup and restore, upgrade, and documentation good enough for another archivist to run an instance unaided.

**Later.** Handwriting recognition and two-sided scans; oversized items photographed rather than scanned; audio and video; narrative and print output; genealogy database matching.

## Status

Early implementation. The HTTP contract for collections and asset upload is defined in [`api/openapi.yaml`](api/openapi.yaml); no handlers exist yet, and the generator has not been run against the spec. This README remains the design document and should be revised as decisions are made rather than left to drift.

**There is no authentication yet.** An instance built from this repository currently exposes an unauthenticated upload endpoint. Bind to localhost and do not expose it publicly until operator authentication and upload limits land. See [`api/README.md`](api/README.md) for detail.
