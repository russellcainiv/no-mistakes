```markdown
# no-mistakes Development Patterns

> Auto-generated skill from repository analysis

## Overview
This skill teaches the core development patterns, coding conventions, and workflows used in the `no-mistakes` Go codebase. The repository emphasizes clear commit practices, modular code organization, and a strong focus on maintaining up-to-date tests and documentation. Whether you're adding features, fixing bugs, or updating documentation, this guide will help you follow established patterns for consistency and quality.

## Coding Conventions

- **File Naming:**  
  Use camelCase for Go source files.  
  _Example:_  
  ```
  internal/myFeature.go
  internal/userProfileHandler.go
  ```

- **Import Style:**  
  Use relative imports within the module.  
  _Example:_  
  ```go
  import (
      "internal/utils"
      "internal/data"
  )
  ```

- **Export Style:**  
  Use named exports for functions, types, and variables that should be accessible outside their package.  
  _Example:_  
  ```go
  // In internal/myFeature.go
  package internal

  func ExportedFunction() {
      // ...
  }
  ```

- **Commit Messages:**  
  Follow [Conventional Commits](https://www.conventionalcommits.org/) with prefixes like `fix`, `docs`, and `feat`.  
  _Example:_  
  ```
  feat: add user authentication with JWT and session fallback
  fix: resolve panic on nil pointer dereference in userProfileHandler
  docs: update README with new environment variable instructions
  ```

## Workflows

### Feature Development with Tests and Docs
**Trigger:** When implementing a new feature or major capability  
**Command:** `/new-feature`

1. Implement the new feature in relevant `internal/**/*.go` files.
2. Add or update corresponding tests in `internal/**/*_test.go`.
3. Update or create documentation in `docs/**/*.md` or `README.md` to describe the new feature.

_Example:_
```go
// internal/userProfile.go
package internal

func UpdateUserProfile(userID string, data UserProfileData) error {
    // implementation
}
```
```go
// internal/userProfile_test.go
package internal

func TestUpdateUserProfile(t *testing.T) {
    // test cases
}
```
```markdown
# docs/user-profile.md

## Update User Profile
Describes how to update a user's profile using the new API.
```

---

### Bugfix with Targeted Tests
**Trigger:** When fixing a bug or regression  
**Command:** `/bugfix`

1. Update implementation files in `internal/**/*.go` to resolve the bug.
2. Add or update tests in `internal/**/*_test.go` to cover the bug scenario.
3. Optionally, update related logic in closely associated files.

_Example:_
```go
// internal/userProfile.go
func UpdateUserProfile(userID string, data UserProfileData) error {
    if userID == "" {
        return errors.New("userID cannot be empty") // bugfix
    }
    // ...
}
```
```go
// internal/userProfile_test.go
func TestUpdateUserProfile_EmptyUserID(t *testing.T) {
    err := UpdateUserProfile("", UserProfileData{})
    if err == nil {
        t.Error("expected error for empty userID")
    }
}
```

---

### Documentation Sync with Code Changes
**Trigger:** When code changes require documentation updates  
**Command:** `/sync-docs`

1. Update or create documentation files in `docs/**/*.md`, `docs/**/*.mjs`, or `README.md` to reflect new or changed features.
2. Ensure the documentation site builds cleanly and all anchor links resolve.
3. Update provider tables, environment variable docs, and configuration references as needed.

_Example:_
```markdown
# README.md

## Environment Variables

- `NO_MISTAKES_API_KEY`: Required for authentication.
```

## Testing Patterns

- **Test File Naming:**  
  Test files follow the pattern `*_test.go` and are located alongside implementation files in the `internal/` directory.

- **Test Structure:**  
  Use Go's standard `testing` package.  
  _Example:_
  ```go
  // internal/example_test.go
  package internal

  import "testing"

  func TestExampleFeature(t *testing.T) {
      // Arrange

      // Act

      // Assert
  }
  ```

- **Test Coverage:**  
  Each new feature or bugfix should be accompanied by targeted tests that verify correct behavior and guard against regressions.

## Commands

| Command      | Purpose                                               |
|--------------|-------------------------------------------------------|
| /new-feature | Start a new feature with code, tests, and docs        |
| /bugfix      | Begin a bugfix with targeted code and test updates    |
| /sync-docs   | Update documentation to match recent code changes     |
```
