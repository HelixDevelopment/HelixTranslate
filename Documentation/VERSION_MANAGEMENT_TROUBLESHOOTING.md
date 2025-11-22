# Version Management Troubleshooting Guide

## Overview

This guide provides solutions for common issues encountered with the version management system in the distributed translation platform. Issues are organized by symptom, with diagnostic steps and resolution procedures.

## Quick Diagnostics

### System Health Check
```bash
# Check coordinator health
curl -s http://localhost:8080/health | jq .

# Check worker health
curl -k -s https://worker01:8443/health | jq .

# Get version information
curl -s http://localhost:8080/api/v1/version | jq .
curl -k -s https://worker01:8443/api/v1/version | jq .
```

### Log Analysis Commands
```bash
# Recent version management events
grep "worker_" /var/log/translator/events.log | tail -20

# Update operation logs
grep "update" /var/log/translator/distributed.log | tail -10

# Backup operations
ls -la /tmp/translator-backups/
ls -la /tmp/translator-updates/
```

## Issue Resolution Guide

### 1. Worker Version Check Failures

#### Symptom: "failed to query worker version"
```
CheckWorkerVersion failed: failed to query worker version: dial tcp 192.168.1.100:8443: connect: connection refused
```

**Possible Causes:**
- Worker service not running
- Network connectivity issues
- Firewall blocking connections
- Incorrect worker configuration

**Diagnostic Steps:**
```bash
# Check if worker is running
ssh worker01 "ps aux | grep translator"

# Test network connectivity
telnet worker01 8443

# Check worker logs
ssh worker01 "tail -f /var/log/translator/worker.log"
```

**Resolution:**
1. **Service Down**: Restart worker service
   ```bash
   ssh worker01 "systemctl restart translator-worker"
   ```

2. **Network Issue**: Check firewall rules
   ```bash
   # On worker
   sudo ufw status
   sudo iptables -L

   # On coordinator
   telnet worker01 8443
   ```

3. **Configuration Error**: Verify worker configuration
   ```bash
   ssh worker01 "cat /etc/translator/worker.yaml"
   ```

#### Symptom: "worker version endpoint returned status 500"
```
CheckWorkerVersion failed: worker version endpoint returned status 500
```

**Diagnostic Steps:**
```bash
# Check worker health
curl -k https://worker01:8443/health

# Check worker logs for errors
ssh worker01 "grep -i error /var/log/translator/worker.log | tail -10"
```

**Resolution:**
- Check worker service health
- Review worker error logs
- Restart worker if necessary

### 2. Update Package Creation Failures

#### Symptom: "failed to create update package: exit status 1"
```
UpdateWorker failed: failed to create update package: exit status 1
```

**Possible Causes:**
- Missing tar command
- Permission issues
- Disk space exhaustion
- Corrupted source files

**Diagnostic Steps:**
```bash
# Check available commands
which tar
tar --version

# Check disk space
df -h /tmp
df -h /

# Check permissions
ls -la /tmp/
```

**Resolution:**
1. **Missing tar**: Install tar utility
   ```bash
   # Ubuntu/Debian
   sudo apt-get install tar

   # CentOS/RHEL
   sudo yum install tar
   ```

2. **Disk Space**: Clean up space
   ```bash
   # Remove old update packages
   rm -rf /tmp/translator-updates/*
   rm -rf /tmp/translator-backups/*
   ```

3. **Permissions**: Fix directory permissions
   ```bash
   sudo chown -R translator:translator /tmp/translator-*
   ```

### 3. Update Upload Failures

#### Symptom: "failed to upload update package: connection reset"
```
UpdateWorker failed: failed to upload update package: connection reset by peer
```

**Possible Causes:**
- Network instability
- Worker rejecting uploads
- SSL/TLS certificate issues
- File size limits

**Diagnostic Steps:**
```bash
# Test basic connectivity
curl -k -I https://worker01:8443/health

# Check SSL certificates
openssl s_client -connect worker01:8443 -servername worker01 < /dev/null

# Check upload endpoint
curl -k -X POST https://worker01:8443/api/v1/update/upload \
  -F "test=@/dev/null"
```

**Resolution:**
1. **SSL Issues**: Check certificate validity
   ```bash
   # On worker
   openssl x509 -in /etc/ssl/certs/translator.crt -text -noout
   ```

2. **Network Issues**: Implement retry logic or check network stability

3. **Size Limits**: Check worker upload limits
   ```bash
   # Check worker configuration
   ssh worker01 "grep -i upload /etc/translator/worker.yaml"
   ```

