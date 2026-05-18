package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ayukumar261/hackathon/go-api/internal/agentphone"
	"github.com/ayukumar261/hackathon/go-api/internal/aigateway"
	"github.com/ayukumar261/hackathon/go-api/internal/models"
	"github.com/ayukumar261/hackathon/go-api/internal/subagent"
	"github.com/ayukumar261/hackathon/go-api/internal/templates"
	"github.com/ayukumar261/hackathon/go-api/internal/transcripts"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgentPhoneWebhookHandler struct {
	DB          *gorm.DB
	Secret      string
	AI          aigateway.Streamer
	Transcripts *transcripts.Store
	Templates   *templates.RedisStore
	SubAgent    *subagent.Runner
	StreamMode  string // "sse" | "ndjson" | "off"
	MaxTurns    int
	ToolLoopMax int

	mu            sync.Mutex
	turnCounts    map[string]int
	conversations map[string]*conversationState
}

type conversationState struct {
	messages     []aigateway.Message
	lastUserText string
	lastUserAt   time.Time
}

const userDedupWindow = 3 * time.Second

type transcriptTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Text    string `json:"text"`
}

type webhookEvent struct {
	Event    string         `json:"event"`
	Type     string         `json:"type"`
	AgentID  string         `json:"agentId"`
	Metadata map[string]any `json:"metadata"`
	Data     struct {
		CallID     string          `json:"callId"`
		From       string          `json:"from"`
		To         string          `json:"to"`
		Transcript json.RawMessage `json:"transcript"`
		Summary    string          `json:"summary"`
		Metadata   map[string]any  `json:"metadata"`
	} `json:"data"`
	RecentHistory []transcriptTurn `json:"recentHistory"`
	Message       struct {
		Text    string `json:"text"`
		Content string `json:"content"`
	} `json:"message"`
}

type agentReply struct {
	Text   string `json:"text,omitempty"`
	Hangup bool   `json:"hangup,omitempty"`
	Action string `json:"action,omitempty"`
}

const (
	safeFallbackReply  = "Sorry, could you repeat that?"
	forcedClosingReply = "Thanks for your time — we'll be in touch with next steps."
	defaultMaxTurns    = 40
	defaultToolLoopMax = 5
)

func (h *AgentPhoneWebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}

	sig := firstNonEmpty(
		r.Header.Get("X-Webhook-Signature"),
		r.Header.Get("X-AgentPhone-Signature"),
		r.Header.Get("X-Signature"),
	)
	ts := r.Header.Get("X-Webhook-Timestamp")
	if !agentphone.VerifySignature(raw, sig, ts, h.Secret) {
		log.Printf("agentphone webhook: bad signature; ts=%s sig=%s body=%s", ts, sig, string(raw))
		writeError(w, http.StatusUnauthorized, "bad signature")
		return
	}

	var evt webhookEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	kind := evt.Event
	if kind == "" {
		kind = evt.Type
	}
	if kind == "" {
		kind = r.Header.Get("X-Webhook-Event")
	}

	switch kind {
	case "agent.message":
		log.Printf("agentphone webhook: agent.message metadata=%v data.metadata=%v to=%q from=%q body=%s",
			evt.Metadata, evt.Data.Metadata, evt.Data.To, evt.Data.From, string(raw))
		h.handleMessage(w, r, &evt)
	case "agent.call_ended":
		h.handleCallEnded(w, r, &evt)
	default:
		log.Printf("agentphone webhook: unhandled event %q body=%s", kind, string(raw))
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *AgentPhoneWebhookHandler) handleMessage(w http.ResponseWriter, r *http.Request, evt *webhookEvent) {
	applicant := h.resolveApplicant(evt)
	tmplContent := templates.DefaultContent
	if h.Templates != nil && applicant != nil {
		if c, ok, err := h.Templates.Get(r.Context(), applicant.ID); err == nil && ok && strings.TrimSpace(c) != "" {
			tmplContent = c
		}
	}
	system := buildSystemPrompt(applicant, tmplContent)

	var applicantID uuid.UUID
	if applicant != nil {
		applicantID = applicant.ID
	}

	userText := strings.TrimSpace(evt.Message.Text)
	if userText == "" {
		userText = strings.TrimSpace(evt.Message.Content)
	}
	if userText == "" && len(evt.RecentHistory) > 0 {
		last := evt.RecentHistory[len(evt.RecentHistory)-1]
		if last.Role == "" || last.Role == "user" || last.Role == "caller" {
			userText = strings.TrimSpace(firstNonEmpty(last.Content, last.Text))
		}
	}

	callID := evt.Data.CallID
	if h.isDuplicateUserText(callID, userText) {
		log.Printf("agentphone webhook: dedup near-duplicate user utterance callId=%s text=%q", callID, userText)
		h.emit(w, agentReply{})
		return
	}

	if h.Transcripts != nil && applicantID != uuid.Nil && userText != "" {
		h.Transcripts.AppendUserUtterance(r.Context(), applicantID, userText)
	}

	maxTurns := h.MaxTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}
	if turns := h.bumpTurn(callID); turns > maxTurns {
		log.Printf("agentphone webhook: turn cap hit for callId=%s turns=%d", callID, turns)
		h.emit(w, agentReply{Text: forcedClosingReply, Hangup: true})
		return
	}

	convo := h.loadConvo(callID, evt, userText)
	updated, reply, err := h.runAgentLoop(r.Context(), applicantID, system, convo)
	if err != nil {
		log.Printf("agentphone webhook: ai error: %v", err)
		h.emit(w, agentReply{Text: safeFallbackReply})
		return
	}
	h.saveConvo(callID, updated, userText)
	h.emit(w, reply)
}

