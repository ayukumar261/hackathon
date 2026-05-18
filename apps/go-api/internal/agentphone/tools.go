package agentphone

import (
	"encoding/json"

	"github.com/ayukumar261/hackathon/go-api/internal/aigateway"
)

const (
	ToolEndCall        = "end_call"
	ToolInvokeSubAgent = "invoke_subagent"
)

var invokeSubAgentParams = json.RawMessage(`{
  "type": "object",
  "properties": {
    "task": {
      "type": "string",
      "description": "Short description of what the sub-agent should update in the screening report (e.g. 'record applicant's years of React experience')."
    }
  },
  "required": ["task"],
  "additionalProperties": false
}`)

var endCallParams = json.RawMessage(`{
  "type": "object",
  "properties": {
    "reason": {
      "type": "string",
      "enum": ["screening_complete", "applicant_declined", "voicemail", "off_topic", "safety"],
      "description": "Why the call is ending."
    },
    "closing_message": {
      "type": "string",
      "description": "Optional short line spoken to the caller before hangup. Omit to hang up silently (use only for voicemail or safety)."
    }
  },
  "required": ["reason"],
  "additionalProperties": false
}`)

func Tools() []aigateway.Tool {
	return []aigateway.Tool{
		{
			Type: "function",
			Function: aigateway.ToolFunction{
				Name:        ToolEndCall,
				Description: "End the phone call. Use when the screening is complete, the applicant asks to end, voicemail is detected, the conversation is off-topic, or for safety reasons.",
				Parameters:  endCallParams,
			},
		},
		{
			Type: "function",
			Function: aigateway.ToolFunction{
				Name:        ToolInvokeSubAgent,
				Description: "Record a fact from the live call into the screening report. Call this after every applicant turn that contains screening-relevant information (compensation, experience, skills, availability, etc.). Returns immediately; do not wait for the result.",
				Parameters:  invokeSubAgentParams,
			},
		},
	}
}

type EndCallArgs struct {
	Reason         string `json:"reason"`
	ClosingMessage string `json:"closing_message"`
}

func ParseEndCall(arguments string) (EndCallArgs, error) {
	var a EndCallArgs
	if arguments == "" {
		return a, nil
	}
	err := json.Unmarshal([]byte(arguments), &a)
	return a, err
}

type InvokeSubAgentArgs struct {
	Task string `json:"task"`
}

func ParseInvokeSubAgent(arguments string) (InvokeSubAgentArgs, error) {
	var a InvokeSubAgentArgs
	if arguments == "" {
		return a, nil
	}
	err := json.Unmarshal([]byte(arguments), &a)
	return a, err
}