### 4. Update Application Failures

#### Symptom: "failed to trigger worker update: command failed"
```
UpdateWorker failed: failed to trigger worker update: exit status 127
```

**Possible Causes:**
- Missing update scripts on worker
- Permission issues
- Corrupted update package

**Diagnostic Steps:**
```bash
# Check if update script exists
ssh worker01 "ls -la /usr/local/bin/translator-update"

# Check worker permissions
ssh worker01 "id translator"

# Verify update package integrity
ssh worker01 "tar -tzf /tmp/update-*.tar.gz | head -10"
```

**Resolution:**
1. **Missing Script**: Ensure update script is installed
   ```bash
   # Deploy update script to worker
   scp scripts/update-worker.sh worker01:/usr/local/bin/
   ssh worker01 "chmod +x /usr/local/bin/translator-update"
   ```

2. **Permissions**: Fix user permissions
   ```bash
   ssh worker01 "usermod -a -G translator $(whoami)"
   ```

### 5. Update Verification Failures

#### Symptom: "update verification failed"
```
UpdateWorker failed: update verification failed
```

**Possible Causes:**
- Update didn't apply correctly
- Version detection issues
- Worker restarted with old version

**Diagnostic Steps:**
```bash
# Check current worker version
curl -k https://worker01:8443/api/v1/version

# Check worker logs during update
ssh worker01 "grep -A 20 -B 5 'update' /var/log/translator/worker.log"

# Verify update package was applied
ssh worker01 "ls -la /usr/local/bin/translator-server"
ssh worker01 "/usr/local/bin/translator-server --version"
```

**Resolution:**
1. **Manual Verification**: Check if update was applied
2. **Restart Worker**: Force restart to pick up new version
3. **Manual Update**: Apply update manually if automated process failed

### 6. Rollback Failures

#### Symptom: "rollback failed to complete: rollback timeout"
```
rollbackWorkerUpdate failed: rollback failed to complete: rollback timeout
```

**Possible Causes:**
- Worker unresponsive during rollback
- Backup corruption
- Rollback script issues

**Diagnostic Steps:**
```bash
# Check worker responsiveness
curl -k https://worker01:8443/health

# Verify backup exists
ls -la /tmp/translator-backups/

# Check rollback logs
ssh worker01 "tail -f /var/log/translator/rollback.log"
```

**Resolution:**
1. **Worker Unresponsive**: Restart worker service
2. **Corrupted Backup**: Use alternative backup or manual recovery
3. **Network Issues**: Wait for network stabilization and retry

#### Symptom: "no backup found for worker"
```
rollbackWorkerUpdate failed: no backup found for worker worker01
```

**Resolution:**
- **Prevention**: Ensure backups are created before updates
- **Recovery**: Manual rollback or worker redeployment
- **Investigation**: Check backup cleanup policies

### 7. Version Drift Issues

#### Symptom: Workers showing different versions
```
Worker worker01: v1.0.0, Worker worker02: v1.1.0
```

**Possible Causes:**
- Staggered update deployment
- Manual worker modifications
- Update failures not properly handled

**Diagnostic Steps:**
```bash
# Check all worker versions
for worker in worker01 worker02 worker03; do
  echo "$worker: $(curl -k -s https://$worker:8443/api/v1/version | jq -r .codebase_version)"
done

# Check update history
grep "worker_update_completed" /var/log/translator/events.log | tail -10
```

**Resolution:**
1. **Trigger Updates**: Force update on drifted workers
2. **Check Configuration**: Ensure all workers use same update source
3. **Monitor Regularly**: Implement automated version drift detection

### 8. Backup Storage Issues

#### Symptom: "failed to create backup: no space left on device"
```
createWorkerBackup failed: failed to create backup directory: mkdir: no space left on device
```

**Diagnostic Steps:**
```bash
# Check disk usage
df -h /tmp

# Check backup directory size
du -sh /tmp/translator-backups/

# Check old backups
find /tmp/translator-backups/ -type d -mtime +1 | wc -l
```

**Resolution:**
1. **Clean Old Backups**:
   ```bash
   # Remove backups older than 24 hours
   find /tmp/translator-backups/ -type d -mtime +1 -exec rm -rf {} \;
   ```

2. **Increase Disk Space**: Add more storage or move backup location

3. **Configure Retention**:
   ```yaml
   # In coordinator config
   version_manager:
     backup_retention_hours: 24
     backup_dir: "/var/backups/translator"
   ```

