package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

func gptGPT3CompleteContext(content string) (ret string) {
	fmt.Printf("Using GPT3\n")
	return ""
}

func gptGPT3CompleteContextImage(content string, url string) (ret string) {
	fmt.Printf("Using GPT3\n")
	return gptCompleteContext(content, url, openai.GPT4VisionPreview)
}

func gptCompleteContext(content string, url string, model string) (ret string) {
	ctx := context.Background()

	// See: https://platform.openai.com/docs/guides/chat
	//			https://github.com/sashabaranov/go-openai
	req := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{{
			Role:    openai.ChatMessageRoleUser,
			MultiContent: []openai.ChatMessagePart{{
				Type:    openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL: url,
				},
			}, {
				Type:    openai.ChatMessagePartTypeText,
				Content: content,
			}},
		}},
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		ret = fmt.Sprintf("Err: %v", err)
	} else {
		// The response contains a list of choices, each with a score.
		// The score is a float between 0 and 1, with 1 being the most likely.
		// The choices are sorted by score, with the first choice being the most likely.
		// So we just take the first choice.
		ret = resp.Choices[0].Message.Content
	}

	return ret
}

// Create image by DALL-E 2
func gptImageCreate(prompt string) (string, error) {
	ctx := context.Background()

	reqUrl := openai.ImageRequest{
		Prompt:         prompt,
		Size:           openai.CreateImageSize256x256,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		N:              1,
	}

	respUrl, err := client.CreateImage(ctx, reqUrl)
	if err != nil {
		fmt.Printf("Image creation error: %v\n", err)
		return "", errors.New("image creation error")
	}
	fmt.Println(respUrl.Data[0].URL)

	return respUrl.Data[0].URL, nil
}
