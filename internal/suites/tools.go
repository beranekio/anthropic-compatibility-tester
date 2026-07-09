package suites

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

const weatherToolName = "get_weather"

func weatherTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{{
		OfTool: &anthropic.ToolParam{
			Name:        weatherToolName,
			Description: anthropic.String("Get the current weather for a location."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "City and state, e.g. San Francisco, CA",
					},
				},
				Required: []string{"location"},
			},
		},
	}}
}

func requiredToolChoice() anthropic.ToolChoiceUnionParam {
	return anthropic.ToolChoiceUnionParam{
		OfAny: &anthropic.ToolChoiceAnyParam{},
	}
}

func autoToolChoice() anthropic.ToolChoiceUnionParam {
	return anthropic.ToolChoiceUnionParam{
		OfAuto: &anthropic.ToolChoiceAutoParam{},
	}
}

func validateWeatherToolInput(suite string, input json.RawMessage) error {
	if len(input) == 0 {
		return fail(suite, "tool_use missing input")
	}
	var parsed map[string]any
	if err := json.Unmarshal(input, &parsed); err != nil {
		return fail(suite, fmt.Sprintf("tool_use input is not valid JSON: %v", err))
	}
	location, ok := parsed["location"]
	if !ok {
		return fail(suite, `tool_use input missing required "location" field`)
	}
	locationStr, ok := location.(string)
	if !ok {
		return fail(suite, `"location" field is not a string`)
	}
	if strings.TrimSpace(locationStr) == "" {
		return fail(suite, `"location" field is empty`)
	}
	return nil
}