### 9. Event System Issues

#### Symptom: Missing version management events
```
No worker_update_* events in logs
```

**Diagnostic Steps:**
```bash
# Check event bus configuration
grep "event_bus" /etc/translator/coordinator.yaml

# Verify event logging
tail -f /var/log/translator/events.log

# Test event emission
curl -X POST http://localhost:8080/api/v1/test-event \
  -H "Content-Type: application/json" \
  -d '{"type":"test","message":"version test"}'
```

**Resolution:**
1. **Event Bus Issues**: Restart coordinator service
2. **Logging Configuration**: Check log levels and destinations
3. **Event Handler**: Verify event handlers are registered

## Emergency Procedures

### Complete Worker Recovery
```bash
# 1. Stop worker service
ssh worker01 "systemctl stop translator-worker"

# 2. Backup current state (if possible)
ssh worker01 "cp -r /usr/local/translator /usr/local/translator.backup"

# 3. Reinstall from known good package
scp latest-translator.tar.gz worker01:/tmp/
ssh worker01 "cd /tmp && tar -xzf latest-translator.tar.gz -C /usr/local/"

# 4. Restart service
ssh worker01 "systemctl start translator-worker"

# 5. Verify version
curl -k https://worker01:8443/api/v1/version
```

### Coordinator Recovery
```bash
# 1. Stop coordinator
systemctl stop translator-coordinator

# 2. Clear corrupted state
rm -rf /tmp/translator-updates/*
rm -rf /tmp/translator-backups/*

# 3. Restart coordinator
systemctl start translator-coordinator

# 4. Trigger worker rediscovery
curl -X POST http://localhost:8080/api/v1/distributed/workers/discover
```

## Monitoring & Alerting

### Key Metrics to Monitor
- Update success/failure rates
- Average update duration
- Rollback frequency
- Backup storage usage
- Version drift detection

### Alert Conditions
- Update failure rate > 5%
- Rollback frequency > 10% of updates
- Version drift > 1 hour old
- Backup storage > 80% full

### Log Monitoring Patterns
```bash
# Failed updates
grep "UpdateWorker.*failed" /var/log/translator/distributed.log

# Rollback events
grep "rollback" /var/log/translator/events.log

# Version mismatches
grep "outdated" /var/log/translator/distributed.log
```

## Prevention Best Practices

### Regular Maintenance
1. **Clean backups weekly**: `find /tmp/translator-backups/ -mtime +7 -delete`
2. **Monitor disk space**: Alert when /tmp usage > 70%
3. **Update testing**: Test updates on staging environment first
4. **Version auditing**: Weekly version consistency checks

### Configuration Best Practices
```yaml
# Recommended coordinator configuration
version_manager:
  update_timeout: 300s
  rollback_timeout: 120s
  backup_retention_hours: 24
  max_concurrent_updates: 3

distributed:
  health_check_interval: 30s
  version_check_interval: 60s
  auto_update_enabled: true
```

### Network Considerations
- Use dedicated management network for updates
- Implement update rate limiting
- Configure proper timeouts for network conditions
- Use SSL/TLS for all update communications

## Support Information

### Log Files
- `/var/log/translator/coordinator.log` - Coordinator operations
- `/var/log/translator/distributed.log` - Distributed operations
- `/var/log/translator/events.log` - Event system logs
- `/var/log/translator/worker.log` - Worker-specific logs

### Diagnostic Commands
```bash
# System information
uname -a
go version
docker --version

# Service status
systemctl status translator-*

# Network diagnostics
netstat -tlnp | grep :8443
ss -tlnp | grep :8443
```

### Contact Information
- **Development Team**: For bugs and feature requests
- **Operations Team**: For production issues
- **Security Team**: For security-related version issues

---

## Quick Reference

### Emergency Commands
```bash
# Force worker update
curl -X POST http://localhost:8080/api/v1/distributed/workers/{id}/update

# Manual rollback
curl -X POST http://localhost:8080/api/v1/update/rollback \
  -H "X-Worker-ID: {worker_id}"

# Check all worker versions
for w in $(curl -s http://localhost:8080/api/v1/distributed/status | jq -r '.workers | keys[]'); do
  echo "$w: $(curl -k -s https://$w:8443/api/v1/version | jq -r .codebase_version)"
done
```

This troubleshooting guide should resolve 95% of version management issues. For unresolved issues, collect diagnostic information and contact the development team.