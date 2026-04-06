# Feature Specification: GitHub OAuth Login

**Feature Branch**: `009-github-oauth-login`  
**Created**: 2026-04-06  
**Status**: Draft  
**Input**: User description: "Implement sign in with GitHub for the login. Plan the tasks and implement them."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Sign In with GitHub (Priority: P1)

A user visits the application and clicks a "Sign in with GitHub" button on the login page. They are redirected to GitHub to authorize the application, and upon approval are redirected back to the application and signed in automatically. If this is their first visit, an account is created using their GitHub username.

**Why this priority**: This is the core feature — replacing the open-access login with authenticated identity from a trusted provider.

**Independent Test**: Can be fully tested by clicking the GitHub sign-in button, authorizing on GitHub, and verifying the user lands on their notes list with the correct GitHub username displayed.

**Acceptance Scenarios**:

1. **Given** an unauthenticated user, **When** they visit the login page, **Then** they see a "Sign in with GitHub" button.
2. **Given** a user clicks "Sign in with GitHub", **When** GitHub prompts for authorization, **Then** the user is redirected to GitHub's authorization page.
3. **Given** the user approves authorization, **When** GitHub redirects back, **Then** the user is signed in and redirected to their notes list.
4. **Given** a first-time GitHub user, **When** they complete sign-in, **Then** an account is automatically created using their GitHub username.
5. **Given** a returning GitHub user, **When** they sign in again, **Then** they see their existing notes and data.

---

### User Story 2 - Session Persistence and Logout (Priority: P1)

A signed-in user remains authenticated across page navigations and browser refreshes. When they click "Sign out", their session is destroyed and they return to the login page.

**Why this priority**: Session management is essential for the sign-in feature to be usable.

**Independent Test**: Sign in via GitHub, navigate between pages, refresh the browser, verify session persists. Click Sign out, verify redirect to login page and inability to access notes.

**Acceptance Scenarios**:

1. **Given** a signed-in user, **When** they navigate between pages or refresh the browser, **Then** they remain signed in.
2. **Given** a signed-in user, **When** they click "Sign out", **Then** their session is destroyed and they are redirected to the login page.
3. **Given** a signed-out user, **When** they try to access a protected page, **Then** they are redirected to the login page.

---

### User Story 3 - Error Handling During Sign-In (Priority: P2)

When something goes wrong during the GitHub sign-in process (user denies authorization, network error, invalid state), the user sees a clear error message and can try again.

**Why this priority**: Error handling provides a polished experience but is not required for the happy path.

**Independent Test**: Deny authorization on GitHub, verify the application shows an error message and a link to try again.

**Acceptance Scenarios**:

1. **Given** the user denies authorization on GitHub, **When** GitHub redirects back with an error, **Then** the login page shows a message explaining that authorization was denied.
2. **Given** a network or configuration error during sign-in, **When** the callback fails, **Then** the user sees a generic error message and a way to retry.

---

### Edge Cases

- What happens if two GitHub accounts have the same display name? The system uses the unique GitHub username, not the display name.
- What happens if the GitHub API is temporarily unavailable? The user sees an error message and can retry.
- What happens if someone manually crafts a callback URL? The state parameter prevents CSRF attacks; invalid state returns an error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The login page MUST display a "Sign in with GitHub" button.
- **FR-002**: Clicking the button MUST redirect the user to GitHub's authorization page with the correct application credentials and a CSRF-protection state parameter.
- **FR-003**: Upon successful authorization, the system MUST exchange the authorization code for user information from GitHub.
- **FR-004**: The system MUST create a new user account if the GitHub username does not already exist.
- **FR-005**: The system MUST sign in existing users when their GitHub username matches a previously created account.
- **FR-006**: The system MUST store the user's session so they remain authenticated across requests.
- **FR-007**: The system MUST validate the state parameter on the callback to prevent CSRF attacks.
- **FR-008**: The system MUST show an error message when authorization is denied or fails.
- **FR-009**: The system MUST remove the old username-only login form (since GitHub becomes the sole authentication method).
- **FR-010**: The application MUST require two configuration values (client ID and client secret) provided at startup, and MUST refuse to start if they are missing.

### Key Entities

- **User**: Identified by GitHub username. Linked to notes, tags, and drawings via their user ID in the database.
- **Session**: Server-side session binding the browser cookie to the authenticated user identity.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can complete the full sign-in flow (click button to seeing notes) in under 10 seconds.
- **SC-002**: 100% of sign-in attempts with a valid GitHub account succeed on the first try.
- **SC-003**: Unauthorized access to any protected page results in a redirect to the login page within 1 second.
- **SC-004**: Denying authorization on GitHub returns the user to the login page with a visible error message.
- **SC-005**: Existing users who previously signed in with a username can sign in via GitHub and access their same notes (assuming their GitHub username matches).

## Assumptions

- The application will be registered as a GitHub OAuth App, and the client ID and secret will be provided as configuration at startup.
- GitHub usernames are unique and stable enough to serve as the user identifier.
- The existing session management infrastructure (cookie-based server-side sessions) will be reused.
- Only GitHub is supported as an identity provider; no other OAuth providers are planned for this feature.
- The old username-only login is removed entirely; users who previously logged in with a username that matches their GitHub username will seamlessly retain their data.