func (h *AgentPhoneWebhookHandler) isDuplicateUserText(callID, text string) bool {
	if callID == "" || text == "" {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.conversations[callID]
	if st == nil || st.lastUserText == "" {
		return false
	}
	if time.Since(st.lastUserAt) > userDedupWindow {
		return false
	}
	a, b := st.lastUserText, text
	if a == b || strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		return true
	}
	return false
}

func (h *AgentPhoneWebhookHandler) loadConvo(callID string, evt *webhookEvent, userText string) []aigateway.Message {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conversations == nil {
		h.conversations = map[string]*conversationState{}
	}
	st := h.conversations[callID]
	if st != nil && len(st.messages) > 0 {
		out := append([]aigateway.Message{}, st.messages...)
		if userText != "" {
			out = append(out, aigateway.Message{Role: "user", Content: userText})
		}
		return out
	}
	// First turn for this call: seed from provider's RecentHistory (or fall back to the message).
	seeded := buildMessages(evt)
	if userText != "" {
		needAppend := true
		if len(seeded) > 0 {
			last := seeded[len(seeded)-1]
			if last.Role == "user" && strings.TrimSpace(last.Content) == userText {
				needAppend = false
			}
		}
		if needAppend {
			seeded = append(seeded, aigateway.Message{Role: "user", Content: userText})
		}
	}
	return seeded
}

func (h *AgentPhoneWebhookHandler) saveConvo(callID string, messages []aigateway.Message, userText string) {
	if callID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conversations == nil {
		h.conversations = map[string]*conversationState{}
	}
	st := h.conversations[callID]
	if st == nil {
		st = &conversationState{}
		h.conversations[callID] = st
	}
	st.messages = messages
	if userText != "" {
		st.lastUserText = userText
		st.lastUserAt = time.Now()
	}
}

func (h *AgentPhoneWebhookHandler) clearConvo(callID string) {
	if callID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conversations, callID)
}

