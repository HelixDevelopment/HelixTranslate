# WebSocket Monitoring System - User Guide

## Getting Started

This guide helps you monitor your translation workflows in real-time using WebSocket-based monitoring.

## What You Can Monitor

- **Translation Progress**: See real-time progress updates as your documents are translated
- **SSH Workers**: Monitor remote translation workers and their performance
- **Error Tracking**: Get immediate alerts when errors occur
- **Session History**: View past translation sessions and their results
- **Performance Metrics**: Track translation speed and worker efficiency

## Step-by-Step Instructions

### Step 1: Start the Monitoring Server

1. Open your terminal
2. Navigate to your project directory:
   ```bash
   cd /Users/milosvasic/Projects/Translate
   ```
3. Start the monitoring server:
   ```bash
   go run ./cmd/monitor-server
   ```
4. You should see output like:
   ```
   🚀 Starting Translation Monitoring Server
   📡 WebSocket server listening on ws://localhost:8090/ws
   🌐 Web dashboard available at http://localhost:8090/monitor
   ```

**Keep this terminal open** - the monitoring server needs to keep running.

### Step 2: Open the Monitoring Dashboard

1. Open your web browser
2. Go to: `http://localhost:8090/monitor`
3. You should see a monitoring dashboard with:
   - Active sessions counter
   - Connection status indicator
   - Event log area
   - Progress tracking section

**For advanced features**, you can also open: `file:///Users/milosvasic/Projects/Translate/enhanced-monitor.html`

### Step 3: Run a Translation with Monitoring

Choose one of the following options:

#### Option A: Quick Demo (Easiest)
1. Open a **new terminal window**
2. Run the basic demo:
   ```bash
   cd /Users/milosvasic/Projects/Translate
   go run demo-translation-with-monitoring-fixed.go
   ```
3. Watch the dashboard update in real-time!

#### Option B: Real AI Translation
1. **Optional**: Set up your OpenAI API key:
   ```bash
   export OPENAI_API_KEY=your_api_key_here
   ```
2. Run the real LLM demo:
   ```bash
   go run demo-real-llm-with-monitoring.go
   ```

#### Option C: SSH Worker Translation
1. **Optional**: Configure SSH worker (if you have remote workers):
   ```bash
   export SSH_WORKER_HOST=localhost
   export SSH_WORKER_USER=milosvasic
   ```
2. Run the SSH worker demo:
   ```bash
   go run demo-ssh-worker-with-monitoring.go
   ```

### Step 4: Monitor Your Translation

As your translation runs, you'll see in the dashboard:

#### Real-time Updates
- **Progress Bar**: Shows overall completion percentage
- **Current Step**: What's happening right now (reading, translating, etc.)
- **Event Log**: Detailed log of all activities
- **Session ID**: Unique identifier for your translation session

#### What the Progress Means
- **0-10%**: Reading and parsing input files
- **10-25%**: Content preparation and conversion
- **25-85%**: Actual translation work
- **85-100%**: Generating output files

#### Color-Coded Status
- 🟡 **Yellow**: Starting up or in progress
- 🟢 **Green**: Successfully completed
- 🔴 **Red**: Error occurred
- ⚪ **Gray**: Pending or idle

### Step 5: Understanding the Results

#### When Translation Completes Successfully
You'll see:
- ✅ Green "Completed" status
- 100% progress bar
- Output file location
- Total time taken
- Any worker information (for SSH translations)

#### If Errors Occur
You'll see:
- 🔴 Red error status
- Detailed error message
- Step where error occurred
- Troubleshooting suggestions

## Advanced Features

### Monitoring Multiple Sessions

1. Run multiple translations in parallel:
   ```bash
   # Terminal 1
   go run demo-translation-with-monitoring-fixed.go
   
   # Terminal 2 (simultaneously)
   go run demo-translation-with-monitoring-fixed.go
   
   # Terminal 3 (simultaneously)
   go run demo-translation-with-monitoring-fixed.go
   ```
2. The dashboard will show all active sessions

### SSH Worker Monitoring

