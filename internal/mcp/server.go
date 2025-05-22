package mcp

import (
    "context"
    "encoding/base64"
    "fmt"
    "os"
    "path/filepath"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

// McpServer starts an MCP server instance
func McpServer() {
    // Create a new MCP server
    s := server.NewMCPServer(
        "AWS Diagram as Code",
        "1.0.0",
        server.WithToolCapabilities(false),
        server.WithRecovery(),
    )

    // Add a calculator tool
    calculatorTool := mcp.NewTool("calculate",
        mcp.WithDescription("Perform basic arithmetic operations"),
        mcp.WithString("operation",
            mcp.Required(),
            mcp.Description("The operation to perform (add, subtract, multiply, divide)"),
            mcp.Enum("add", "subtract", "multiply", "divide"),
        ),
        mcp.WithNumber("x",
            mcp.Required(),
            mcp.Description("First number"),
        ),
        mcp.WithNumber("y",
            mcp.Required(),
            mcp.Description("Second number"),
        ),
    )

    // Add the calculator handler
    s.AddTool(calculatorTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Using helper functions for type-safe argument access
        op, err := request.RequireString("operation")
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        x, err := request.RequireFloat("x")
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        y, err := request.RequireFloat("y")
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        var result float64
        switch op {
        case "add":
            result = x + y
        case "subtract":
            result = x - y
        case "multiply":
            result = x * y
        case "divide":
            if y == 0 {
                return mcp.NewToolResultError("cannot divide by zero"), nil
            }
            result = x / y
        }

        return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
    })

    // Add image tool
    imageTool := mcp.NewTool("image",
        mcp.WithDescription("Load and return an image file"),
        mcp.WithString("path",
            mcp.Required(),
            mcp.Description("Path to the image file"),
        ),
    )

    // Add the image handler
    s.AddTool(imageTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Get the image path
        imagePath, err := request.RequireString("path")
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        // Read the image file
        imageData, err := os.ReadFile(imagePath)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to read image: %v", err)), nil
        }

        // Encode the image as base64
        base64Data := base64.StdEncoding.EncodeToString(imageData)
        
        // Determine MIME type based on file extension
        mimeType := "image/png" // Default
        ext := filepath.Ext(imagePath)
        switch ext {
        case ".jpg", ".jpeg":
            mimeType = "image/jpeg"
        case ".gif":
            mimeType = "image/gif"
        case ".svg":
            mimeType = "image/svg+xml"
        }
        
        // Return the image as a data URL
        return mcp.NewToolResultImage("Image loaded successfully", base64Data, mimeType), nil
    })

    // Start the server
    if err := server.ServeStdio(s); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}

