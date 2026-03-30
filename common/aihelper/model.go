package aihelper

import (
	"GopherAI/common/websearch"
	appconfig "GopherAI/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type StreamCallback func(msg string)

type ChatOptions struct {
	EnableWebSearch bool
}

type AIModel interface {
	GenerateResponse(ctx context.Context, messages []*schema.Message, options ChatOptions) (*schema.Message, error)
	StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback, options ChatOptions) (string, error)
	GetModelType() string
}

type OpenAIModel struct {
	llm einomodel.ToolCallingChatModel
}

func NewOpenAIModel(ctx context.Context) (*OpenAIModel, error) {
	conf := appconfig.GetConfig()
	key := conf.AIConfig.APIKey
	modelName := conf.AIConfig.ModelName
	baseURL := conf.AIConfig.BaseURL

	llm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  key,
	})
	if err != nil {
		return nil, fmt.Errorf("create openai model failed: %v", err)
	}
	return &OpenAIModel{llm: llm}, nil
}

func (o *OpenAIModel) GenerateResponse(ctx context.Context, messages []*schema.Message, _ ChatOptions) (*schema.Message, error) {
	resp, err := o.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("openai generate failed: %v", err)
	}
	return resp, nil
}

func (o *OpenAIModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback, _ ChatOptions) (string, error) {
	stream, err := o.llm.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("openai stream failed: %v", err)
	}
	defer stream.Close()

	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("openai stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}

	return fullResp.String(), nil
}

func (o *OpenAIModel) GetModelType() string { return "openai" }

type OllamaModel struct {
	llm einomodel.ToolCallingChatModel
}

