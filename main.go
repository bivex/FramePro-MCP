package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// FrameProData represents the structure of FramePro JSON files
// Supports both frame_analysis.json and functions_analysis.json formats
type FrameProData struct {
	SessionName     string                `json:"SessionName"`
	TotalFrames     int                   `json:"TotalFrames"`
	TotalFunctions  int                   `json:"TotalFunctions,omitempty"`
	Frames          []FrameProFrame       `json:"Frames,omitempty"`
	Functions       []FrameProFunction    `json:"Functions,omitempty"`
}

type FrameProFrame struct {
	FrameNumber int                  `json:"FrameNumber"`
	Functions   []FrameProFunction   `json:"Functions,omitempty"`
}

type FrameProFunction struct {
	FunctionName              string  `json:"FunctionName"`
	ThreadID                  int     `json:"ThreadId"`
	ThreadName                string  `json:"ThreadName"`
	TimeMs                    float64 `json:"TimeMs,omitempty"`          // Time in current frame
	Count                     int     `json:"Count,omitempty"`           // Count in current frame
	TotalTimeMs               float64 `json:"TotalTimeMs"`               // Total time across all frames
	TotalCount                int     `json:"TotalCount"`                // Total count across all frames
	MaxTimeMs                 float64 `json:"MaxTimeMs,omitempty"`
	MaxTimePerFrameMs         float64 `json:"MaxTimePerFrameMs"`
	MaxCountPerFrame          int     `json:"MaxCountPerFrame"`
	AvgTimePerFrameMs         float64 `json:"AvgTimePerFrameMs"`
	AvgCountPerFrame          float64 `json:"AvgCountPerFrame"`
	ThreadUtilizationPercent  float64 `json:"ThreadUtilizationPercent"`
	IsMainThread              bool    `json:"IsMainThread"`
	IsRenderThread            bool    `json:"IsRenderThread"`
	IsWorkerThread            bool    `json:"IsWorkerThread"`
	ThreadPriority            int     `json:"ThreadPriority"`
}

