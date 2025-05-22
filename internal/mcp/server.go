package mcp

import (
    "context"
    "encoding/base64"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/awslabs/diagram-as-code/internal/ctl"
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
    
    // Add dac tool
    dacTool := mcp.NewTool("dac",
        mcp.WithDescription("Convert DAC file to image and return it"),
        mcp.WithString("content",
            mcp.Required(),
            mcp.Description("DAC file content"),
        ),
        mcp.WithString("outputFormat",
            mcp.Description("Output format (png)"),
        ),
    )
    
    // Add cfn tool
    cfnTool := mcp.NewTool("cfn",
        mcp.WithDescription("Convert CloudFormation template to DAC file"),
        mcp.WithString("content",
            mcp.Required(),
            mcp.Description("CloudFormation template content"),
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
    
    // Add the DAC handler
    s.AddTool(dacTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Get the DAC content
        dacContent, err := request.RequireString("content")
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }
        
        // Create a temporary file for the DAC content
        tempDacFile, err := os.CreateTemp("", "dac-*.yaml")
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to create temporary file: %v", err)), nil
        }
        defer os.Remove(tempDacFile.Name())
        
        // Write the DAC content to the temporary file
        if _, err := tempDacFile.WriteString(dacContent); err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to write to temporary file: %v", err)), nil
        }
        tempDacFile.Close()
        
        // Create a temporary file for the output image
        tempImageFile, err := os.CreateTemp("", "dac-output-*.png")
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to create temporary output file: %v", err)), nil
        }
        defer os.Remove(tempImageFile.Name())
        tempImageFile.Close()
        
        outputPath := tempImageFile.Name()
        
        // Process the DAC file and generate the diagram
        opts := &ctl.CreateOptions{
            IsGoTemplate: false,
        }
        
        // 標準出力のリダイレクトを安全に行う
        oldStdout := os.Stdout
        nullFile, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to open null device: %v", err)), nil
        }
        os.Stdout = nullFile
        
        // パニックをキャッチする
        defer func() {
            // 標準出力を元に戻す
            nullFile.Close()
            os.Stdout = oldStdout
            
            if r := recover(); r != nil {
                fmt.Printf("Recovered from panic: %v\n", r)
            }
        }()
        
        // 安全に実行を試みる
        func() {
            defer func() {
                if r := recover(); r != nil {
                    fmt.Printf("Panic in CreateDiagramFromDacFile: %v\n", r)
                }
            }()
            ctl.CreateDiagramFromDacFile(tempDacFile.Name(), &outputPath, opts)
        }()
        
        // 生成された画像ファイルが存在するか確認
        if _, err := os.Stat(outputPath); os.IsNotExist(err) {
            return mcp.NewToolResultError("Failed to generate diagram: output file not created"), nil
        }
        
        // Read the generated image
        imageData, err := os.ReadFile(outputPath)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to read generated image: %v", err)), nil
        }
        
        // Encode the image as base64
        base64Data := base64.StdEncoding.EncodeToString(imageData)
        
        // Save base64 encoded data to file
        if err := os.WriteFile("/tmp/base64encoded.data", []byte(base64Data), 0644); err != nil {
            fmt.Printf("Failed to save base64 data: %v\n", err)
        } else {
            fmt.Printf("Base64 encoded data saved to /tmp/base64encoded.data\n")
        }
        
        // Return the image
        return mcp.NewToolResultImage("Diagram generated successfully", base64Data, "image/png"), nil
    })

    // Add the CFN handler
    s.AddTool(cfnTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Get the CloudFormation template content
        cfnContent, err := request.RequireString("content")
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }
        
        // Create a temporary file for the CloudFormation content
        tempCfnFile, err := os.CreateTemp("", "cfn-*.yaml")
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to create temporary file: %v", err)), nil
        }
        defer os.Remove(tempCfnFile.Name())
        
        // Write the CloudFormation content to the temporary file
        if _, err := tempCfnFile.WriteString(cfnContent); err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to write to temporary file: %v", err)), nil
        }
        tempCfnFile.Close()
        
        // Create a temporary file for the output DAC file
        tempDacFile, err := os.CreateTemp("", "dac-output-*.yaml")
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to create temporary output file: %v", err)), nil
        }
        defer os.Remove(tempDacFile.Name())
        tempDacFile.Close()
        
        outputPath := tempDacFile.Name()
        
        // Process the CloudFormation file and generate the DAC file
        opts := &ctl.CreateOptions{
            IsGoTemplate: false,
        }
        
        // 標準出力のリダイレクトを安全に行う
        oldStdout := os.Stdout
        nullFile, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to open null device: %v", err)), nil
        }
        os.Stdout = nullFile
        
        // パニックをキャッチする
        defer func() {
            // 標準出力を元に戻す
            nullFile.Close()
            os.Stdout = oldStdout
            
            if r := recover(); r != nil {
                fmt.Printf("Recovered from panic: %v\n", r)
            }
        }()
        
        // 安全に実行を試みる
        func() {
            defer func() {
                if r := recover(); r != nil {
                    fmt.Printf("Panic in CreateDiagramFromCFnTemplate: %v\n", r)
                }
            }()
            // Generate DAC file from CloudFormation template
            // outputPathはPNG画像のパスとして使われるが、YAMLファイルは同じベース名で.yaml拡張子で生成される
            fmt.Printf("Generating DAC file from CloudFormation template, output path: %s\n", outputPath)
            ctl.CreateDiagramFromCFnTemplate(tempCfnFile.Name(), &outputPath, true, opts)
        }()
        
        // CreateDiagramFromCFnTemplateは、outputPathと同じベース名で.yaml拡張子のファイルを生成する
        // 例: outputPath=/tmp/test.png → DACファイル=/tmp/test.yaml
        dacFilePath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".yaml"
        
        fmt.Printf("Looking for DAC file at: %s\n", dacFilePath)
        
        // 生成されたDACファイルが存在するか確認
        if _, err := os.Stat(dacFilePath); os.IsNotExist(err) {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to generate DAC file: output file %s not created", dacFilePath)), nil
        }
        
        // Read the generated DAC file
        dacData, err := os.ReadFile(dacFilePath)
        if err != nil {
            return mcp.NewToolResultError(fmt.Sprintf("Failed to read generated DAC file: %v", err)), nil
        }
        
        // Save DAC file content to a fixed location for debugging
        if err := os.WriteFile("/tmp/dac-output.yaml", dacData, 0644); err != nil {
            fmt.Printf("Failed to save DAC data: %v\n", err)
        } else {
            fmt.Printf("DAC file content saved to /tmp/dac-output.yaml\n")
        }
        
        // Return the DAC file content
        return mcp.NewToolResultText(string(dacData)), nil
    })

    // Start the server
    if err := server.ServeStdio(s); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}

