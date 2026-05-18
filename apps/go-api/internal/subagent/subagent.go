// Package subagent runs a small LLM tool-loop that reads the live call
// transcript and screening report, then patches the report in Redis.
package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/aigateway"
	"github.com/ayukumar261/hackathon/go-api/internal/supermemory"
	"github.com/ayukumar261/hackathon/go-api/internal/templates"
	"github.com/ayukumar261/hackathon/go-api/internal/transcripts"
	"github.com/google/uuid"
)

const (
	toolReadTranscript = "read_transcript"
	toolReadTemplate   = "read_template"
	toolPatchTemplate  = "patch_template"
	toolSearchResume   = "search_resume"
	defaultLoopMax     = 10
	transcriptTailN    = 50
	summaryMaxChars    = 200
)

type Runner struct {
	AI          aigateway.Streamer
	Transcripts *transcripts.Store
	Templates   *templates.RedisStore
	Supermemory *supermemory.Client
	LoopMax     int
}

func New(ai aigateway.Streamer, ts *transcripts.Store, tpl *templates.RedisStore, sm *supermemory.Client) *Runner {
	return &Runner{AI: ai, Transcripts: ts, Templates: tpl, Supermemory: sm}
}

func tools() []aigateway.Tool {
	return []aigateway.Tool{
		{
			Type: "function",
			Function: aigateway.ToolFunction{
				Name:        toolReadTranscript,
				Description: "Read the most recent entries from the live call transcript.",
				Parameters:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
			},
		},
		{
			Type: "function",
			Function: aigateway.ToolFunction{
				Name:        toolReadTemplate,
				Description: "Read the current screening report markdown.",
				Parameters:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
			},
		},
		{
			Type: "function",
			Function: aigateway.ToolFunction{
				Name:        toolSearchResume,
				Description: "Semantic search over the candidate's resume. Use to verify claims from the transcript or fill in details (past roles, dates, skills, education) before patching the report.",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Short natural-language query."}},"required":["query"],"additionalProperties":false}`),
			},
		},
		{
			Type: "function",
			Function: aigateway.ToolFunction{
				Name:        toolPatchTemplate,
				Description: "Replace the body of an EXISTING section in the screening report. `section` must exactly match (case-insensitive) the heading text of a section already present in the template — call `read_template` first to discover the available section headings. `new_content` is the new markdown body for that section. Unknown sections are rejected; this tool will never create a new section.",
				Parameters: json.RawMessage(`{
  "type":"object",
  "properties":{
    "section":{"type":"string","description":"Heading text of the section to replace (case-insensitive)."},
    "new_content":{"type":"string","description":"New markdown body for the section, excluding the heading line."}
  },
  "required":["section","new_content"],
  "additionalProperties":false
}`),
			},
		},
	}
}

func systemPrompt(task string) string {
	return `You are a sub-agent for a live phone screening. The parent agent just delegated this task to you:

  TASK: ` + task + `

Workflow:
1. ALWAYS call read_template FIRST to see the current report and the exact list of section headings that already exist.
2. Call read_transcript to gather the relevant transcript context.
3. Decide which EXISTING section heading the new information belongs under. The screening report has fixed sections (e.g. "5. Logistics" holds compensation, location, start date, and work authorization). Map the fact to the best existing section — do NOT invent new section names.
4. Call patch_template with section = the exact heading text from the template (case-insensitive match) and new_content = the rewritten body for that whole section (keep prior answers from that section intact; only add or update what changed).

Hard rules:
- Never pass a section name that is not already a heading in the template. The tool will reject unknown sections; if nothing fits, make no patch and reply with one sentence explaining why.
- Replace, do not append. Your new_content becomes the entire body of that section, so include the existing content you want to keep alongside the new answer.
- Edit one section per turn. Prefer the smallest change that captures the new information.

When done, reply with a single short sentence summarising what you changed (or why you made no change).`
}

func (r *Runner) Run(ctx context.Context, applicantID uuid.UUID, task string) (string, error) {
	if r == nil || r.AI == nil {
		return "", fmt.Errorf("subagent: not configured")
	}
	loopMax := r.LoopMax
	if loopMax <= 0 {
		loopMax = defaultLoopMax
	}
	sys := systemPrompt(task)
	msgs := []aigateway.Message{{Role: "user", Content: task}}
	tls := tools()

	for i := 0; i < loopMax; i++ {
		content, toolCalls, err := r.AI.Chat(ctx, sys, msgs, tls)
		if err != nil {
			return "", err
		}
		if len(toolCalls) == 0 {
			return truncate(strings.TrimSpace(content), summaryMaxChars), nil
		}
		msgs = append(msgs, aigateway.Message{Role: "assistant", Content: content, ToolCalls: toolCalls})
		for _, tc := range toolCalls {
			result := r.dispatchTool(ctx, applicantID, tc)
			msgs = append(msgs, aigateway.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result,
			})
		}
	}
	return "sub-agent stopped: tool loop cap reached", nil
}

func (r *Runner) dispatchTool(ctx context.Context, applicantID uuid.UUID, tc aigateway.ToolCall) string {
	switch tc.Function.Name {
	case toolReadTranscript:
		entries, err := r.Transcripts.ReadRecent(ctx, applicantID, transcriptTailN)
		if err != nil {
			return jsonErr(err)
		}
		var b strings.Builder
		for _, e := range entries {
			if e.Text == "" {
				continue
			}
			b.WriteString(e.Role)
			b.WriteString(": ")
			b.WriteString(e.Text)
			b.WriteByte('\n')
		}
		return jsonOK(map[string]string{"transcript": b.String()})
	case toolReadTemplate:
		v, _, err := r.Templates.Get(ctx, applicantID)
		if err != nil {
			return jsonErr(err)
		}
		return jsonOK(map[string]string{"template": v})
	case toolSearchResume:
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return jsonErr(err)
		}
		if r.Supermemory == nil || !r.Supermemory.Enabled() {
			return `{"error":"resume search not configured"}`
		}
		if r.Transcripts != nil {
			r.Transcripts.AppendResumeSearch(ctx, applicantID, args.Query)
		}
		sctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		snippet, err := r.Supermemory.Search(sctx, "applicant:"+applicantID.String(), args.Query, 5)
		if err != nil {
			return jsonErr(err)
		}
		if strings.TrimSpace(snippet) == "" {
			return jsonOK(map[string]string{"results": "", "note": "no relevant resume content found"})
		}
		return jsonOK(map[string]string{"results": snippet})
	case toolPatchTemplate:
		var args struct {
			Section    string `json:"section"`
			NewContent string `json:"new_content"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return jsonErr(err)
		}
		cur, _, err := r.Templates.Get(ctx, applicantID)
		if err != nil {
			return jsonErr(err)
		}
		patched, err := PatchSection(cur, args.Section, args.NewContent)
		if err != nil {
			return jsonErr(err)
		}
		if err := r.Templates.Set(ctx, applicantID, patched); err != nil {
			return jsonErr(err)
		}
		log.Printf("subagent: patched section %q for applicant %s", args.Section, applicantID)
		return jsonOK(map[string]string{"status": "ok", "section": args.Section})
	default:
		return `{"error":"unknown tool"}`
	}
}

func jsonOK(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"error":"marshal failed"}`
	}
	return string(b)
}

func jsonErr(err error) string {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	return string(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
