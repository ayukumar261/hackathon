# shortlist

An AI phone screener that calls job applicants, runs a tool-using voice agent on the line, and writes a structured screening report that fills in live during the call.

**Demo:** [youtube.com/watch?v=_DTNmJikm6s](https://www.youtube.com/watch?v=_DTNmJikm6s)

---

## What it does

Upload a position and a list of applicants (name, phone, resume PDF). The system dials each applicant and runs the screening with a voice agent. While the call is in progress, the agent uses tools to look things up and record what it learns:

- **`search_resume`** — semantic search over the applicant's resume (stored in Cloudflare R2, indexed via Supermemory). The agent calls this before asking follow-ups so questions reference real details from the resume.
- **`invoke_subagent`** — fire-and-forget delegation. After each applicant turn, the voice agent hands a short task ("record their years of React experience") to a parallel sub-agent that reads the transcript, looks up the resume if needed, and patches a single section of the screening report.
- **`end_call`** — hang up when screening is complete, the applicant declines, voicemail is detected, or for safety.

The screening report is a fixed Markdown template with predefined sections (logistics, experience, skills, etc.). The sub-agent can only **replace** an existing section — it can't invent new ones — which keeps the output structured and comparable across candidates.

Transcript and report both stream to the dashboard over Server-Sent Events, so you can watch the call unfold and the report fill in live.

## Architecture

```
   PSTN ─── AgentPhone ───►  go-api  ──── AI Gateway (LLM)
                                │
                                ├── voice-agent tools
                                │     • search_resume  ──► Supermemory
                                │     • invoke_subagent ──► sub-agent loop
                                │     • end_call
                                │
                                ├── sub-agent tools (own LLM loop)
                                │     • read_transcript  ◄── Redis
                                │     • read_template    ◄── Redis
                                │     • search_resume    ──► Supermemory
                                │     • patch_template   ──► Redis
                                │
                                ├── Postgres   (applicants, positions, sessions, users)
                                └── R2         (resume PDFs)

   browser ── SWR + SSE ───►  next-js-app  ◄── go-api  (transcript + template streams)
```

## Stack

A [Turborepo](https://turborepo.com) monorepo running locally via Docker Compose.

| Service       | Tech                                     | Port |
| ------------- | ---------------------------------------- | ---- |
| `next-js-app` | Next.js + SWR, Google OAuth, Sonner      | 3000 |
| `go-api`      | Go + Chi + GORM                          | 8080 |
| `postgres`    | Postgres 16 (alpine)                     | 5432 |
| `redis`       | Redis 7 (alpine) — transcripts + reports | 6379 |

External services: **AgentPhone** (telephony + voice), **Vercel AI Gateway** (LLM), **Supermemory** (resume embeddings), **Cloudflare R2** (resume storage), **Google OAuth** (user auth).

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (or Docker Engine + Compose v2)
- `make`

## Quick start

```sh
cp apps/go-api/.env.example apps/go-api/.env   # fill in API keys
make up                                        # build + start the stack
make migrate                                   # run GORM auto-migrations
```

- Frontend → http://localhost:3000
- API → http://localhost:8080

When you're done: `make down`. To wipe Postgres/Redis volumes too: `make clean`.

## Common commands

| Command        | What it does                                |
| -------------- | ------------------------------------------- |
| `make up`      | Build and start the full stack              |
| `make down`    | Stop and remove containers (data preserved) |
| `make clean`   | Stop containers and wipe volumes            |
| `make migrate` | Run GORM auto-migrations                    |
| `make logs`    | Tail logs from all services                 |
| `make restart` | Restart containers                          |

## Repo layout

```
apps/
  next-js-app/             # Dashboard: positions, applicants, live transcripts and reports
  go-api/
    cmd/
      api/                 # Main HTTP server
      migrate/             # GORM AutoMigrate
      seed/                # Local fixtures
      backfill-supermemory/# Re-index existing resumes into Supermemory
      reset-transcripts/   # Wipe Redis transcripts
      smoketest/           # Quick wiring check
    internal/
      handlers/            # HTTP + SSE handlers, AgentPhone webhook
      agentphone/          # Tool schemas exposed to the voice agent
      subagent/            # Sub-agent tool loop + markdown section patcher
      supermemory/         # Resume semantic search client
      aigateway/           # Vercel AI Gateway client (chat + tools)
      storage/             # R2 resume uploads
      transcripts/         # Redis-backed transcript store
      templates/           # Redis-backed screening report store
      oauth/               # Google OAuth
      db/, models/, middleware/, config/
```
