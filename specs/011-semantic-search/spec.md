# Feature Specification: Semantic Search

**Feature Branch**: `011-semantic-search`  
**Created**: 2026-04-05  
**Status**: Draft  
**Input**: User description: "We want semantic search instead of simple text and fuzzy search. Use asg017/sqlite-vec. Package all dependencies in Docker. Search should work fast and scale for 100s of notes. Implement API-level integration tests without mocking, using testcontainers or Docker-based setup. Add test cases for all endpoints and build/make scripts."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Semantic Note Search (Priority: P1)

A user types a conceptual query into the search box -- for example, "how to deploy containers" -- and the system returns notes that are semantically related, even if those exact words don't appear in the note. Notes about "Docker packaging" or "Kubernetes setup" should rank highly. The search replaces the existing fuzzy text search with meaning-based matching.

**Why this priority**: This is the core value of the feature. Without semantic search working, nothing else matters.

**Independent Test**: Can be tested by creating several notes with known content, searching with a conceptual query, and verifying that semantically related notes appear in the results ranked by relevance.

**Acceptance Scenarios**:

1. **Given** a user has notes titled "Docker deployment guide" and "Kubernetes cluster setup", **When** the user searches for "container orchestration", **Then** both notes appear in the results, ranked by semantic relevance.
2. **Given** a user has a note about "cooking pasta recipes", **When** the user searches for "container orchestration", **Then** the cooking note does not appear in the results.
3. **Given** a user has 200 notes, **When** the user performs a semantic search, **Then** results are returned within 2 seconds.

---

### User Story 2 - Embedding Generation on Note Save (Priority: P1)

When a user creates or updates a note, the system automatically generates a vector embedding from the note's title and body content. This embedding is stored and used for future searches. The user does not need to take any action -- embedding generation is seamless and invisible.

**Why this priority**: Without embeddings being generated and stored, semantic search cannot function. This is a prerequisite for User Story 1.

**Independent Test**: Can be tested by saving a note and verifying that an embedding record is created in the database for that note.

**Acceptance Scenarios**:

1. **Given** a user creates a new note with title "Meeting notes" and body content, **When** the note is saved, **Then** a vector embedding is generated and stored for that note.
2. **Given** a user updates an existing note's body, **When** the note is saved, **Then** the embedding is regenerated to reflect the updated content.
3. **Given** a user archives a note, **When** searching active notes, **Then** the archived note does not appear in active search results.

---

### User Story 3 - Bulk Embedding Rebuild (Priority: P2)

When the system starts up or when embeddings need to be regenerated (e.g., after a model change or data migration), the system can rebuild all embeddings for all existing notes in one operation. This ensures all notes are searchable even if they were created before semantic search was added.

**Why this priority**: Essential for initial deployment and ongoing maintenance, but not part of the day-to-day user experience.

**Independent Test**: Can be tested by starting the application with existing notes that have no embeddings and verifying that all notes get embeddings generated during startup.

**Acceptance Scenarios**:

1. **Given** 100 existing notes with no embeddings, **When** the system performs a bulk rebuild, **Then** all 100 notes have embeddings generated.
2. **Given** a bulk rebuild is in progress, **When** the process completes, **Then** all notes are searchable via semantic search.

---

### User Story 4 - Search Across Active and Archived Notes (Priority: P2)

The search respects the existing separation between active and archived notes. When a user searches in the active notes section, only active notes are returned. When searching in the archive section, only archived notes are returned. Semantic search works identically in both contexts.

**Why this priority**: Maintains consistency with existing application behavior. Important but builds on the core search capability.

**Independent Test**: Can be tested by creating active and archived notes, performing searches in each section, and verifying proper filtering.

**Acceptance Scenarios**:

1. **Given** a user has both active and archived notes about "project planning", **When** searching in the active section, **Then** only active notes appear in results.
2. **Given** a user has archived notes about "project planning", **When** searching in the archive section, **Then** only archived notes appear in results.

---

### User Story 5 - Docker Distribution with All Dependencies (Priority: P2)

The Docker image includes everything needed for semantic search to work out of the box -- the embedding model, vector search extensions, and all runtime dependencies. A user pulling and running the Docker image should have semantic search working without any additional setup, external services, or API keys.

**Why this priority**: Critical for distribution and ease of use, but only relevant after the core search functionality works.

**Independent Test**: Can be tested by building the Docker image, running a container, creating notes, and verifying that semantic search works without any external dependencies.

**Acceptance Scenarios**:

1. **Given** a freshly pulled Docker image, **When** a user runs the container and creates notes, **Then** semantic search works without additional configuration.
2. **Given** the Docker image, **When** inspecting it, **Then** no external API keys or network-dependent embedding services are required.

---

### User Story 6 - API-Level Integration Tests (Priority: P1)

All search and note endpoints are covered by integration tests that run against a real application instance (no mocking). Tests use a Docker-based setup (e.g., testcontainers) to spin up the application and exercise the API end-to-end. The test suite is integrated into the build system via Makefile targets.

**Why this priority**: Testing infrastructure is essential for confidence in the implementation and ongoing maintenance. It validates all other stories actually work correctly.

**Independent Test**: Can be tested by running the integration test suite and verifying all tests pass.

**Acceptance Scenarios**:

1. **Given** the integration test suite, **When** tests are run against a fresh application instance, **Then** all search endpoint tests pass.
2. **Given** the integration test suite, **When** tests are run, **Then** note CRUD endpoint tests also pass (create, read, update, delete, archive).
3. **Given** a developer wants to run integration tests, **When** they invoke the appropriate Makefile target, **Then** the Docker-based test environment is set up and tests execute automatically.

---

### Edge Cases

