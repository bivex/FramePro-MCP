# FramePro-MCP: Performance Analysis Server

MCP (Model Context Protocol) server for analyzing FramePro profiling data with senior-level performance optimization insights.

## Features

### 4 Analysis Tools

1. **analyze_performance** - Comprehensive performance analysis
   - Detects CPU hotspots, frame issues, thread saturation
   - Focus areas: `cpu`, `frames`, `threads`, or `all`
   - Severity-based prioritization (critical/high/medium/low)

2. **find_hotspots** - Top N most expensive functions
   - Ranked by total time consumption
   - Detailed metrics: time, calls, utilization
   - Function-specific optimization suggestions

3. **analyze_frame_times** - Frame performance analysis
   - FPS estimation based on main thread work
   - Frame spike detection
   - Main thread bottleneck identification

4. **compare_profiles** - Profile comparison
   - Detects performance regressions and improvements
   - Shows percentage changes
   - Identifies new and removed functions

### Expert Analysis Capabilities

- **Thread-Aware**: Identifies Main Thread, Render Thread, Worker Threads
- **Pattern Recognition**: Detects Wait, Lock, Physics, Render, Update patterns
- **Variance Analysis**: Finds inconsistent performance (stuttering)
- **Automatic Prioritization**: Critical issues flagged first
- **Context-Specific Suggestions**: Tailored recommendations per function

## Installation

### Prerequisites
- Go 1.21+ (for building from source)
- FramePro JSON exports

### Quick Setup

1. **The executable is already built**:
   ```
   c:\Program Files\PureDevSoftware\FramePro\FrameProReader\FramePro-MCP\framepro-mcp.exe
   ```

2. **Configure Claude Desktop**:

   Edit `%APPDATA%\Claude\claude_desktop_config.json`:
   ```json
   {
     "mcpServers": {
       "framepro": {
         "command": "cmd",
         "args": [
           "/c",
           "c:\\Program Files\\PureDevSoftware\\FramePro\\FrameProReader\\FramePro-MCP\\framepro-mcp.exe"
         ],
         "env": {
           "FRAMEPRO_DATA_DIR": "c:\\Program Files\\PureDevSoftware\\FramePro"
         }
       }
     }
   }
   ```

   Or for Cursor/Claude Code, edit `c:\Users\Admin\.cursor\mcp.json`:
   ```json
   {
     "mcpServers": {
       "framepro": {
         "command": "cmd",
         "args": [
           "/c",
           "c:\\Program Files\\PureDevSoftware\\FramePro\\FrameProReader\\FramePro-MCP\\framepro-mcp.exe"
         ],
         "env": {
           "FRAMEPRO_DATA_DIR": "c:\\Program Files\\PureDevSoftware\\FramePro"
         },
         "disabled": false,
         "autoApprove": []
       }
     }
   }
   ```

3. **Restart your editor**

4. **Verify server is running**:
   - Check MCP servers list
   - Should see: "Found 4 tools, 0 prompts, and 0 resources"

## Usage

### Supported File Formats

The server works with **real FramePro JSON exports**:

- ✅ `*_functions_analysis.json` - Aggregated function data (recommended)
- ✅ `*_frame_analysis.json` - Per-frame detailed data

### File Path Options

**Relative paths** (automatically resolved):
```
"02_10_2025+03_33_23_functions_analysis.json"
```
→ Searches in `FRAMEPRO_DATA_DIR`

**Absolute paths**:
```
"c:\Program Files\PureDevSoftware\FramePro\02_10_2025+03_33_23_functions_analysis.json"
```

### Natural Language Usage

Simply ask Claude/Cursor in natural language:

**Example 1: Full Analysis**
```
Analyze the performance in "02_10_2025+03_33_23_functions_analysis.json"
```

**Example 2: Find Hotspots**
```
Find the top 15 performance hotspots in "02_10_2025+03_33_23_functions_analysis.json"
```

**Example 3: Frame Analysis**
```
Analyze frame times with target FPS 60
```

**Example 4: Compare Profiles**
```
Compare "baseline_functions_analysis.json" with "current_functions_analysis.json"
```

## Performance Thresholds

### Critical Issues ⚠️
- **Main thread** functions >16.67ms per frame
- **Any function** >500ms total time
- **Thread utilization** >95%

### High Priority 🔶
- **Main thread** functions >5ms average
- **Any function** >100ms total time
- **Thread utilization** >80%
- **Frame spikes** on main thread

### Medium Priority 🔸
- **High variance** (max/avg ratio >5x)
- **High call count** (>10,000 calls) with >50ms total
- **Thread imbalance** (>2:1 ratio between threads)