func (h *AgentPhoneWebhookHandler) runAgentLoop(ctx context.Context, applicantID uuid.UUID, system string, msgs []aigateway.Message) ([]aigateway.Message, agentReply, error) {
	loopMax := h.ToolLoopMax
	if loopMax <= 0 {
		loopMax = defaultToolLoopMax
	}
	tools := agentphone.Tools()
	convo := append([]aigateway.Message{}, msgs...)

	for i := 0; i < loopMax; i++ {
		onToken := func(tok string) error {
			if h.Transcripts != nil && applicantID != uuid.Nil {
				h.Transcripts.AppendAssistantToken(ctx, applicantID, tok)
			}
			return nil
		}
		content, toolCalls, err := h.AI.StreamChatWithTools(ctx, system, convo, tools, onToken)
		if err != nil {
			return convo, agentReply{}, err
		}
		if h.Transcripts != nil && applicantID != uuid.Nil && strings.TrimSpace(content) != "" {
			h.Transcripts.AppendAssistantTurnEnd(ctx, applicantID, content)
		}
		if len(toolCalls) == 0 {
			text := strings.TrimSpace(content)
			if text == "" {
				text = safeFallbackReply
			}
			convo = append(convo, aigateway.Message{Role: "assistant", Content: content})
			return convo, agentReply{Text: text}, nil
		}

		// Record the assistant turn with its tool_calls so any tool messages we append are valid.
		convo = append(convo, aigateway.Message{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		})

		var terminalReply *agentReply
		for _, tc := range toolCalls {
			if tc.Function.Name == agentphone.ToolEndCall {
				args, perr := agentphone.ParseEndCall(tc.Function.Arguments)
				if perr != nil {
					log.Printf("agentphone webhook: end_call parse error: %v args=%q", perr, tc.Function.Arguments)
				}
				rep := buildEndCallReply(args)
				if h.Transcripts != nil && applicantID != uuid.Nil && rep.Text != "" {
					h.Transcripts.AppendAssistantToken(ctx, applicantID, rep.Text)
					h.Transcripts.AppendAssistantTurnEnd(ctx, applicantID, rep.Text)
				}
				terminalReply = &rep
				break
			}
			if tc.Function.Name == agentphone.ToolInvokeSubAgent {
				args, perr := agentphone.ParseInvokeSubAgent(tc.Function.Arguments)
				if perr != nil {
					log.Printf("agentphone webhook: invoke_subagent parse error: %v args=%q", perr, tc.Function.Arguments)
				}
				if h.Transcripts != nil && applicantID != uuid.Nil {
					h.Transcripts.AppendSubAgentInvoked(context.Background(), applicantID, args.Task)
				}
				go func(task string) {
					ctx2 := context.Background()
					log.Printf("agentphone webhook: sub-agent invoked task=%q", task)
					var summary string
					if h.SubAgent == nil {
						summary = "sub-agent not configured"
					} else {
						s, err := h.SubAgent.Run(ctx2, applicantID, task)
						if err != nil {
							summary = "error: " + err.Error()
						} else if s == "" {
							summary = "done"
						} else {
							summary = s
						}
					}
					log.Printf("agentphone webhook: sub-agent finished task=%q summary=%q", task, summary)
					if h.Transcripts != nil && applicantID != uuid.Nil {
						h.Transcripts.AppendSubAgentCompleted(ctx2, applicantID, summary)
					}
				}(args.Task)
				convo = append(convo, aigateway.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
					Content:    `{"status":"started","note":"sub-agent running asynchronously; do not wait for result"}`,
				})
				continue
			}
			// Unknown / non-terminal tool: return a stub result so the model can recover.
			convo = append(convo, aigateway.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    `{"error":"tool not implemented"}`,
			})
		}
		if terminalReply != nil {
			return convo, *terminalReply, nil
		}
	}
	log.Printf("agentphone webhook: tool loop cap exhausted")
	return convo, agentReply{Text: forcedClosingReply, Hangup: true}, nil
}

func buildEndCallReply(args agentphone.EndCallArgs) agentReply {
	closing := strings.TrimSpace(args.ClosingMessage)
	if closing == "" {
		return agentReply{Action: "hangup"}
	}
	return agentReply{Text: closing, Hangup: true}
}