If you have remote SSH workers configured:
- You'll see worker host and connection info
- Worker model and capacity details
- Real-time worker performance
- Connection status for each worker

### Custom Session Monitoring

1. In the dashboard, enter a specific session ID
2. Click "Monitor Session" to focus on one translation
3. View detailed progress and events for that session

## Troubleshooting Common Issues

### "Cannot connect to WebSocket server"
**Solution**: Make sure the monitoring server is running:
```bash
# Check if server is running
lsof -i :8090

# If not running, start it:
go run ./cmd/monitor-server
```

### "No events appearing in dashboard"
**Solutions**:
1. Check that a translation is actually running
2. Verify the translation process includes monitoring code
3. Check browser console for JavaScript errors

### "SSH worker connection failed"
**Solutions**:
1. Verify SSH credentials are correct
2. Test SSH connection manually:
   ```bash
   ssh milosvasic@localhost 'echo "SSH works"'
   ```
3. Check worker configuration in config files

### "LLM API authentication failed"
**Solutions**:
1. Verify your API key is set:
   ```bash
   echo $OPENAI_API_KEY
   ```
2. Set the API key:
   ```bash
   export OPENAI_API_KEY=your_actual_api_key
   ```

### "Port already in use"
**Solution**: Kill the process using port 8090:
```bash
# Find the process
lsof -i :8090

# Kill it (replace PID with actual process ID)
kill -9 <PID>

# Restart server
go run ./cmd/monitor-server
```

## Performance Tips

### For Best Experience
- Use a modern web browser (Chrome, Firefox, Safari)
- Keep the monitoring server on a stable network
- Don't run too many translations simultaneously on a slow computer

### Monitoring Multiple Workers
- Use the enhanced dashboard for better multi-worker support
- Consider increasing WebSocket timeout for slow connections
- Monitor worker capacity to avoid overloading

### Large Translation Files
- Progress updates may be less frequent for large files
- Monitor memory usage on the translation machine
- Consider breaking very large files into smaller chunks

## FAQ

**Q: Do I need to keep the monitoring server running?**
A: Yes, the monitoring server must stay running to track translations and serve the dashboard.

**Q: Can I monitor translations running on different machines?**
A: Yes! Configure the translation clients to connect to your monitoring server's IP address instead of localhost.

**Q: How long does the monitoring history last?**
A: Session history is stored in memory and lasts until you restart the monitoring server.

**Q: Is my data secure?**
A: For development, data is sent over plain WebSocket. In production, consider using WSS (WebSocket Secure) and authentication.

**Q: Can I export the monitoring data?**
A: Currently, data is displayed in the dashboard only. Export functionality is planned for future versions.

## Getting Help

If you encounter issues:

1. **Check the logs** in your terminal windows
2. **Look at browser console** (F12 → Console tab)
3. **Try the basic demo first** to ensure the system works
4. **Review the technical documentation** in `docs/WebSocket_Monitoring_Guide.md`
5. **Check configuration files** for any incorrect settings

## Keyboard Shortcuts (Dashboard)

- **Ctrl+R**: Refresh dashboard
- **Ctrl+L**: Clear event log
- **F5**: Full page refresh

## Quick Reference

| Command | Purpose |
|---------|---------|
| `go run ./cmd/monitor-server` | Start monitoring server |
| `go run demo-translation-with-monitoring-fixed.go` | Run basic demo |
| `go run demo-real-llm-with-monitoring.go` | Run real LLM demo |
| `go run demo-ssh-worker-with-monitoring.go` | Run SSH worker demo |
| `http://localhost:8090/monitor` | Basic dashboard |
| `enhanced-monitor.html` | Advanced dashboard |

## Success Checklist

Before you start monitoring real translations:

- [ ] Monitoring server running without errors
- [ ] Dashboard accessible in browser
- [ ] At least one demo runs successfully
- [ ] Real-time updates appear in dashboard
- [ ] No connection errors in browser console
- [ ] Translation output files are created

When all items are checked, you're ready to monitor your actual translation workflows!

---

**Happy Monitoring! 🚀**