# Epic: Architectural Refactoring - Separation of Concerns

## Epic Goal
Transform the Tesla Road Trip Game codebase from a monolithic architecture to a clean, layered architecture with REST API as the single source of truth, reducing code by 40% and eliminating all business logic duplication.

## Epic Description

### Existing System Context
- **Current architecture:** 1,200-line monolithic `main.go` mixing HTTP, WebSocket, session management, and game logic
- **Technology stack:** Go, Gorilla WebSocket, MCP integration, HTTP REST API
- **Integration points:** Three separate MCP implementations (HTTP proxy, embedded, stdio), WebSocket hub, HTTP handlers
- **Current issues:** Severe code duplication (~30-40%), mixed concerns, global state management

### Enhancement Details
- **What's being added/changed:** Complete architectural refactoring to introduce clean service layer, dedicated session management, and unified MCP implementation
- **How it integrates:** REST API becomes single source of truth; MCP servers become thin HTTP clients; all business logic consolidated in service layer
- **Success criteria:** 40% code reduction, 100% test coverage for service layer, zero business logic in transport layers

## Stories

### Story 1: Extract GameService Layer
*Priority: High | Estimated: 3 days*

Create `game/service/` package with clean interfaces consolidating all business logic from HTTP handlers and MCP implementations.

**Acceptance Criteria:**
- [ ] GameService interface defined with all game operations
- [ ] Service implementation using existing engine package
- [ ] All business logic moved from handlers to service
- [ ] Unit tests achieving 90%+ coverage
- [ ] Zero breaking changes to existing APIs

### Story 2: Implement Session Manager Package
*Priority: High | Estimated: 2 days*

Create dedicated `game/session/` package to centralize session management currently scattered across three files.

**Acceptance Criteria:**
- [ ] Session Manager interface with CRUD operations
- [ ] Thread-safe session storage implementation
- [ ] Session lifecycle management (creation, expiration)
- [ ] Migration of existing session logic without data loss
- [ ] Performance benchmarks showing no degradation

### Story 3: Refactor REST API as Single Source of Truth
*Priority: High | Estimated: 3 days*

Transform existing HTTP handlers into thin transport layer delegating all logic to GameService.

**Acceptance Criteria:**
- [ ] Clean REST routes in `api/server.go`
- [ ] Standardized error responses
- [ ] Request validation middleware
- [ ] All handlers delegate to GameService
- [ ] API documentation updated
- [ ] Integration tests for all endpoints

### Story 4: Convert MCP Servers to Thin HTTP Clients
*Priority: Medium | Estimated: 2 days*

Refactor both MCP implementations to be thin clients calling REST API, eliminating all duplicate business logic.

**Acceptance Criteria:**
- [ ] MCP HTTP proxy server calls REST API only
- [ ] MCP stdio server reuses HTTP client logic
- [ ] All business logic removed from MCP packages
- [ ] Both MCP modes tested through REST API
- [ ] Code duplication eliminated (target: -1,500 lines)

### Story 5: Consolidate WebSocket Hub
*Priority: Medium | Estimated: 1 day*

Extract WebSocket hub to dedicated package integrated with API server for real-time updates.

**Acceptance Criteria:**
- [ ] WebSocket hub in `transport/websocket/` package
- [ ] Clean integration with API server
- [ ] Session-aware broadcasting
- [ ] Connection lifecycle management
- [ ] No breaking changes to WebSocket protocol

### Story 6: Clean Up Main Package
*Priority: Low | Estimated: 1 day*

Reduce main.go to orchestration only, moving all remaining logic to appropriate packages.

**Acceptance Criteria:**
- [ ] main.go reduced to <200 lines
- [ ] Only orchestration and configuration in main
- [ ] All type aliases removed (use engine types directly)
- [ ] Clean dependency injection setup
- [ ] Application startup simplified

## Compatibility Requirements

- [x] All existing REST API endpoints remain unchanged
- [x] WebSocket protocol maintains backward compatibility
- [x] MCP tool interfaces remain identical
- [x] Configuration file format unchanged
- [x] Save file format preserved

## Risk Mitigation

### Primary Risks

1. **Breaking existing integrations**
   - **Mitigation:** Comprehensive integration tests before each phase
   - **Rollback:** Feature flags for gradual rollout

2. **Performance degradation from additional layers**
   - **Mitigation:** Benchmark critical paths, optimize hot spots
   - **Rollback:** Keep old implementation available via config flag

3. **Session data loss during migration**
   - **Mitigation:** Parallel run of old and new session managers
   - **Rollback:** Session export/import functionality

## Definition of Done

- [ ] All 6 stories completed with acceptance criteria met
- [ ] Code reduction achieved: Target 40% (from 6,000 to ~3,600 lines)
- [ ] Test coverage: Service layer >90%, Overall >80%
- [ ] Zero business logic in transport layers (HTTP, WebSocket, MCP)
- [ ] Performance benchmarks show no degradation
- [ ] Documentation updated with new architecture
- [ ] Clean code analysis passing (no critical issues)
- [ ] Team code review completed
- [ ] Deploy to staging environment successful

## Implementation Sequence

**Phase 1 (Week 1):** Stories 1-2 (Service Layer & Session Manager)
- Foundation work with no external API changes
- Can be tested in isolation

**Phase 2 (Week 2):** Stories 3-4 (REST API & MCP Refactor)
- Visible changes but backward compatible
- High impact on code reduction

**Phase 3 (Week 3):** Stories 5-6 (WebSocket & Cleanup)
- Final consolidation
- Achieve target metrics

## Success Metrics

- **Code Quality:** 40% reduction in total LOC
- **Maintainability:** Cyclomatic complexity <10 for all methods
- **Test Coverage:** >80% overall, >90% for service layer
- **Performance:** <5% latency increase for any operation
- **Developer Experience:** New feature implementation time reduced by 50%