type ollamaWebSearchArgs struct {
	Query      string `json:"query"`
	Focus      string `json:"focus,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

const (
	ollamaWebSearchToolName = "web_search"
	ollamaWebSearchPrompt   = "如果用户的问题需要最新信息、联网信息、新闻、实时状态或外部资料，请优先调用 web_search 工具。拿到工具结果后，必须直接回答最后一个用户问题，不要无意义地反问。若用户要求“列出几条并注明来源”，就按条目输出标题、来源、链接和一句简述。"
	ollamaWebSearchFallback = "下面是后端刚刚联网检索到的结果。你必须优先基于这些检索结果，直接回答最后一个用户问题，不要说“问题不明确”或再去反问。若用户要求最新新闻或要求列出几条，请从结果里挑选最相关的条目，按“标题 / 来源 / 链接 / 简述”输出；如果结果不足以支撑结论，请明确说明不确定。"
	maxToolRounds           = 3
)

func NewOllamaModel(ctx context.Context, baseURL, modelName string) (*OllamaModel, error) {
	llm, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("create ollama model failed: %v", err)
	}
	return &OllamaModel{llm: llm}, nil
}

func (o *OllamaModel) GenerateResponse(ctx context.Context, messages []*schema.Message, options ChatOptions) (*schema.Message, error) {
	if !options.EnableWebSearch || !appconfig.GetConfig().SearchConfig.Enabled {
		resp, err := o.llm.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("ollama generate failed: %v", err)
		}
		return resp, nil
	}

	resp, toolUsed, err := o.generateWithToolCalls(ctx, messages)
	if err != nil {
		log.Printf("[websearch] ollama tool path failed: %v", err)
	} else if toolUsed {
		return resp, nil
	}

	augmentedMessages, err := o.buildSearchFallbackConversation(ctx, messages)
	if err != nil {
		log.Printf("[websearch] search fallback failed: %v", err)
		return schema.AssistantMessage(buildSearchUnavailableMessage(err), nil), nil
	}

	resp, err = o.llm.Generate(ctx, augmentedMessages)
	if err != nil {
		return nil, fmt.Errorf("ollama generate with search fallback failed: %v", err)
	}

	return resp, nil
}

func (o *OllamaModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback, options ChatOptions) (string, error) {
	if !options.EnableWebSearch || !appconfig.GetConfig().SearchConfig.Enabled {
		return o.streamWithModel(ctx, o.llm, messages, cb)
	}

	conversation, toolUsed, err := o.resolveToolConversation(ctx, messages)
	if err != nil {
		log.Printf("[websearch] ollama stream tool path failed: %v", err)
	} else if toolUsed {
		return o.streamWithModel(ctx, o.llm, conversation, cb)
	}

	augmentedMessages, err := o.buildSearchFallbackConversation(ctx, messages)
	if err != nil {
		content := buildSearchUnavailableMessage(err)
		cb(content)
		return content, nil
	}

	return o.streamWithModel(ctx, o.llm, augmentedMessages, cb)
}

func (o *OllamaModel) GetModelType() string { return "ollama" }

func (o *OllamaModel) prepareToolConversation(messages []*schema.Message) (einomodel.ToolCallingChatModel, []*schema.Message, error) {
	toolModel, err := o.llm.WithTools(buildWebSearchTools())
	if err != nil {
		return nil, nil, fmt.Errorf("bind ollama tools failed: %v", err)
	}

	conversation := make([]*schema.Message, 0, len(messages)+1)
	conversation = append(conversation, schema.SystemMessage(ollamaWebSearchPrompt))
	conversation = append(conversation, messages...)

	return toolModel, conversation, nil
}

func (o *OllamaModel) generateWithToolCalls(ctx context.Context, messages []*schema.Message) (*schema.Message, bool, error) {
	toolModel, conversation, err := o.prepareToolConversation(messages)
	if err != nil {
		return nil, false, err
	}

	toolUsed := false
	for round := 0; round < maxToolRounds; round++ {
		resp, err := toolModel.Generate(ctx, conversation)
		if err != nil {
			return nil, toolUsed, fmt.Errorf("ollama tool generate failed: %v", err)
		}

		log.Printf("[websearch] ollama tool round=%d tool_calls=%d", round+1, len(resp.ToolCalls))
		if len(resp.ToolCalls) == 0 {
			return resp, toolUsed, nil
		}

		toolUsed = true
		conversation = append(conversation, schema.AssistantMessage(resp.Content, resp.ToolCalls))
		conversation = append(conversation, o.executeToolCalls(ctx, resp.ToolCalls)...)
	}

	return nil, toolUsed, fmt.Errorf("ollama tool call rounds exceeded limit")
}

func (o *OllamaModel) resolveToolConversation(ctx context.Context, messages []*schema.Message) ([]*schema.Message, bool, error) {
	toolModel, conversation, err := o.prepareToolConversation(messages)
	if err != nil {
		return nil, false, err
	}

	toolUsed := false
	for round := 0; round < maxToolRounds; round++ {
		resp, err := toolModel.Generate(ctx, conversation)
		if err != nil {
			return nil, toolUsed, fmt.Errorf("ollama tool generate failed: %v", err)
		}

		log.Printf("[websearch] ollama stream tool round=%d tool_calls=%d", round+1, len(resp.ToolCalls))
		if len(resp.ToolCalls) == 0 {
			return conversation, toolUsed, nil
		}

		toolUsed = true
		conversation = append(conversation, schema.AssistantMessage(resp.Content, resp.ToolCalls))
		conversation = append(conversation, o.executeToolCalls(ctx, resp.ToolCalls)...)
	}

	return nil, toolUsed, fmt.Errorf("ollama tool call rounds exceeded limit")
}

func (o *OllamaModel) streamWithModel(ctx context.Context, chatModel einomodel.ToolCallingChatModel, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := chatModel.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("ollama stream failed: %v", err)
	}
	defer stream.Close()

	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("ollama stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}

	return fullResp.String(), nil
}

func (o *OllamaModel) executeToolCalls(ctx context.Context, toolCalls []schema.ToolCall) []*schema.Message {
	toolMessages := make([]*schema.Message, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		content := fmt.Sprintf("tool %s is not implemented", toolCall.Function.Name)
		if toolCall.Function.Name == ollamaWebSearchToolName {
			content = executeWebSearchTool(ctx, toolCall.Function.Arguments)
		}

		toolMessages = append(toolMessages, schema.ToolMessage(
			content,
			toolCall.ID,
			schema.WithToolName(toolCall.Function.Name),
		))
	}
	return toolMessages
}

func (o *OllamaModel) buildSearchFallbackConversation(ctx context.Context, messages []*schema.Message) ([]*schema.Message, error) {
	query := extractLastUserQuery(messages)
	if query == "" {
		return nil, fmt.Errorf("no user query found for web search")
	}

	resp, err := websearch.Search(ctx, query, websearch.FocusAuto, 0)
	if err != nil {
		return nil, err
	}

	log.Printf("[websearch] fallback search query=%q focus=%s results=%d", query, resp.Focus, len(resp.Results))

	conversation := make([]*schema.Message, 0, len(messages)+2)
	conversation = append(conversation, schema.SystemMessage(ollamaWebSearchFallback))
	conversation = append(conversation, schema.SystemMessage(websearch.FormatToolResult(resp)))
	conversation = append(conversation, messages...)

	return conversation, nil
}

func extractLastUserQuery(messages []*schema.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil {
			continue
		}
		if msg.Role == schema.User && strings.TrimSpace(msg.Content) != "" {
			return strings.TrimSpace(msg.Content)
		}
	}
	return ""
}

func buildSearchUnavailableMessage(err error) string {
	return fmt.Sprintf("当前无法完成联网搜索：%v。请稍后再试，或先关闭“联网搜索”后仅使用本地模型回答。", err)
}

func executeWebSearchTool(ctx context.Context, arguments string) string {
	var args ollamaWebSearchArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return fmt.Sprintf("联网搜索参数解析失败: %v", err)
	}

	if strings.TrimSpace(args.Query) == "" {
		return "联网搜索失败: query 不能为空"
	}

	resp, err := websearch.Search(ctx, args.Query, websearch.Focus(args.Focus), args.MaxResults)
	if err != nil {
		return fmt.Sprintf("联网搜索失败: %v", err)
	}

	return websearch.FormatToolResult(resp)
}

func buildWebSearchTools() []*schema.ToolInfo {
	return []*schema.ToolInfo{
		{
			Name: ollamaWebSearchToolName,
			Desc: "Search the internet for up-to-date public information. Use this tool when the user asks for latest, current, recent, online, news, or external facts.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"query": {
					Type:     schema.String,
					Desc:     "The search query in the user's language.",
					Required: true,
				},
				"focus": {
					Type: schema.String,
					Desc: "Search focus. Use 'news' for time-sensitive queries, 'general' for encyclopedic queries, or 'auto' if unsure.",
					Enum: []string{"auto", "general", "news"},
				},
				"max_results": {
					Type: schema.Integer,
					Desc: "Maximum number of search results to return. Keep this small, usually 3 to 5.",
				},
			}),
		},
	}
}