func (h *AgentPhoneWebhookHandler) emit(w http.ResponseWriter, reply agentReply) {
	switch strings.ToLower(h.StreamMode) {
	case "sse":
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		b, _ := json.Marshal(reply)
		fmt.Fprintf(w, "data: %s\n\n", b)
		fmt.Fprint(w, "event: done\ndata: {}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	case "ndjson":
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		b, _ := json.Marshal(reply)
		fmt.Fprintf(w, "%s\n", b)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	default:
		writeJSON(w, http.StatusOK, reply)
	}
}

func (h *AgentPhoneWebhookHandler) bumpTurn(callID string) int {
	if callID == "" {
		return 1
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.turnCounts == nil {
		h.turnCounts = map[string]int{}
	}
	h.turnCounts[callID]++
	return h.turnCounts[callID]
}

func (h *AgentPhoneWebhookHandler) clearTurns(callID string) {
	if callID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.turnCounts, callID)
}

func (h *AgentPhoneWebhookHandler) handleCallEnded(w http.ResponseWriter, r *http.Request, evt *webhookEvent) {
	h.clearTurns(evt.Data.CallID)
	h.clearConvo(evt.Data.CallID)
	applicant := h.resolveApplicant(evt)
	if applicant == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.Transcripts != nil {
		h.Transcripts.AppendCallEnded(r.Context(), applicant.ID, "Call ended")
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"call_summary":  evt.Data.Summary,
		"call_ended_at": &now,
	}
	if err := h.DB.Model(&models.Applicant{}).Where("id = ?", applicant.ID).Updates(updates).Error; err != nil {
		log.Printf("agentphone webhook: persist transcript: %v", err)
	}
	if h.SubAgent != nil {
		task := "The call has ended. You MUST update the existing \"Agent Notes (post-call)\" section " +
			"of the screening report — this is not optional. Call read_template first to see the exact " +
			"bullet structure, then call patch_template ONCE with section=\"Agent Notes (post-call)\" " +
			"and new_content that keeps the same bullet fields, filling each one.\n\n" +
			"How to fill each field:\n" +
			"- Candidate name, Role, Date of call: copy from the report header / today's date.\n" +
			"- Compensation expectations, Earliest start date, Work authorization: extract verbatim from the transcript. Use [unknown] only if truly never discussed.\n" +
			"- Overall fit (1–5), Strengths, Concerns or gaps, Recommendation (Advance / Hold / Pass), Additional notes: YOU must form a judgment from the transcript. Do not leave these as [unknown] or placeholders — make a call based on the evidence, even if the signal is thin. For Recommendation, pick exactly one of Advance / Hold / Pass and mark it with [x].\n\n" +
			"Do NOT add new headings, sub-sections, or extra bullets — only fill in the existing fields.\n\n" +
			"Agent's own call summary: " + evt.Data.Summary
		applicantID := applicant.ID
		if h.Transcripts != nil {
			h.Transcripts.AppendSubAgentInvoked(context.Background(), applicantID, task)
		}
		go func() {
			ctx2 := context.Background()
			summary, err := h.SubAgent.Run(ctx2, applicantID, task)
			if err != nil {
				summary = "error: " + err.Error()
			} else if summary == "" {
				summary = "done"
			}
			log.Printf("agentphone webhook: post-call sub-agent finished summary=%q", summary)
			if h.Transcripts != nil {
				h.Transcripts.AppendSubAgentCompleted(ctx2, applicantID, summary)
			}
		}()
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AgentPhoneWebhookHandler) resolveApplicant(evt *webhookEvent) *models.Applicant {
	if h.DB == nil {
		return nil
	}
	candidates := []map[string]any{evt.Metadata, evt.Data.Metadata}
	for _, m := range candidates {
		if m == nil {
			continue
		}
		if id, ok := m["applicantId"].(string); ok && id != "" {
			if uid, err := uuid.Parse(id); err == nil {
				var a models.Applicant
				if err := h.DB.Preload("Position").First(&a, "id = ?", uid).Error; err == nil {
					return &a
				}
			}
		}
	}
	if evt.Data.To != "" {
		var a models.Applicant
		if err := h.DB.Preload("Position").Where("phone = ?", evt.Data.To).First(&a).Error; err == nil {
			return &a
		}
		digits := digitsOnly(evt.Data.To)
		if digits != "" {
			if err := h.DB.Preload("Position").
				Where("regexp_replace(phone, '[^0-9]', '', 'g') = ?", digits).
				First(&a).Error; err == nil {
				return &a
			}
		}
	}
	return nil
}

func digitsOnly(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b = append(b, s[i])
		}
	}
	return string(b)
}

