# Feature Specification: Product Landing Page with Feature Showcase

**Feature Branch**: `019-landing-page`  
**Created**: 2026-04-29  
**Status**: Draft  
**Input**: User description: "Specify all the product features in elegant way on the login page. Keep it simple, straight to the point, attractive for developers and power users, highlight all capabilities. Add elements in the home page for SEO. Give attribution to tldraw, excalidraw and mermaid (just links to those products github page should be enough)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Feature Showcase on Login Page (Priority: P1)

A visitor (unauthenticated user) arrives at the YANT login page and immediately sees a concise, well-structured list of product capabilities. The page is designed for developers and power users: no marketing fluff, just a clear statement of what the tool does and why it matters. The visitor can scan the feature list in seconds and decide whether to sign in.

**Why this priority**: The login page is the only page unauthenticated visitors see. It is the primary conversion surface and the only opportunity to communicate value before sign-in.

**Independent Test**: Can be fully tested by visiting the root URL without authentication and verifying that all product capabilities are listed, the page is visually appealing, and the sign-in button remains prominently accessible.

**Acceptance Scenarios**:

1. **Given** an unauthenticated visitor, **When** they navigate to the application root, **Then** they see a landing/login page that lists all major product features in a scannable format alongside the sign-in option.
2. **Given** an unauthenticated visitor viewing the login page, **When** they read the feature list, **Then** each feature is described in one short line (no paragraphs), grouped by category.
3. **Given** an unauthenticated visitor, **When** they view the page, **Then** they see attribution links to tldraw, Excalidraw, and Mermaid (linking to their respective GitHub pages).
4. **Given** an authenticated user, **When** they navigate to the root URL, **Then** they are redirected to `/notes` as before (login page is not shown).

---

### User Story 2 - SEO-Optimized Home Page (Priority: P2)

The login/landing page includes standard SEO elements so that search engines can discover, index, and surface the application for relevant queries (e.g., "self-hosted markdown notes", "developer note-taking tool").

**Why this priority**: SEO drives organic discovery. Without proper meta tags and semantic HTML, the page will not appear in search results.

**Independent Test**: Can be validated by inspecting the page source for the presence of meta description, Open Graph tags, semantic HTML headings, and structured content.

**Acceptance Scenarios**:

1. **Given** the landing page HTML, **When** inspected, **Then** it contains a `<meta name="description">` tag with a concise product summary.
2. **Given** the landing page HTML, **When** inspected, **Then** it contains Open Graph tags (`og:title`, `og:description`, `og:type`).
3. **Given** the landing page, **When** rendered, **Then** it uses semantic HTML (`<h1>`, `<h2>`, `<section>`, `<footer>`) for the feature sections.
4. **Given** the landing page, **When** inspected, **Then** the page title (`<title>`) is descriptive (e.g., "YANT - Self-hosted Markdown Notes for Developers").

---

### User Story 3 - Third-Party Attribution (Priority: P2)

The landing page gives visible attribution to the open-source drawing and diagramming tools used in the product: tldraw, Excalidraw, and Mermaid. Attribution appears as simple text with links to each project's GitHub repository.

**Why this priority**: Attribution is a license compliance and community good-citizenship requirement. It must be visible but should not distract from the product's own feature list.

**Independent Test**: Can be tested by verifying that three external links are present and point to the correct GitHub repositories.

**Acceptance Scenarios**:

1. **Given** the landing page, **When** rendered, **Then** it displays attribution text with links to `https://github.com/tldraw/tldraw`, `https://github.com/excalidraw/excalidraw`, and `https://github.com/mermaid-js/mermaid`.
2. **Given** the attribution links, **When** clicked, **Then** each opens the correct GitHub repository page (links use `target="_blank"` and `rel="noopener"`).
3. **Given** the landing page layout, **When** viewed, **Then** the attribution section is visually distinct from the feature list (e.g., in a footer or "Powered by" section) and does not dominate the page.

---

### Edge Cases

- What happens when the GitHub OAuth is not configured? The page should still render the feature list and attribution but hide the sign-in button (or show a "Sign-in unavailable" message).
- What happens on very narrow screens (mobile)? The feature list and attribution should remain readable and not overflow.
- What happens when a logged-in user navigates directly to `/login`? They should be redirected to `/notes`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The login page MUST display a product name and one-line tagline that communicates the core value proposition (self-hosted, Markdown, developer-focused).
- **FR-002**: The login page MUST list all major product features, grouped into logical categories (e.g., Writing, Search, Drawing, Collaboration, Organization).
- **FR-003**: Each feature MUST be described in a single short phrase (no multi-sentence descriptions).
- **FR-004**: The page MUST include a `<meta name="description">` tag summarizing the product.
- **FR-005**: The page MUST include Open Graph meta tags (`og:title`, `og:description`, `og:type` at minimum).
- **FR-006**: The page MUST use semantic HTML elements (`<h1>`, `<h2>`, `<section>`, `<footer>`) for content structure.
- **FR-007**: The page title MUST be descriptive and include the product name and primary use case.
- **FR-008**: The page MUST display attribution links to tldraw (https://github.com/tldraw/tldraw), Excalidraw (https://github.com/excalidraw/excalidraw), and Mermaid (https://github.com/mermaid-js/mermaid).
- **FR-009**: Attribution links MUST open in a new tab with `rel="noopener"`.
- **FR-010**: The sign-in button MUST remain prominent and accessible without scrolling on standard desktop viewports.
- **FR-011**: The page MUST be responsive and readable on mobile devices.
- **FR-012**: Authenticated users visiting the login page MUST be redirected to `/notes`.

### Key Entities

- **Feature Category**: A grouping of related capabilities (e.g., "Writing", "Search", "Drawing"). Each category has a short name and a list of feature descriptions.
- **Attribution Entry**: A third-party project name and its GitHub URL.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All product capabilities listed in the README "Features" section are represented on the landing page.
- **SC-002**: The page passes basic SEO validation (meta description present, Open Graph tags present, semantic HTML structure, descriptive page title).
- **SC-003**: Attribution links for tldraw, Excalidraw, and Mermaid are present and point to the correct GitHub repositories.
- **SC-004**: The page renders correctly on viewports from 320px to 1920px wide without horizontal scrolling or content overflow.
- **SC-005**: A visitor can identify all product capabilities and reach the sign-in button within 10 seconds of page load.
- **SC-006**: The page loads in under 2 seconds on a standard connection (no heavy assets required for the landing page).

## Assumptions

- The landing page is the same URL as the current login page (`/login`). No separate marketing site is needed.
- The feature list is static content rendered server-side (no JavaScript required for the feature showcase).
- The visual design should be clean and minimal, matching the existing application aesthetic (dark nav, light content area, system fonts).
- Only the three specified projects (tldraw, Excalidraw, Mermaid) require attribution. EasyMDE and other libraries do not need visible attribution on this page.
- The landing page does not need analytics tracking or cookie consent banners.
