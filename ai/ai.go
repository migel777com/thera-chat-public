package ai

import (
	"chatgpt/config"
	"chatgpt/models"
	"context"
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"time"
)

type AI struct {
	client    *openai.Client
	assistant *openai.Assistant
	model     *string
}

func NewAI(config *config.Config) *AI {
	client := openai.NewClient(config.OpenAiAuthToken)

	assistant, err := client.RetrieveAssistant(context.Background(), config.OpenAiAssistantId)
	if err != nil {
		fmt.Printf("Assistant error: %v\n", err)
		panic(err)
	}

	// TODO what model or it comes from the assistant
	model := openai.GPT3Dot5Turbo1106

	return &AI{
		client:    client,
		assistant: &assistant,
		model:     &model,
	}
}

func (a *AI) NewThread(ctx context.Context) (openai.Thread, error) {
	thread, err := a.client.CreateThread(ctx, openai.ThreadRequest{})
	if err != nil {
		return openai.Thread{}, err
	}

	return thread, nil
}

func (a *AI) NewMessage(ctx context.Context, threadId string, text string) (string, error) {
	_, err := a.client.CreateMessage(ctx, threadId, openai.MessageRequest{
		Role:    openai.ChatMessageRoleUser,
		Content: text,
	})
	if err != nil {
		return "", err
	}

	run, err := a.client.CreateRun(ctx, threadId, openai.RunRequest{
		AssistantID: a.assistant.ID,
		Model:       a.model,
	})
	if err != nil {
		return "", err
	}

	for {
		run, err = a.client.RetrieveRun(ctx, threadId, run.ID)
		if err != nil {
			return "", err
		}
		switch run.Status {
		case "in_progress":
			time.Sleep(time.Second * 5)
		case "completed":
			return a.GetLastMessage(ctx, threadId)
		case "requires_action":
			return "required action", nil
		case "expired":
			return "", errors.New("run expired")
		case "cancelling":
			return "", errors.New("run cancelling")
		case "cancelled":
			return "", errors.New("run cancelled")
		case "failed":
			return "", fmt.Errorf("run failed: %s, code: %s", run.LastError.Message, run.LastError.Code)

		}
	}
}

func (a *AI) GetLastMessage(ctx context.Context, threadId string) (string, error) {
	msg, err := a.client.ListMessage(ctx, threadId, nil, nil, nil, nil)
	if err != nil {
		return "", err
	}

	if msg.Messages[0].Content[0].Text != nil {
		return msg.Messages[0].Content[0].Text.Value, nil
	}

	return "", errors.New("no response")
}

func (a *AI) GetMessages(ctx context.Context, threadId string) ([]models.Message, error) {
	msg, err := a.client.ListMessage(ctx, threadId, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	if len(msg.Messages) > 0 {
		var messages []models.Message

		for _, m := range msg.Messages {
			messages = append(messages, models.Message{Role: m.Role, Text: m.Content[0].Text.Value})
		}

		return messages, nil
	}

	// TODO return error
	//return nil, errors.New("empty conversation")
	messages := make([]models.Message, 0)
	return messages, nil
}