func buildSystemPrompt(a *models.Applicant, template string) string {
	var b strings.Builder
	if a == nil {
		b.WriteString("You are a friendly phone screener for a job application. Keep replies short and conversational.\n")
	} else {
		b.WriteString("You are a friendly phone screener calling ")
		b.WriteString(a.Name)
		b.WriteString(" about a job application. Keep replies short, conversational, and one or two sentences. Ask one question at a time.\n")
		if a.Position.Title != "" {
			b.WriteString("Position: ")
			b.WriteString(a.Position.Title)
			b.WriteString("\n")
		}
		if a.Position.Description != "" {
			b.WriteString("Role description: ")
			b.WriteString(a.Position.Description)
			b.WriteString("\n")
		}
		if a.Resume != "" {
			b.WriteString("Candidate resume notes: ")
			b.WriteString(a.Resume)
			b.WriteString("\n")
		}
	}
	if strings.TrimSpace(template) != "" {
		b.WriteString("\nScreening guide (work through these in order, one question per turn, follow up naturally before moving on; skip a question if the candidate already answered it):\n")
		b.WriteString(template)
		b.WriteString("\n")
		b.WriteString("When all guide items are covered, call end_call with reason=screening_complete.\n")
	}
	b.WriteString("\nYou have two tools available:\n")
	b.WriteString("- invoke_subagent(task): delegates a background write to a smaller agent that reads the live transcript and patches the screening report. Returns immediately; do not wait.\n")
	b.WriteString("- end_call(reason, closing_message?): end the phone call.\n")
	b.WriteString("\nCall invoke_subagent after EVERY applicant turn that yields a fact relevant to the screening guide — compensation expectations, years of experience, specific skills, availability/start date, location, work authorization, willingness to relocate, etc. Emit the invoke_subagent tool call in the SAME turn as your spoken reply, with `task` describing exactly what to record. Example: if the applicant says \"I want five hundred thousand dollars a year\", call invoke_subagent with task=\"record applicant's total compensation expectation: $500,000/year\". If a single turn contains multiple distinct facts, emit one invoke_subagent call per fact. Do not wait for results; your spoken reply stays one short sentence and moves to the next guide question. Skip invoke_subagent only when the applicant turn is pure smalltalk, filler, or a clarifying question with no new information.\n")
	b.WriteString("\nAlso call invoke_subagent eagerly whenever the caller asks you to look something up, research a topic, think through options, draft something, or \"spin up\" / \"use\" sub-agents. If the caller explicitly asks for multiple sub-agents (e.g. \"use two sub agents to research X and Y\"), emit one invoke_subagent tool call per topic in the same turn. Do this even if the topic is off-topic from the screening — fire the sub-agents, then in your spoken reply briefly acknowledge it and steer back to the job questions in one short sentence. Never refuse to spin up sub-agents.\n")
	b.WriteString("\nCall end_call when: the screening questions are all answered (reason=screening_complete, include a brief closing_message thanking the candidate), the applicant asks to stop or declines (applicant_declined, with a polite closing_message), you detect voicemail or a long silence (voicemail, omit closing_message), or the conversation is hostile/off-topic and not recoverable (off_topic or safety).\n")
	return b.String()
}

func buildMessages(evt *webhookEvent) []aigateway.Message {
	if len(evt.RecentHistory) > 0 {
		out := make([]aigateway.Message, 0, len(evt.RecentHistory))
		for _, m := range evt.RecentHistory {
			content := m.Content
			if content == "" {
				content = m.Text
			}
			role := m.Role
			switch role {
			case "agent":
				role = "assistant"
			case "":
				role = "user"
			}
			out = append(out, aigateway.Message{Role: role, Content: content})
		}
		return out
	}
	text := evt.Message.Text
	if text == "" {
		text = evt.Message.Content
	}
	if text == "" {
		text = "(caller said nothing)"
	}
	return []aigateway.Message{{Role: "user", Content: text}}
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
