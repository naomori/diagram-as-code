package mcp

import (
	"encoding/base64"
	"fmt"
	"os"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

type Content struct {
	Title       string  `json:"title" jsonschema:"required,description=The title to submit"`
	Description *string `json:"description" jsonschema:"description=The description to submit"`
}
type MyFunctionsArguments struct {
	Submitter string  `json:"submitter" jsonschema:"required,description=The name of the thing calling this tool (openai, google, claude, etc)"`
	Content   Content `json:"content" jsonschema:"required,description=The content of the message"`
}

func McpServer() {
	done := make(chan struct{})

	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())
	err := server.RegisterTool("hello", "Return image data", func(arguments MyFunctionsArguments) (*mcp_golang.ToolResponse, error) {
		// 実際の画像ファイルを読み込む
		imagePath := "/home/ubuntu/diagram-as-code.git/doc/static/desired-image.png"
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Error reading image file: %v", err))), nil
		}
		
		// 画像データをBase64エンコードする
		base64Data := base64.StdEncoding.EncodeToString(imageData)
		
		// Base64エンコードされたデータをファイルに保存
		err = os.WriteFile("output.base64", []byte(base64Data), 0644)
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Error writing output.base64: %v", err))), nil
		}
		
		// 元の画像データもファイルに保存
		err = os.WriteFile("output.png", imageData, 0644)
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Error writing output.png: %v", err))), nil
		}
		
		return mcp_golang.NewToolResponse(mcp_golang.NewImageContent(base64Data, "image/png")), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.RegisterPrompt("prompt_test", "This is a test prompt", func(arguments Content) (*mcp_golang.PromptResponse, error) {
		return mcp_golang.NewPromptResponse("description", mcp_golang.NewPromptMessage(mcp_golang.NewTextContent(fmt.Sprintf("Hello, %s!", arguments.Title)), mcp_golang.RoleUser)), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.RegisterResource("test://resource", "resource_test", "This is a test resource", "application/json", func() (*mcp_golang.ResourceResponse, error) {
		return mcp_golang.NewResourceResponse(mcp_golang.NewTextEmbeddedResource("test://resource", "This is a test resource", "application/json")), nil
	})

	err = server.RegisterResource("file://app_logs", "app_logs", "The app logs", "text/plain", func() (*mcp_golang.ResourceResponse, error) {
		return mcp_golang.NewResourceResponse(mcp_golang.NewTextEmbeddedResource("file://app_logs", "This is a test resource", "text/plain")), nil
	})

	err = server.Serve()
	if err != nil {
		panic(err)
	}

	<-done
}