// PerformanceIssue represents a detected performance problem
type PerformanceIssue struct {
	Severity    string  `json:"severity"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Suggestion  string  `json:"suggestion"`
	Value       float64 `json:"value,omitempty"`
}

var dataDir string

func main() {
	// Get data directory from environment or use default
	dataDir = os.Getenv("FRAMEPRO_DATA_DIR")
	if dataDir == "" {
		exe, err := os.Executable()
		if err == nil {
			dataDir = filepath.Dir(exe)
		} else {
			dataDir = "."
		}
	}

	// Create MCP server
	s := server.NewMCPServer(
		"FramePro Performance Analyzer",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register tools
	analyzePerformanceTool := mcp.NewTool("analyze_performance",
		mcp.WithDescription("Analyzes FramePro JSON data and identifies performance bottlenecks, hotspots, and optimization opportunities"),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the FramePro JSON file to analyze")),
		mcp.WithString("focus",
			mcp.Description("Optional focus area: 'cpu', 'memory', 'frames', 'threads', or 'all' (default: 'all')")),
	)

	findHotspotsTool := mcp.NewTool("find_hotspots",
		mcp.WithDescription("Identifies the top performance hotspots (most expensive functions) in the FramePro data"),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the FramePro JSON file")),
		mcp.WithNumber("top_n",
			mcp.Description("Number of top hotspots to return (default: 10)")),
	)

	frameAnalysisTool := mcp.NewTool("analyze_frame_times",
		mcp.WithDescription("Analyzes frame timing data to detect stuttering, spikes, and frame rate issues"),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the FramePro JSON file")),
		mcp.WithNumber("target_fps",
			mcp.Description("Target FPS for comparison (default: 60)")),
	)

	compareProfilesTool := mcp.NewTool("compare_profiles",
		mcp.WithDescription("Compares two FramePro profiles to identify performance regressions or improvements"),
		mcp.WithString("baseline_path",
			mcp.Required(),
			mcp.Description("Path to the baseline FramePro JSON file")),
		mcp.WithString("current_path",
			mcp.Required(),
			mcp.Description("Path to the current FramePro JSON file")),
	)

	s.AddTool(analyzePerformanceTool, analyzePerformanceHandler)
	s.AddTool(findHotspotsTool, findHotspotsHandler)
	s.AddTool(frameAnalysisTool, frameAnalysisHandler)
	s.AddTool(compareProfilesTool, compareProfilesHandler)

	// Note: Resources disabled to avoid null array error
	// Tools provide all necessary functionality

	// Start server using stdio
	if err := server.ServeStdio(s); err != nil {
		log.Fatal(err)
	}
}

// Tool handlers

func analyzePerformanceHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments format"), nil
	}

	filePath, _ := args["file_path"].(string)
	focus, _ := args["focus"].(string)
	if focus == "" {
		focus = "all"
	}

	data, err := loadFrameProData(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load FramePro data: %v", err)), nil
	}

	issues := []PerformanceIssue{}

	// Analyze based on focus area
	if focus == "all" || focus == "cpu" {
		issues = append(issues, analyzeCPUPerformance(data)...)
	}
	if focus == "all" || focus == "frames" {
		issues = append(issues, analyzeFramePerformance(data)...)
	}
	if focus == "all" || focus == "threads" {
		issues = append(issues, analyzeThreadPerformance(data)...)
	}

	// Sort by severity
	sort.Slice(issues, func(i, j int) bool {
		severityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		return severityOrder[issues[i].Severity] < severityOrder[issues[j].Severity]
	})

	result, _ := json.MarshalIndent(map[string]interface{}{
		"file":          filePath,
		"focus":         focus,
		"issuesFound":   len(issues),
		"issues":        issues,
		"summary":       generateSummary(issues),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

func findHotspotsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments format"), nil
	}

	filePath, _ := args["file_path"].(string)
	topN := 10
	if n, ok := args["top_n"].(float64); ok {
		topN = int(n)
	}

	data, err := loadFrameProData(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load FramePro data: %v", err)), nil
	}

	// Sort functions by total time
	functions := data.Functions
	sort.Slice(functions, func(i, j int) bool {
		return functions[i].TotalTimeMs > functions[j].TotalTimeMs
	})

	if topN > len(functions) {
		topN = len(functions)
	}

	hotspots := functions[:topN]

	// Generate optimization suggestions for each hotspot
	analysis := make([]map[string]interface{}, len(hotspots))
	for i, fn := range hotspots {
		avgTimePerCall := fn.TotalTimeMs / float64(fn.TotalCount+1)

		analysis[i] = map[string]interface{}{
			"rank":                  i + 1,
			"functionName":          fn.FunctionName,
			"threadName":            fn.ThreadName,
			"threadId":              fn.ThreadID,
			"isMainThread":          fn.IsMainThread,
			"isRenderThread":        fn.IsRenderThread,
			"totalTimeMs":           fn.TotalTimeMs,
			"avgTimePerFrameMs":     fn.AvgTimePerFrameMs,
			"maxTimePerFrameMs":     fn.MaxTimePerFrameMs,
			"totalCount":            fn.TotalCount,
			"avgCountPerFrame":      fn.AvgCountPerFrame,
			"avgTimePerCallMs":      avgTimePerCall,
			"threadUtilization":     fn.ThreadUtilizationPercent,
			"suggestions":           generateFunctionSuggestions(fn),
		}
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"file":     filePath,
		"topN":     topN,
		"hotspots": analysis,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

func frameAnalysisHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments format"), nil
	}

	filePath, _ := args["file_path"].(string)
	targetFPS := 60.0
	if fps, ok := args["target_fps"].(float64); ok {
		targetFPS = fps
	}

	data, err := loadFrameProData(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load FramePro data: %v", err)), nil
	}

	targetFrameTime := 1000.0 / targetFPS // in milliseconds

	// Analyze main thread functions for frame issues
	var mainThreadFunctions []FrameProFunction
	var renderThreadFunctions []FrameProFunction
	var problemFunctions []map[string]interface{}

	for _, fn := range data.Functions {
		if fn.IsMainThread {
			mainThreadFunctions = append(mainThreadFunctions, fn)
			if fn.MaxTimePerFrameMs > targetFrameTime {
				problemFunctions = append(problemFunctions, map[string]interface{}{
					"function":          fn.FunctionName,
					"maxTimePerFrame":   fn.MaxTimePerFrameMs,
					"avgTimePerFrame":   fn.AvgTimePerFrameMs,
					"threadUtilization": fn.ThreadUtilizationPercent,
					"impact":            "Blocks main thread, causes frame drops",
				})
			}
		}
		if fn.IsRenderThread {
			renderThreadFunctions = append(renderThreadFunctions, fn)
		}
	}

	// Calculate approximate FPS based on main thread work
	var mainThreadTotalAvgTime float64
	for _, fn := range mainThreadFunctions {
		mainThreadTotalAvgTime += fn.AvgTimePerFrameMs
	}
	estimatedFPS := 1000.0 / mainThreadTotalAvgTime
	if estimatedFPS > 1000.0 {
		estimatedFPS = 1000.0 // Cap at reasonable value
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"file":                    filePath,
		"sessionName":             data.SessionName,
		"totalFrames":             data.TotalFrames,
		"targetFPS":               targetFPS,
		"estimatedFPS":            estimatedFPS,
		"mainThreadAvgWorkMs":     mainThreadTotalAvgTime,
		"targetFrameTimeMs":       targetFrameTime,
		"problemFunctions":        problemFunctions,
		"mainThreadFunctionCount": len(mainThreadFunctions),
		"analysis":                analyzeFrameIssues(len(problemFunctions), 0, estimatedFPS, targetFPS),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

func compareProfilesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments format"), nil
	}

	baselinePath, _ := args["baseline_path"].(string)
	currentPath, _ := args["current_path"].(string)

	baseline, err := loadFrameProData(baselinePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load baseline data: %v", err)), nil
	}

	current, err := loadFrameProData(currentPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load current data: %v", err)), nil
	}

	// Compare functions
	baselineFuncs := make(map[string]FrameProFunction)
	for _, fn := range baseline.Functions {
		key := fmt.Sprintf("%s:%d", fn.FunctionName, fn.ThreadID)
		baselineFuncs[key] = fn
	}

	regressions := []map[string]interface{}{}
	improvements := []map[string]interface{}{}
	newFunctions := []map[string]interface{}{}

	for _, currentFn := range current.Functions {
		key := fmt.Sprintf("%s:%d", currentFn.FunctionName, currentFn.ThreadID)
		if baselineFn, exists := baselineFuncs[key]; exists {
			timeDiff := currentFn.TotalTimeMs - baselineFn.TotalTimeMs
			percentChange := (timeDiff / (baselineFn.TotalTimeMs + 0.001)) * 100

			avgTimeDiff := currentFn.AvgTimePerFrameMs - baselineFn.AvgTimePerFrameMs
			avgPercentChange := (avgTimeDiff / (baselineFn.AvgTimePerFrameMs + 0.001)) * 100

			if percentChange > 10.0 { // Regression threshold
				severity := "medium"
				if percentChange > 50.0 {
					severity = "high"
				}
				if currentFn.IsMainThread {
					severity = "critical"
				}

				regressions = append(regressions, map[string]interface{}{
					"severity":             severity,
					"function":             currentFn.FunctionName,
					"threadName":           currentFn.ThreadName,
					"isMainThread":         currentFn.IsMainThread,
					"baselineTotalMs":      baselineFn.TotalTimeMs,
					"currentTotalMs":       currentFn.TotalTimeMs,
					"totalTimeDiffMs":      timeDiff,
					"totalPercentChange":   percentChange,
					"baselineAvgMs":        baselineFn.AvgTimePerFrameMs,
					"currentAvgMs":         currentFn.AvgTimePerFrameMs,
					"avgTimeDiffMs":        avgTimeDiff,
					"avgPercentChange":     avgPercentChange,
					"baselineUtilization":  baselineFn.ThreadUtilizationPercent,
					"currentUtilization":   currentFn.ThreadUtilizationPercent,
				})
			} else if percentChange < -10.0 { // Improvement threshold
				improvements = append(improvements, map[string]interface{}{
					"function":           currentFn.FunctionName,
					"threadName":         currentFn.ThreadName,
					"baselineTotalMs":    baselineFn.TotalTimeMs,
					"currentTotalMs":     currentFn.TotalTimeMs,
					"totalTimeDiffMs":    timeDiff,
					"totalPercentChange": percentChange,
					"avgPercentChange":   avgPercentChange,
				})
			}
			delete(baselineFuncs, key)
		} else {
			// New function not in baseline
			if currentFn.TotalTimeMs > 10.0 { // Only report significant new functions
				newFunctions = append(newFunctions, map[string]interface{}{
					"function":   currentFn.FunctionName,
					"threadName": currentFn.ThreadName,
					"totalMs":    currentFn.TotalTimeMs,
					"avgMs":      currentFn.AvgTimePerFrameMs,
				})
			}
		}
	}

	// Functions that disappeared
	removedFunctions := []map[string]interface{}{}
	for _, fn := range baselineFuncs {
		if fn.TotalTimeMs > 10.0 {
			removedFunctions = append(removedFunctions, map[string]interface{}{
				"function":   fn.FunctionName,
				"threadName": fn.ThreadName,
				"totalMs":    fn.TotalTimeMs,
			})
		}
	}

	// Sort regressions by severity and impact
	sort.Slice(regressions, func(i, j int) bool {
		severityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		si := severityOrder[regressions[i]["severity"].(string)]
		sj := severityOrder[regressions[j]["severity"].(string)]
		if si != sj {
			return si < sj
		}
		return regressions[i]["totalPercentChange"].(float64) > regressions[j]["totalPercentChange"].(float64)
	})

	result, _ := json.MarshalIndent(map[string]interface{}{
		"baseline":         baselinePath,
		"baselineSession":  baseline.SessionName,
		"current":          currentPath,
		"currentSession":   current.SessionName,
		"regressions":      regressions,
		"improvements":     improvements,
		"newFunctions":     newFunctions,
		"removedFunctions": removedFunctions,
		"summary": fmt.Sprintf("Found %d regressions (%d critical), %d improvements, %d new functions, %d removed functions",
			len(regressions), countBySeverity(regressions, "critical"), len(improvements), len(newFunctions), len(removedFunctions)),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// Resource handler
func resourceHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract path from URI (framepro://path/to/file.json)
	path := strings.TrimPrefix(request.Params.URI, "framepro://")

	fullPath := filepath.Join(dataDir, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := mcp.TextResourceContents{
		URI:      request.Params.URI,
		MIMEType: "application/json",
		Text:     string(data),
	}

	// Convert to ResourceContents interface
	var result []mcp.ResourceContents
	result = append(result, content)
	return result, nil
}

// Helper functions

func loadFrameProData(filePath string) (*FrameProData, error) {
	// Try absolute path first
	fullPath := filePath

	// If file doesn't exist and path is not absolute, try with dataDir
	if !filepath.IsAbs(filePath) {
		// Try in dataDir
		fullPath = filepath.Join(dataDir, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// Try in current directory
			fullPath = filePath
		}
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file (tried: %s, %s): %w", filePath, fullPath, err)
	}

	var frameProData FrameProData
	if err := json.Unmarshal(data, &frameProData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &frameProData, nil
}

func analyzeCPUPerformance(data *FrameProData) []PerformanceIssue {
	issues := []PerformanceIssue{}

	// Find expensive functions
	for _, fn := range data.Functions {
		// Critical: functions taking more than 100ms total
		if fn.TotalTimeMs > 100.0 {
			severity := "high"
			if fn.TotalTimeMs > 500.0 {
				severity = "critical"
			}

			threadInfo := fn.ThreadName
			if fn.IsMainThread {
				threadInfo += " (MAIN THREAD - blocks rendering!)"
				severity = "critical"
			} else if fn.IsRenderThread {
				threadInfo += " (RENDER THREAD - affects FPS!)"
			}

			issues = append(issues, PerformanceIssue{
				Severity:    severity,
				Category:    "CPU Hotspot",
				Description: fmt.Sprintf("Function '%s' on %s consumes excessive CPU time", fn.FunctionName, threadInfo),
				Impact:      fmt.Sprintf("%.2fms total (%.2fms avg/frame), %d total calls, %.1f%% thread utilization",
					fn.TotalTimeMs, fn.AvgTimePerFrameMs, fn.TotalCount, fn.ThreadUtilizationPercent),
				Suggestion:  generateOptimizationSuggestion(fn),
				Value:       fn.TotalTimeMs,
			})
		}

		// High call count with significant time
		if fn.TotalCount > 10000 && fn.TotalTimeMs > 50.0 {
			issues = append(issues, PerformanceIssue{
				Severity:    "medium",
				Category:    "Call Frequency",
				Description: fmt.Sprintf("Function '%s' called very frequently on %s", fn.FunctionName, fn.ThreadName),
				Impact:      fmt.Sprintf("%d total calls (%.1f avg/frame), %.2fms total time",
					fn.TotalCount, fn.AvgCountPerFrame, fn.TotalTimeMs),
				Suggestion:  "Consider caching results, batching calls, or reducing call frequency",
				Value:       float64(fn.TotalCount),
			})
		}

		// High per-frame spikes
		if fn.MaxTimePerFrameMs > 16.67 && fn.TotalCount > 100 { // Longer than 1 frame at 60fps
			issues = append(issues, PerformanceIssue{
				Severity:    "high",
				Category:    "Frame Spike",
				Description: fmt.Sprintf("Function '%s' causes frame spikes", fn.FunctionName),
				Impact:      fmt.Sprintf("Max %.2fms in single frame (avg: %.2fms) on %s",
					fn.MaxTimePerFrameMs, fn.AvgTimePerFrameMs, fn.ThreadName),
				Suggestion:  "Investigate why this function occasionally takes much longer. Consider spreading work across frames",
				Value:       fn.MaxTimePerFrameMs,
			})
		}

		// Very high thread utilization (>95%)
		if fn.ThreadUtilizationPercent > 95.0 && fn.TotalTimeMs > 100.0 {
			issues = append(issues, PerformanceIssue{
				Severity:    "critical",
				Category:    "Thread Saturation",
				Description: fmt.Sprintf("Function '%s' saturates %s", fn.FunctionName, fn.ThreadName),
				Impact:      fmt.Sprintf("%.1f%% thread utilization, %.2fms total time",
					fn.ThreadUtilizationPercent, fn.TotalTimeMs),
				Suggestion:  "Thread is completely saturated. Critical optimization needed or work redistribution to other threads",
				Value:       fn.ThreadUtilizationPercent,
			})
		}
	}

	return issues
}

func analyzeFramePerformance(data *FrameProData) []PerformanceIssue {
	issues := []PerformanceIssue{}

	// Analyze based on total frames and function data
	if data.TotalFrames > 0 {
		// Look for functions with high max time per frame
		for _, fn := range data.Functions {
			// Frame spike detection
			if fn.MaxTimePerFrameMs > 33.0 && fn.IsMainThread { // Slower than 30 FPS
				issues = append(issues, PerformanceIssue{
					Severity:    "critical",
					Category:    "Frame Spike - Main Thread",
					Description: fmt.Sprintf("Function '%s' causes critical frame spikes on main thread", fn.FunctionName),
					Impact:      fmt.Sprintf("Max %.2fms per frame (target: 16.67ms for 60fps), avg %.2fms",
						fn.MaxTimePerFrameMs, fn.AvgTimePerFrameMs),
					Suggestion:  "This blocks the main thread and causes stuttering. Move to worker thread or optimize urgently",
					Value:       fn.MaxTimePerFrameMs,
				})
			} else if fn.MaxTimePerFrameMs > 16.67 && fn.IsMainThread {
				issues = append(issues, PerformanceIssue{
					Severity:    "high",
					Category:    "Frame Performance",
					Description: fmt.Sprintf("Function '%s' on main thread exceeds 60fps budget", fn.FunctionName),
					Impact:      fmt.Sprintf("Max %.2fms per frame (target: 16.67ms), avg %.2fms",
						fn.MaxTimePerFrameMs, fn.AvgTimePerFrameMs),
					Suggestion:  "Optimize or move to worker thread to maintain 60fps",
					Value:       fn.MaxTimePerFrameMs,
				})
			}

			// Inconsistent frame times (high variance)
			variance := fn.MaxTimePerFrameMs / (fn.AvgTimePerFrameMs + 0.001) // Avoid div by 0
			if variance > 5.0 && fn.AvgTimePerFrameMs > 1.0 {
				issues = append(issues, PerformanceIssue{
					Severity:    "medium",
					Category:    "Inconsistent Performance",
					Description: fmt.Sprintf("Function '%s' has highly variable frame times", fn.FunctionName),
					Impact:      fmt.Sprintf("Max/Avg ratio: %.1fx (max: %.2fms, avg: %.2fms)",
						variance, fn.MaxTimePerFrameMs, fn.AvgTimePerFrameMs),
					Suggestion:  "Inconsistent performance causes stuttering. Investigate what causes occasional slowdowns",
					Value:       variance,
				})
			}
		}

		// Session-level analysis
		if data.TotalFrames > 0 {
			issues = append(issues, PerformanceIssue{
				Severity:    "info",
				Category:    "Session Info",
				Description: fmt.Sprintf("Profiling session: %s", data.SessionName),
				Impact:      fmt.Sprintf("Captured %d frames with %d unique functions",
					data.TotalFrames, data.TotalFunctions),
				Suggestion:  "Analysis based on this profiling session",
				Value:       float64(data.TotalFrames),
			})
		}
	}

	return issues
}

func analyzeThreadPerformance(data *FrameProData) []PerformanceIssue {
	issues := []PerformanceIssue{}

	// Group functions by thread
	threadStats := make(map[string]*ThreadStats)

	for _, fn := range data.Functions {
		threadKey := fmt.Sprintf("%s (ID:%d)", fn.ThreadName, fn.ThreadID)
		if _, exists := threadStats[threadKey]; !exists {
			threadStats[threadKey] = &ThreadStats{
				ThreadName: fn.ThreadName,
				ThreadID:   fn.ThreadID,
				IsMainThread: fn.IsMainThread,
				IsRenderThread: fn.IsRenderThread,
				Functions: []FrameProFunction{},
			}
		}
		threadStats[threadKey].TotalTime += fn.TotalTimeMs
		threadStats[threadKey].Functions = append(threadStats[threadKey].Functions, fn)
		if fn.ThreadUtilizationPercent > threadStats[threadKey].MaxUtilization {
			threadStats[threadKey].MaxUtilization = fn.ThreadUtilizationPercent
		}
	}

	// Analyze each thread
	var mainThreadTime, renderThreadTime float64
	for _, stats := range threadStats {
		if stats.IsMainThread {
			mainThreadTime = stats.TotalTime
		}
		if stats.IsRenderThread {
			renderThreadTime = stats.TotalTime
		}

		// Check for saturated threads
		if stats.MaxUtilization > 90.0 {
			severity := "medium"
			if stats.IsMainThread || stats.IsRenderThread {
				severity = "high"
			}

			issues = append(issues, PerformanceIssue{
				Severity:    severity,
				Category:    "Thread Saturation",
				Description: fmt.Sprintf("Thread '%s' is heavily saturated", stats.ThreadName),
				Impact:      fmt.Sprintf("%.1f%% utilization with %.2fms total work across %d functions",
					stats.MaxUtilization, stats.TotalTime, len(stats.Functions)),
				Suggestion:  "Thread is running at capacity. Consider redistributing work or optimizing top functions",
				Value:       stats.MaxUtilization,
			})
		}
	}

	// Check main thread vs render thread balance
	if mainThreadTime > 0 && renderThreadTime > 0 {
		ratio := mainThreadTime / renderThreadTime
		if ratio > 2.0 || ratio < 0.5 {
			issues = append(issues, PerformanceIssue{
				Severity:    "medium",
				Category:    "Thread Balance",
				Description: "Imbalance between main thread and render thread",
				Impact:      fmt.Sprintf("Main thread: %.2fms, Render thread: %.2fms (ratio: %.2f:1)",
					mainThreadTime, renderThreadTime, ratio),
				Suggestion:  "Consider redistributing work between main and render threads for better parallelization",
				Value:       ratio,
			})
		}
	}

	return issues
}

type ThreadStats struct {
	ThreadName     string
	ThreadID       int
	IsMainThread   bool
	IsRenderThread bool
	TotalTime      float64
	MaxUtilization float64
	Functions      []FrameProFunction
}

func generateSummary(issues []PerformanceIssue) string {
	counts := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0, "info": 0}
	for _, issue := range issues {
		counts[issue.Severity]++
	}

	summary := fmt.Sprintf("Performance Analysis Summary: %d critical, %d high, %d medium, %d low priority issues detected",
		counts["critical"], counts["high"], counts["medium"], counts["low"])

	if counts["critical"] > 0 {
		summary += " - IMMEDIATE ACTION REQUIRED"
	} else if counts["high"] > 0 {
		summary += " - Optimization recommended"
	} else if counts["medium"] > 0 {
		summary += " - Moderate optimization opportunities"
	}

	return summary
}

func countBySeverity(items []map[string]interface{}, severity string) int {
	count := 0
	for _, item := range items {
		if sev, ok := item["severity"].(string); ok && sev == severity {
			count++
		}
	}
	return count
}

func generateOptimizationSuggestion(fn FrameProFunction) string {
	suggestions := []string{}

	// Thread-specific suggestions
	if fn.IsMainThread {
		suggestions = append(suggestions, "MAIN THREAD: Move to worker thread if possible")
	}
	if fn.IsRenderThread {
		suggestions = append(suggestions, "RENDER THREAD: Optimize GPU calls and state changes")
	}

	// High call count
	if fn.TotalCount > 10000 {
		suggestions = append(suggestions, "High call count - consider caching or batching")
	}

	// High thread utilization
	if fn.ThreadUtilizationPercent > 80.0 {
		suggestions = append(suggestions, fmt.Sprintf("%.1f%% thread utilization - critical optimization target", fn.ThreadUtilizationPercent))
	}

	// Variance analysis
	variance := fn.MaxTimePerFrameMs / (fn.AvgTimePerFrameMs + 0.001)
	if variance > 3.0 {
		suggestions = append(suggestions, fmt.Sprintf("High variance (%.1fx) - investigate occasional slowdowns", variance))
	}

	// Function name analysis
	funcLower := strings.ToLower(fn.FunctionName)
	if strings.Contains(funcLower, "wait") || strings.Contains(funcLower, "sleep") {
		suggestions = append(suggestions, "WAIT/SLEEP detected - may indicate synchronization issues or idle time")
	}
	if strings.Contains(funcLower, "lock") || strings.Contains(funcLower, "mutex") {
		suggestions = append(suggestions, "Lock contention possible - review synchronization strategy")
	}
	if strings.Contains(funcLower, "physics") {
		suggestions = append(suggestions, "Physics calculation - review collision detection and simulation complexity")
	}
	if strings.Contains(funcLower, "render") || strings.Contains(funcLower, "draw") {
		suggestions = append(suggestions, "Rendering function - check draw calls, batching, and GPU state changes")
	}
	if strings.Contains(funcLower, "audio") {
		suggestions = append(suggestions, "Audio processing - ensure streaming and buffering are optimized")
	}
	if strings.Contains(funcLower, "update") {
		suggestions = append(suggestions, "Update loop - review what systems are being updated and their frequency")
	}

	if len(suggestions) == 0 {
		return "Review algorithm complexity and consider profiling child functions"
	}

	return strings.Join(suggestions, "; ")
}

func generateFunctionSuggestions(fn FrameProFunction) []string {
	suggestions := []string{}

	// High call count
	if fn.TotalCount > 10000 {
		suggestions = append(suggestions, "Consider caching or memoization to reduce repeated calculations")
		suggestions = append(suggestions, "Evaluate if call frequency can be reduced through batching")
	}

	// High thread utilization
	if fn.ThreadUtilizationPercent > 90.0 {
		suggestions = append(suggestions, fmt.Sprintf("Thread %.1f%% saturated - this is a critical optimization target", fn.ThreadUtilizationPercent))
	}

	// Main thread specific
	if fn.IsMainThread && fn.AvgTimePerFrameMs > 5.0 {
		suggestions = append(suggestions, "Main thread function taking significant time - consider moving to worker thread")
	}

	// Frame spike analysis
	variance := fn.MaxTimePerFrameMs / (fn.AvgTimePerFrameMs + 0.001)
	if variance > 3.0 {
		suggestions = append(suggestions, fmt.Sprintf("Inconsistent performance (max/avg: %.1fx) - investigate occasional slowdowns", variance))
	}

	// Average time per call
	avgTimePerCall := fn.TotalTimeMs / float64(fn.TotalCount+1)
	if avgTimePerCall > 0.1 && fn.TotalCount > 1000 {
		suggestions = append(suggestions, fmt.Sprintf("High avg time per call (%.3fms) - review algorithm complexity", avgTimePerCall))
	}

	// Function name-based suggestions
	funcLower := strings.ToLower(fn.FunctionName)
	if strings.Contains(funcLower, "event") && strings.Contains(funcLower, "wait") {
		suggestions = append(suggestions, "Event waiting - may indicate thread synchronization overhead or idle time")
	}
	if strings.Contains(funcLower, "physics") {
		suggestions = append(suggestions, "Physics - review collision detection, spatial partitioning, and simulation timestep")
	}
	if strings.Contains(funcLower, "render") || strings.Contains(funcLower, "draw") {
		suggestions = append(suggestions, "Rendering - optimize draw calls, use instancing, check GPU state changes")
	}
	if strings.Contains(funcLower, "update") {
		suggestions = append(suggestions, "Update function - profile child systems and consider update frequency")
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Profile child functions to identify specific bottlenecks")
	}

	return suggestions
}

func analyzeFrameIssues(slowFrames, stutters int, actualFPS, targetFPS float64) []string {
	issues := []string{}

	if actualFPS < targetFPS*0.8 {
		issues = append(issues, fmt.Sprintf("FPS is %.1f%% below target - significant optimization needed", (1-actualFPS/targetFPS)*100))
	}

	if slowFrames > 0 {
		issues = append(issues, fmt.Sprintf("%d frames exceeded target frame time", slowFrames))
	}

	if stutters > 0 {
		issues = append(issues, fmt.Sprintf("%d stutter events detected - investigate sudden workload spikes", stutters))
	}

	if len(issues) == 0 {
		issues = append(issues, "Frame performance is within acceptable parameters")
	}

	return issues
}