- What happens when a note has no body content (only a title)? The embedding should be generated from the title alone.
- What happens when a note body is extremely long (>10,000 words)? The system should handle it gracefully, potentially truncating to the model's input limit.
- What happens when the embedding model fails to generate an embedding for a note? The note should still be saved and will appear in search results via text-based fallback matching until the embedding is successfully generated.
- What happens when two notes have very similar content? Both should appear in results, ranked by their individual relevance scores.
- What happens when the search query is empty or only whitespace? The system should return an empty result set or all notes (consistent with current behavior).
- What happens when no notes meet the similarity threshold? The system should return an empty result set.
- What happens when a note is deleted? Its embedding should also be removed.

## Clarifications

### Session 2026-04-05

- Q: Should semantic search trigger on every keystroke (as today), after a debounce pause, or only on explicit submit? → A: Debounce after user pauses typing, with a configurable delay value.
- Q: How should search results be filtered for relevance? → A: Apply both a minimum similarity threshold and a maximum result cap.
- Q: The current build uses CGO_ENABLED=0. sqlite-vec and local embedding inference require CGO. Enable CGO or find pure-Go alternatives? → A: Enable CGO for the build.
- Q: Should notes without embeddings be excluded from search, or fall back to text matching? → A: Fall back to text-based matching (title/tag) for notes lacking embeddings. Additionally, provide a feature toggle to disable semantic search entirely and use text matching instead. Embeddings should still be generated regardless of the toggle.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST generate vector embeddings from note title and body content when a note is created or updated.
- **FR-002**: System MUST store vector embeddings in a way that supports efficient similarity search (using sqlite-vec).
- **FR-003**: System MUST perform semantic similarity search when a user submits a search query, returning notes ranked by semantic relevance.
- **FR-004**: System MUST use semantic search by default on the search endpoints, replacing the existing fuzzy search. When the semantic search toggle is disabled, the system MUST use text-based matching instead.
- **FR-005**: System MUST respect the active/archived note separation during search (same behavior as current fuzzy search).
- **FR-006**: System MUST generate embeddings using a local model bundled within the application -- no external API calls or services required.
- **FR-007**: System MUST rebuild all embeddings on startup for notes that do not yet have embeddings (backfill for existing notes).
- **FR-008**: System MUST remove embeddings when a note is deleted.
- **FR-009**: The Docker image MUST include all dependencies for semantic search (embedding model, sqlite-vec extension, runtime libraries).
- **FR-010**: The build system MUST include Makefile targets for running integration tests.
- **FR-011**: Integration tests MUST cover all note and search API endpoints without mocking, using a Docker-based test environment.
- **FR-012**: System MUST handle notes with no body content by generating embeddings from the title alone.
- **FR-013**: System MUST handle very long note content gracefully (truncation to model input limits).
- **FR-014**: Search MUST use a debounce mechanism, triggering only after the user pauses typing. The debounce delay MUST be configurable.
- **FR-015**: Search results MUST be filtered by a minimum similarity threshold, excluding notes below it. Results MUST also be capped at a maximum number. Both values should be reasonable defaults.
- **FR-016**: For notes that do not yet have embeddings (e.g., during bulk rebuild), the system MUST fall back to text-based matching (title/tag) so those notes still appear in search results.
- **FR-017**: System MUST provide a feature toggle to disable semantic search and use text-based matching instead. When the toggle is off, embeddings MUST still be generated and maintained in the background.
- **FR-018**: Embedding generation MUST continue regardless of whether semantic search is enabled or disabled, so that switching the toggle on takes effect immediately without a rebuild.

### Key Entities

- **Note Embedding**: A vector representation of a note's content (title + body), associated one-to-one with a note. Used for similarity comparison during search.
- **Embedding Model**: A local machine learning model used to convert text into vector embeddings. Bundled with the application, requiring no external services.
- **Search Query Embedding**: A vector representation of the user's search query, generated on-the-fly and compared against stored note embeddings.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users searching for a concept find relevant notes even when exact keywords don't match, with at least 80% of top-5 results being relevant to the query intent.
- **SC-002**: Search results are returned within 2 seconds for a corpus of 500 notes.
- **SC-003**: The Docker image runs semantic search out of the box with zero external dependencies or configuration.
- **SC-004**: All API endpoints (notes CRUD + search) are covered by integration tests that pass consistently.
- **SC-005**: Embedding generation on note save adds no more than 3 seconds to the save operation.
- **SC-006**: The integration test suite can be invoked via a single Makefile target and completes within 5 minutes.

## Assumptions

- The embedding model will be a lightweight, local model suitable for running on modest hardware (no GPU required). The specific model choice is an implementation detail, but it must be small enough to bundle in a Docker image (target: under 100 MB for the model).
- The existing note title autocomplete endpoint (`/notes/autocomplete`) will continue to use the current SQL LIKE-based approach, as it serves a different purpose (quick prefix matching for wiki-links) and doesn't benefit from semantic search.
- The search UI (the search box and results display) will remain largely unchanged -- the only behavioral change is switching from keystroke-by-keystroke triggering to a debounced trigger to account for the heavier semantic search operation.
- The application is single-user or low-concurrency (consistent with the existing GitHub OAuth single-user design), so embedding generation does not need to handle high-concurrency write scenarios.
- sqlite-vec will be used as a SQLite extension for vector storage and search, keeping the single-database architecture.
- The Go build will switch from CGO_ENABLED=0 to CGO_ENABLED=1 to support sqlite-vec and local embedding inference. The Docker image and CI pipeline will be updated accordingly (e.g., Alpine with musl for static linking).
- Integration tests will use the existing Docker image to run the application, ensuring tests validate the same artifact that gets deployed.