## What You'll Get

### Analysis Output Example

```json
{
  "severity": "critical",
  "category": "CPU Hotspot",
  "description": "Function 'Event Wait' on TaskGraph Render Thread 2 (RENDER THREAD - affects FPS!) consumes excessive CPU time",
  "impact": "37150.96ms total (146.84ms avg/frame), 6426 total calls, 100.0% thread utilization",
  "suggestion": "RENDER THREAD: Optimize GPU calls and state changes; 100.0% thread utilization - critical optimization target; WAIT/SLEEP detected - may indicate synchronization issues or idle time",
  "value": 37150.9578
}
```

### Optimization Suggestions

The tool provides context-aware recommendations:

- **For Main Thread**: "Move to worker thread if possible"
- **For High Variance**: "Investigate occasional slowdowns causing stuttering"
- **For Frequent Calls**: "Consider caching or batching"
- **For Wait/Sleep**: "May indicate synchronization issues"
- **For Physics**: "Review collision detection and spatial partitioning"
- **For Render**: "Optimize draw calls, use instancing, check GPU state changes"

## Real Data Format

FramePro exports contain:

```json
{
  "SessionName": "02_10_2025+03_33_23",
  "TotalFrames": 254,
  "TotalFunctions": 212,
  "Functions": [
    {
      "FunctionName": "Event Wait",
      "ThreadId": 5032,
      "ThreadName": "TaskGraph Render Thread 2",
      "TotalTimeMs": 37150.9578,
      "TotalCount": 6426,
      "MaxTimePerFrameMs": 519.1692,
      "MaxCountPerFrame": 38,
      "AvgTimePerFrameMs": 146.84173,
      "AvgCountPerFrame": 25.399,
      "ThreadUtilizationPercent": 100.0,
      "IsMainThread": false,
      "IsRenderThread": true,
      "IsWorkerThread": true,
      "ThreadPriority": 0
    }
  ]
}
```

## Workflow

### Optimization Process

1. **Capture Baseline**
   - Profile your application with FramePro
   - Export to JSON

2. **Analyze**
   - Use `analyze_performance` or `find_hotspots`
   - Review prioritized issues

3. **Optimize**
   - Apply suggested improvements
   - Focus on critical main thread issues first

4. **Verify**
   - Re-profile after optimization
   - Use `compare_profiles` to verify improvements

5. **Iterate**
   - Continue until performance targets met

## Troubleshooting

### Server Not Starting
- Check path in config has no spaces issues (use `cmd /c`)
- Verify executable exists at specified location
- Check Cursor/Claude logs for errors

### File Not Found
- Use absolute path to test
- Verify `FRAMEPRO_DATA_DIR` is set correctly
- Check file exists: `dir "c:\Program Files\PureDevSoftware\FramePro\*.json"`

### JSON Parse Errors
- Ensure file is valid JSON (not binary .framepro file)
- Export from FramePro to JSON format
- Check file has `Functions` array with required fields

### No Analysis Results
- Verify file contains function data
- Check `TotalFunctions` > 0
- Ensure functions have `TotalTimeMs`, `FunctionName`, etc.

## Technical Details

### Architecture
- **Language**: Go 1.21+
- **Framework**: mcp-go (official MCP implementation)
- **Protocol**: Model Context Protocol (MCP)
- **Interface**: stdio-based communication

### Performance
- Fast JSON parsing (handles 26MB+ files)
- Efficient sorting and analysis algorithms
- Minimal memory footprint
- Smart path resolution

### Environment Variables
- `FRAMEPRO_DATA_DIR` - Base directory for FramePro JSON files

## Building from Source

```bash
cd "c:\Program Files\PureDevSoftware\FramePro\FrameProReader\FramePro-MCP"
go build -o framepro-mcp.exe
```

## Documentation Files

- **README.md** (this file) - Complete documentation
- **README_RU.md** - Russian documentation
- **ИНСТРУКЦИЯ.md** - Quick start guide (Russian)
- **REAL_FORMAT.md** - FramePro data format details
- **QUICKSTART.md** - Quick start guide (English)
- **ACTIVATION.md** - Configuration and activation
- **SUCCESS.md** - Server status and testing

## License

This tool is designed for use with FramePro profiler data.

## Support

For issues or questions:
- Check the troubleshooting section above
- Review log files in your editor
- Verify JSON format matches FramePro export structure

---

**Ready to optimize your application!** 🚀

Built with senior-level performance analysis expertise.
