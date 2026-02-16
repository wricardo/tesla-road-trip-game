# Integration Points Documentation

## Current File Structure Issues

### Conflicting Files
The following files have conflicts between old and new architecture:

1. **main.go** (old, 1200 lines) vs **main_new.go** (new, 194 lines)
   - Both declare `main()` function
   - Resolution: Backup main.go, rename main_new.go to main.go

2. **mcp_embedded.go** (old)
   - Contains duplicate MCP server implementation
   - Conflicts with new transport/mcp/client.go
   - Resolution: Remove after backup

3. **mcp_server.go** (old)
   - Another duplicate MCP implementation
   - Resolution: Remove after backup

4. **cli.go** (old)
   - Contains old CLI parsing incompatible with new architecture
   - Resolution: Remove or update to work with service layer

## Integration Points Between Old and New

### What's Preserved
- `game/engine/` - Core game logic (unchanged)
- `configs/` - Game configuration files
- `templates/` - HTML templates (if any)
- Test files (may need updates)

### What's Replaced
| Old Component | New Component | Location |
|--------------|---------------|----------|
| Session management in main.go | SessionManager | game/session/manager.go |
| Config loading in main.go | ConfigManager | game/config/manager.go |
| Business logic in handlers | GameService | game/service/game_service.go |
| HTTP handlers in main.go | REST API Server | api/server.go |
| WebSocket in main.go | WebSocket Hub | transport/websocket/hub.go |
| MCP servers (3 versions) | MCP Client | transport/mcp/client.go |

## Critical Integration Points

### 1. Session ID Format
- Old: Various formats (numeric, alphanumeric)
- New: Standardized format `sess_<timestamp>`
- Compatibility: SessionManager handles case-insensitive lookups

### 2. API Endpoints
- Old: `/api?sessionId=xxx`
- New: `/api/sessions/{id}/operation`
- Bridge: Both patterns supported during migration

### 3. WebSocket Protocol
- Old: Global broadcast
- New: Session-specific channels
- Protocol: Same JSON structure maintained

### 4. MCP Communication
- Old: Direct game engine manipulation
- New: HTTP calls to REST API
- Impact: MCP server must have HTTP server running

## Dependencies Between Components

```
main.go
  ├── api.Server (requires GameService)
  │     └── GameService (requires SessionManager, ConfigManager)
  │           ├── SessionManager
  │           └── ConfigManager
  ├── websocket.Hub (standalone)
  └── mcp.Client (requires HTTP server URL)
        └── Makes HTTP calls to api.Server
```

## Environment Variables

- `GAME_HTTP_SERVER` - URL for MCP to connect to REST API
- Default: `http://localhost:8080`

## Configuration Files

### Required Configs
- `configs/classic.json` - Default configuration
- `configs/easy.json` - Easy mode
- `configs/medium_maze.json` - Medium difficulty
- `configs/challenge.json` - Challenge mode

### Config Loading Order
1. Command line flag `-config`
2. Default config in ConfigManager
3. Per-session config override

## Testing Integration Points

### Unit Test Updates Needed
- `game_engine_test.go` - Should work as-is
- `validation_test.go` - May need service layer mocks
- `game_features_test.go` - May need session manager mocks

### Integration Tests Needed
- REST API endpoints
- WebSocket broadcasting
- MCP to REST API communication
- Session isolation

## Migration Checklist

- [ ] Backup old files
- [ ] Remove conflicting files
- [ ] Update imports in test files
- [ ] Verify all configs load
- [ ] Test session creation
- [ ] Test game operations
- [ ] Test WebSocket updates
- [ ] Test MCP integration
- [ ] Update CI/CD scripts
- [ ] Update documentation

## Rollback Plan

If issues arise:
1. Stop new server
2. Restore from backup directory
3. Rebuild with old main.go
4. Restart services

## Post-Migration Cleanup

After successful migration:
1. Remove backup files after 1 week
2. Archive old architecture documentation
3. Update README with new architecture
4. Remove this integration document